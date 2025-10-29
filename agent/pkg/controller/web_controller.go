package controller

import (
	"net/http"

	"devops-manager/agent/pkg/service"

	"github.com/gin-gonic/gin"
)

// WebController Web页面控制器
type WebController struct {
	hostService *service.HostAgent
	taskService *service.TaskService
	fileService *service.FileService
}

// NewWebController 创建Web控制器
func NewWebController() *WebController {
	return &WebController{
		taskService: service.NewTaskService(),
		fileService: service.NewFileService("./uploads", "./downloads"),
	}
}

// SetHostService 设置主机服务
func (wc *WebController) SetHostService(hostService *service.HostAgent) {
	wc.hostService = hostService
}

// Index 首页
func (wc *WebController) Index(c *gin.Context) {
	LogHTTPRequest(c)

	data := gin.H{
		"title":   "DevOps Agent",
		"version": "1.0.0",
		"status":  "running",
	}

	// 如果有模板文件，使用模板渲染
	if c.Request.Header.Get("Accept") == "application/json" {
		SuccessResponse(c, data)
	} else {
		// 返回简单的HTML页面
		c.HTML(http.StatusOK, "index.html", data)
	}
}

// Status 状态页面
func (wc *WebController) Status(c *gin.Context) {
	LogHTTPRequest(c)

	// 获取系统状态
	status := gin.H{
		"agent_status": "running",
		"connected":    wc.hostService != nil,
		"uptime":       "unknown",
		"last_report":  "unknown",
	}

	// 获取运行中的任务
	runningTasks := wc.taskService.GetRunningTasks()
	status["running_tasks"] = len(runningTasks)

	if c.Request.Header.Get("Accept") == "application/json" {
		SuccessResponse(c, status)
	} else {
		c.HTML(http.StatusOK, "status.html", gin.H{
			"title":  "Agent Status",
			"status": status,
		})
	}
}

// Tasks 任务页面
func (wc *WebController) Tasks(c *gin.Context) {
	LogHTTPRequest(c)

	// 获取任务列表
	runningTasks := wc.taskService.GetRunningTasks()

	var tasks []gin.H
	for _, taskID := range runningTasks {
		execution, exists := wc.taskService.GetTaskStatus(taskID)
		if !exists {
			continue
		}

		taskInfo := gin.H{
			"task_id":    execution.TaskID,
			"command":    execution.Command,
			"status":     execution.Status,
			"start_time": execution.StartTime.Format("2006-01-02 15:04:05"),
		}

		if execution.EndTime != nil {
			taskInfo["end_time"] = execution.EndTime.Format("2006-01-02 15:04:05")
		}

		tasks = append(tasks, taskInfo)
	}

	data := gin.H{
		"tasks": tasks,
		"total": len(tasks),
	}

	if c.Request.Header.Get("Accept") == "application/json" {
		SuccessResponse(c, data)
	} else {
		c.HTML(http.StatusOK, "tasks.html", gin.H{
			"title": "Task Management",
			"data":  data,
		})
	}
}

// Files 文件页面
func (wc *WebController) Files(c *gin.Context) {
	LogHTTPRequest(c)

	// 获取上传文件列表
	uploadFiles, _ := wc.fileService.ListFiles("upload")
	downloadFiles, _ := wc.fileService.ListFiles("download")

	data := gin.H{
		"upload_files":   uploadFiles,
		"download_files": downloadFiles,
		"upload_count":   len(uploadFiles),
		"download_count": len(downloadFiles),
	}

	if c.Request.Header.Get("Accept") == "application/json" {
		SuccessResponse(c, data)
	} else {
		c.HTML(http.StatusOK, "files.html", gin.H{
			"title": "File Management",
			"data":  data,
		})
	}
}
