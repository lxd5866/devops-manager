package controller

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HTTPController HTTP基础控制器，不实现具体业务逻辑
type HTTPController struct {
	router *gin.Engine
}

// NewHTTPController 创建HTTP控制器
func NewHTTPController() *HTTPController {
	router := gin.Default()

	return &HTTPController{
		router: router,
	}
}

// RegisterRoutes 注册所有HTTP路由
func (hc *HTTPController) RegisterRoutes() {
	// 注册主机相关路由
	RegisterHostHTTPRoutes(hc.router)

	// 注册任务相关路由
	RegisterTaskHTTPRoutes(hc.router)

	// 注册文件相关路由
	RegisterFileHTTPRoutes(hc.router)

	// 注册Web页面路由
	RegisterWebRoutes(hc.router)

	log.Println("All HTTP routes registered successfully")
}

// GetRouter 获取Gin路由器实例
func (hc *HTTPController) GetRouter() *gin.Engine {
	return hc.router
}

// RegisterHostHTTPRoutes 注册主机HTTP路由
func RegisterHostHTTPRoutes(r *gin.Engine) {
	hostController := NewHostHTTPController()

	api := r.Group("/api/v1")
	{
		api.GET("/host/info", hostController.GetHostInfo)
		api.GET("/host/status", hostController.GetHostStatus)
		api.POST("/host/update", hostController.UpdateHostInfo)
	}

	log.Println("Host HTTP routes registered")
}

// RegisterTaskHTTPRoutes 注册任务HTTP路由
func RegisterTaskHTTPRoutes(r *gin.Engine) {
	taskController := NewTaskHTTPController()

	api := r.Group("/api/v1")
	{
		api.POST("/task/execute", taskController.ExecuteTask)
		api.GET("/task/status/:id", taskController.GetTaskStatus)
		api.POST("/task/cancel/:id", taskController.CancelTask)
		api.GET("/task/list", taskController.ListTasks)
	}

	log.Println("Task HTTP routes registered")
}

// RegisterFileHTTPRoutes 注册文件HTTP路由
func RegisterFileHTTPRoutes(r *gin.Engine) {
	fileController := NewFileHTTPController()

	api := r.Group("/api/v1")
	{
		api.POST("/file/upload", fileController.UploadFile)
		api.GET("/file/download/:name", fileController.DownloadFile)
		api.GET("/file/list", fileController.ListFiles)
		api.DELETE("/file/:name", fileController.DeleteFile)
		api.GET("/file/info/:name", fileController.GetFileInfo)
	}

	log.Println("File HTTP routes registered")
}

// RegisterWebRoutes 注册Web页面路由
func RegisterWebRoutes(r *gin.Engine) {
	webController := NewWebController()

	// 静态文件
	r.Static("/static", "./agent/web/static")
	r.LoadHTMLGlob("agent/web/templates/*")

	// Web页面路由
	r.GET("/", webController.Index)
	r.GET("/status", webController.Status)
	r.GET("/tasks", webController.Tasks)
	r.GET("/files", webController.Files)

	log.Println("Web routes registered")
}

// LogHTTPRequest 记录HTTP请求日志
func LogHTTPRequest(c *gin.Context) {
	log.Printf("HTTP Request - Method: %s, Path: %s, IP: %s",
		c.Request.Method, c.Request.URL.Path, c.ClientIP())
}

// LogHTTPResponse 记录HTTP响应日志
func LogHTTPResponse(c *gin.Context, statusCode int, message string) {
	log.Printf("HTTP Response - Status: %d, Path: %s, Message: %s",
		statusCode, c.Request.URL.Path, message)
}

// ErrorResponse 统一错误响应格式
func ErrorResponse(c *gin.Context, code int, message string) {
	LogHTTPResponse(c, code, message)
	c.JSON(code, gin.H{
		"error":   true,
		"message": message,
		"code":    code,
	})
}

// SuccessResponse 统一成功响应格式
func SuccessResponse(c *gin.Context, data interface{}) {
	LogHTTPResponse(c, http.StatusOK, "success")
	c.JSON(http.StatusOK, gin.H{
		"error": false,
		"data":  data,
	})
}
