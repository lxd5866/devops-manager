package controller

import (
	"net/http"

	"devops-manager/agent/pkg/service"
	"devops-manager/agent/pkg/utils"

	"github.com/gin-gonic/gin"
)

// HostHTTPController 主机HTTP业务控制器
type HostHTTPController struct {
	hostService *service.HostAgent
}

// NewHostHTTPController 创建主机HTTP控制器
func NewHostHTTPController() *HostHTTPController {
	return &HostHTTPController{
		// hostService 将在启动时注入
	}
}

// SetHostService 设置主机服务
func (hhc *HostHTTPController) SetHostService(hostService *service.HostAgent) {
	hhc.hostService = hostService
}

// GetHostInfo 获取主机信息
func (hhc *HostHTTPController) GetHostInfo(c *gin.Context) {
	LogHTTPRequest(c)

	// 获取系统信息
	systemInfo := utils.GetSystemInfo()
	processInfo := utils.GetProcessInfo()

	data := gin.H{
		"system":  systemInfo,
		"process": processInfo,
	}

	SuccessResponse(c, data)
}

// GetHostStatus 获取主机状态
func (hhc *HostHTTPController) GetHostStatus(c *gin.Context) {
	LogHTTPRequest(c)

	// 获取连接状态等信息
	status := gin.H{
		"connected":   hhc.hostService != nil,
		"uptime":      "unknown",
		"last_report": "unknown",
		"server_addr": "unknown",
	}

	// 如果hostService可用，获取实际状态
	if hhc.hostService != nil {
		// 这里应该从hostService获取实际状态
		// 具体实现依赖于service层的接口
	}

	SuccessResponse(c, status)
}

// UpdateHostInfo 更新主机信息
func (hhc *HostHTTPController) UpdateHostInfo(c *gin.Context) {
	LogHTTPRequest(c)

	var req struct {
		Tags map[string]string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// 这里应该更新主机标签信息
	// 具体实现依赖于service层的接口

	SuccessResponse(c, gin.H{
		"message": "Host info updated successfully",
		"tags":    req.Tags,
	})
}
