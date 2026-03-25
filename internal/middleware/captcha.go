package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"knowledge-maker/internal/logger"
	"knowledge-maker/internal/model"
	"knowledge-maker/internal/service"

	"github.com/gin-gonic/gin"
)

// sessionTokenSecret 用于签名 session token 的密钥（运行时随机生成更安全，这里使用固定密钥简化实现）
// 每次服务重启后旧 token 自动失效
var sessionTokenSecret = generateSessionSecret()

// sessionTokenExpiry session token 有效期（5 分钟，足够完成一轮 MCP Function Calling 交互）
const sessionTokenExpiry = 5 * time.Minute

// generateSessionSecret 生成 session token 签名密钥
func generateSessionSecret() string {
	// 使用启动时间作为种子，确保每次重启后密钥不同
	h := hmac.New(sha256.New, []byte("knowledge-maker-captcha-session"))
	h.Write([]byte(time.Now().String()))
	return hex.EncodeToString(h.Sum(nil))
}

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

		// 优先检查 session token（用于 MCP 多轮调用场景，避免验证码重复弹出）
		sessionToken := c.GetHeader("X-Session-Token")
		if sessionToken != "" {
			if m.verifySessionToken(sessionToken, c.ClientIP()) {
				logger.Info("Session token 验证通过，跳过验证码验证")
				c.Next()
				return
			}
			// session token 无效，继续走验证码验证流程
			logger.Info("Session token 无效或已过期，继续验证码验证")
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

		// 验证码通过后，生成 session token 返回给前端
		// 前端后续请求（如 MCP Function Calling 多轮调用）可携带此 token 跳过验证码
		token := m.generateSessionToken(c.ClientIP())
		c.Header("X-Session-Token", token)

		c.Next()
	}
}

// generateSessionToken 生成带签名的 session token
// 格式: {clientIP}|{expireTimestamp}|{signature}
func (m *CaptchaMiddleware) generateSessionToken(clientIP string) string {
	expireAt := time.Now().Add(sessionTokenExpiry).Unix()
	payload := fmt.Sprintf("%s|%d", clientIP, expireAt)

	// 使用 HMAC-SHA256 签名
	h := hmac.New(sha256.New, []byte(sessionTokenSecret))
	h.Write([]byte(payload))
	signature := hex.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("%s|%s", payload, signature)
}

// verifySessionToken 验证 session token 的有效性
func (m *CaptchaMiddleware) verifySessionToken(token string, clientIP string) bool {
	parts := strings.SplitN(token, "|", 3)
	if len(parts) != 3 {
		return false
	}

	tokenIP := parts[0]
	expireStr := parts[1]
	signature := parts[2]

	// 验证 IP 是否匹配
	if tokenIP != clientIP {
		logger.Warn("Session token IP 不匹配: token=%s, client=%s", tokenIP, clientIP)
		return false
	}

	// 验证是否过期
	expireAt, err := strconv.ParseInt(expireStr, 10, 64)
	if err != nil {
		return false
	}
	if time.Now().Unix() > expireAt {
		logger.Info("Session token 已过期")
		return false
	}

	// 验证签名
	payload := fmt.Sprintf("%s|%s", tokenIP, expireStr)
	h := hmac.New(sha256.New, []byte(sessionTokenSecret))
	h.Write([]byte(payload))
	expectedSig := hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		logger.Warn("Session token 签名验证失败")
		return false
	}

	return true
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
