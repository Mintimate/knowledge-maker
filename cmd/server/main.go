package main

import (
	"log"
	"net/http"
	"strings"

	"knowledge-maker/internal/config"
	"knowledge-maker/internal/handler"
	"knowledge-maker/internal/logger"
	"knowledge-maker/internal/middleware"
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
		"version":     "v3.0.0",
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
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志系统
	if err := logger.Init(&cfg.Log); err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}
	defer logger.Close()

	logger.Info("应用启动中...")
	logger.Info("配置加载完成 - 服务端口: %s, 模式: %s", cfg.Server.Port, cfg.Server.Mode)

	// 设置 Gin 模式
	gin.SetMode(cfg.Server.Mode)

	// 创建 Gin 路由器
	r := gin.Default()

	// 设置信任的代理以解决 GIN 代理警告
	trustedProxies := []string{"127.0.0.1", "::1"}
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		logger.Warn("设置信任代理失败: %v", err)
	}

	// 添加 CORS 中间件
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowOrigin := "*"

		// 检查域名配置
		if len(cfg.Server.AllowDomains) > 0 {
			for _, domain := range cfg.Server.AllowDomains {
				if domain != "" && (origin == domain || origin == strings.TrimSuffix(domain, "/")) {
					allowOrigin = origin
					break
				}
			}
		}

		c.Header("Access-Control-Allow-Origin", allowOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Captcha-Ticket, X-Captcha-Randstr, X-Geetest-Lot-Number, X-Geetest-Captcha-Output, X-Geetest-Pass-Token, X-Geetest-Gen-Time, X-Recaptcha-Token, X-Recaptcha-Action, X-Cf-Turnstile-Token")

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

	// 初始化验证码服务
	captchaService, err := service.NewCaptchaService(&cfg.Captcha)
	if err != nil {
		logger.Warn("验证码服务初始化失败，将跳过验证码验证: %v", err)
		captchaService = nil
	} else {
		logger.Info("验证码服务初始化成功")
	}

	// 初始化处理器
	ragHandler := handler.NewRAGHandler(ragService)

	// 初始化验证码中间件
	captchaMiddleware := middleware.NewCaptchaMiddleware(captchaService)

	// 注册路由
	api := r.Group("/api/v1")
	{
		// 使用验证码中间件保护聊天接口
		api.POST("/chat", captchaMiddleware.VerifyCaptcha(), ragHandler.HandleChat)
		api.POST("/chat/stream", captchaMiddleware.VerifyCaptcha(), ragHandler.HandleStreamChat)
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
