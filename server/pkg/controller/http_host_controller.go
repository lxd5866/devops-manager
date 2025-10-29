package controller

import (
	"net/http"

	"devops-manager/api/protobuf"
	"devops-manager/server/pkg/service"

	"github.com/gin-gonic/gin"
)

// HTTPHostController 主机 HTTP 控制器
type HTTPHostController struct {
	hostService *service.HostService
}

// NewHTTPHostController 创建新的主机 HTTP 控制器
func NewHTTPHostController() *HTTPHostController {
	return &HTTPHostController{
		hostService: service.GetHostService(),
	}
}

// RegisterHostHTTPRoutes 注册主机相关 HTTP 路由
func RegisterHostHTTPRoutes(r *gin.Engine) {
	controller := NewHTTPHostController()

	api := r.Group("/api/v1")
	{
		// 主机管理
		api.POST("/hosts/register", controller.RegisterHost)
		api.GET("/hosts", controller.GetHosts)
		api.GET("/hosts/:id", controller.GetHost)
		api.PUT("/hosts/:id", controller.UpdateHost)
		api.DELETE("/hosts/:id", controller.DeleteHost)

		// 主机状态
		api.POST("/hosts/:id/status", controller.ReportHostStatus)
		api.GET("/hosts/:id/status", controller.GetHostStatus)

		// 准入管理
		api.GET("/pending-hosts", controller.GetPendingHosts)
		api.GET("/pending-hosts/count", controller.GetPendingHostsCount)
		api.POST("/pending-hosts/:id/approve", controller.ApproveHost)
		api.POST("/pending-hosts/:id/reject", controller.RejectHost)
	}
}

// RegisterHost 注册主机
// @Summary      注册新主机
// @Description  注册一个新的主机到系统中
// @Tags         主机管理
// @Accept       json
// @Produce      json
// @Param        host  body      models.HostRegisterRequest  true  "主机信息"
// @Success      200   {object}  models.APIResponse
// @Failure      400   {object}  models.APIResponse
// @Failure      500   {object}  models.APIResponse
// @Router       /hosts/register [post]
func (hc *HTTPHostController) RegisterHost(c *gin.Context) {
	var hostInfo protobuf.HostInfo
	if err := c.ShouldBindJSON(&hostInfo); err != nil {
		SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// 注册主机
	err := hc.hostService.RegisterHost(&hostInfo)
	if err != nil {
		SendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccessResponse(c, gin.H{"assigned_id": hostInfo.Id})
}

// GetHosts 获取所有主机
// @Summary      获取主机列表
// @Description  获取系统中所有已准入的主机信息
// @Tags         主机管理
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.HostListResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /hosts [get]
func (hc *HTTPHostController) GetHosts(c *gin.Context) {
	hosts := hc.hostService.GetAllHosts()
	SendSuccessResponse(c, hosts)
}

// GetHost 获取单个主机
// @Summary      获取主机详情
// @Description  根据主机ID获取主机详细信息
// @Tags         主机管理
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "主机ID"
// @Success      200  {object}  models.APIResponse{data=models.HostInfoResponse}
// @Failure      404  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /hosts/{id} [get]
func (hc *HTTPHostController) GetHost(c *gin.Context) {
	id := c.Param("id")
	host, exists := hc.hostService.GetHost(id)
	if !exists {
		SendErrorResponse(c, http.StatusNotFound, "Host not found")
		return
	}

	SendSuccessResponse(c, host)
}

// UpdateHost 更新主机信息
func (hc *HTTPHostController) UpdateHost(c *gin.Context) {
	id := c.Param("id")

	var hostInfo protobuf.HostInfo
	if err := c.ShouldBindJSON(&hostInfo); err != nil {
		SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	hostInfo.Id = id
	err := hc.hostService.UpdateHost(&hostInfo)
	if err != nil {
		if err == service.ErrHostNotFound {
			SendErrorResponse(c, http.StatusNotFound, "Host not found")
		} else {
			SendErrorResponse(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	SendSuccessResponse(c, &hostInfo)
}

// DeleteHost 删除主机
func (hc *HTTPHostController) DeleteHost(c *gin.Context) {
	id := c.Param("id")

	err := hc.hostService.DeleteHost(id)
	if err != nil {
		if err == service.ErrHostNotFound {
			SendErrorResponse(c, http.StatusNotFound, "Host not found")
		} else {
			SendErrorResponse(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	SendMessageResponse(c, "Host deleted successfully")
}

// GetPendingHosts 获取待准入主机列表
// @Summary      获取待准入主机列表
// @Description  获取所有等待管理员准入的主机列表
// @Tags         主机管理
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.PendingHostListResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /pending-hosts [get]
func (hc *HTTPHostController) GetPendingHosts(c *gin.Context) {
	pendingHosts, err := hc.hostService.GetPendingHosts()
	if err != nil {
		SendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccessResponse(c, pendingHosts)
}

// GetPendingHostsCount 获取待准入主机数量
func (hc *HTTPHostController) GetPendingHostsCount(c *gin.Context) {
	count, err := hc.hostService.GetPendingHostsCount()
	if err != nil {
		SendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccessResponse(c, gin.H{"count": count})
}

// ApproveHost 准入主机
// @Summary      准入主机
// @Description  管理员准入一个待准入的主机
// @Tags         主机管理
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "主机ID"
// @Success      200  {object}  models.APIResponse
// @Failure      404  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /pending-hosts/{id}/approve [post]
func (hc *HTTPHostController) ApproveHost(c *gin.Context) {
	hostID := c.Param("id")

	err := hc.hostService.ApproveHost(hostID)
	if err != nil {
		SendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	SendMessageResponse(c, "Host approved successfully")
}

// RejectHost 拒绝主机准入
func (hc *HTTPHostController) RejectHost(c *gin.Context) {
	hostID := c.Param("id")

	err := hc.hostService.RejectHost(hostID)
	if err != nil {
		SendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	SendMessageResponse(c, "Host rejected successfully")
}

// ReportHostStatus 主机状态上报
func (hc *HTTPHostController) ReportHostStatus(c *gin.Context) {
	hostID := c.Param("id")

	var status protobuf.HostStatus
	if err := c.ShouldBindJSON(&status); err != nil {
		SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// 设置主机ID
	status.HostId = hostID

	// 处理状态上报
	err := hc.hostService.ReportHostStatus(&status)
	if err != nil {
		SendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	SendMessageResponse(c, "Status report received successfully")
}

// GetHostStatus 获取主机状态
func (hc *HTTPHostController) GetHostStatus(c *gin.Context) {
	hostID := c.Param("id")

	status, err := hc.hostService.GetHostStatus(hostID)
	if err != nil {
		SendErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	SendSuccessResponse(c, status)
}
