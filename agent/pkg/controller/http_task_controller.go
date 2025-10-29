package controller

import (
	"net/http"
	"strconv"
	"time"

	"devops-manager/agent/pkg/service"

	"github.com/gin-gonic/gin"
)

// TaskHTTPController 任务HTTP业务控制器
type TaskHTTPController struct {
	taskService *service.TaskService
}

// NewTaskHTTPController 创建任务HTTP控制器
func NewTaskHTTPController() *TaskHTTPController {
	return &TaskHTTPController{
		taskService: service.NewTaskService(),
	}
}

// ExecuteTask 执行任务
func (thc *TaskHTTPController) ExecuteTask(c *gin.Context) {
	LogHTTPRequest(c)

	var req struct {
		TaskID  string `json:"task_id" binding:"required"`
		Command string `json:"command" binding:"required"`
		Timeout int    `json:"timeout"` // 秒
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// 设置默认超时时间
	timeout := 30 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	// 执行任务
	result, err := thc.taskService.ExecuteTask(req.TaskID, req.Command, timeout)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	SuccessResponse(c, gin.H{
		"task_id":   req.TaskID,
		"command":   req.Command,
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
		"exit_code": result.ExitCode,
		"duration":  result.Duration.String(),
		"error":     result.Error,
	})
}

// GetTaskStatus 获取任务状态
func (thc *TaskHTTPController) GetTaskStatus(c *gin.Context) {
	LogHTTPRequest(c)

	taskID := c.Param("id")
	if taskID == "" {
		ErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	execution, exists := thc.taskService.GetTaskStatus(taskID)
	if !exists {
		ErrorResponse(c, http.StatusNotFound, "Task not found")
		return
	}

	data := gin.H{
		"task_id":    execution.TaskID,
		"command":    execution.Command,
		"status":     execution.Status,
		"start_time": execution.StartTime.Unix(),
	}

	if execution.EndTime != nil {
		data["end_time"] = execution.EndTime.Unix()
	}

	if execution.Result != nil {
		data["result"] = gin.H{
			"stdout":    execution.Result.Stdout,
			"stderr":    execution.Result.Stderr,
			"exit_code": execution.Result.ExitCode,
			"duration":  execution.Result.Duration.String(),
			"error":     execution.Result.Error,
		}
	}

	SuccessResponse(c, data)
}

// CancelTask 取消任务
func (thc *TaskHTTPController) CancelTask(c *gin.Context) {
	LogHTTPRequest(c)

	taskID := c.Param("id")
	if taskID == "" {
		ErrorResponse(c, http.StatusBadRequest, "Task ID is required")
		return
	}

	err := thc.taskService.CancelTask(taskID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	SuccessResponse(c, gin.H{
		"message": "Task canceled successfully",
		"task_id": taskID,
	})
}

// ListTasks 列出任务
func (thc *TaskHTTPController) ListTasks(c *gin.Context) {
	LogHTTPRequest(c)

	// 获取查询参数
	statusFilter := c.Query("status")
	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	// 获取运行中的任务
	runningTasks := thc.taskService.GetRunningTasks()

	var tasks []gin.H
	count := 0

	for _, taskID := range runningTasks {
		if count >= limit {
			break
		}

		execution, exists := thc.taskService.GetTaskStatus(taskID)
		if !exists {
			continue
		}

		// 状态过滤
		if statusFilter != "" && execution.Status != statusFilter {
			continue
		}

		taskInfo := gin.H{
			"task_id":    execution.TaskID,
			"command":    execution.Command,
			"status":     execution.Status,
			"start_time": execution.StartTime.Unix(),
		}

		if execution.EndTime != nil {
			taskInfo["end_time"] = execution.EndTime.Unix()
		}

		tasks = append(tasks, taskInfo)
		count++
	}

	SuccessResponse(c, gin.H{
		"tasks": tasks,
		"total": len(tasks),
		"limit": limit,
	})
}
