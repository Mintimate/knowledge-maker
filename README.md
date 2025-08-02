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
├── bin/                  # 构建输出目录
├── go.mod               # Go 模块文件
├── go.sum               # Go 依赖锁定文件
├── Makefile            # 构建脚本
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

## 快速开始

### 环境要求
- Docker 和 Docker Compose
- 网络连接（用于访问知识库和 AI 服务）

### 使用 Docker Compose 部署（推荐）

1. 复制环境变量模板：
```bash
cp .env.example .env
```

2. 编辑 `.env` 文件，填入实际的配置值：
```bash
# 必须设置的环境变量
AI_API_KEY=your_actual_api_key
KNOWLEDGE_BASE_URL=your_knowledge_base_url
KNOWLEDGE_TOKEN=your_knowledge_token
```

3. 启动服务：
```bash
docker-compose up -d
```

4. 查看服务状态：
```bash
docker-compose ps
docker-compose logs -f knowledge-maker
```

### 使用 Docker 部署

1. 构建镜像：
```bash
docker build -t knowledge-maker .
```

2. 运行容器：
```bash
docker run -d \
  --name knowledge-maker \
  -p 8081:8081 \
  -e AI_API_KEY=your_api_key \
  -e KNOWLEDGE_BASE_URL=your_knowledge_base_url \
  -e KNOWLEDGE_TOKEN=your_knowledge_token \
  knowledge-maker
```

### 本地开发

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
export SERVER_PORT=8081
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
3. 默认值（最低优先级）

生产环境建议使用环境变量或 `config.prod.yml` 模板。

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

## Docker 管理命令

```bash
# 启动服务
docker-compose up -d

# 停止服务
docker-compose down

# 查看日志
docker-compose logs -f knowledge-maker

# 重启服务
docker-compose restart knowledge-maker

# 查看服务状态
docker-compose ps

# 进入容器
docker-compose exec knowledge-maker sh

# 重新构建并启动
docker-compose up -d --build
```

## 本地开发命令

```bash
# 下载依赖
go mod download && go mod tidy

# 开发模式运行
go run ./cmd/server/main.go

# 构建应用
go build -o bin/knowledge-maker ./cmd/server

# 运行测试
go test -v ./...

# 代码格式化
go fmt ./...

# 代码检查
go vet ./...
```

## 项目优势

1. **清晰的架构**: 采用标准的 Go 项目结构，分层清晰，易于维护
2. **配置化**: 支持环境变量配置，便于部署和管理
3. **模块化**: 各个组件职责单一，便于测试和扩展
4. **可扩展**: 易于添加新的服务和功能
5. **生产就绪**: 包含错误处理、日志记录等生产环境必需功能

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