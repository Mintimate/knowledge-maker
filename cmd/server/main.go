package main

import (
	"log"
	"net/http"

	"knowledge-maker/internal/config"
	"knowledge-maker/internal/handler"
	"knowledge-maker/internal/service"

	"github.com/gin-gonic/gin"
)

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
