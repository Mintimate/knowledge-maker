package captcha

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"knowledge-maker/internal/config"
)

// CloudflareCaptchaProvider Cloudflare Turnstile 验证码提供者
type CloudflareCaptchaProvider struct {
	siteKey   string
	secretKey string
	verifyURL string
	enabled   bool
	client    *http.Client
}

// NewCloudflareCaptchaProvider 创建 Cloudflare Turnstile 验证码提供者
func NewCloudflareCaptchaProvider(cfg *config.CaptchaConfig) (*CloudflareCaptchaProvider, error) {
	if cfg.CloudflareSecretKey == "" {
		return &CloudflareCaptchaProvider{enabled: false}, nil
	}

	return &CloudflareCaptchaProvider{
		siteKey:   cfg.CloudflareSiteKey,
		secretKey: cfg.CloudflareSecretKey,
		verifyURL: cfg.CloudflareURL,
		enabled:   true,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// Verify 验证 Cloudflare Turnstile 验证码
func (p *CloudflareCaptchaProvider) Verify(params map[string]string) (bool, error) {
	if !p.enabled {
		return true, nil // 如果未启用，直接返回成功
	}

	token := params["token"]
	userIP := params["userIP"]

	if token == "" {
		return false, fmt.Errorf("Cloudflare Turnstile token 不能为空")
	}

	// 检查是否是容灾票据
	if len(token) > 12 && token[:12] == "cf_fallback_" {
		// 容灾票据，直接返回成功
		return true, nil
	}

	// 构建请求参数
	data := url.Values{}
	data.Set("secret", p.secretKey)
	data.Set("response", token)
	if userIP != "" {
		data.Set("remoteip", userIP)
	}

	// 发送验证请求
	resp, err := p.client.PostForm(p.verifyURL, data)
	if err != nil {
		return false, fmt.Errorf("Cloudflare Turnstile 验证请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("读取 Cloudflare Turnstile 响应失败: %v", err)
	}

	// 解析响应
	var result CloudflareTurnstileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("解析 Cloudflare Turnstile 响应失败: %v", err)
	}

	// 检查验证结果
	if !result.Success {
		errorMsg := "Cloudflare Turnstile 验证失败"
		if len(result.ErrorCodes) > 0 {
			errorMsg += fmt.Sprintf(": %v", result.ErrorCodes)
		}
		return false, fmt.Errorf(errorMsg)
	}

	return true, nil
}

// IsEnabled 检查是否启用
func (p *CloudflareCaptchaProvider) IsEnabled() bool {
	return p.enabled
}

// GetType 获取类型
func (p *CloudflareCaptchaProvider) GetType() string {
	return "cloudflare"
}
