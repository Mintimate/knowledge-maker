package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	"knowledge-maker/internal/logger"
	"knowledge-maker/internal/model"
	"knowledge-maker/internal/service"

	"github.com/gin-gonic/gin"
)

// CaptchaMiddleware 验证码中间件
type CaptchaMiddleware struct {
	captchaService *service.CaptchaService
}

// NewCaptchaMiddleware 创建验证码中间件实例
func NewCaptchaMiddleware(captchaService *service.CaptchaService) *CaptchaMiddleware {
	return &CaptchaMiddleware{
		captchaService: captchaService,
	}
}

// VerifyCaptcha 验证码验证中间件
func (m *CaptchaMiddleware) VerifyCaptcha() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果验证码服务未启用，跳过验证
		if m.captchaService == nil || !m.captchaService.IsEnabled() {
			logger.Info("验证码服务未启用，跳过验证")
			c.Next()
			return
		}

		// 验证验证码
		if err := m.verifyCaptcha(c, c.ClientIP()); err != nil {
			c.JSON(http.StatusBadRequest, model.ChatResponse{
				Success: false,
				Message: err.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// verifyCaptcha 验证验证码
func (m *CaptchaMiddleware) verifyCaptcha(c *gin.Context, clientIP string) error {
	// 根据验证码类型进行不同的验证
	captchaType := m.captchaService.GetCaptchaType()

	switch captchaType {
	case "tencent":
		return m.verifyTencentCaptcha(c, clientIP)
	case "geetest":
		return m.verifyGeetestCaptcha(c)
	case "google_v2", "google_v3":
		return m.verifyGoogleRecaptcha(c, clientIP)
	case "cloudflare":
		return m.verifyCloudflareTurnstile(c, clientIP)
	case "aliyun":
		return m.verifyAliyunCaptcha(c)
	default:
		return fmt.Errorf("不支持的验证码类型: %s", captchaType)
	}
}

// verifyTencentCaptcha 验证腾讯云验证码
func (m *CaptchaMiddleware) verifyTencentCaptcha(c *gin.Context, clientIP string) error {
	// 从 Header 中获取腾讯云验证码信息
	captchaTicket := c.GetHeader("X-Captcha-Ticket")
	captchaRandstr := c.GetHeader("X-Captcha-Randstr")

	// 如果请求中没有腾讯云验证码信息，要求提供验证码
	if captchaTicket == "" || captchaRandstr == "" {
		return fmt.Errorf("请完成腾讯云验证码验证")
	}

	// 验证验证码
	isValid, err := m.captchaService.VerifyCaptcha(captchaTicket, captchaRandstr, clientIP)
	if err != nil {
		logger.Error("腾讯云验证码验证失败: %v", err)
		return fmt.Errorf("验证码验证失败: %v", err)
	}

	if !isValid {
		return fmt.Errorf("验证码验证失败，请重新验证")
	}

	logger.Info("腾讯云验证码验证成功")
	return nil
}

// verifyGeetestCaptcha 验证极验验证码
func (m *CaptchaMiddleware) verifyGeetestCaptcha(c *gin.Context) error {
	// 从 Header 中获取极验验证码信息
	lotNumber := c.GetHeader("X-Geetest-Lot-Number")
	captchaOutput := c.GetHeader("X-Geetest-Captcha-Output")
	passToken := c.GetHeader("X-Geetest-Pass-Token")
	genTime := c.GetHeader("X-Geetest-Gen-Time")

	// 如果请求中没有极验验证码信息，要求提供验证码
	if lotNumber == "" || captchaOutput == "" || passToken == "" || genTime == "" {
		return fmt.Errorf("请完成极验验证码验证")
	}

	// 验证验证码
	isValid, err := m.captchaService.VerifyGeetestCaptcha(lotNumber, captchaOutput, passToken, genTime)
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
func (m *CaptchaMiddleware) verifyGoogleRecaptcha(c *gin.Context, clientIP string) error {
	// 从 Header 中获取 Google reCAPTCHA 信息
	recaptchaToken := c.GetHeader("X-Recaptcha-Token")
	recaptchaAction := c.GetHeader("X-Recaptcha-Action")

	// 如果请求中没有 Google reCAPTCHA token，要求提供验证码
	if recaptchaToken == "" {
		return fmt.Errorf("请完成 Google reCAPTCHA 验证")
	}

	// 验证验证码
	isValid, err := m.captchaService.VerifyGoogleRecaptcha(recaptchaToken, recaptchaAction, clientIP)
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
func (m *CaptchaMiddleware) verifyCloudflareTurnstile(c *gin.Context, clientIP string) error {
	// 从 Header 中获取 Cloudflare Turnstile token
	cfToken := c.GetHeader("X-Cf-Turnstile-Token")

	// 如果请求中没有 Cloudflare Turnstile token，要求提供验证码
	if cfToken == "" {
		return fmt.Errorf("请完成 Cloudflare Turnstile 验证")
	}

	// 验证验证码
	isValid, err := m.captchaService.VerifyCloudflareTurnstile(cfToken, clientIP)
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

// verifyAliyunCaptcha 验证阿里云验证码
func (m *CaptchaMiddleware) verifyAliyunCaptcha(c *gin.Context) error {
	// 从 Header 中获取验证码信息（兼容前端格式）
	captchaTicket := c.GetHeader("X-Captcha-Ticket")
	captchaRandstr := c.GetHeader("X-Captcha-Randstr")

	// 如果没有标准格式，尝试阿里云专用格式
	if captchaTicket == "" {
		captchaTicket = c.GetHeader("X-Aliyun-Captcha-Param")
		captchaRandstr = c.GetHeader("X-Aliyun-Scene")
	}

	// 如果请求中没有阿里云验证码信息，要求提供验证码
	if captchaTicket == "" {
		return fmt.Errorf("请完成阿里云验证码验证")
	}

	// 解析阿里云验证码数据
	var captchaParam string
	var scene string
	var appId string

	// 阿里云验证码要求CaptchaVerifyParam必须是前端传来的完整JSON字符串，不能做任何修改
	captchaParam = captchaTicket

	// 尝试从JSON中提取sceneId用于场景标识
	if captchaTicket[0] == '{' {
		var ticketData map[string]interface{}
		if err := json.Unmarshal([]byte(captchaTicket), &ticketData); err == nil {
			if sceneId, ok := ticketData["sceneId"].(string); ok {
				scene = sceneId
			}
		}
	}

	// 如果没有场景ID，使用randstr或默认值
	if scene == "" {
		if captchaRandstr != "" && captchaRandstr != "default" {
			scene = captchaRandstr
		} else {
			scene = "default"
		}
	}

	logger.Info("阿里云验证码参数: captchaParam=%s, scene=%s", captchaParam, scene)

	// 验证验证码
	isValid, err := m.captchaService.VerifyAliyunCaptcha(captchaParam, scene, appId)
	if err != nil {
		logger.Error("阿里云验证码验证失败: %v", err)
		return fmt.Errorf("验证码验证失败: %v", err)
	}

	if !isValid {
		return fmt.Errorf("验证码验证失败，请重新验证")
	}

	logger.Info("阿里云验证码验证成功")
	return nil
}
