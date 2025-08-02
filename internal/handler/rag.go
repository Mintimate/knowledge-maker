package handler

import (
	"fmt"
	"net/http"

	"knowledge-maker/internal/model"
	"knowledge-maker/internal/service"

	"github.com/gin-gonic/gin"
)

// RAGHandler RAG 处理器
type RAGHandler struct {
	ragService *service.RAGService
}

// NewRAGHandler 创建 RAG 处理器实例
func NewRAGHandler(ragService *service.RAGService) *RAGHandler {
	return &RAGHandler{
		ragService: ragService,
	}
}

// HandleChat 处理聊天请求
func (h *RAGHandler) HandleChat(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ChatResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 调用服务层处理请求
	response, err := h.ragService.ProcessChat(req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, *response)
		return
	}

	// 如果需要调试，可以包含知识库上下文
	if gin.Mode() == gin.DebugMode {
		// 这里可以添加调试信息，但需要从服务层获取
	}

	c.JSON(http.StatusOK, *response)
}

// HandleStreamChat 处理流式聊天请求
func (h *RAGHandler) HandleStreamChat(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ChatResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 调用服务层处理流式请求
	responseChan, errorChan, err := h.ragService.ProcessStreamChat(req.Query)
	if err != nil {
		c.SSEvent("error", gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 发送流式数据
	for {
		select {
		case content, ok := <-responseChan:
			if !ok {
				// 流结束
				c.SSEvent("done", gin.H{
					"success": true,
					"message": "回答完成",
				})
				return
			}
			// 发送内容片段
			c.SSEvent("data", gin.H{
				"content": content,
			})
			c.Writer.Flush()

		case err := <-errorChan:
			if err != nil {
				c.SSEvent("error", gin.H{
					"success": false,
					"message": fmt.Sprintf("流式响应错误: %v", err),
				})
				return
			}

		case <-c.Request.Context().Done():
			// 客户端断开连接
			return
		}
	}
}