package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	AI        AIConfig        `yaml:"ai"`
	Knowledge KnowledgeConfig `yaml:"knowledge"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `yaml:"port"`
	Mode string `yaml:"mode"` // debug, release, test
}

// AIConfig AI 服务配置
type AIConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
}

// KnowledgeConfig 知识库配置
type KnowledgeConfig struct {
	BaseURL string `yaml:"base_url"`
	Token   string `yaml:"token"`
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
