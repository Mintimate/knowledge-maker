package main

import (
	"log"
	"net/http"

	"knowledge-maker/internal/config"
	"knowledge-maker/internal/handler"
	"knowledge-maker/internal/service"

	"github.com/gin-gonic/gin"
)

// setupCommonRoutes 设置常见的路由处理，避免产生大量404日志
func setupCommonRoutes(r *gin.Engine) {
	// 根路径 - 服务信息
	r.GET("/", handleRoot)

	// robots.txt - 搜索引擎爬虫规则
	r.GET("/robots.txt", handleRobots)

	// favicon.ico - 浏览器图标请求
	r.GET("/favicon.ico", handleFavicon)

	// favicon.svg - SVG 图标
	r.GET("/favicon.svg", handleFaviconSVG)
}

// handleRoot 处理根路径请求
func handleRoot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":     "知识库 RAG 服务",
		"version":     "v2.1.0",
		"status":      "running",
		"description": "基于 RAG 技术的智能问答服务",
		"endpoints": gin.H{
			"health": "/api/v1/health",
			"chat":   "/api/v1/chat",
			"stream": "/api/v1/chat/stream",
		},
	})
}

// handleRobots 处理 robots.txt 请求
func handleRobots(c *gin.Context) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, `User-agent: *
Disallow: /
Allow: /api/v1/health

# 知识库 RAG 服务
# 仅允许访问健康检查接口`)
}

// handleFavicon 处理 favicon.ico 请求
func handleFavicon(c *gin.Context) {
	// 重定向到 SVG favicon
	c.Redirect(http.StatusMovedPermanently, "/favicon.svg")
}

// handleFaviconSVG 提供 SVG favicon
func handleFaviconSVG(c *gin.Context) {
	c.Header("Content-Type", "image/svg+xml")
	c.Header("Cache-Control", "public, max-age=86400") // 缓存1天
	c.File("static/favicon.svg")
}

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatal("加载配置失败:", err)
	}

	// 设置 Gin 模式
	gin.SetMode(cfg.Server.Mode)

	// 创建 Gin 路由器
	r := gin.Default()

	// 添加 CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// 注册通用路由
	setupCommonRoutes(r)

	// 初始化服务
	knowledgeService := service.NewKnowledgeService(cfg)
	aiService := service.NewAIService(cfg)
	ragService := service.NewRAGService(knowledgeService, aiService, cfg)

	// 初始化处理器
	ragHandler := handler.NewRAGHandler(ragService)

	// 注册路由
	api := r.Group("/api/v1")
	{
		api.POST("/chat", ragHandler.HandleChat)
		api.POST("/chat/stream", ragHandler.HandleStreamChat)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"message": "知识库 RAG 服务运行正常",
			})
		})
	}

	// 启动服务器
	log.Printf("服务器启动在端口 :%s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
