package model

// ChatResponse 聊天响应结构
type ChatResponse struct {
	Success          bool   `json:"success"`
	Answer           string `json:"answer"`
	KnowledgeContext string `json:"knowledge_context,omitempty"`
	Message          string `json:"message,omitempty"`
}

// KnowledgeResponse 知识库查询响应
type KnowledgeResponse struct {
	Success bool   `json:"success"`
	Data    string `json:"data"`
	Message string `json:"message"`
}