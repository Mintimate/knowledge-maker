package captcha

import (
	"fmt"
	"net"
	"strings"

	"knowledge-maker/internal/config"
)

// CaptchaService 验证码服务
type CaptchaService struct {
	provider CaptchaProvider
	config   *config.CaptchaConfig
}

// NewCaptchaService 创建验证码服务实例
func NewCaptchaService(cfg *config.CaptchaConfig) (*CaptchaService, error) {
	var provider CaptchaProvider
	var err error

	// 根据验证码类型创建对应的提供者
	switch strings.ToLower(cfg.Type) {
	case "tencent":
		provider, err = NewTencentCaptchaProvider(cfg)
	case "geetest":
		provider, err = NewGeetestCaptchaProvider(cfg)
	case "google_v2", "google_v3":
		provider, err = NewGoogleCaptchaProvider(cfg)
	default:
		return nil, fmt.Errorf("不支持的验证码类型: %s", cfg.Type)
	}

	if err != nil {
		return nil, err
	}

	return &CaptchaService{
		provider: provider,
		config:   cfg,
	}, nil
}

// VerifyCaptcha 验证验证码（腾讯云）
func (s *CaptchaService) VerifyCaptcha(ticket, randstr, userIP string) (bool, error) {
	params := map[string]string{
		"ticket":  ticket,
		"randstr": randstr,
		"userIP":  userIP,
	}
	return s.provider.Verify(params)
}

// VerifyGeetestCaptcha 验证极验验证码
func (s *CaptchaService) VerifyGeetestCaptcha(lotNumber, captchaOutput, passToken, genTime string) (bool, error) {
	params := map[string]string{
		"lotNumber":     lotNumber,
		"captchaOutput": captchaOutput,
		"passToken":     passToken,
		"genTime":       genTime,
	}
	return s.provider.Verify(params)
}

// VerifyGoogleRecaptcha 验证 Google reCAPTCHA v2/v3
func (s *CaptchaService) VerifyGoogleRecaptcha(token, action, userIP string) (bool, error) {
	params := map[string]string{
		"token":  token,
		"action": action,
		"userIP": userIP,
	}
	return s.provider.Verify(params)
}

// IsEnabled 检查验证码服务是否启用
func (s *CaptchaService) IsEnabled() bool {
	return s.provider.IsEnabled()
}

// GetCaptchaType 获取验证码类型
func (s *CaptchaService) GetCaptchaType() string {
	return s.provider.GetType()
}

// getLocalIP 获取本地IP地址
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
