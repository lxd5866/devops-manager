package controller

import (
	"net/http"
	"strconv"

	"devops-manager/server/pkg/models"
	"devops-manager/server/pkg/service"

	"github.com/gin-gonic/gin"
)

// HTTPTaskController 任务 HTTP 控制器
type HTTPTaskController struct {
	taskService *service.TaskService
}

// NewHTTPTaskController 创建新的任务 HTTP 控制器
func NewHTTPTaskController() *HTTPTaskController {
	return &HTTPTaskController{
		taskService: service.GetTaskService(),
	}
}

// RegisterTaskHTTPRoutes 注册任务相关 HTTP 路由
func RegisterTaskHTTPRoutes(r *gin.Engine) {
	controller := NewHTTPTaskController()

	api := r.Group("/api/v1")
	{
		// 任务管理
		api.POST("/tasks", controller.CreateTask)
		api.GET("/tasks", controller.GetTasks)
		api.GET("/tasks/:id", controller.GetTask)
	}
}

// CreateTask 创建任务
// @Summary      创建新任务
// @Description  创建一个新的执行任务
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        task  body      models.CreateTaskRequest  true  "任务信息"
// @Success      200   {object}  models.APIResponse{data=models.TaskResponse}
// @Failure      400   {object}  models.APIResponse
// @Failure      500   {object}  models.APIResponse
// @Router       /tasks [post]
func (tc *HTTPTaskController) CreateTask(c *gin.Context) {
	LogGRPCRequest("CreateTask", c.Request.Method+" "+c.Request.URL.Path)

	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		LogGRPCResponse("CreateTask", false, "Invalid request body: "+err.Error())
		SendErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// 验证请求参数
	if req.Name == "" {
		LogGRPCResponse("CreateTask", false, "Task name is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task name is required")
		return
	}

	if len(req.HostIDs) == 0 {
		LogGRPCResponse("CreateTask", false, "At least one host is required")
		SendErrorResponse(c, http.StatusBadRequest, "At least one host is required")
		return
	}

	if req.Command == "" {
		LogGRPCResponse("CreateTask", false, "Command is required")
		SendErrorResponse(c, http.StatusBadRequest, "Command is required")
		return
	}

	// 创建任务
	task, err := tc.taskService.CreateTask(
		req.Name,
		req.Description,
		req.HostIDs,
		req.Command,
		req.Timeout,
		req.Parameters,
		"admin", // TODO: 从认证信息中获取用户
	)

	if err != nil {
		LogGRPCResponse("CreateTask", false, "Failed to create task: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to create task: "+err.Error())
		return
	}

	// 构建响应
	response := models.TaskResponse{
		ID:             task.ID,
		TaskID:         task.TaskID,
		Name:           task.Name,
		Description:    task.Description,
		Status:         string(task.Status),
		TotalHosts:     task.TotalHosts,
		CompletedHosts: task.CompletedHosts,
		FailedHosts:    task.FailedHosts,
		CreatedBy:      task.CreatedBy,
		CreatedAt:      task.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      task.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	LogGRPCResponse("CreateTask", true, "Task created successfully: "+task.TaskID)
	SendSuccessResponse(c, response)
}

// GetTasks 获取任务列表
// @Summary      获取任务列表
// @Description  获取系统中的任务列表，支持分页和筛选
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        page    query     int     false  "页码"        default(1)
// @Param        size    query     int     false  "每页数量"     default(20)
// @Param        status  query     string  false  "任务状态筛选"
// @Param        name    query     string  false  "任务名称筛选"
// @Success      200     {object}  models.TaskListResponse
// @Failure      500     {object}  models.APIResponse
// @Router       /tasks [get]
func (tc *HTTPTaskController) GetTasks(c *gin.Context) {
	LogGRPCRequest("GetTasks", c.Request.Method+" "+c.Request.URL.Path)

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.Query("status")
	name := c.Query("name")

	// 参数验证
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	// 获取任务列表
	tasks, total, err := tc.taskService.GetTasks(page, size, status, name)
	if err != nil {
		LogGRPCResponse("GetTasks", false, "Failed to get tasks: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to get tasks: "+err.Error())
		return
	}

	// 构建响应
	var taskResponses []models.TaskResponse
	for _, task := range tasks {
		taskResponses = append(taskResponses, models.TaskResponse{
			ID:             task.ID,
			TaskID:         task.TaskID,
			Name:           task.Name,
			Description:    task.Description,
			Status:         string(task.Status),
			TotalHosts:     task.TotalHosts,
			CompletedHosts: task.CompletedHosts,
			FailedHosts:    task.FailedHosts,
			CreatedBy:      task.CreatedBy,
			CreatedAt:      task.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:      task.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	response := gin.H{
		"tasks": taskResponses,
		"pagination": gin.H{
			"page":  page,
			"size":  size,
			"total": total,
		},
	}

	LogGRPCResponse("GetTasks", true, "Retrieved "+strconv.Itoa(len(tasks))+" tasks")
	SendSuccessResponse(c, response)
}

// GetTask 获取单个任务
// @Summary      获取任务详情
// @Description  根据任务ID获取任务详细信息
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "任务ID"
// @Success      200  {object}  models.APIResponse{data=models.TaskResponse}
// @Failure      404  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/{id} [get]
func (tc *HTTPTaskController) GetTask(c *gin.Context) {
	LogGRPCRequest("GetTask", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	if taskID == "" {
		LogGRPCResponse("GetTask", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	task, err := tc.taskService.GetTask(taskID)
	if err != nil {
		LogGRPCResponse("GetTask", false, "Task not found: "+err.Error())
		SendErrorResponse(c, http.StatusNotFound, "Task not found: "+err.Error())
		return
	}

	response := models.TaskResponse{
		ID:             task.ID,
		TaskID:         task.TaskID,
		Name:           task.Name,
		Description:    task.Description,
		Status:         string(task.Status),
		TotalHosts:     task.TotalHosts,
		CompletedHosts: task.CompletedHosts,
		FailedHosts:    task.FailedHosts,
		CreatedBy:      task.CreatedBy,
		CreatedAt:      task.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      task.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	LogGRPCResponse("GetTask", true, "Task retrieved: "+taskID)
	SendSuccessResponse(c, response)
}
