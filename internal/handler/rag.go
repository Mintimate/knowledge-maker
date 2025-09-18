package handler

import (
	"fmt"
	"net/http"

	"knowledge-maker/internal/logger"
	"knowledge-maker/internal/model"
	"knowledge-maker/internal/service"

	"github.com/gin-gonic/gin"
)

// RAGHandler RAG 处理器
type RAGHandler struct {
	ragService     *service.RAGService
	captchaService *service.CaptchaService
}

// NewRAGHandler 创建 RAG 处理器实例
func NewRAGHandler(ragService *service.RAGService, captchaService *service.CaptchaService) *RAGHandler {
	return &RAGHandler{
		ragService:     ragService,
		captchaService: captchaService,
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

	// 验证码验证
	if err := h.verifyCaptcha(req, c.ClientIP()); err != nil {
		c.JSON(http.StatusBadRequest, model.ChatResponse{
			Success: false,
			Message: err.Error(),
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

	// 验证码验证
	if err := h.verifyCaptcha(req, c.ClientIP()); err != nil {
		c.JSON(http.StatusBadRequest, model.ChatResponse{
			Success: false,
			Message: err.Error(),
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

	// 立即发送连接建立确认
	c.SSEvent("connected", gin.H{
		"success": true,
		"message": "连接已建立，开始处理...",
	})
	c.Writer.Flush()

	// 发送流式数据 - 简化版本，直接转发 ai.go 中的标记和内容
	for {
		select {
		case streamContent, ok := <-responseChan:
			if !ok {
				// 流结束
				c.SSEvent("done", gin.H{
					"success": true,
					"message": "回答完成",
				})
				return
			}

			// 检查是否包含思考内容
			if streamContent.ReasoningContent != "" {
				// 发送思考内容
				c.SSEvent("data", gin.H{
					"content": streamContent.ReasoningContent,
				})
				c.Writer.Flush()
			}

			// 检查是否有回答内容（包括标记）
			if streamContent.Content != "" {
				// 发送回答内容（包括标记）
				c.SSEvent("data", gin.H{
					"content": streamContent.Content,
				})
				c.Writer.Flush()
			}

		case err := <-errorChan:
			if err != nil {
				logger.Error("流式响应错误: %v", err)
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

// verifyCaptcha 验证验证码
func (h *RAGHandler) verifyCaptcha(req model.ChatRequest, clientIP string) error {
	// 如果验证码服务未启用，跳过验证
	if h.captchaService == nil || !h.captchaService.IsEnabled() {
		logger.Info("验证码服务未启用，跳过验证")
		return nil
	}

	// 根据验证码类型进行不同的验证
	captchaType := h.captchaService.GetCaptchaType()

	switch captchaType {
	case "tencent":
		return h.verifyTencentCaptcha(req, clientIP)
	case "geetest":
		return h.verifyGeetestCaptcha(req)
	case "google_v2", "google_v3":
		return h.verifyGoogleRecaptcha(req, clientIP)
	case "cloudflare":
		return h.verifyCloudflareTurnstile(req, clientIP)
	default:
		return fmt.Errorf("不支持的验证码类型: %s", captchaType)
	}
}

// verifyTencentCaptcha 验证腾讯云验证码
func (h *RAGHandler) verifyTencentCaptcha(req model.ChatRequest, clientIP string) error {
	// 如果请求中没有腾讯云验证码信息，要求提供验证码
	if req.CaptchaTicket == "" || req.CaptchaRandstr == "" {
		return fmt.Errorf("请完成腾讯云验证码验证")
	}

	// 验证验证码
	isValid, err := h.captchaService.VerifyCaptcha(req.CaptchaTicket, req.CaptchaRandstr, clientIP)
	if err != nil {
		logger.Error("腾讯云验证码验证失败: %v", err)
		return fmt.Errorf("验证码验证失败: %v", err)
	}

	if !isValid {
		return fmt.Errorf("验证码验证失败，请重新验证")
	}
	return nil
}

// verifyGeetestCaptcha 验证极验验证码
func (h *RAGHandler) verifyGeetestCaptcha(req model.ChatRequest) error {
	// 如果请求中没有极验验证码信息，要求提供验证码
	if req.LotNumber == "" || req.CaptchaOutput == "" || req.PassToken == "" || req.GenTime == "" {
		return fmt.Errorf("请完成极验验证码验证")
	}

	// 验证验证码
	isValid, err := h.captchaService.VerifyGeetestCaptcha(req.LotNumber, req.CaptchaOutput, req.PassToken, req.GenTime)
	if err != nil {
		logger.Error("极验验证码验证失败: %v", err)
		return fmt.Errorf("验证码验证失败: %v", err)
	}

	if !isValid {
		return fmt.Errorf("验证码验证失败，请重新验证")
	}

	logger.Info("极验验证码验证成功")
	return nil
}

// verifyGoogleRecaptcha 验证 Google reCAPTCHA
func (h *RAGHandler) verifyGoogleRecaptcha(req model.ChatRequest, clientIP string) error {
	// 如果请求中没有 Google reCAPTCHA token，要求提供验证码
	if req.RecaptchaToken == "" {
		return fmt.Errorf("请完成 Google reCAPTCHA 验证")
	}

	// 验证验证码
	isValid, err := h.captchaService.VerifyGoogleRecaptcha(req.RecaptchaToken, req.RecaptchaAction, clientIP)
	if err != nil {
		logger.Error("Google reCAPTCHA 验证失败: %v", err)
		return fmt.Errorf("验证码验证失败: %v", err)
	}

	if !isValid {
		return fmt.Errorf("验证码验证失败，请重新验证")
	}

	logger.Info("Google reCAPTCHA 验证成功")
	return nil
}

// verifyCloudflareTurnstile 验证 Cloudflare Turnstile 验证码
func (h *RAGHandler) verifyCloudflareTurnstile(req model.ChatRequest, clientIP string) error {
	// 如果请求中没有 Cloudflare Turnstile token，要求提供验证码
	if req.CFToken == "" {
		return fmt.Errorf("请完成 Cloudflare Turnstile 验证")
	}

	// 验证验证码
	isValid, err := h.captchaService.VerifyCloudflareTurnstile(req.CFToken, clientIP)
	if err != nil {
		logger.Error("Cloudflare Turnstile 验证失败: %v", err)
		return fmt.Errorf("验证码验证失败: %v", err)
	}

	if !isValid {
		return fmt.Errorf("验证码验证失败，请重新验证")
	}

	logger.Info("Cloudflare Turnstile 验证成功")
	return nil
}
