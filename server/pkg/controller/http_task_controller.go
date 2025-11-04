package controller

import (
	"net/http"
	"strconv"
	"time"

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

		// 任务状态监控
		api.GET("/tasks/:id/status", controller.GetTaskStatus)
		api.GET("/tasks/:id/progress", controller.GetTaskProgress)

		// 任务控制
		api.POST("/tasks/:id/start", controller.StartTask)
		api.POST("/tasks/:id/stop", controller.StopTask)
		api.POST("/tasks/:id/cancel", controller.CancelTask)

		// 任务统计和报告
		api.GET("/tasks/statistics", controller.GetTaskStatistics)
		api.GET("/tasks/execution-statistics", controller.GetExecutionStatistics)
		api.GET("/tasks/audit-summary", controller.GetAuditSummary)
		api.GET("/tasks/log-statistics", controller.GetLogStatistics)
		api.GET("/tasks/by-host/:hostId", controller.GetTasksByHost)
		api.GET("/tasks/by-status/:status", controller.GetTasksByStatus)
		api.GET("/tasks/by-date", controller.GetTasksByDateRange)

		// 任务主机管理
		api.GET("/tasks/:id/hosts", controller.GetTaskHosts)
		api.POST("/tasks/:id/hosts", controller.AddTaskHosts)
		api.DELETE("/tasks/:id/hosts/:hostId", controller.RemoveTaskHost)

		// 任务日志和详情
		api.GET("/tasks/:id/logs", controller.GetTaskLogs)
		api.GET("/tasks/:id/logs/detailed", controller.GetDetailedTaskLogs)
		api.GET("/tasks/:id/audit", controller.GetTaskAuditTrail)
		api.GET("/tasks/:id/timeline", controller.GetTaskExecutionTimeline)
		api.GET("/tasks/:id/summary", controller.GetTaskExecutionSummary)

		// 异常处理和超时管理
		api.GET("/tasks/failed-commands", controller.GetFailedCommands)
		api.POST("/tasks/commands/:commandId/retry", controller.RetryFailedCommand)
		api.POST("/tasks/commands/:commandId/check-timeout", controller.CheckCommandTimeout)
		api.GET("/tasks/timeout-statistics", controller.GetTimeoutStatistics)
		api.GET("/tasks/error-statistics", controller.GetErrorStatistics)

		// 数据库优化和维护
		api.GET("/tasks/database-statistics", controller.GetDatabaseStatistics)
		api.POST("/tasks/cleanup-old-records", controller.CleanupOldRecords)
		api.POST("/tasks/cleanup-old-logs", controller.CleanupOldLogs)
		api.POST("/tasks/optimize-tables", controller.OptimizeTables)

		// 日志搜索和分析
		api.GET("/tasks/search-logs", controller.SearchLogs)
		api.POST("/tasks/update-daily-statistics", controller.UpdateDailyStatistics)
		api.GET("/tasks/table-sizes", controller.AnalyzeTableSizes)
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

// GetTaskStatus 获取任务状态
// @Summary      获取任务状态
// @Description  获取任务的详细状态信息，包括执行进度和统计数据
// @Tags         任务监控
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "任务ID"
// @Success      200  {object}  models.APIResponse
// @Failure      404  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/{id}/status [get]
func (tc *HTTPTaskController) GetTaskStatus(c *gin.Context) {
	LogGRPCRequest("GetTaskStatus", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	if taskID == "" {
		LogGRPCResponse("GetTaskStatus", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	status, err := tc.taskService.GetTaskStatus(taskID)
	if err != nil {
		LogGRPCResponse("GetTaskStatus", false, "Failed to get task status: "+err.Error())
		SendErrorResponse(c, http.StatusNotFound, "Failed to get task status: "+err.Error())
		return
	}

	LogGRPCResponse("GetTaskStatus", true, "Task status retrieved: "+taskID)
	SendSuccessResponse(c, status)
}

// GetTaskProgress 获取任务进度
// @Summary      获取任务进度
// @Description  获取任务的详细进度信息，包括各主机的执行状态
// @Tags         任务监控
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "任务ID"
// @Success      200  {object}  models.APIResponse
// @Failure      404  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/{id}/progress [get]
func (tc *HTTPTaskController) GetTaskProgress(c *gin.Context) {
	LogGRPCRequest("GetTaskProgress", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	if taskID == "" {
		LogGRPCResponse("GetTaskProgress", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	progress, err := tc.taskService.GetTaskProgress(taskID)
	if err != nil {
		LogGRPCResponse("GetTaskProgress", false, "Failed to get task progress: "+err.Error())
		SendErrorResponse(c, http.StatusNotFound, "Failed to get task progress: "+err.Error())
		return
	}

	LogGRPCResponse("GetTaskProgress", true, "Task progress retrieved: "+taskID)
	SendSuccessResponse(c, progress)
}

// StartTask 启动任务
// @Summary      启动任务
// @Description  启动指定的任务，开始向目标主机下发命令
// @Tags         任务控制
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "任务ID"
// @Success      200  {object}  models.APIResponse
// @Failure      400  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/{id}/start [post]
func (tc *HTTPTaskController) StartTask(c *gin.Context) {
	LogGRPCRequest("StartTask", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	if taskID == "" {
		LogGRPCResponse("StartTask", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	err := tc.taskService.StartTask(taskID)
	if err != nil {
		LogGRPCResponse("StartTask", false, "Failed to start task: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to start task: "+err.Error())
		return
	}

	LogGRPCResponse("StartTask", true, "Task started: "+taskID)
	SendSuccessResponse(c, gin.H{"message": "Task started successfully"})
}

// StopTask 停止任务
// @Summary      停止任务
// @Description  停止正在运行的任务
// @Tags         任务控制
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "任务ID"
// @Success      200  {object}  models.APIResponse
// @Failure      400  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/{id}/stop [post]
func (tc *HTTPTaskController) StopTask(c *gin.Context) {
	LogGRPCRequest("StopTask", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	if taskID == "" {
		LogGRPCResponse("StopTask", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	err := tc.taskService.StopTask(taskID)
	if err != nil {
		LogGRPCResponse("StopTask", false, "Failed to stop task: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to stop task: "+err.Error())
		return
	}

	LogGRPCResponse("StopTask", true, "Task stopped: "+taskID)
	SendSuccessResponse(c, gin.H{"message": "Task stopped successfully"})
}

// CancelTask 取消任务
// @Summary      取消任务
// @Description  取消任务执行，包括正在运行和待执行的命令
// @Tags         任务控制
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "任务ID"
// @Success      200  {object}  models.APIResponse
// @Failure      400  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/{id}/cancel [post]
func (tc *HTTPTaskController) CancelTask(c *gin.Context) {
	LogGRPCRequest("CancelTask", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	if taskID == "" {
		LogGRPCResponse("CancelTask", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	err := tc.taskService.CancelTask(taskID)
	if err != nil {
		LogGRPCResponse("CancelTask", false, "Failed to cancel task: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to cancel task: "+err.Error())
		return
	}

	LogGRPCResponse("CancelTask", true, "Task canceled: "+taskID)
	SendSuccessResponse(c, gin.H{"message": "Task canceled successfully"})
}

// GetTaskStatistics 获取任务统计信息
// @Summary      获取任务统计信息
// @Description  获取系统任务的统计信息，包括状态分布、执行统计等
// @Tags         任务统计
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/statistics [get]
func (tc *HTTPTaskController) GetTaskStatistics(c *gin.Context) {
	LogGRPCRequest("GetTaskStatistics", c.Request.Method+" "+c.Request.URL.Path)

	statistics, err := tc.taskService.GetTaskStatistics()
	if err != nil {
		LogGRPCResponse("GetTaskStatistics", false, "Failed to get task statistics: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to get task statistics: "+err.Error())
		return
	}

	LogGRPCResponse("GetTaskStatistics", true, "Task statistics retrieved")
	SendSuccessResponse(c, statistics)
}

// GetTasksByHost 按主机筛选任务
// @Summary      按主机筛选任务
// @Description  获取指定主机相关的任务列表
// @Tags         任务查询
// @Accept       json
// @Produce      json
// @Param        hostId  path      string  true   "主机ID"
// @Param        page    query     int     false  "页码"        default(1)
// @Param        size    query     int     false  "每页数量"     default(20)
// @Param        status  query     string  false  "任务状态筛选"
// @Success      200     {object}  models.TaskListResponse
// @Failure      500     {object}  models.APIResponse
// @Router       /tasks/by-host/{hostId} [get]
func (tc *HTTPTaskController) GetTasksByHost(c *gin.Context) {
	LogGRPCRequest("GetTasksByHost", c.Request.Method+" "+c.Request.URL.Path)

	hostID := c.Param("hostId")
	if hostID == "" {
		LogGRPCResponse("GetTasksByHost", false, "Host ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Host ID is required")
		return
	}

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.Query("status")

	// 参数验证
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	tasks, total, err := tc.taskService.GetTasksByHost(hostID, page, size, status)
	if err != nil {
		LogGRPCResponse("GetTasksByHost", false, "Failed to get tasks by host: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to get tasks by host: "+err.Error())
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
		"host_id": hostID,
	}

	LogGRPCResponse("GetTasksByHost", true, "Retrieved "+strconv.Itoa(len(tasks))+" tasks for host "+hostID)
	SendSuccessResponse(c, response)
}

// GetTasksByStatus 按状态筛选任务
// @Summary      按状态筛选任务
// @Description  获取指定状态的任务列表
// @Tags         任务查询
// @Accept       json
// @Produce      json
// @Param        status  path      string  true   "任务状态"
// @Param        page    query     int     false  "页码"        default(1)
// @Param        size    query     int     false  "每页数量"     default(20)
// @Success      200     {object}  models.TaskListResponse
// @Failure      500     {object}  models.APIResponse
// @Router       /tasks/by-status/{status} [get]
func (tc *HTTPTaskController) GetTasksByStatus(c *gin.Context) {
	LogGRPCRequest("GetTasksByStatus", c.Request.Method+" "+c.Request.URL.Path)

	status := c.Param("status")
	if status == "" {
		LogGRPCResponse("GetTasksByStatus", false, "Status is required")
		SendErrorResponse(c, http.StatusBadRequest, "Status is required")
		return
	}

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	// 参数验证
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	tasks, total, err := tc.taskService.GetTasksByStatus(status, page, size)
	if err != nil {
		LogGRPCResponse("GetTasksByStatus", false, "Failed to get tasks by status: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to get tasks by status: "+err.Error())
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
		"status": status,
	}

	LogGRPCResponse("GetTasksByStatus", true, "Retrieved "+strconv.Itoa(len(tasks))+" tasks with status "+status)
	SendSuccessResponse(c, response)
}

// GetTasksByDateRange 按日期范围筛选任务
// @Summary      按日期范围筛选任务
// @Description  获取指定日期范围内的任务列表
// @Tags         任务查询
// @Accept       json
// @Produce      json
// @Param        start_date  query     string  true   "开始日期 (YYYY-MM-DD)"
// @Param        end_date    query     string  true   "结束日期 (YYYY-MM-DD)"
// @Param        page        query     int     false  "页码"        default(1)
// @Param        size        query     int     false  "每页数量"     default(20)
// @Param        status      query     string  false  "任务状态筛选"
// @Success      200         {object}  models.TaskListResponse
// @Failure      400         {object}  models.APIResponse
// @Failure      500         {object}  models.APIResponse
// @Router       /tasks/by-date [get]
func (tc *HTTPTaskController) GetTasksByDateRange(c *gin.Context) {
	LogGRPCRequest("GetTasksByDateRange", c.Request.Method+" "+c.Request.URL.Path)

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		LogGRPCResponse("GetTasksByDateRange", false, "start_date and end_date are required")
		SendErrorResponse(c, http.StatusBadRequest, "start_date and end_date are required")
		return
	}

	// 解析日期
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		LogGRPCResponse("GetTasksByDateRange", false, "Invalid start_date format: "+err.Error())
		SendErrorResponse(c, http.StatusBadRequest, "Invalid start_date format, use YYYY-MM-DD")
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		LogGRPCResponse("GetTasksByDateRange", false, "Invalid end_date format: "+err.Error())
		SendErrorResponse(c, http.StatusBadRequest, "Invalid end_date format, use YYYY-MM-DD")
		return
	}

	// 设置时间范围为全天
	endDate = endDate.Add(24*time.Hour - time.Nanosecond)

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.Query("status")

	// 参数验证
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	tasks, total, err := tc.taskService.GetTasksByDateRange(startDate, endDate, page, size, status)
	if err != nil {
		LogGRPCResponse("GetTasksByDateRange", false, "Failed to get tasks by date range: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to get tasks by date range: "+err.Error())
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
		"date_range": gin.H{
			"start_date": startDateStr,
			"end_date":   endDateStr,
		},
	}

	LogGRPCResponse("GetTasksByDateRange", true, "Retrieved "+strconv.Itoa(len(tasks))+" tasks in date range")
	SendSuccessResponse(c, response)
}

// GetTaskHosts 获取任务主机列表
// @Summary      获取任务主机列表
// @Description  获取任务关联的所有主机及其执行状态
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "任务ID"
// @Success      200  {object}  models.APIResponse
// @Failure      404  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/{id}/hosts [get]
func (tc *HTTPTaskController) GetTaskHosts(c *gin.Context) {
	LogGRPCRequest("GetTaskHosts", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	if taskID == "" {
		LogGRPCResponse("GetTaskHosts", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	hosts, err := tc.taskService.GetTaskHosts(taskID)
	if err != nil {
		LogGRPCResponse("GetTaskHosts", false, "Failed to get task hosts: "+err.Error())
		SendErrorResponse(c, http.StatusNotFound, "Failed to get task hosts: "+err.Error())
		return
	}

	LogGRPCResponse("GetTaskHosts", true, "Task hosts retrieved: "+taskID)
	SendSuccessResponse(c, hosts)
}

// AddTaskHosts 添加任务主机
// @Summary      添加任务主机
// @Description  向现有任务添加新的目标主机
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id       path      string                    true  "任务ID"
// @Param        hosts    body      models.AddTaskHostsRequest true  "主机列表"
// @Success      200      {object}  models.APIResponse
// @Failure      400      {object}  models.APIResponse
// @Failure      500      {object}  models.APIResponse
// @Router       /tasks/{id}/hosts [post]
func (tc *HTTPTaskController) AddTaskHosts(c *gin.Context) {
	LogGRPCRequest("AddTaskHosts", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	if taskID == "" {
		LogGRPCResponse("AddTaskHosts", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	var req struct {
		HostIDs []string `json:"host_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		LogGRPCResponse("AddTaskHosts", false, "Invalid request body: "+err.Error())
		SendErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if len(req.HostIDs) == 0 {
		LogGRPCResponse("AddTaskHosts", false, "At least one host ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "At least one host ID is required")
		return
	}

	err := tc.taskService.AddTaskHosts(taskID, req.HostIDs)
	if err != nil {
		LogGRPCResponse("AddTaskHosts", false, "Failed to add task hosts: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to add task hosts: "+err.Error())
		return
	}

	LogGRPCResponse("AddTaskHosts", true, "Task hosts added: "+taskID)
	SendSuccessResponse(c, gin.H{"message": "Hosts added successfully"})
}

// RemoveTaskHost 移除任务主机
// @Summary      移除任务主机
// @Description  从任务中移除指定的主机
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        id      path      string  true  "任务ID"
// @Param        hostId  path      string  true  "主机ID"
// @Success      200     {object}  models.APIResponse
// @Failure      400     {object}  models.APIResponse
// @Failure      500     {object}  models.APIResponse
// @Router       /tasks/{id}/hosts/{hostId} [delete]
func (tc *HTTPTaskController) RemoveTaskHost(c *gin.Context) {
	LogGRPCRequest("RemoveTaskHost", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	hostID := c.Param("hostId")

	if taskID == "" || hostID == "" {
		LogGRPCResponse("RemoveTaskHost", false, "Task ID and Host ID are required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID and Host ID are required")
		return
	}

	err := tc.taskService.RemoveTaskHost(taskID, hostID)
	if err != nil {
		LogGRPCResponse("RemoveTaskHost", false, "Failed to remove task host: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to remove task host: "+err.Error())
		return
	}

	LogGRPCResponse("RemoveTaskHost", true, "Task host removed: "+taskID+"/"+hostID)
	SendSuccessResponse(c, gin.H{"message": "Host removed successfully"})
}

// GetTaskLogs 获取任务日志
// @Summary      获取任务日志
// @Description  获取任务执行的详细日志信息
// @Tags         任务监控
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "任务ID"
// @Success      200  {object}  models.APIResponse
// @Failure      404  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/{id}/logs [get]
func (tc *HTTPTaskController) GetTaskLogs(c *gin.Context) {
	LogGRPCRequest("GetTaskLogs", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	if taskID == "" {
		LogGRPCResponse("GetTaskLogs", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	logs, err := tc.taskService.GetTaskLogs(taskID)
	if err != nil {
		LogGRPCResponse("GetTaskLogs", false, "Failed to get task logs: "+err.Error())
		SendErrorResponse(c, http.StatusNotFound, "Failed to get task logs: "+err.Error())
		return
	}

	LogGRPCResponse("GetTaskLogs", true, "Task logs retrieved: "+taskID)
	SendSuccessResponse(c, logs)
}

// GetTaskExecutionSummary 获取任务执行摘要
// @Summary      获取任务执行摘要
// @Description  获取任务执行的详细摘要报告，包括统计信息和错误分析
// @Tags         任务报告
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "任务ID"
// @Success      200  {object}  models.APIResponse
// @Failure      404  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/{id}/summary [get]
func (tc *HTTPTaskController) GetTaskExecutionSummary(c *gin.Context) {
	LogGRPCRequest("GetTaskExecutionSummary", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	if taskID == "" {
		LogGRPCResponse("GetTaskExecutionSummary", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	summary, err := tc.taskService.GetTaskExecutionSummary(taskID)
	if err != nil {
		LogGRPCResponse("GetTaskExecutionSummary", false, "Failed to get task execution summary: "+err.Error())
		SendErrorResponse(c, http.StatusNotFound, "Failed to get task execution summary: "+err.Error())
		return
	}

	LogGRPCResponse("GetTaskExecutionSummary", true, "Task execution summary retrieved: "+taskID)
	SendSuccessResponse(c, summary)
}

// GetFailedCommands 获取失败的命令列表
// @Summary      获取失败的命令列表
// @Description  获取系统中执行失败的命令列表，支持分页和主机筛选
// @Tags         异常处理
// @Accept       json
// @Produce      json
// @Param        page    query     int     false  "页码"        default(1)
// @Param        size    query     int     false  "每页数量"     default(20)
// @Param        host_id query     string  false  "主机ID筛选"
// @Success      200     {object}  models.APIResponse
// @Failure      500     {object}  models.APIResponse
// @Router       /tasks/failed-commands [get]
func (tc *HTTPTaskController) GetFailedCommands(c *gin.Context) {
	LogGRPCRequest("GetFailedCommands", c.Request.Method+" "+c.Request.URL.Path)

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	hostID := c.Query("host_id")

	// 参数验证
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	commands, total, err := tc.taskService.GetFailedCommands(page, size, hostID)
	if err != nil {
		LogGRPCResponse("GetFailedCommands", false, "Failed to get failed commands: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to get failed commands: "+err.Error())
		return
	}

	response := gin.H{
		"commands": commands,
		"pagination": gin.H{
			"page":  page,
			"size":  size,
			"total": total,
		},
	}

	LogGRPCResponse("GetFailedCommands", true, "Retrieved "+strconv.Itoa(len(commands))+" failed commands")
	SendSuccessResponse(c, response)
}

// RetryFailedCommand 重试失败的命令
// @Summary      重试失败的命令
// @Description  重新执行指定的失败命令
// @Tags         异常处理
// @Accept       json
// @Produce      json
// @Param        commandId  path      string  true  "命令ID"
// @Success      200        {object}  models.APIResponse
// @Failure      400        {object}  models.APIResponse
// @Failure      500        {object}  models.APIResponse
// @Router       /tasks/commands/{commandId}/retry [post]
func (tc *HTTPTaskController) RetryFailedCommand(c *gin.Context) {
	LogGRPCRequest("RetryFailedCommand", c.Request.Method+" "+c.Request.URL.Path)

	commandID := c.Param("commandId")
	if commandID == "" {
		LogGRPCResponse("RetryFailedCommand", false, "Command ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Command ID is required")
		return
	}

	err := tc.taskService.RetryFailedCommand(commandID)
	if err != nil {
		LogGRPCResponse("RetryFailedCommand", false, "Failed to retry command: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to retry command: "+err.Error())
		return
	}

	LogGRPCResponse("RetryFailedCommand", true, "Command retry initiated: "+commandID)
	SendSuccessResponse(c, gin.H{"message": "Command retry initiated successfully"})
}

// CheckCommandTimeout 检查命令超时
// @Summary      检查命令超时
// @Description  手动检查指定命令是否超时并处理
// @Tags         超时管理
// @Accept       json
// @Produce      json
// @Param        commandId  path      string  true  "命令ID"
// @Success      200        {object}  models.APIResponse
// @Failure      400        {object}  models.APIResponse
// @Failure      500        {object}  models.APIResponse
// @Router       /tasks/commands/{commandId}/check-timeout [post]
func (tc *HTTPTaskController) CheckCommandTimeout(c *gin.Context) {
	LogGRPCRequest("CheckCommandTimeout", c.Request.Method+" "+c.Request.URL.Path)

	commandID := c.Param("commandId")
	if commandID == "" {
		LogGRPCResponse("CheckCommandTimeout", false, "Command ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Command ID is required")
		return
	}

	err := tc.taskService.CheckCommandTimeout(commandID)
	if err != nil {
		LogGRPCResponse("CheckCommandTimeout", false, "Failed to check command timeout: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to check command timeout: "+err.Error())
		return
	}

	LogGRPCResponse("CheckCommandTimeout", true, "Command timeout check completed: "+commandID)
	SendSuccessResponse(c, gin.H{"message": "Command timeout check completed"})
}

// GetTimeoutStatistics 获取超时统计信息
// @Summary      获取超时统计信息
// @Description  获取系统命令执行超时的统计信息
// @Tags         超时管理
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/timeout-statistics [get]
func (tc *HTTPTaskController) GetTimeoutStatistics(c *gin.Context) {
	LogGRPCRequest("GetTimeoutStatistics", c.Request.Method+" "+c.Request.URL.Path)

	statistics, err := tc.taskService.GetTimeoutStatistics()
	if err != nil {
		LogGRPCResponse("GetTimeoutStatistics", false, "Failed to get timeout statistics: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to get timeout statistics: "+err.Error())
		return
	}

	LogGRPCResponse("GetTimeoutStatistics", true, "Timeout statistics retrieved")
	SendSuccessResponse(c, statistics)
}

// GetErrorStatistics 获取错误统计信息
// @Summary      获取错误统计信息
// @Description  获取系统命令执行错误的统计信息
// @Tags         异常处理
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/error-statistics [get]
func (tc *HTTPTaskController) GetErrorStatistics(c *gin.Context) {
	LogGRPCRequest("GetErrorStatistics", c.Request.Method+" "+c.Request.URL.Path)

	statistics, err := tc.taskService.GetErrorStatistics()
	if err != nil {
		LogGRPCResponse("GetErrorStatistics", false, "Failed to get error statistics: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to get error statistics: "+err.Error())
		return
	}

	LogGRPCResponse("GetErrorStatistics", true, "Error statistics retrieved")
	SendSuccessResponse(c, statistics)
}

// GetDatabaseStatistics 获取数据库统计信息
// @Summary      获取数据库统计信息
// @Description  获取数据库表大小、索引使用情况等统计信息
// @Tags         数据库优化
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/database-statistics [get]
func (tc *HTTPTaskController) GetDatabaseStatistics(c *gin.Context) {
	LogGRPCRequest("GetDatabaseStatistics", c.Request.Method+" "+c.Request.URL.Path)

	statistics, err := tc.taskService.GetDatabaseStatistics()
	if err != nil {
		LogGRPCResponse("GetDatabaseStatistics", false, "Failed to get database statistics: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to get database statistics: "+err.Error())
		return
	}

	LogGRPCResponse("GetDatabaseStatistics", true, "Database statistics retrieved")
	SendSuccessResponse(c, statistics)
}

// CleanupOldRecords 清理旧记录
// @Summary      清理旧记录
// @Description  清理指定天数之前的旧记录，释放存储空间
// @Tags         数据库优化
// @Accept       json
// @Produce      json
// @Param        retention_days  query     int     false  "保留天数"  default(30)
// @Success      200             {object}  models.APIResponse
// @Failure      400             {object}  models.APIResponse
// @Failure      500             {object}  models.APIResponse
// @Router       /tasks/cleanup-old-records [post]
func (tc *HTTPTaskController) CleanupOldRecords(c *gin.Context) {
	LogGRPCRequest("CleanupOldRecords", c.Request.Method+" "+c.Request.URL.Path)

	retentionDays, _ := strconv.Atoi(c.DefaultQuery("retention_days", "30"))
	if retentionDays < 1 {
		LogGRPCResponse("CleanupOldRecords", false, "Invalid retention days")
		SendErrorResponse(c, http.StatusBadRequest, "Retention days must be greater than 0")
		return
	}

	err := tc.taskService.CleanupOldRecords(retentionDays)
	if err != nil {
		LogGRPCResponse("CleanupOldRecords", false, "Failed to cleanup old records: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to cleanup old records: "+err.Error())
		return
	}

	LogGRPCResponse("CleanupOldRecords", true, "Old records cleanup completed")
	SendSuccessResponse(c, gin.H{
		"message":        "Old records cleanup completed successfully",
		"retention_days": retentionDays,
	})
}

// OptimizeTables 优化数据库表
// @Summary      优化数据库表
// @Description  执行数据库表优化操作，提升查询性能
// @Tags         数据库优化
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/optimize-tables [post]
func (tc *HTTPTaskController) OptimizeTables(c *gin.Context) {
	LogGRPCRequest("OptimizeTables", c.Request.Method+" "+c.Request.URL.Path)

	err := tc.taskService.OptimizeTables()
	if err != nil {
		LogGRPCResponse("OptimizeTables", false, "Failed to optimize tables: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to optimize tables: "+err.Error())
		return
	}

	LogGRPCResponse("OptimizeTables", true, "Tables optimization completed")
	SendSuccessResponse(c, gin.H{"message": "Tables optimization completed successfully"})
}

// AnalyzeTableSizes 分析表大小
// @Summary      分析表大小
// @Description  分析数据库表的大小和记录数统计
// @Tags         数据库优化
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/table-sizes [get]
func (tc *HTTPTaskController) AnalyzeTableSizes(c *gin.Context) {
	LogGRPCRequest("AnalyzeTableSizes", c.Request.Method+" "+c.Request.URL.Path)

	analysis, err := tc.taskService.AnalyzeTableSizes()
	if err != nil {
		LogGRPCResponse("AnalyzeTableSizes", false, "Failed to analyze table sizes: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to analyze table sizes: "+err.Error())
		return
	}

	LogGRPCResponse("AnalyzeTableSizes", true, "Table sizes analysis completed")
	SendSuccessResponse(c, analysis)
}

// GetDetailedTaskLogs 获取详细任务日志
// @Summary      获取详细任务日志
// @Description  获取任务执行的详细日志信息，包含完整输出
// @Tags         任务监控
// @Accept       json
// @Produce      json
// @Param        id         path      string  true   "任务ID"
// @Param        command_id query     string  false  "命令ID"
// @Param        host_id    query     string  false  "主机ID"
// @Success      200        {object}  models.APIResponse
// @Failure      404        {object}  models.APIResponse
// @Failure      500        {object}  models.APIResponse
// @Router       /tasks/{id}/logs/detailed [get]
func (tc *HTTPTaskController) GetDetailedTaskLogs(c *gin.Context) {
	LogGRPCRequest("GetDetailedTaskLogs", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	commandID := c.Query("command_id")
	hostID := c.Query("host_id")

	if taskID == "" {
		LogGRPCResponse("GetDetailedTaskLogs", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	logs, err := tc.taskService.GetDetailedTaskLogs(taskID, commandID, hostID)
	if err != nil {
		LogGRPCResponse("GetDetailedTaskLogs", false, "Failed to get detailed task logs: "+err.Error())
		SendErrorResponse(c, http.StatusNotFound, "Failed to get detailed task logs: "+err.Error())
		return
	}

	LogGRPCResponse("GetDetailedTaskLogs", true, "Detailed task logs retrieved: "+taskID)
	SendSuccessResponse(c, logs)
}

// GetTaskAuditTrail 获取任务审计追踪
// @Summary      获取任务审计追踪
// @Description  获取任务的完整审计追踪记录
// @Tags         任务监控
// @Accept       json
// @Produce      json
// @Param        id    path      string  true   "任务ID"
// @Param        page  query     int     false  "页码" default(1)
// @Param        size  query     int     false  "每页大小" default(20)
// @Success      200   {object}  models.APIResponse
// @Failure      404   {object}  models.APIResponse
// @Failure      500   {object}  models.APIResponse
// @Router       /tasks/{id}/audit [get]
func (tc *HTTPTaskController) GetTaskAuditTrail(c *gin.Context) {
	LogGRPCRequest("GetTaskAuditTrail", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	if taskID == "" {
		LogGRPCResponse("GetTaskAuditTrail", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	auditTrail, err := tc.taskService.GetTaskAuditTrail(taskID, page, size)
	if err != nil {
		LogGRPCResponse("GetTaskAuditTrail", false, "Failed to get task audit trail: "+err.Error())
		SendErrorResponse(c, http.StatusNotFound, "Failed to get task audit trail: "+err.Error())
		return
	}

	LogGRPCResponse("GetTaskAuditTrail", true, "Task audit trail retrieved: "+taskID)
	SendSuccessResponse(c, auditTrail)
}

// GetTaskExecutionTimeline 获取任务执行时间线
// @Summary      获取任务执行时间线
// @Description  获取任务执行的完整时间线，包含所有事件
// @Tags         任务监控
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "任务ID"
// @Success      200  {object}  models.APIResponse
// @Failure      404  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/{id}/timeline [get]
func (tc *HTTPTaskController) GetTaskExecutionTimeline(c *gin.Context) {
	LogGRPCRequest("GetTaskExecutionTimeline", c.Request.Method+" "+c.Request.URL.Path)

	taskID := c.Param("id")
	if taskID == "" {
		LogGRPCResponse("GetTaskExecutionTimeline", false, "Task ID is required")
		SendErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	timeline, err := tc.taskService.GetTaskExecutionTimeline(taskID)
	if err != nil {
		LogGRPCResponse("GetTaskExecutionTimeline", false, "Failed to get task execution timeline: "+err.Error())
		SendErrorResponse(c, http.StatusNotFound, "Failed to get task execution timeline: "+err.Error())
		return
	}

	response := gin.H{
		"task_id":  taskID,
		"timeline": timeline,
	}

	LogGRPCResponse("GetTaskExecutionTimeline", true, "Task execution timeline retrieved: "+taskID)
	SendSuccessResponse(c, response)
}

// GetExecutionStatistics 获取执行统计信息
// @Summary      获取执行统计信息
// @Description  获取指定时间范围内的执行统计信息
// @Tags         任务统计
// @Accept       json
// @Produce      json
// @Param        start_date  query     string  false  "开始日期 (YYYY-MM-DD)"
// @Param        end_date    query     string  false  "结束日期 (YYYY-MM-DD)"
// @Param        stat_type   query     string  false  "统计类型"
// @Success      200         {object}  models.APIResponse
// @Failure      400         {object}  models.APIResponse
// @Failure      500         {object}  models.APIResponse
// @Router       /tasks/execution-statistics [get]
func (tc *HTTPTaskController) GetExecutionStatistics(c *gin.Context) {
	LogGRPCRequest("GetExecutionStatistics", c.Request.Method+" "+c.Request.URL.Path)

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	statType := c.Query("stat_type")

	// 默认查询最近7天的数据
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	if startDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = parsed
		}
	}

	if endDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = parsed
		}
	}

	statistics, err := tc.taskService.GetExecutionStatistics(startDate, endDate, statType)
	if err != nil {
		LogGRPCResponse("GetExecutionStatistics", false, "Failed to get execution statistics: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to get execution statistics: "+err.Error())
		return
	}

	response := gin.H{
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
		"stat_type":  statType,
		"statistics": statistics,
	}

	LogGRPCResponse("GetExecutionStatistics", true, "Execution statistics retrieved")
	SendSuccessResponse(c, response)
}

// GetAuditSummary 获取审计摘要
// @Summary      获取审计摘要
// @Description  获取指定天数内的审计摘要信息
// @Tags         任务统计
// @Accept       json
// @Produce      json
// @Param        days  query     int  false  "天数" default(7)
// @Success      200   {object}  models.APIResponse
// @Failure      500   {object}  models.APIResponse
// @Router       /tasks/audit-summary [get]
func (tc *HTTPTaskController) GetAuditSummary(c *gin.Context) {
	LogGRPCRequest("GetAuditSummary", c.Request.Method+" "+c.Request.URL.Path)

	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	summary, err := tc.taskService.GetAuditSummary(days)
	if err != nil {
		LogGRPCResponse("GetAuditSummary", false, "Failed to get audit summary: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to get audit summary: "+err.Error())
		return
	}

	LogGRPCResponse("GetAuditSummary", true, "Audit summary retrieved")
	SendSuccessResponse(c, summary)
}

// GetLogStatistics 获取日志统计信息
// @Summary      获取日志统计信息
// @Description  获取系统日志的统计信息
// @Tags         任务统计
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/log-statistics [get]
func (tc *HTTPTaskController) GetLogStatistics(c *gin.Context) {
	LogGRPCRequest("GetLogStatistics", c.Request.Method+" "+c.Request.URL.Path)

	statistics, err := tc.taskService.GetLogStatistics()
	if err != nil {
		LogGRPCResponse("GetLogStatistics", false, "Failed to get log statistics: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to get log statistics: "+err.Error())
		return
	}

	LogGRPCResponse("GetLogStatistics", true, "Log statistics retrieved")
	SendSuccessResponse(c, statistics)
}

// SearchLogs 搜索日志
// @Summary      搜索日志
// @Description  根据关键词搜索审计日志和执行日志
// @Tags         任务监控
// @Accept       json
// @Produce      json
// @Param        keyword    query     string  false  "搜索关键词"
// @Param        log_type   query     string  false  "日志类型 (audit/execution)"
// @Param        start_time query     string  false  "开始时间 (RFC3339格式)"
// @Param        end_time   query     string  false  "结束时间 (RFC3339格式)"
// @Param        page       query     int     false  "页码" default(1)
// @Param        size       query     int     false  "每页大小" default(20)
// @Success      200        {object}  models.APIResponse
// @Failure      400        {object}  models.APIResponse
// @Failure      500        {object}  models.APIResponse
// @Router       /tasks/search-logs [get]
func (tc *HTTPTaskController) SearchLogs(c *gin.Context) {
	LogGRPCRequest("SearchLogs", c.Request.Method+" "+c.Request.URL.Path)

	keyword := c.Query("keyword")
	logType := c.Query("log_type")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	var startTime, endTime *time.Time

	if startTimeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &parsed
		} else {
			SendErrorResponse(c, http.StatusBadRequest, "Invalid start_time format, use RFC3339")
			return
		}
	}

	if endTimeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &parsed
		} else {
			SendErrorResponse(c, http.StatusBadRequest, "Invalid end_time format, use RFC3339")
			return
		}
	}

	results, err := tc.taskService.SearchLogs(keyword, logType, startTime, endTime, page, size)
	if err != nil {
		LogGRPCResponse("SearchLogs", false, "Failed to search logs: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to search logs: "+err.Error())
		return
	}

	LogGRPCResponse("SearchLogs", true, "Log search completed")
	SendSuccessResponse(c, results)
}

// CleanupOldLogs 清理旧日志
// @Summary      清理旧日志
// @Description  清理指定天数之前的审计日志和执行日志
// @Tags         系统维护
// @Accept       json
// @Produce      json
// @Param        retention_days  query     int  false  "保留天数" default(30)
// @Success      200             {object}  models.APIResponse
// @Failure      400             {object}  models.APIResponse
// @Failure      500             {object}  models.APIResponse
// @Router       /tasks/cleanup-old-logs [post]
func (tc *HTTPTaskController) CleanupOldLogs(c *gin.Context) {
	LogGRPCRequest("CleanupOldLogs", c.Request.Method+" "+c.Request.URL.Path)

	retentionDays, _ := strconv.Atoi(c.DefaultQuery("retention_days", "30"))

	if retentionDays < 1 {
		LogGRPCResponse("CleanupOldLogs", false, "Retention days must be greater than 0")
		SendErrorResponse(c, http.StatusBadRequest, "Retention days must be greater than 0")
		return
	}

	err := tc.taskService.CleanupOldLogs(retentionDays)
	if err != nil {
		LogGRPCResponse("CleanupOldLogs", false, "Failed to cleanup old logs: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to cleanup old logs: "+err.Error())
		return
	}

	response := gin.H{
		"message":        "Old logs cleaned up successfully",
		"retention_days": retentionDays,
	}

	LogGRPCResponse("CleanupOldLogs", true, "Old logs cleaned up successfully")
	SendSuccessResponse(c, response)
}

// UpdateDailyStatistics 更新每日统计
// @Summary      更新每日统计
// @Description  手动触发每日统计信息的更新
// @Tags         系统维护
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.APIResponse
// @Failure      500  {object}  models.APIResponse
// @Router       /tasks/update-daily-statistics [post]
func (tc *HTTPTaskController) UpdateDailyStatistics(c *gin.Context) {
	LogGRPCRequest("UpdateDailyStatistics", c.Request.Method+" "+c.Request.URL.Path)

	err := tc.taskService.UpdateDailyStatistics()
	if err != nil {
		LogGRPCResponse("UpdateDailyStatistics", false, "Failed to update daily statistics: "+err.Error())
		SendErrorResponse(c, http.StatusInternalServerError, "Failed to update daily statistics: "+err.Error())
		return
	}

	response := gin.H{
		"message": "Daily statistics updated successfully",
		"date":    time.Now().Format("2006-01-02"),
	}

	LogGRPCResponse("UpdateDailyStatistics", true, "Daily statistics updated successfully")
	SendSuccessResponse(c, response)
}
