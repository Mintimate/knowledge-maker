package service

import (
	"context"
	"fmt"
	"io"

	"knowledge-maker/internal/config"
	"knowledge-maker/internal/logger"
	"knowledge-maker/internal/model"

	"github.com/sashabaranov/go-openai"
)

// AIService AI 服务
type AIService struct {
	client *openai.Client
	model  string
}

// NewAIService 创建 AI 服务实例
func NewAIService(cfg *config.Config) *AIService {
	openaiConfig := openai.DefaultConfig(cfg.AI.APIKey)
	openaiConfig.BaseURL = cfg.AI.BaseURL

	client := openai.NewClientWithConfig(openaiConfig)

	return &AIService{
		client: client,
		model:  cfg.AI.Model,
	}
}

// GenerateResponse 生成 AI 回复
func (ai *AIService) GenerateResponse(systemPrompt, userQuery, knowledgeContext string) (string, error) {
	// 构建消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	// 如果有知识库上下文，添加到消息中
	if knowledgeContext != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("参考知识库内容：\n%s\n\n用户问题：%s", knowledgeContext, userQuery),
		})
	} else {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: userQuery,
		})
	}

	// 创建聊天完成请求
	req := openai.ChatCompletionRequest{
		Model:       ai.model,
		Messages:    messages,
		MaxTokens:   2000,
		Temperature: 0.7,
	}

	// 调用 AI API
	resp, err := ai.client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("AI 生成回复失败: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("AI 未返回任何回复")
	}

	return resp.Choices[0].Message.Content, nil
}

// GenerateStreamResponse 生成流式 AI 回复
func (ai *AIService) GenerateStreamResponse(systemPrompt, userQuery, knowledgeContext string) (*openai.ChatCompletionStream, error) {
	// 构建消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	// 如果有知识库上下文，添加到消息中
	if knowledgeContext != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("参考知识库内容：\n%s\n\n用户问题：%s", knowledgeContext, userQuery),
		})
	} else {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: userQuery,
		})
	}

	// 创建流式聊天完成请求
	req := openai.ChatCompletionRequest{
		Model:       ai.model,
		Messages:    messages,
		MaxTokens:   2000,
		Temperature: 0.7,
		Stream:      true, // 启用流式输出
	}

	// 调用流式 AI API
	stream, err := ai.client.CreateChatCompletionStream(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("AI 流式生成回复失败: %v", err)
	}

	return stream, nil
}

// ProcessStreamResponse 处理流式响应并通过通道发送
func (ai *AIService) ProcessStreamResponse(stream *openai.ChatCompletionStream, responseChan chan<- model.StreamContent, errorChan chan<- error, userQuery, knowledgeContext string) {
	defer close(responseChan)
	defer close(errorChan)
	defer stream.Close()

	// 使用统一日志系统记录流式处理信息
	logger.Info("ProcessStreamResponse 开始处理")
	logger.Info("用户问题: %s", userQuery)
	
	if knowledgeContext != "" {
		logger.Info("知识库上下文长度: %d", len(knowledgeContext))
	} else {
		logger.Info("知识库上下文长度: 0")
	}

	var reasoningStarted bool
	var reasoningEnded bool
	var answerStarted bool
	var hasReasoningContent bool

	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				// 记录是否有思考内容和回答结束
				logger.Info("是否有思考内容: %v", hasReasoningContent)
				logger.Info("内容回答结束")
				
				// 流结束，如果还在思考阶段，发送结束标记
				if reasoningStarted && !reasoningEnded {
					responseChan <- model.StreamContent{Content: "</think>"}
					reasoningEnded = true
				}
				// 如果还没开始答案，发送答案开始标记
				if !answerStarted {
					responseChan <- model.StreamContent{Content: "<answer>"}
				}
				return
			}
			logger.Error("接收流式响应失败: %v", err)
			errorChan <- fmt.Errorf("接收流式响应失败: %v", err)
			return
		}

		if len(response.Choices) > 0 {
			choice := response.Choices[0]
			streamContent := model.StreamContent{}

			// 使用新版本 go-openai 的 reasoning_content 字段
			if choice.Delta.ReasoningContent != "" {
				logger.Debug("收到思考内容: %s", choice.Delta.ReasoningContent[:min(50, len(choice.Delta.ReasoningContent))])
				
				// 第一次收到 reasoning_content 时发送开始标记
				if !reasoningStarted {
					logger.Info("发送思考开始标记")
					responseChan <- model.StreamContent{Content: "<think>"}
					reasoningStarted = true
				}
				
				// 标记找到了思考内容
				hasReasoningContent = true
				
				// 发送 reasoning_content 内容
				streamContent.ReasoningContent = choice.Delta.ReasoningContent
				
				// 立即发送思考内容
				responseChan <- streamContent
			}

			// 处理普通内容
			if choice.Delta.Content != "" {
				logger.Debug("收到普通内容: %s", choice.Delta.Content[:min(20, len(choice.Delta.Content))])
				
				// 如果之前有 reasoning_content 但现在开始有普通内容，说明思考结束
				if reasoningStarted && !reasoningEnded && choice.Delta.ReasoningContent == "" {
					logger.Info("思考阶段结束，发送结束标记")
					responseChan <- model.StreamContent{Content: "</think>"}
					reasoningEnded = true
				}
				
				// 如果思考已结束且还没开始答案，发送答案开始标记
				if reasoningEnded && !answerStarted {
					logger.Info("内容回答开始")
					responseChan <- model.StreamContent{Content: "<answer>"}
					answerStarted = true
				} else if !reasoningStarted && !answerStarted {
					// 如果没有思考阶段，直接开始答案
					logger.Info("内容回答开始（无思考阶段）")
					responseChan <- model.StreamContent{Content: "<answer>"}
					answerStarted = true
				}
				
				streamContent.Content = choice.Delta.Content
				
				// 发送普通内容
				responseChan <- streamContent
			}
		}
	}
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}