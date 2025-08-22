package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LogConfig 日志配置
type LogConfig struct {
	Dir        string `yaml:"dir"`
	Level      string `yaml:"level"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// KnowledgeConfig 知识库配置
type KnowledgeConfig struct {
	BaseURL  string         `yaml:"base_url"`
	Token    string         `yaml:"token"`
	VectorDB VectorDBConfig `yaml:"vector_db"`
}

// VectorDBConfig 向量数据库配置
type VectorDBConfig struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

// Config 应用配置
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	AI        AIConfig        `yaml:"ai"`
	RAG       RAGConfig       `yaml:"rag"`
	Database  DatabaseConfig  `yaml:"database"`
	Knowledge KnowledgeConfig `yaml:"knowledge"`
	Log       LogConfig       `yaml:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port        string `yaml:"port"`
	Mode        string `yaml:"mode"`
	AllowDomain string `yaml:"allow_domain"`
}

// AIConfig AI 服务配置
type AIConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
}


// RAGConfig RAG 服务配置
type RAGConfig struct {
	SystemPrompt string `yaml:"system_prompt"`
}

// LoadConfig 加载配置
func LoadConfig(configPath string) (*Config, error) {
	// 如果没有指定配置文件路径，尝试默认路径
	if configPath == "" {
		configPath = "config.yml"

		// 获取当前工作目录
		wd, err := os.Getwd()
		if err == nil {
			// 尝试在项目根目录查找
			rootConfigPath := filepath.Join(wd, "config.yml")
			if fileExists(rootConfigPath) {
				configPath = rootConfigPath
			}
		}
	}

	config := &Config{}

	// 如果配置文件存在，从文件加载
	if fileExists(configPath) {
		if err := loadFromFile(configPath, config); err != nil {
			return nil, fmt.Errorf("加载配置文件失败: %v", err)
		}
	}

	// 环境变量覆盖配置文件设置
	overrideWithEnv(config)

	return config, nil
}

// loadFromFile 从文件加载配置
func loadFromFile(path string, config *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	return nil
}

// overrideWithEnv 使用环境变量覆盖配置
func overrideWithEnv(config *Config) {
	// 服务器配置
	if port := os.Getenv("SERVER_PORT"); port != "" {
		config.Server.Port = port
	}
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		config.Server.Mode = mode
	}
	if allowDomain := os.Getenv("ALLOW_DOMAIN"); allowDomain != "" {
		config.Server.AllowDomain = allowDomain
	}

	// AI 配置
	if baseURL := os.Getenv("AI_BASE_URL"); baseURL != "" {
		config.AI.BaseURL = baseURL
	}
	if apiKey := os.Getenv("AI_API_KEY"); apiKey != "" {
		config.AI.APIKey = apiKey
	}
	if model := os.Getenv("AI_MODEL"); model != "" {
		config.AI.Model = model
	}

	// 知识库配置
	if baseURL := os.Getenv("KNOWLEDGE_BASE_URL"); baseURL != "" {
		config.Knowledge.BaseURL = baseURL
	}
	if token := os.Getenv("KNOWLEDGE_TOKEN"); token != "" {
		config.Knowledge.Token = token
	}

	// RAG 配置
	if systemPrompt := os.Getenv("RAG_SYSTEM_PROMPT"); systemPrompt != "" {
		config.RAG.SystemPrompt = systemPrompt
	}

	// 日志配置
	if logDir := os.Getenv("LOG_DIR"); logDir != "" {
		config.Log.Dir = logDir
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
