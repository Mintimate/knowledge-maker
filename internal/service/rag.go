package service

import (
	"fmt"
	"log"

	"knowledge-maker/internal/model"
)

// RAGService RAG 服务，整合知识库和 AI
type RAGService struct {
	knowledgeService *KnowledgeService
	aiService        *AIService
}

// NewRAGService 创建 RAG 服务实例
func NewRAGService(knowledgeService *KnowledgeService, aiService *AIService) *RAGService {
	return &RAGService{
		knowledgeService: knowledgeService,
		aiService:        aiService,
	}
}

// ProcessChat 处理聊天请求的核心逻辑
func (rs *RAGService) ProcessChat(query string) (*model.ChatResponse, error) {
	log.Printf("收到用户查询: %s", query)

	// 1. 查询知识库
	knowledgeContext, err := rs.knowledgeService.QueryKnowledge(query)
	if err != nil {
		log.Printf("知识库查询失败: %v", err)
		// 知识库查询失败时，仍然可以使用 AI 直接回答
		knowledgeContext = ""
	} else {
		log.Printf("知识库查询成功，获得上下文长度: %d", len(knowledgeContext))
	}

	// 2. 构建系统提示词
	systemPrompt := rs.getSystemPrompt()

	// 3. 调用 AI 生成回复
	answer, err := rs.aiService.GenerateResponse(systemPrompt, query, knowledgeContext)
	if err != nil {
		log.Printf("AI 生成回复失败: %v", err)
		return &model.ChatResponse{
			Success: false,
			Message: "AI 服务暂时不可用，请稍后重试",
		}, err
	}

	log.Printf("AI 回复生成成功，长度: %d", len(answer))

	// 4. 返回结果
	response := &model.ChatResponse{
		Success: true,
		Answer:  answer,
	}

	return response, nil
}

// ProcessStreamChat 处理流式聊天请求的核心逻辑
func (rs *RAGService) ProcessStreamChat(query string) (chan string, chan error, error) {
	log.Printf("收到流式查询: %s", query)

	// 1. 查询知识库
	knowledgeContext, err := rs.knowledgeService.QueryKnowledge(query)
	if err != nil {
		log.Printf("知识库查询失败: %v", err)
		knowledgeContext = ""
	} else {
		log.Printf("知识库查询成功，获得上下文长度: %d", len(knowledgeContext))
	}

	// 2. 构建系统提示词
	systemPrompt := rs.getSystemPrompt()

	// 3. 获取流式响应
	stream, err := rs.aiService.GenerateStreamResponse(systemPrompt, query, knowledgeContext)
	if err != nil {
		log.Printf("AI 流式生成失败: %v", err)
		return nil, nil, fmt.Errorf("AI 服务暂时不可用，请稍后重试")
	}

	// 4. 创建通道
	responseChan := make(chan string, 100)
	errorChan := make(chan error, 1)

	// 5. 启动协程处理流式响应
	go rs.aiService.ProcessStreamResponse(stream, responseChan, errorChan)

	return responseChan, errorChan, nil
}

// getSystemPrompt 获取系统提示词
func (rs *RAGService) getSystemPrompt() string {
	return `你是 AI 助手，专门检索 Rime 和薄荷输入法有关内容，拒绝其他内容（如：情感咨询、数学计算、作文写作和政治主张）。

请严格遵循以下规则：
1. 只回答与 Rime 输入法框架和薄荷输入法相关的技术问题
2. 对于非相关问题，礼貌拒绝并说明只能回答 Rime 和薄荷输入法相关问题
3. 基于提供的知识库内容进行回答，确保信息准确
4. 使用简体中文回答
5. 内容应包含适当的标题、段落、列表等标签来提升可读性

请为用户提供准确的回答。`
}
