package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	TopK     int            `yaml:"top_k"`
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
	Captcha   CaptchaConfig   `yaml:"captcha"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         string   `yaml:"port"`
	Mode         string   `yaml:"mode"`
	AllowDomains []string `yaml:"allow_domains"`
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

// CaptchaConfig 验证码配置
type CaptchaConfig struct {
	// 验证码类型: "tencent" 或 "geetest"
	Type         string `yaml:"type"`
	// 腾讯云验证码配置
	SecretID     string `yaml:"secret_id"`
	SecretKey    string `yaml:"secret_key"`
	CaptchaAppID uint64 `yaml:"captcha_app_id"`
	AppSecretKey string `yaml:"app_secret_key"`
	Endpoint     string `yaml:"endpoint"`
	CaptchaType  uint64 `yaml:"captcha_type"`
	// 极验验证码配置
	GeetestID    string `yaml:"geetest_id"`
	GeetestKey   string `yaml:"geetest_key"`
	GeetestURL   string `yaml:"geetest_url"`
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

	// 设置默认值
	setDefaults(config)

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
	if allowDomains := os.Getenv("ALLOW_DOMAINS"); allowDomains != "" {
		// 支持逗号分隔的多个域名
		config.Server.AllowDomains = strings.Split(allowDomains, ",")
		for i, domain := range config.Server.AllowDomains {
			config.Server.AllowDomains[i] = strings.TrimSpace(domain)
		}
	}
	// 向后兼容：如果设置了 ALLOW_DOMAIN 但没有设置 ALLOW_DOMAINS，则转换为数组
	if allowDomain := os.Getenv("ALLOW_DOMAIN"); allowDomain != "" && len(config.Server.AllowDomains) == 0 {
		config.Server.AllowDomains = []string{allowDomain}
	}
	if allowDomains := os.Getenv("ALLOW_DOMAINS"); allowDomains != "" {
		// 支持逗号分隔的多个域名
		config.Server.AllowDomains = strings.Split(allowDomains, ",")
		for i, domain := range config.Server.AllowDomains {
			config.Server.AllowDomains[i] = strings.TrimSpace(domain)
		}
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
	if topK := os.Getenv("KNOWLEDGE_TOP_K"); topK != "" {
		if k, err := strconv.Atoi(topK); err == nil {
			config.Knowledge.TopK = k
		}
	}

	// RAG 配置
	if systemPrompt := os.Getenv("RAG_SYSTEM_PROMPT"); systemPrompt != "" {
		config.RAG.SystemPrompt = systemPrompt
	}

	// 日志配置
	if logDir := os.Getenv("LOG_DIR"); logDir != "" {
		config.Log.Dir = logDir
	}

	// 验证码配置
	if captchaType := os.Getenv("CAPTCHA_TYPE"); captchaType != "" {
		config.Captcha.Type = captchaType
	}
	// 腾讯云验证码配置
	if secretID := os.Getenv("TENCENTCLOUD_SECRET_ID"); secretID != "" {
		config.Captcha.SecretID = secretID
	}
	if secretKey := os.Getenv("TENCENTCLOUD_SECRET_KEY"); secretKey != "" {
		config.Captcha.SecretKey = secretKey
	}
	if captchaAppID := os.Getenv("CAPTCHA_APP_ID"); captchaAppID != "" {
		if appID, err := strconv.ParseUint(captchaAppID, 10, 64); err == nil {
			config.Captcha.CaptchaAppID = appID
		}
	}
	if appSecretKey := os.Getenv("CAPTCHA_APP_SECRET_KEY"); appSecretKey != "" {
		config.Captcha.AppSecretKey = appSecretKey
	}
	if endpoint := os.Getenv("CAPTCHA_ENDPOINT"); endpoint != "" {
		config.Captcha.Endpoint = endpoint
	}
	if tencentCaptchaType := os.Getenv("TENCENT_CAPTCHA_TYPE"); tencentCaptchaType != "" {
		if cType, err := strconv.ParseUint(tencentCaptchaType, 10, 64); err == nil {
			config.Captcha.CaptchaType = cType
		}
	}
	// 极验验证码配置
	if geetestID := os.Getenv("GEETEST_ID"); geetestID != "" {
		config.Captcha.GeetestID = geetestID
	}
	if geetestKey := os.Getenv("GEETEST_KEY"); geetestKey != "" {
		config.Captcha.GeetestKey = geetestKey
	}
	if geetestURL := os.Getenv("GEETEST_URL"); geetestURL != "" {
		config.Captcha.GeetestURL = geetestURL
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// setDefaults 设置默认配置值
func setDefaults(config *Config) {
	// 知识库默认配置
	if config.Knowledge.TopK == 0 {
		config.Knowledge.TopK = 3
	}

	// 验证码默认配置 - 如果没有设置验证类型，则不进行验证码校验
	// 不再设置默认的验证码类型，保持为空表示不启用验证码
	if config.Captcha.Endpoint == "" {
		config.Captcha.Endpoint = "captcha.tencentcloudapi.com"
	}
	if config.Captcha.CaptchaType == 0 {
		config.Captcha.CaptchaType = 9 // 默认使用滑动验证码
	}
	if config.Captcha.GeetestURL == "" {
		config.Captcha.GeetestURL = "http://gcaptcha4.geetest.com/validate"
	}
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
