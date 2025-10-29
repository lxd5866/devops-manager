package controller

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type WebController struct{}

func NewWebController() *WebController {
	return &WebController{}
}

// RegisterWebRoutes 注册 Web 路由
func RegisterWebRoutes(r *gin.Engine) {
	// 添加 CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 静态文件服务 - 优先服务前端构建文件
	r.Static("/assets", filepath.Join("server", "web", "dist", "assets"))
	r.StaticFile("/favicon.ico", filepath.Join("server", "web", "dist", "favicon.ico"))

	// 旧版静态文件（向后兼容）
	r.Static("/static", filepath.Join("server", "web", "static"))

	controller := NewWebController()

	// SPA 路由处理 - 所有非 API 路由都返回 index.html
	r.NoRoute(controller.ServeIndex)
}

// ServeIndex 服务 SPA 应用的 index.html
func (wc *WebController) ServeIndex(c *gin.Context) {
	// 检查是否是 API 请求
	if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
		c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
		return
	}

	// 服务前端应用
	indexPath := filepath.Join("server", "web", "dist", "index.html")
	c.File(indexPath)
}
