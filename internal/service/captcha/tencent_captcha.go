package captcha

import (
	"fmt"
	"strings"

	"knowledge-maker/internal/config"
	"knowledge-maker/internal/logger"

	captcha "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/captcha/v20190722"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// TencentCaptchaProvider 腾讯云验证码提供者
type TencentCaptchaProvider struct {
	client *captcha.Client
	config *config.CaptchaConfig
}

// NewTencentCaptchaProvider 创建腾讯云验证码提供者
func NewTencentCaptchaProvider(cfg *config.CaptchaConfig) (*TencentCaptchaProvider, error) {
	if cfg.SecretID == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("腾讯云验证码服务配置不完整：缺少 SecretID 或 SecretKey")
	}

	// 实例化认证对象
	credential := common.NewCredential(cfg.SecretID, cfg.SecretKey)

	// 实例化客户端配置对象
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = cfg.Endpoint

	// 实例化要请求产品的client对象
	client, err := captcha.NewClient(credential, "", cpf)
	if err != nil {
		return nil, fmt.Errorf("创建腾讯云验证码客户端失败: %v", err)
	}

	return &TencentCaptchaProvider{
		client: client,
		config: cfg,
	}, nil
}

// Verify 验证腾讯云验证码
func (p *TencentCaptchaProvider) Verify(params map[string]string) (bool, error) {
	ticket := params["ticket"]
	randstr := params["randstr"]
	userIP := params["userIP"]

	if ticket == "" || randstr == "" {
		return false, fmt.Errorf("验证码参数不能为空")
	}

	// 如果没有提供用户IP，尝试获取本地IP
	if userIP == "" {
		userIP = getLocalIP()
	}

	// 实例化请求对象
	request := captcha.NewDescribeCaptchaResultRequest()
	request.CaptchaType = common.Uint64Ptr(p.config.CaptchaType)
	request.Ticket = common.StringPtr(ticket)
	request.UserIp = common.StringPtr(userIP)
	request.Randstr = common.StringPtr(randstr)
	request.CaptchaAppId = common.Uint64Ptr(p.config.CaptchaAppID)
	request.AppSecretKey = common.StringPtr(p.config.AppSecretKey)

	// 发送请求
	response, err := p.client.DescribeCaptchaResult(request)
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

// IsEnabled 检查腾讯云验证码是否启用
func (p *TencentCaptchaProvider) IsEnabled() bool {
	return p.config.SecretID != "" && p.config.SecretKey != "" && p.config.CaptchaAppID != 0
}

// GetType 获取验证码类型
func (p *TencentCaptchaProvider) GetType() string {
	return strings.ToLower(p.config.Type)
}
