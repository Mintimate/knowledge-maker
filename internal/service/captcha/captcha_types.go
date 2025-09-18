package captcha

// CaptchaProvider 验证码提供者接口
type CaptchaProvider interface {
	// Verify 验证验证码
	Verify(params map[string]string) (bool, error)
	// IsEnabled 检查是否启用
	IsEnabled() bool
	// GetType 获取类型
	GetType() string
}

// GeetestResponse 极验响应结构
type GeetestResponse struct {
	Result      string                 `json:"result"`
	Reason      string                 `json:"reason"`
	CaptchaArgs map[string]interface{} `json:"captcha_args"`
}

// GoogleRecaptchaResponse Google reCAPTCHA 响应结构
type GoogleRecaptchaResponse struct {
	Success     bool     `json:"success"`
	Score       float64  `json:"score,omitempty"`       // v3 专用
	Action      string   `json:"action,omitempty"`      // v3 专用
	ChallengeTS string   `json:"challenge_ts"`          // 验证时间戳
	Hostname    string   `json:"hostname"`              // 验证的主机名
	ErrorCodes  []string `json:"error-codes,omitempty"` // 错误代码
}

// CloudflareTurnstileResponse Cloudflare Turnstile 响应结构
type CloudflareTurnstileResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`          // 验证时间戳
	Hostname    string   `json:"hostname"`              // 验证的主机名
	ErrorCodes  []string `json:"error-codes,omitempty"` // 错误代码
	Action      string   `json:"action,omitempty"`      // 操作类型
	CData       string   `json:"cdata,omitempty"`       // 自定义数据
}
