# Knowledge Maker - 知识库 RAG 服务

基于 Go 和 Gin 框架构建的知识库检索增强生成（RAG）服务，专门用于 Rime 输入法和薄荷输入法相关内容的智能问答。

## 项目结构

```
knowledge-maker/
├── cmd/                    # 应用程序入口
│   └── server/            # 服务器主程序
│       └── main.go        # 主入口文件
├── internal/              # 内部包（不对外暴露）
│   ├── config/           # 配置管理
│   │   └── config.go     # 配置结构和加载
│   ├── handler/          # HTTP 处理器
│   │   └── rag.go        # RAG 相关路由处理
│   ├── model/            # 数据模型
│   │   ├── request.go    # 请求模型
│   │   └── response.go   # 响应模型
│   └── service/          # 业务逻辑服务
│       ├── ai.go         # AI 服务
│       ├── knowledge.go  # 知识库服务
│       └── rag.go        # RAG 核心服务
├── go.mod               # Go 模块文件
├── go.sum               # Go 依赖锁定文件
└── README.md           # 项目说明
```

## 架构设计

### 分层架构
- **Handler 层**: 处理 HTTP 请求和响应，负责参数验证和错误处理
- **Service 层**: 核心业务逻辑，包含 RAG、AI 和知识库服务
- **Model 层**: 数据结构定义，包含请求和响应模型
- **Config 层**: 配置管理，支持环境变量和默认值

### 服务组件
1. **KnowledgeService**: 负责与外部知识库 API 交互
2. **AIService**: 负责与 AI 模型（DeepSeek）交互，支持普通和流式响应
3. **RAGService**: 整合知识库检索和 AI 生成的核心服务

## 功能特性

- ✅ 知识库检索增强生成（RAG）
- ✅ 支持普通和流式聊天响应
- ✅ 专门针对 Rime 和薄荷输入法的智能问答
- ✅ 配置化管理，支持环境变量
- ✅ 清晰的分层架构和模块化设计
- ✅ CORS 支持，便于前端集成
- ✅ 健康检查接口

## 本地开发

如果需要本地开发，确保安装了 Go 1.24+：

```bash
# 下载依赖
go mod download

# 开发模式运行
go run ./cmd/server/main.go
```

## 配置说明

项目支持两种配置方式：

### 1. 配置文件（推荐）

使用 `config.yml` 文件进行配置：

```yaml
# 服务器配置
server:
  port: "8081"
  mode: "debug"  # debug, release, test

# AI 服务配置
ai:
  base_url: "https://api.deepseek.com/v1"
  api_key: "your_api_key_here"
  model: "deepseek-chat"

# 知识库配置
knowledge:
  base_url: "https://api.cnb.cool/Mintimate/rime/DocVitePressOMR/-/knowledge/base/query"
  token: "your_token_here"
```

### 2. 环境变量（优先级更高）

环境变量会覆盖配置文件中的设置：

```bash
# 服务器配置
export SERVER_PORT=8082
export GIN_MODE=release

# 知识库配置
export KNOWLEDGE_BASE_URL="your_knowledge_base_url"
export KNOWLEDGE_TOKEN="your_token_here"

# AI 服务配置
export AI_API_KEY="your_api_key_here"
export AI_BASE_URL="https://api.deepseek.com/v1"
export AI_MODEL="deepseek-chat"
```

### 配置管理

配置文件会按以下优先级加载：
1. 环境变量（最高优先级）
2. `config.yml` 配置文件


## API 接口

### 1. 健康检查
```
GET /api/v1/health
```

### 2. 普通聊天
```
POST /api/v1/chat
Content-Type: application/json

{
  "query": "如何配置 Rime 输入法？"
}
```

### 3. 流式聊天
```
POST /api/v1/chat/stream
Content-Type: application/json

{
  "query": "薄荷输入法有什么特色功能？"
}
```

## 技术栈

- **框架**: Gin (HTTP 框架)
- **AI 集成**: OpenAI Go SDK (兼容 DeepSeek API)
- **配置管理**: YAML 配置文件 + 环境变量
- **部署方式**: Docker + Docker Compose
- **构建工具**: Docker 多阶段构建

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证。