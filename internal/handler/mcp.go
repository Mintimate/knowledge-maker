package handler

import (
	"fmt"
	"net/http"

	"knowledge-maker/internal/logger"
	"knowledge-maker/internal/model"
	"knowledge-maker/internal/service"

	"github.com/gin-gonic/gin"
)

// MCPHandler MCP 处理器
type MCPHandler struct {
	mcpService *service.MCPService
}

// NewMCPHandler 创建 MCP 处理器实例
func NewMCPHandler(mcpService *service.MCPService) *MCPHandler {
	return &MCPHandler{
		mcpService: mcpService,
	}
}

// HandleListTools 处理获取工具列表请求
func (h *MCPHandler) HandleListTools(c *gin.Context) {
	tools := h.mcpService.ListTools()
	c.JSON(http.StatusOK, model.MCPToolsListResponse{
		Success: true,
		Tools:   tools,
	})
}

// HandleCallTool 处理工具调用请求
func (h *MCPHandler) HandleCallTool(c *gin.Context) {
	var req model.MCPToolCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.MCPToolCallResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	logger.Info("[MCP Handler] 工具调用请求: %s", req.ToolName)

	result, err := h.mcpService.CallTool(req.ToolName, req.Arguments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.MCPToolCallResponse{
			Success: false,
			Message: fmt.Sprintf("工具调用失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, model.MCPToolCallResponse{
		Success: true,
		Result:  result,
	})
}

// HandleLLMChat 处理 LLM 聊天请求（支持流式和非流式，支持 Function Calling）
func (h *MCPHandler) HandleLLMChat(c *gin.Context) {
	var req model.LLMChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.LLMChatResponse{
			Success: false,
			Error:   "请求参数错误: " + err.Error(),
		})
		return
	}

	logger.Info("[MCP Handler] LLM 聊天请求，流式: %v，消息数: %d", req.Stream, len(req.Messages))

	if req.Stream {
		h.handleLLMStreamChat(c, req)
	} else {
		h.handleLLMNonStreamChat(c, req)
	}
}

// handleLLMNonStreamChat 处理非流式 LLM 聊天
func (h *MCPHandler) handleLLMNonStreamChat(c *gin.Context, req model.LLMChatRequest) {
	resp, err := h.mcpService.LLMChat(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// handleLLMStreamChat 处理流式 LLM 聊天
func (h *MCPHandler) handleLLMStreamChat(c *gin.Context, req model.LLMChatRequest) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	chunkChan, errorChan, err := h.mcpService.LLMStreamChat(req)
	if err != nil {
		c.SSEvent("error", gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 发送连接确认
	c.SSEvent("connected", gin.H{
		"success": true,
		"message": "LLM 流式连接已建立",
	})
	c.Writer.Flush()

	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				c.SSEvent("done", gin.H{
					"success": true,
					"message": "回答完成",
				})
				c.Writer.Flush()
				return
			}

			if chunk.FinishReason == "stop" && chunk.Delta == nil {
				c.SSEvent("done", gin.H{
					"success":       true,
					"message":       "回答完成",
					"finish_reason": "stop",
				})
				c.Writer.Flush()
				return
			}

			if chunk.FinishReason == "tool_calls" {
				c.SSEvent("tool_calls", gin.H{
					"success":       true,
					"finish_reason": "tool_calls",
				})
				c.Writer.Flush()
				continue
			}

			c.SSEvent("data", chunk)
			c.Writer.Flush()

		case err := <-errorChan:
			if err != nil {
				logger.Error("[MCP Handler] 流式响应错误: %v", err)
				c.SSEvent("error", gin.H{
					"success": false,
					"message": fmt.Sprintf("流式响应错误: %v", err),
				})
				c.Writer.Flush()
				return
			}

		case <-c.Request.Context().Done():
			logger.Info("[MCP Handler] 客户端断开连接")
			return
		}
	}
}
