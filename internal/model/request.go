package model

// ChatMessage 聊天消息结构
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Query   string        `json:"Query" binding:"required"`
	History []ChatMessage `json:"History,omitempty"`
}

// KnowledgeQuery 知识库查询请求
type KnowledgeQuery struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"`
}
