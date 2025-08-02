package model

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Query string `json:"query" binding:"required"`
}

// KnowledgeQuery 知识库查询请求
type KnowledgeQuery struct {
	Query string `json:"query"`
}