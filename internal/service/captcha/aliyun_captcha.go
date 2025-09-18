package captcha

import (
	"fmt"
	"strings"

	"knowledge-maker/internal/config"
	"knowledge-maker/internal/logger"

	captcha20230305 "github.com/alibabacloud-go/captcha-20230305/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

// AliyunCaptchaProvider 阿里云验证码提供者
type AliyunCaptchaProvider struct {
	client *captcha20230305.Client
	config *config.CaptchaConfig
}

// AliyunCaptchaResponse 阿里云验证码响应结构
type AliyunCaptchaResponse struct {
	Result    string `json:"result"`
	Reason    string `json:"reason"`
	BizResult string `json:"bizResult"`
}

// NewAliyunCaptchaProvider 创建阿里云验证码提供者
func NewAliyunCaptchaProvider(cfg *config.CaptchaConfig) (*AliyunCaptchaProvider, error) {
	if cfg.AliyunAccessKeyID == "" || cfg.AliyunAccessKeySecret == "" {
		return nil, fmt.Errorf("阿里云验证码服务配置不完整：缺少 AccessKeyID 或 AccessKeySecret")
	}

	// 配置阿里云客户端
	openApiConfig := &openapi.Config{
		AccessKeyId:     tea.String(cfg.AliyunAccessKeyID),
		AccessKeySecret: tea.String(cfg.AliyunAccessKeySecret),
	}

	// 设置访问的域名
	if cfg.AliyunEndpoint != "" {
		openApiConfig.Endpoint = tea.String(cfg.AliyunEndpoint)
	} else {
		openApiConfig.Endpoint = tea.String("captcha.cn-shanghai.aliyuncs.com")
	}

	// 创建客户端
	client, err := captcha20230305.NewClient(openApiConfig)
	if err != nil {
		return nil, fmt.Errorf("创建阿里云验证码客户端失败: %v", err)
	}

	return &AliyunCaptchaProvider{
		client: client,
		config: cfg,
	}, nil
}

// Verify 验证阿里云验证码
func (p *AliyunCaptchaProvider) Verify(params map[string]string) (bool, error) {
	captchaParam := params["captcha_param"]
	scene := params["scene"]
	appId := params["app_id"]

	if captchaParam == "" {
		return false, fmt.Errorf("验证码参数不能为空")
	}

	// 如果没有提供场景ID，使用默认值
	if scene == "" {
		scene = "default"
	}

	// 如果没有提供应用ID，使用配置中的默认值
	if appId == "" {
		appId = p.config.AliyunCaptchaAppID
	}

	// 构建验证请求
	verifyIntelligentCaptchaRequest := &captcha20230305.VerifyIntelligentCaptchaRequest{
		CaptchaVerifyParam: tea.String(captchaParam),
		SceneId:            tea.String(scene),
	}

	// 创建运行时选项
	runtime := &util.RuntimeOptions{}

	try := func() (_result *captcha20230305.VerifyIntelligentCaptchaResponse, _e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				_e = r
			}
		}()

		// 发送验证请求
		_result, _e = p.client.VerifyIntelligentCaptchaWithOptions(verifyIntelligentCaptchaRequest, runtime)
		return _result, _e
	}

	response, err := try()
	if err != nil {
		logger.Error("阿里云验证码验证请求失败: %v", err)
		return false, fmt.Errorf("验证码验证请求失败: %v", err)
	}

	// 检查响应
	if response == nil || response.Body == nil {
		return false, fmt.Errorf("验证码响应格式错误")
	}

	// 检查API调用是否成功
	if response.Body.Success == nil || !*response.Body.Success {
		message := "验证码验证失败"
		if response.Body.Message != nil {
			message = *response.Body.Message
		}
		logger.Error("阿里云验证码API调用失败: %s", message)
		return false, fmt.Errorf("验证码验证失败: %s", message)
	}

	// 检查验证结果
	if response.Body.Result == nil {
		return false, fmt.Errorf("验证码响应缺少结果字段")
	}

	result := response.Body.Result
	logger.Info("阿里云验证码验证结果: %+v", result)

	// 检查验证结果
	// VerifyResult 字段为 true 表示验证通过
	if result.VerifyResult != nil && *result.VerifyResult {
		logger.Info("阿里云验证码验证成功")
		return true, nil
	}

	// 验证失败，返回具体原因
	reason := "验证码验证失败"
	if result.VerifyCode != nil {
		reason = fmt.Sprintf("验证码验证失败，错误码: %s", *result.VerifyCode)
	}

	logger.Warn("阿里云验证码验证失败: %s", reason)
	return false, fmt.Errorf("%s", reason)
}

// IsEnabled 检查阿里云验证码是否启用
func (p *AliyunCaptchaProvider) IsEnabled() bool {
	return p.config.AliyunAccessKeyID != "" &&
		p.config.AliyunAccessKeySecret != "" &&
		p.config.AliyunCaptchaAppID != ""
}

// GetType 获取验证码类型
func (p *AliyunCaptchaProvider) GetType() string {
	return strings.ToLower(p.config.Type)
}
