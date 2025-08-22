package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"knowledge-maker/internal/config"
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
func (ai *AIService) ProcessStreamResponse(stream *openai.ChatCompletionStream, responseChan chan<- model.StreamContent, errorChan chan<- error) {
	defer close(responseChan)
	defer close(errorChan)
	defer stream.Close()

	// 创建日志文件
	logFile, err := os.OpenFile("stream_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("无法创建日志文件: %v", err)
	} else {
		defer logFile.Close()
	}

	logger := log.New(logFile, "", log.LstdFlags)
	logger.Printf("=== 开始新的流式响应处理 ===")

	var reasoningStarted bool
	var reasoningEnded bool
	var answerStarted bool

	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				logger.Printf("流结束 - reasoningStarted: %v, reasoningEnded: %v, answerStarted: %v", 
					reasoningStarted, reasoningEnded, answerStarted)
				
				// 流结束，如果还在思考阶段，发送结束标记
				if reasoningStarted && !reasoningEnded {
					logger.Printf("发送思考结束标记")
					responseChan <- model.StreamContent{Content: "</think>"}
					reasoningEnded = true
				}
				// 如果还没开始答案，发送答案开始标记
				if !answerStarted {
					logger.Printf("发送答案开始标记")
					responseChan <- model.StreamContent{Content: "<answer>"}
				}
				return
			}
			logger.Printf("接收流式响应失败: %v", err)
			errorChan <- fmt.Errorf("接收流式响应失败: %v", err)
			return
		}

		// 记录原始响应
		if responseBytes, err := json.Marshal(response); err == nil {
			logger.Printf("原始响应: %s", string(responseBytes))
		}

		if len(response.Choices) > 0 {
			choice := response.Choices[0]
			streamContent := model.StreamContent{}

			// 记录 choice 详情
			if choiceBytes, err := json.Marshal(choice); err == nil {
				logger.Printf("Choice详情: %s", string(choiceBytes))
			}

			// 处理思考内容 - 多种方式尝试获取 reasoning_content
			var hasReasoningContent bool
			var reasoningStr string

			// 方法1: 直接尝试访问可能的字段
			if choiceBytes, err := json.Marshal(choice); err == nil {
				var choiceMap map[string]interface{}
				if err := json.Unmarshal(choiceBytes, &choiceMap); err == nil {
					logger.Printf("Choice Map: %+v", choiceMap)
					
					if delta, exists := choiceMap["delta"]; exists {
						if deltaMap, ok := delta.(map[string]interface{}); ok {
							logger.Printf("Delta Map: %+v", deltaMap)
							
							// 尝试多种可能的字段名
							possibleFields := []string{"reasoning_content", "reasoning", "thought", "thinking"}
							for _, field := range possibleFields {
								if reasoningContent, exists := deltaMap[field]; exists {
									if str, ok := reasoningContent.(string); ok && str != "" {
										hasReasoningContent = true
										reasoningStr = str
										logger.Printf("找到 %s 字段: %s", field, str)
										break
									}
								}
							}
						}
					}
				}
			}

			// 如果找到了 reasoning_content
			if hasReasoningContent {
				// 第一次收到 reasoning_content 时发送开始标记
				if !reasoningStarted {
					logger.Printf("第一次收到思考内容，发送开始标记")
					responseChan <- model.StreamContent{Content: "<think>"}
					reasoningStarted = true
				}
				
				// 发送 reasoning_content 内容
				streamContent.ReasoningContent = reasoningStr
				logger.Printf("发送思考内容: %s", reasoningStr)
				
				// 立即发送思考内容
				responseChan <- streamContent
			}

			// 处理普通内容
			if choice.Delta.Content != "" {
				logger.Printf("收到普通内容: %s", choice.Delta.Content)
				
				// 如果之前有 reasoning_content 但现在开始有普通内容，说明思考结束
				if reasoningStarted && !reasoningEnded && !hasReasoningContent {
					logger.Printf("思考阶段结束，发送结束标记")
					responseChan <- model.StreamContent{Content: "</think>"}
					reasoningEnded = true
				}
				
				// 如果思考已结束且还没开始答案，发送答案开始标记
				if reasoningEnded && !answerStarted {
					logger.Printf("开始答案阶段")
					responseChan <- model.StreamContent{Content: "<answer>"}
					answerStarted = true
				} else if !reasoningStarted && !answerStarted {
					// 如果没有思考阶段，直接开始答案
					logger.Printf("没有思考阶段，直接开始答案")
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
