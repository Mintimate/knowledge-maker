package service

import (
	"fmt"
	"net"

	"knowledge-maker/internal/config"
	"knowledge-maker/internal/logger"

	captcha "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/captcha/v20190722"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// CaptchaService 验证码服务
type CaptchaService struct {
	client *captcha.Client
	config *config.CaptchaConfig
}

// NewCaptchaService 创建验证码服务实例
func NewCaptchaService(cfg *config.CaptchaConfig) (*CaptchaService, error) {
	if cfg.SecretID == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("验证码服务配置不完整：缺少 SecretID 或 SecretKey")
	}

	// 实例化认证对象
	credential := common.NewCredential(cfg.SecretID, cfg.SecretKey)

	// 实例化客户端配置对象
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = cfg.Endpoint

	// 实例化要请求产品的client对象
	client, err := captcha.NewClient(credential, "", cpf)
	if err != nil {
		return nil, fmt.Errorf("创建验证码客户端失败: %v", err)
	}

	return &CaptchaService{
		client: client,
		config: cfg,
	}, nil
}

// VerifyCaptcha 验证验证码
func (s *CaptchaService) VerifyCaptcha(ticket, randstr, userIP string) (bool, error) {
	if ticket == "" || randstr == "" {
		return false, fmt.Errorf("验证码参数不能为空")
	}

	// 如果没有提供用户IP，尝试获取本地IP
	if userIP == "" {
		userIP = s.getLocalIP()
	}

	// 实例化请求对象
	request := captcha.NewDescribeCaptchaResultRequest()
	request.CaptchaType = common.Uint64Ptr(s.config.CaptchaType)
	request.Ticket = common.StringPtr(ticket)
	request.UserIp = common.StringPtr(userIP)
	request.Randstr = common.StringPtr(randstr)
	request.CaptchaAppId = common.Uint64Ptr(s.config.CaptchaAppID)
	request.AppSecretKey = common.StringPtr(s.config.AppSecretKey)

	// 发送请求
	response, err := s.client.DescribeCaptchaResult(request)
	if err != nil {
		if sdkErr, ok := err.(*errors.TencentCloudSDKError); ok {
			logger.Error("验证码验证API错误: Code=%s, Message=%s", sdkErr.Code, sdkErr.Message)
			return false, fmt.Errorf("验证码验证失败: %s", sdkErr.Message)
		}
		logger.Error("验证码验证请求失败: %v", err)
		return false, fmt.Errorf("验证码验证请求失败: %v", err)
	}

	// 检查验证结果
	if response.Response.CaptchaCode == nil {
		return false, fmt.Errorf("验证码响应格式错误")
	}

	captchaCode := *response.Response.CaptchaCode
	logger.Info("验证码验证结果: Code=%d", captchaCode)

	// 验证码验证成功的状态码是1
	if captchaCode == 1 {
		return true, nil
	}

	// 根据不同的错误码返回相应的错误信息
	var errorMsg string
	switch captchaCode {
	case 6:
		errorMsg = "验证码已过期"
	case 7:
		errorMsg = "验证码已使用"
	case 8:
		errorMsg = "验证码验证失败"
	case 9:
		errorMsg = "验证码参数错误"
	case 10:
		errorMsg = "验证码配置错误"
	case 100:
		errorMsg = "验证码AppID不存在"
	default:
		errorMsg = fmt.Sprintf("验证码验证失败，错误码: %d", captchaCode)
	}

	return false, fmt.Errorf(errorMsg)
}

// getLocalIP 获取本地IP地址
func (s *CaptchaService) getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// IsEnabled 检查验证码服务是否启用
func (s *CaptchaService) IsEnabled() bool {
	return s.config.SecretID != "" && s.config.SecretKey != "" && s.config.CaptchaAppID != 0
}