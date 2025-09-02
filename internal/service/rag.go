package service

import (
	"fmt"

	"knowledge-maker/internal/config"
	"knowledge-maker/internal/logger"
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

// queryKnowledgeWithDetailedLogging 统一的知识库查询方法，包含详细日志
func (rs *RAGService) queryKnowledgeWithDetailedLogging(query string) (string, error) {
	logger.Info("开始查询知识库，查询内容: %s", query)
	
	knowledgeContext, err := rs.knowledgeService.QueryKnowledge(query)
	if err != nil {
		logger.Error("知识库查询失败: %v", err)
		return "", err
	}
	
	// 记录详细的知识库内容
	if knowledgeContext != "" {
		logger.Info("知识库查询成功，上下文长度: %d", len(knowledgeContext))
		
		// 记录知识库内容的前200个字符作为预览
		preview := knowledgeContext
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		logger.Info("知识库内容预览: %s", preview)
		
		// 记录完整的知识库内容到日志
		logger.Info("=== 知识库完整内容开始 ===")
		logger.Info("查询: %s", query)
		logger.Info(fmt.Sprintf("限制查询最优匹配: %d", rs.config.Knowledge.TopK))
		logger.Info("=== 知识库完整内容结束 ===")
	} else {
		logger.Info("知识库查询结果为空")
	}
	
	return knowledgeContext, nil
}

// ProcessChat 处理聊天请求的核心逻辑
func (rs *RAGService) ProcessChat(query string) (*model.ChatResponse, error) {
	logger.Info("收到用户查询: %s", query)

	// 1. 查询知识库
	knowledgeContext, err := rs.queryKnowledgeWithDetailedLogging(query)
	if err != nil {
		logger.Error("知识库查询失败: %v", err)
		// 知识库查询失败时，仍然可以使用 AI 直接回答
		knowledgeContext = ""
	}

	// 2. 构建系统提示词
	systemPrompt := rs.getSystemPrompt()

	// 3. 调用 AI 生成回复
	answer, err := rs.aiService.GenerateResponse(systemPrompt, query, knowledgeContext)
	if err != nil {
		logger.Error("AI 生成回复失败: %v", err)
		return &model.ChatResponse{
			Success: false,
			Message: "AI 服务暂时不可用，请稍后重试",
		}, err
	}

	logger.Info("AI 回复生成成功，长度: %d", len(answer))

	// 4. 返回结果
	response := &model.ChatResponse{
		Success: true,
		Answer:  answer,
	}

	return response, nil
}

// ProcessStreamChat 处理流式聊天请求的核心逻辑
func (rs *RAGService) ProcessStreamChat(query string) (chan model.StreamContent, chan error, error) {
	logger.Info("收到流式查询: %s", query)

	// 1. 先同步查询知识库（因为很快，2秒内完成）
	knowledgeContext, err := rs.queryKnowledgeWithDetailedLogging(query)
	if err != nil {
		logger.Error("知识库查询失败: %v", err)
		knowledgeContext = ""
	}

	// 2. 构建系统提示词
	systemPrompt := rs.getSystemPrompt()

	// 3. 立即获取流式响应
	logger.Info("准备调用 AI 流式服务")
	stream, err := rs.aiService.GenerateStreamResponse(systemPrompt, query, knowledgeContext)
	if err != nil {
		logger.Error("AI 流式生成失败: %v", err)
		return nil, nil, fmt.Errorf("AI 服务暂时不可用，请稍后重试")
	}
	logger.Info("AI 流式服务调用成功，开始处理响应")

	// 4. 创建通道
	responseChan := make(chan model.StreamContent, 1)
	errorChan := make(chan error, 1)

	// 5. 启动协程处理流式响应
	go func() {
		defer close(responseChan)
		defer close(errorChan)
		rs.aiService.ProcessStreamResponse(stream, responseChan, errorChan, query, knowledgeContext)
	}()

	return responseChan, errorChan, nil
}

// getSystemPrompt 获取系统提示词
func (rs *RAGService) getSystemPrompt() string {
	return rs.config.RAG.SystemPrompt
}
