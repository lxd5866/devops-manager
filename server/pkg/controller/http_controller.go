package controller

import (
	"net/http"

	"devops-manager/api/protobuf"
	"devops-manager/server/pkg/service"

	"github.com/gin-gonic/gin"
)

// HTTPController HTTP 控制器基础结构
type HTTPController struct {
	hostService *service.HostService
}

// NewHTTPController 创建新的 HTTP 控制器
func NewHTTPController() *HTTPController {
	return &HTTPController{
		hostService: service.GetHostService(),
	}
}

// RegisterHTTPRoutes 注册所有 HTTP API 路由
func RegisterHTTPRoutes(r *gin.Engine) {
	// 注册主机相关路由
	RegisterHostHTTPRoutes(r)

	// 注册任务相关路由
	RegisterTaskHTTPRoutes(r)

	// 注册命令相关路由
	RegisterCommandHTTPRoutes(r)
}

// RegisterCommandHTTPRoutes 注册命令相关路由
func RegisterCommandHTTPRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		// 命令管理
		api.POST("/commands", nil)             // 创建命令
		api.GET("/commands", nil)              // 获取命令列表
		api.GET("/commands/:id", nil)          // 获取单个命令
		api.PUT("/commands/:id", nil)          // 更新命令
		api.DELETE("/commands/:id", nil)       // 删除命令
		api.POST("/commands/:id/execute", nil) // 执行命令
		api.GET("/commands/:id/result", nil)   // 获取命令结果
	}
}

// CommonResponse 通用响应结构
type CommonResponse struct {
	Success      bool        `json:"success"`
	Data         interface{} `json:"data,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
	Message      string      `json:"message,omitempty"`
}

// SendSuccessResponse 发送成功响应
func SendSuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, CommonResponse{
		Success: true,
		Data:    data,
	})
}

// SendErrorResponse 发送错误响应
func SendErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, CommonResponse{
		Success:      false,
		ErrorMessage: message,
	})
}

// SendMessageResponse 发送消息响应
func SendMessageResponse(c *gin.Context, message string) {
	c.JSON(http.StatusOK, CommonResponse{
		Success: true,
		Message: message,
	})
}

// RegisterHost 注册主机
func (hc *HTTPController) RegisterHost(c *gin.Context) {
	var hostInfo protobuf.HostInfo
	if err := c.ShouldBindJSON(&hostInfo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":       false,
			"error_message": err.Error(),
		})
		return
	}

	// 注册主机
	err := hc.hostService.RegisterHost(&hostInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":       false,
			"error_message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"assigned_id": hostInfo.Id,
	})
}

// GetHosts 获取所有主机
func (hc *HTTPController) GetHosts(c *gin.Context) {
	hosts := hc.hostService.GetAllHosts()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    hosts,
	})
}

// GetHost 获取单个主机
func (hc *HTTPController) GetHost(c *gin.Context) {
	id := c.Param("id")
	host, exists := hc.hostService.GetHost(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success":       false,
			"error_message": "Host not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    host,
	})
}

// UpdateHost 更新主机信息
func (hc *HTTPController) UpdateHost(c *gin.Context) {
	id := c.Param("id")

	var hostInfo protobuf.HostInfo
	if err := c.ShouldBindJSON(&hostInfo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":       false,
			"error_message": err.Error(),
		})
		return
	}

	hostInfo.Id = id
	err := hc.hostService.UpdateHost(&hostInfo)
	if err != nil {
		if err == service.ErrHostNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success":       false,
				"error_message": "Host not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success":       false,
				"error_message": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &hostInfo,
	})
}

// DeleteHost 删除主机
func (hc *HTTPController) DeleteHost(c *gin.Context) {
	id := c.Param("id")

	err := hc.hostService.DeleteHost(id)
	if err != nil {
		if err == service.ErrHostNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success":       false,
				"error_message": "Host not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success":       false,
				"error_message": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// GetPendingHosts 获取待准入主机列表
func (hc *HTTPController) GetPendingHosts(c *gin.Context) {
	pendingHosts, err := hc.hostService.GetPendingHosts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":       false,
			"error_message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    pendingHosts,
	})
}

// GetPendingHostsCount 获取待准入主机数量
func (hc *HTTPController) GetPendingHostsCount(c *gin.Context) {
	count, err := hc.hostService.GetPendingHostsCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":       false,
			"error_message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"count": count,
		},
	})
}

// ApproveHost 准入主机
func (hc *HTTPController) ApproveHost(c *gin.Context) {
	hostID := c.Param("id")

	err := hc.hostService.ApproveHost(hostID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":       false,
			"error_message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Host approved successfully",
	})
}

// RejectHost 拒绝主机准入
func (hc *HTTPController) RejectHost(c *gin.Context) {
	hostID := c.Param("id")

	err := hc.hostService.RejectHost(hostID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":       false,
			"error_message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Host rejected successfully",
	})
}

// ReportHostStatus 主机状态上报
func (hc *HTTPController) ReportHostStatus(c *gin.Context) {
	hostID := c.Param("id")

	var status protobuf.HostStatus
	if err := c.ShouldBindJSON(&status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":       false,
			"error_message": err.Error(),
		})
		return
	}

	// 设置主机ID
	status.HostId = hostID

	// 处理状态上报
	err := hc.hostService.ReportHostStatus(&status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":       false,
			"error_message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Status report received successfully",
	})
}

// GetHostStatus 获取主机状态
func (hc *HTTPController) GetHostStatus(c *gin.Context) {
	hostID := c.Param("id")

	status, err := hc.hostService.GetHostStatus(hostID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":       false,
			"error_message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}
