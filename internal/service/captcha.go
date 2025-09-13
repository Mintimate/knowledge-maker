package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

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

// GeetestResponse 极验响应结构
type GeetestResponse struct {
	Result      string                 `json:"result"`
	Reason      string                 `json:"reason"`
	CaptchaArgs map[string]interface{} `json:"captcha_args"`
}

// NewCaptchaService 创建验证码服务实例
func NewCaptchaService(cfg *config.CaptchaConfig) (*CaptchaService, error) {
	service := &CaptchaService{
		config: cfg,
	}

	// 根据验证码类型进行不同的初始化
	switch strings.ToLower(cfg.Type) {
	case "tencent":
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
		service.client = client

	case "geetest":
		if cfg.GeetestID == "" || cfg.GeetestKey == "" {
			return nil, fmt.Errorf("极验验证码服务配置不完整：缺少 GeetestID 或 GeetestKey")
		}

	default:
		return nil, fmt.Errorf("不支持的验证码类型: %s", cfg.Type)
	}

	return service, nil
}

// VerifyCaptcha 验证验证码（腾讯云）
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

// VerifyGeetestCaptcha 验证极验验证码
func (s *CaptchaService) VerifyGeetestCaptcha(lotNumber, captchaOutput, passToken, genTime string) (bool, error) {
	if lotNumber == "" || captchaOutput == "" || passToken == "" || genTime == "" {
		return false, fmt.Errorf("极验验证码参数不能为空")
	}

	// 生成签名
	signToken := s.hmacEncode(s.config.GeetestKey, lotNumber)

	// 构建请求参数
	formData := url.Values{}
	formData.Set("lot_number", lotNumber)
	formData.Set("captcha_output", captchaOutput)
	formData.Set("pass_token", passToken)
	formData.Set("gen_time", genTime)
	formData.Set("sign_token", signToken)

	// 构建完整的URL
	requestURL := fmt.Sprintf("%s?captcha_id=%s", s.config.GeetestURL, s.config.GeetestID)

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

// hmacEncode 使用HMAC-SHA256生成签名
func (s *CaptchaService) hmacEncode(key, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
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
	switch strings.ToLower(s.config.Type) {
	case "tencent":
		return s.config.SecretID != "" && s.config.SecretKey != "" && s.config.CaptchaAppID != 0
	case "geetest":
		return s.config.GeetestID != "" && s.config.GeetestKey != ""
	default:
		return false
	}
}

// GetCaptchaType 获取验证码类型
func (s *CaptchaService) GetCaptchaType() string {
	return strings.ToLower(s.config.Type)
}
