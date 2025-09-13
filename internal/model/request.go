package model

// ChatMessage 聊天消息结构
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Query          string        `json:"Query" binding:"required"`
	History        []ChatMessage `json:"History,omitempty"`
	// 腾讯云验证码字段
	CaptchaTicket  string        `json:"CaptchaTicket,omitempty"`
	CaptchaRandstr string        `json:"CaptchaRandstr,omitempty"`
	// 极验验证码字段
	LotNumber     string        `json:"lot_number,omitempty"`
	CaptchaOutput string        `json:"captcha_output,omitempty"`
	PassToken     string        `json:"pass_token,omitempty"`
	GenTime       string        `json:"gen_time,omitempty"`
}

// KnowledgeQuery 知识库查询请求
type KnowledgeQuery struct {
	Query string `json:"query"`
	TopK int `json:"top_k"`
}