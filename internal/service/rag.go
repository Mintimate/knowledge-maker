package service

import (
	"fmt"
	"log"

	"knowledge-maker/internal/config"
	"knowledge-maker/internal/model"
)

// RAGService RAG 服务，整合知识库和 AI
type RAGService struct {
	knowledgeService *KnowledgeService
	aiService        *AIService
	config           *config.Config
}

// NewRAGService 创建 RAG 服务实例
func NewRAGService(knowledgeService *KnowledgeService, aiService *AIService, cfg *config.Config) *RAGService {
	return &RAGService{
		knowledgeService: knowledgeService,
		aiService:        aiService,
		config:           cfg,
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
func (rs *RAGService) ProcessStreamChat(query string) (chan model.StreamContent, chan error, error) {
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
	responseChan := make(chan model.StreamContent, 100)
	errorChan := make(chan error, 1)

	// 5. 启动协程处理流式响应
	go rs.aiService.ProcessStreamResponse(stream, responseChan, errorChan, query, knowledgeContext)

	return responseChan, errorChan, nil
}

// getSystemPrompt 获取系统提示词
func (rs *RAGService) getSystemPrompt() string {
	return rs.config.RAG.SystemPrompt
}
