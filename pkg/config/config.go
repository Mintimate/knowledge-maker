package config

import (
	"os"
)

// Config 应用配置
type Config struct {
	Server    ServerConfig    `json:"server"`
	AI        AIConfig        `json:"ai"`
	Knowledge KnowledgeConfig `json:"knowledge"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `json:"port"`
	Mode string `json:"mode"`
}

// AIConfig AI 服务配置
type AIConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

// KnowledgeConfig 知识库配置
type KnowledgeConfig struct {
	BaseURL string `json:"base_url"`
	Token   string `json:"token"`
}

// LoadConfig 加载配置
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8081"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		AI: AIConfig{
			BaseURL: getEnv("AI_BASE_URL", "https://api.deepseek.com/v1"),
			APIKey:  getEnv("AI_API_KEY", "sk-Your-api-key-here"),
			Model:   getEnv("AI_MODEL", "deepseek-chat"),
		},
		Knowledge: KnowledgeConfig{
			BaseURL: getEnv("KNOWLEDGE_BASE_URL", "https://knowledge.example.com/api/v1/query"),
			Token:   getEnv("KNOWLEDGE_TOKEN", "Bearer your-knowledge-token-here"),
		},
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
