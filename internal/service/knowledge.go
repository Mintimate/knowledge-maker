package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"knowledge-maker/internal/config"
	"knowledge-maker/internal/model"
)

// KnowledgeService 知识库服务
type KnowledgeService struct {
	baseURL string
	token   string
	config  *config.Config
}

// NewKnowledgeService 创建知识库服务实例
func NewKnowledgeService(cfg *config.Config) *KnowledgeService {
	return &KnowledgeService{
		baseURL: cfg.Knowledge.BaseURL,
		token:   cfg.Knowledge.Token,
		config:  cfg,
	}
}

// QueryKnowledge 查询知识库
func (ks *KnowledgeService) QueryKnowledge(query string) (string, error) {
	// 构建请求体
	requestBody := model.KnowledgeQuery{
		Query: query,
		TopK:  ks.config.Knowledge.TopK,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求数据失败: %v", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", ks.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", ks.token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("知识库查询失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应（这里假设直接返回文本内容，您可能需要根据实际 API 响应格式调整）
	return string(body), nil
}