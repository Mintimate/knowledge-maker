# Build stage
FROM golang:1.24.5-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的系统依赖
RUN apk add --no-cache gcc musl-dev

# 复制 go.mod 和 go.sum 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod tidy && go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=1 GOOS=linux go build -o knowledge-maker ./cmd/server

# Final stage
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk add --no-cache ca-certificates && \
    rm -rf /var/cache/apk/*

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/knowledge-maker .
COPY config.yml .
COPY static .

# 暴露端口
EXPOSE 8082

ENV TZ=Asia/Shanghai
ENV GIN_MODE=release

# 设置入口点和命令
CMD ["./knowledge-maker"]