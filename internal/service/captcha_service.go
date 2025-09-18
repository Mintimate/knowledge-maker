package service

import (
	"knowledge-maker/internal/config"
	"knowledge-maker/internal/service/captcha"
)

// CaptchaService 验证码服务（对外接口）
type CaptchaService struct {
	service *captcha.CaptchaService
}

// NewCaptchaService 创建验证码服务实例
func NewCaptchaService(cfg *config.CaptchaConfig) (*CaptchaService, error) {
	captchaService, err := captcha.NewCaptchaService(cfg)
	if err != nil {
		return nil, err
	}

	return &CaptchaService{
		service: captchaService,
	}, nil
}

// VerifyCaptcha 验证验证码（腾讯云）
func (s *CaptchaService) VerifyCaptcha(ticket, randstr, userIP string) (bool, error) {
	return s.service.VerifyCaptcha(ticket, randstr, userIP)
}

// VerifyGeetestCaptcha 验证极验验证码
func (s *CaptchaService) VerifyGeetestCaptcha(lotNumber, captchaOutput, passToken, genTime string) (bool, error) {
	return s.service.VerifyGeetestCaptcha(lotNumber, captchaOutput, passToken, genTime)
}

// VerifyGoogleRecaptcha 验证 Google reCAPTCHA v2/v3
func (s *CaptchaService) VerifyGoogleRecaptcha(token, action, userIP string) (bool, error) {
	return s.service.VerifyGoogleRecaptcha(token, action, userIP)
}

// VerifyCloudflareTurnstile 验证 Cloudflare Turnstile 验证码
func (s *CaptchaService) VerifyCloudflareTurnstile(token, userIP string) (bool, error) {
	return s.service.VerifyCloudflareTurnstile(token, userIP)
}

// IsEnabled 检查验证码服务是否启用
func (s *CaptchaService) IsEnabled() bool {
	return s.service.IsEnabled()
}

// GetCaptchaType 获取验证码类型
func (s *CaptchaService) GetCaptchaType() string {
	return s.service.GetCaptchaType()
}
