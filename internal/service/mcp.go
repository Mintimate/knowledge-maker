package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"knowledge-maker/internal/config"
	"knowledge-maker/internal/logger"
	"knowledge-maker/internal/model"

	"github.com/sashabaranov/go-openai"
)

// MCPService MCP 服务，提供知识库工具和 LLM 聊天接口
type MCPService struct {
	knowledgeService *KnowledgeService
	aiService        *AIService
	config           *config.Config
}

// NewMCPService 创建 MCP 服务实例
func NewMCPService(knowledgeService *KnowledgeService, aiService *AIService, cfg *config.Config) *MCPService {
	return &MCPService{
		knowledgeService: knowledgeService,
		aiService:        aiService,
		config:           cfg,
	}
}

// ListTools 返回可用的 MCP 工具列表
func (ms *MCPService) ListTools() []model.MCPTool {
	return []model.MCPTool{
		{
			Name:        "query_knowledge_base",
			Description: "查询知识库，根据用户的问题在知识库中检索相关内容。适用于需要查找 Rime 输入法和薄荷输入法相关技术资料的场景。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "需要在知识库中查询的问题或关键词",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// CallTool 调用指定的 MCP 工具
func (ms *MCPService) CallTool(toolName string, arguments map[string]interface{}) (interface{}, error) {
	switch toolName {
	case "query_knowledge_base":
		return ms.callQueryKnowledgeBase(arguments)
	default:
		return nil, fmt.Errorf("未知的工具: %s", toolName)
	}
}

// callQueryKnowledgeBase 调用知识库查询工具
func (ms *MCPService) callQueryKnowledgeBase(arguments map[string]interface{}) (interface{}, error) {
	query, ok := arguments["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("缺少必要参数: query")
	}

	logger.Info("[MCP] 知识库查询工具被调用，查询: %s", query)

	result, err := ms.knowledgeService.QueryKnowledge(query)
	if err != nil {
		logger.Error("[MCP] 知识库查询失败: %v", err)
		return nil, fmt.Errorf("知识库查询失败: %v", err)
	}

	if result == "" {
		logger.Info("[MCP] 知识库查询结果为空")
		return map[string]interface{}{
			"found":   false,
			"content": "未找到相关知识库内容",
		}, nil
	}

	// 记录查询结果预览
	preview := result
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	logger.Info("[MCP] 知识库查询成功，结果预览: %s", preview)

	return map[string]interface{}{
		"found":   true,
		"content": result,
	}, nil
}

// LLMChat LLM 非流式聊天（支持 Function Calling）
func (ms *MCPService) LLMChat(req model.LLMChatRequest) (*model.LLMChatResponse, error) {
	logger.Info("[MCP] LLM 非流式聊天请求，消息数: %d，工具数: %d", len(req.Messages), len(req.Tools))

	// 构建 OpenAI 消息
	messages := ms.buildOpenAIMessages(req.Messages)

	// 构建请求
	chatReq := openai.ChatCompletionRequest{
		Model:       ms.aiService.model,
		Messages:    messages,
		MaxTokens:   2000,
		Temperature: 0.7,
	}

	// 如果有工具定义，添加到请求中
	if len(req.Tools) > 0 {
		chatReq.Tools = ms.buildOpenAITools(req.Tools)
	}

	// 调用 AI API
	resp, err := ms.aiService.client.CreateChatCompletion(context.Background(), chatReq)
	if err != nil {
		logger.Error("[MCP] LLM 聊天失败: %v", err)
		return &model.LLMChatResponse{
			Success: false,
			Error:   fmt.Sprintf("LLM 调用失败: %v", err),
		}, err
	}

	if len(resp.Choices) == 0 {
		return &model.LLMChatResponse{
			Success: false,
			Error:   "LLM 未返回任何回复",
		}, fmt.Errorf("LLM 未返回任何回复")
	}

	choice := resp.Choices[0]
	responseMsg := &model.LLMChatMessage{
		Role:    string(choice.Message.Role),
		Content: choice.Message.Content,
	}

	// 处理工具调用
	if len(choice.Message.ToolCalls) > 0 {
		var toolCalls []model.LLMToolCall
		for _, tc := range choice.Message.ToolCalls {
			toolCalls = append(toolCalls, model.LLMToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: model.LLMFunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		responseMsg.ToolCalls = toolCalls
		logger.Info("[MCP] LLM 请求工具调用，工具数: %d", len(toolCalls))
	}

	return &model.LLMChatResponse{
		Success: true,
		Message: responseMsg,
	}, nil
}

// LLMStreamChat LLM 流式聊天（支持 Function Calling）
func (ms *MCPService) LLMStreamChat(req model.LLMChatRequest) (chan model.LLMStreamChunk, chan error, error) {
	logger.Info("[MCP] LLM 流式聊天请求，消息数: %d，工具数: %d", len(req.Messages), len(req.Tools))

	// 构建 OpenAI 消息
	messages := ms.buildOpenAIMessages(req.Messages)

	// 构建请求
	chatReq := openai.ChatCompletionRequest{
		Model:       ms.aiService.model,
		Messages:    messages,
		MaxTokens:   2000,
		Temperature: 0.7,
		Stream:      true,
	}

	// 如果有工具定义，添加到请求中
	if len(req.Tools) > 0 {
		chatReq.Tools = ms.buildOpenAITools(req.Tools)
	}

	// 调用流式 AI API
	stream, err := ms.aiService.client.CreateChatCompletionStream(context.Background(), chatReq)
	if err != nil {
		logger.Error("[MCP] LLM 流式聊天失败: %v", err)
		return nil, nil, fmt.Errorf("LLM 流式调用失败: %v", err)
	}

	chunkChan := make(chan model.LLMStreamChunk, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errorChan)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					// 发送结束信号
					chunkChan <- model.LLMStreamChunk{
						FinishReason: "stop",
					}
					return
				}
				logger.Error("[MCP] 流式接收失败: %v", err)
				errorChan <- fmt.Errorf("流式接收失败: %v", err)
				return
			}

			if len(response.Choices) > 0 {
				choice := response.Choices[0]
				chunk := model.LLMStreamChunk{}

				delta := &model.LLMChatMessageDelta{}
				hasContent := false

				// 处理角色
				if choice.Delta.Role != "" {
					delta.Role = string(choice.Delta.Role)
					hasContent = true
				}

				// 处理普通内容
				if choice.Delta.Content != "" {
					delta.Content = choice.Delta.Content
					hasContent = true
				}

				// 处理思考内容（reasoning_content）
				if choice.Delta.ReasoningContent != "" {
					logger.Debug("[MCP] 收到思考内容: %s", choice.Delta.ReasoningContent[:min(50, len(choice.Delta.ReasoningContent))])
					delta.ReasoningContent = choice.Delta.ReasoningContent
					hasContent = true
				}

				// 处理工具调用
				if len(choice.Delta.ToolCalls) > 0 {
					var toolCallDeltas []model.LLMToolCallDelta
					for _, tc := range choice.Delta.ToolCalls {
						tcd := model.LLMToolCallDelta{
							Index: *tc.Index,
						}
						if tc.ID != "" {
							tcd.ID = tc.ID
						}
						if tc.Type != "" {
							tcd.Type = string(tc.Type)
						}
						if tc.Function.Name != "" || tc.Function.Arguments != "" {
							tcd.Function = &model.LLMFunctionCallDelta{
								Name:      tc.Function.Name,
								Arguments: tc.Function.Arguments,
							}
						}
						toolCallDeltas = append(toolCallDeltas, tcd)
					}
					delta.ToolCalls = toolCallDeltas
					hasContent = true
				}

				// 处理 finish_reason
				if choice.FinishReason != "" {
					chunk.FinishReason = string(choice.FinishReason)
					hasContent = true
				}

				if hasContent {
					chunk.Delta = delta
					chunkChan <- chunk
				}
			}
		}
	}()

	return chunkChan, errorChan, nil
}

// buildOpenAIMessages 将 LLMChatMessage 转换为 OpenAI 消息格式
func (ms *MCPService) buildOpenAIMessages(messages []model.LLMChatMessage) []openai.ChatCompletionMessage {
	var openaiMessages []openai.ChatCompletionMessage

	// 如果第一条不是 system 消息，自动添加系统提示词
	hasSystemPrompt := false
	for _, msg := range messages {
		if msg.Role == "system" {
			hasSystemPrompt = true
			break
		}
	}

	if !hasSystemPrompt && ms.config.RAG.SystemPrompt != "" {
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: ms.config.RAG.SystemPrompt,
		})
	}

	for _, msg := range messages {
		openaiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// 处理工具调用（assistant 消息）
		if len(msg.ToolCalls) > 0 {
			var toolCalls []openai.ToolCall
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolType(tc.Type),
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
			openaiMsg.ToolCalls = toolCalls
		}

		// 处理工具结果消息
		if msg.Role == "tool" {
			openaiMsg.ToolCallID = msg.ToolCallID
			openaiMsg.Name = msg.Name
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	return openaiMessages
}

// buildOpenAITools 将 LLMToolDef 转换为 OpenAI 工具格式
func (ms *MCPService) buildOpenAITools(tools []model.LLMToolDef) []openai.Tool {
	var openaiTools []openai.Tool

	for _, tool := range tools {
		// 将 Parameters map 转换为 JSON 再解析为 jsonschema.Definition
		paramBytes, err := json.Marshal(tool.Function.Parameters)
		if err != nil {
			logger.Warn("[MCP] 工具参数序列化失败: %v", err)
			continue
		}

		var paramDef json.RawMessage = paramBytes

		openaiTools = append(openaiTools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  paramDef,
			},
		})
	}

	return openaiTools
}
