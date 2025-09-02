# Knowledge Maker - RAG 知识库问答服务

基于 RAG（Retrieval-Augmented Generation）技术的智能问答服务，支持流式响应和思考内容展示。

## ✨ 主要特性

- 🤖 **智能问答**：基于知识库检索的 AI 问答服务
- 🌊 **流式响应**：支持实时流式输出，提升用户体验
- 🧠 **思考过程展示**：支持 reasoning_content 解析，展示 AI 思考过程
- 📝 **统一日志系统**：配置化的日志管理，支持按日期分文件存储
- 🔒 **CORS 安全配置**：支持配置化的跨域访问控制
- ⚙️ **灵活配置**：支持配置文件和环境变量双重配置方式

## 🚀 快速开始

### 环境要求

- 支持的 AI 服务（如 DeepSeek、混元等）
- 知识库服务

### 安装运行

1. **克隆项目**
```bash
git clone <repository-url>
cd knowledge-maker
```

2. **安装依赖**
```bash
go mod tidy
```

3. **配置服务**
```bash
# 编辑配置文件
vim config.yml
```

4. **启动服务**
```bash
go run cmd/server/main.go
```

服务将在 `http://localhost:8082` 启动。

## ⚙️ 配置说明

### 配置文件 (config.yml)

```yaml
# 服务器配置
server:
  port: "8082"                    # 服务端口
  mode: "debug"                   # 运行模式: debug, release, test
  # 支持多个域名配置，留空表示允许所有域名
  allow_domains:
    - "https://www.mintimate.cc"
    - "https://mintimate.cc"

# AI 服务配置
ai:
  base_url: "https://api.example.com/v1"  # AI 服务地址
  api_key: "your-api-key"                 # API 密钥
  model: "your-model"                     # 使用的模型

# 知识库配置
knowledge:
  base_url: "https://knowledge.example.com/query"  # 知识库查询地址
  token: "your-knowledge-token"                     # 知识库访问令牌
  top_k: 5                                          # 单次查询返回的最大结果数量

# RAG 配置
rag:
  system_prompt: |
    你是 AI 助手，专门检索相关内容...
    # 系统提示词配置

# 日志配置
log:
  dir: "logs"          # 日志目录
  level: "info"        # 日志级别: debug, info, warn, error
```

### 环境变量配置

环境变量优先级高于配置文件：

```bash
# 服务器配置
export SERVER_PORT="8082"
export GIN_MODE="release"
# 支持多个域名，用逗号分隔
export ALLOW_DOMAINS="https://www.mintimate.cc,https://mintimate.cc"
# 向后兼容：单域名配置（如果没有设置 ALLOW_DOMAINS）
export ALLOW_DOMAIN="https://yourdomain.com"

# AI 服务配置
export AI_BASE_URL="https://api.example.com/v1"
export AI_API_KEY="your-api-key"
export AI_MODEL="your-model"

# 知识库配置
export KNOWLEDGE_BASE_URL="https://knowledge.example.com/query"
export KNOWLEDGE_TOKEN="your-knowledge-token"
export KNOWLEDGE_TOP_K="5"

# RAG 配置
export RAG_SYSTEM_PROMPT="你是 AI 助手..."

# 日志配置
export LOG_DIR="./logs"
```

## 📡 API 接口

### 健康检查
```http
GET /api/v1/health
```

### 普通问答
```http
POST /api/v1/chat
Content-Type: application/json

{
  "query": "你的问题"
}
```

### 流式问答
```http
POST /api/v1/chat/stream
Content-Type: application/json

{
  "query": "你的问题"
}
```

流式响应格式：
```
event: data
data: {"content": "<think>"}

event: data
data: {"content": "AI 的思考内容..."}

event: data
data: {"content": "</think>"}

event: data
data: {"content": "<answer>"}

event: data
data: {"content": "AI 的回答内容..."}

event: done
data: {"success": true, "message": "回答完成"}
```

## 🛠️ 开发指南

### 项目结构
```
knowledge-maker/
├── cmd/server/          # 主程序入口
├── internal/
│   ├── config/         # 配置管理
│   ├── handler/        # HTTP 处理器
│   ├── logger/         # 日志系统
│   ├── model/          # 数据模型
│   └── service/        # 业务逻辑
├── logs/               # 日志文件
├── static/             # 静态资源
└── config.yml          # 配置文件
```

### 添加新功能
1. 在 `internal/service/` 中添加业务逻辑
2. 在 `internal/handler/` 中添加 HTTP 处理
3. 在 `internal/model/` 中定义数据结构
4. 更新配置文件和环境变量支持

## 📄 许可证

本项目采用 GPL-3.0 许可证。详见 [LICENSE](LICENSE) 文件。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📞 支持

如有问题，请提交 Issue 或联系维护者。