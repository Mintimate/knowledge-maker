package captcha

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

// GeetestCaptchaProvider 极验验证码提供者
type GeetestCaptchaProvider struct {
	config *config.CaptchaConfig
}

// NewGeetestCaptchaProvider 创建极验验证码提供者
func NewGeetestCaptchaProvider(cfg *config.CaptchaConfig) (*GeetestCaptchaProvider, error) {
	if cfg.GeetestID == "" || cfg.GeetestKey == "" {
		return nil, fmt.Errorf("极验验证码服务配置不完整：缺少 GeetestID 或 GeetestKey")
	}

	return &GeetestCaptchaProvider{
		config: cfg,
	}, nil
}

// Verify 验证极验验证码
func (p *GeetestCaptchaProvider) Verify(params map[string]string) (bool, error) {
	lotNumber := params["lotNumber"]
	captchaOutput := params["captchaOutput"]
	passToken := params["passToken"]
	genTime := params["genTime"]

	if lotNumber == "" || captchaOutput == "" || passToken == "" || genTime == "" {
		return false, fmt.Errorf("极验验证码参数不能为空")
	}

	// 生成签名
	signToken := p.hmacEncode(p.config.GeetestKey, lotNumber)

	// 构建请求参数
	formData := url.Values{}
	formData.Set("lot_number", lotNumber)
	formData.Set("captcha_output", captchaOutput)
	formData.Set("pass_token", passToken)
	formData.Set("gen_time", genTime)
	formData.Set("sign_token", signToken)

	// 构建完整的URL
	requestURL := fmt.Sprintf("%s?captcha_id=%s", p.config.GeetestURL, p.config.GeetestID)

	// 创建HTTP客户端，设置5秒超时
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 发起POST请求
	resp, err := client.PostForm(requestURL, formData)
	if err != nil {
		logger.Error("极验验证码请求失败: %v", err)
		// 当请求发生异常时，应放行通过，以免阻塞业务
		logger.Info("极验服务异常，放行通过")
		return true, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Error("极验验证码响应状态码异常: %d", resp.StatusCode)
		// 当请求发生异常时，应放行通过，以免阻塞业务
		logger.Info("极验服务异常，放行通过")
		return true, nil
	}

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取极验响应失败: %v", err)
		return true, nil // 异常时放行
	}

	// 解析JSON响应
	var geetestResp GeetestResponse
	if err := json.Unmarshal(body, &geetestResp); err != nil {
		logger.Error("解析极验响应JSON失败: %v", err)
		return true, nil // 异常时放行
	}

	// 检查验证结果
	if geetestResp.Result == "success" {
		logger.Info("极验验证码验证成功")
		return true, nil
	} else {
		logger.Info("极验验证码验证失败: %s", geetestResp.Reason)
		return false, fmt.Errorf("验证失败: %s", geetestResp.Reason)
	}
}

// IsEnabled 检查极验验证码是否启用
func (p *GeetestCaptchaProvider) IsEnabled() bool {
	return p.config.GeetestID != "" && p.config.GeetestKey != ""
}

// GetType 获取验证码类型
func (p *GeetestCaptchaProvider) GetType() string {
	return strings.ToLower(p.config.Type)
}

// hmacEncode 使用HMAC-SHA256生成签名
func (p *GeetestCaptchaProvider) hmacEncode(key, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
