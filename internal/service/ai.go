package service

import (
	"context"
	"fmt"
	"io"

	"knowledge-maker/internal/config"

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
func (ai *AIService) ProcessStreamResponse(stream *openai.ChatCompletionStream, responseChan chan<- string, errorChan chan<- error) {
	defer close(responseChan)
	defer close(errorChan)
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				// 流结束
				return
			}
			errorChan <- fmt.Errorf("接收流式响应失败: %v", err)
			return
		}

		if len(response.Choices) > 0 {
			delta := response.Choices[0].Delta
			if delta.Content != "" {
				responseChan <- delta.Content
			}
		}
	}
}