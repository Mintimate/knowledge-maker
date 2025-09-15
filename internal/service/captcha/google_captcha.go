package captcha

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"knowledge-maker/internal/config"
	"knowledge-maker/internal/logger"
)

// GoogleCaptchaProvider Google reCAPTCHA 提供者
type GoogleCaptchaProvider struct {
	config *config.CaptchaConfig
}

// NewGoogleCaptchaProvider 创建 Google reCAPTCHA 提供者
func NewGoogleCaptchaProvider(cfg *config.CaptchaConfig) (*GoogleCaptchaProvider, error) {
	if cfg.GoogleRecaptchaKey == "" {
		return nil, fmt.Errorf("Google reCAPTCHA 验证码服务配置不完整：缺少 GoogleRecaptchaKey")
	}

	return &GoogleCaptchaProvider{
		config: cfg,
	}, nil
}

// Verify 验证 Google reCAPTCHA v2/v3
func (p *GoogleCaptchaProvider) Verify(params map[string]string) (bool, error) {
	token := params["token"]
	action := params["action"]
	userIP := params["userIP"]

	if token == "" {
		return false, fmt.Errorf("Google reCAPTCHA token 不能为空")
	}

	// 如果没有提供用户IP，尝试获取本地IP
	if userIP == "" {
		userIP = getLocalIP()
	}

	// 构建请求参数
	formData := url.Values{}
	formData.Set("secret", p.config.GoogleRecaptchaKey)
	formData.Set("response", token)
	if userIP != "" {
		formData.Set("remoteip", userIP)
	}

	// Google reCAPTCHA 验证接口
	verifyURL := "https://www.google.com/recaptcha/api/siteverify"

	// 创建HTTP客户端，设置10秒超时
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 发起POST请求
	resp, err := client.PostForm(verifyURL, formData)
	if err != nil {
		logger.Error("Google reCAPTCHA 验证请求失败: %v", err)
		// 当请求发生异常时，应放行通过，以免阻塞业务
		logger.Info("Google reCAPTCHA 服务异常，放行通过")
		return true, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Error("Google reCAPTCHA 验证响应状态码异常: %d", resp.StatusCode)
		// 当请求发生异常时，应放行通过，以免阻塞业务
		logger.Info("Google reCAPTCHA 服务异常，放行通过")
		return true, nil
	}

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取 Google reCAPTCHA 响应失败: %v", err)
		return true, nil // 异常时放行
	}

	// 解析JSON响应
	var recaptchaResp GoogleRecaptchaResponse
	if err := json.Unmarshal(body, &recaptchaResp); err != nil {
		logger.Error("解析 Google reCAPTCHA 响应JSON失败: %v", err)
		return true, nil // 异常时放行
	}

	// 检查基本验证结果
	if !recaptchaResp.Success {
		logger.Info("Google reCAPTCHA 验证失败，错误代码: %v", recaptchaResp.ErrorCodes)
		return false, fmt.Errorf("验证失败，错误代码: %v", recaptchaResp.ErrorCodes)
	}

	// 对于 v3，需要检查分数和动作
	if strings.ToLower(p.config.Type) == "google_v3" {
		// 检查动作是否匹配（如果提供了动作参数）
		if action != "" && recaptchaResp.Action != action {
			logger.Info("Google reCAPTCHA v3 动作不匹配，期望: %s, 实际: %s", action, recaptchaResp.Action)
			return false, fmt.Errorf("验证动作不匹配")
		}

		// 检查分数是否达到阈值
		if recaptchaResp.Score < p.config.GoogleMinScore {
			logger.Info("Google reCAPTCHA v3 分数过低: %.2f < %.2f", recaptchaResp.Score, p.config.GoogleMinScore)
			return false, fmt.Errorf("验证分数过低: %.2f", recaptchaResp.Score)
		}

		logger.Info("Google reCAPTCHA v3 验证成功，分数: %.2f, 动作: %s", recaptchaResp.Score, recaptchaResp.Action)
	} else {
		logger.Info("Google reCAPTCHA v2 验证成功")
	}

	return true, nil
}

// IsEnabled 检查 Google reCAPTCHA 是否启用
func (p *GoogleCaptchaProvider) IsEnabled() bool {
	return p.config.GoogleRecaptchaKey != ""
}

// GetType 获取验证码类型
func (p *GoogleCaptchaProvider) GetType() string {
	return strings.ToLower(p.config.Type)
}
