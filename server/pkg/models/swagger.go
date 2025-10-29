package models

import "time"

// SwaggerHostInfo Swagger主机信息模型
type SwaggerHostInfo struct {
	ID       string            `json:"id" example:"agent-host-001"`
	Hostname string            `json:"hostname" example:"web-server-01"`
	IP       string            `json:"ip" example:"192.168.1.100"`
	OS       string            `json:"os" example:"linux"`
	Tags     map[string]string `json:"tags" example:"role:web,env:prod"`
	LastSeen int64             `json:"last_seen" example:"1640995200"`
}

// SwaggerTask Swagger任务模型
type SwaggerTask struct {
	ID             uint      `json:"id" example:"1"`
	TaskID         string    `json:"task_id" example:"task-001"`
	Name           string    `json:"name" example:"部署应用"`
	Description    string    `json:"description" example:"部署Web应用到生产环境"`
	Status         string    `json:"status" example:"completed"`
	TotalHosts     int       `json:"total_hosts" example:"5"`
	CompletedHosts int       `json:"completed_hosts" example:"5"`
	FailedHosts    int       `json:"failed_hosts" example:"0"`
	CreatedBy      string    `json:"created_by" example:"admin"`
	StartedAt      time.Time `json:"started_at" example:"2024-01-01T10:00:00Z"`
	FinishedAt     time.Time `json:"finished_at" example:"2024-01-01T10:05:00Z"`
	CreatedAt      time.Time `json:"created_at" example:"2024-01-01T09:55:00Z"`
	UpdatedAt      time.Time `json:"updated_at" example:"2024-01-01T10:05:00Z"`
}

// SwaggerPendingHost Swagger待准入主机模型
type SwaggerPendingHost struct {
	HostID    string            `json:"host_id" example:"agent-pending-001"`
	Hostname  string            `json:"hostname" example:"new-server-01"`
	IP        string            `json:"ip" example:"192.168.1.200"`
	OS        string            `json:"os" example:"linux"`
	Tags      map[string]string `json:"tags" example:"role:app,env:test"`
	FirstSeen int64             `json:"first_seen" example:"1640995000"`
	LastSeen  int64             `json:"last_seen" example:"1640995200"`
}

// SwaggerCreateTaskRequest 创建任务请求模型
type SwaggerCreateTaskRequest struct {
	Name        string            `json:"name" example:"执行脚本任务" binding:"required"`
	Description string            `json:"description" example:"在指定主机上执行部署脚本"`
	HostIDs     []string          `json:"host_ids" example:"agent-host-001,agent-host-002" binding:"required"`
	Command     string            `json:"command" example:"bash deploy.sh" binding:"required"`
	Timeout     int               `json:"timeout" example:"300"`
	Parameters  map[string]string `json:"parameters" example:"env:prod,version:1.2.3"`
}

// SwaggerErrorResponse 错误响应模型
type SwaggerErrorResponse struct {
	Success      bool   `json:"success" example:"false"`
	ErrorMessage string `json:"error_message" example:"参数验证失败"`
	Message      string `json:"message" example:"请求处理失败"`
}

// SwaggerSuccessResponse 成功响应模型
type SwaggerSuccessResponse struct {
	Success bool        `json:"success" example:"true"`
	Data    interface{} `json:"data"`
	Message string      `json:"message" example:"操作成功"`
}

// SwaggerHostListResponse 主机列表响应模型
type SwaggerHostListResponse struct {
	Success bool              `json:"success" example:"true"`
	Data    []SwaggerHostInfo `json:"data"`
}

// SwaggerTaskListResponse 任务列表响应模型
type SwaggerTaskListResponse struct {
	Success bool          `json:"success" example:"true"`
	Data    []SwaggerTask `json:"data"`
}

// SwaggerPendingHostListResponse 待准入主机列表响应模型
type SwaggerPendingHostListResponse struct {
	Success bool                 `json:"success" example:"true"`
	Data    []SwaggerPendingHost `json:"data"`
}
