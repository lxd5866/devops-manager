package models

// APIResponse 标准API响应结构
type APIResponse struct {
	Success      bool        `json:"success" example:"true"`
	Data         interface{} `json:"data,omitempty"`
	Message      string      `json:"message,omitempty" example:"操作成功"`
	ErrorMessage string      `json:"error_message,omitempty" example:"参数验证失败"`
}

// HostInfoResponse 主机信息响应
type HostInfoResponse struct {
	ID       string            `json:"id" example:"agent-host-001"`
	Hostname string            `json:"hostname" example:"web-server-01"`
	IP       string            `json:"ip" example:"192.168.1.100"`
	OS       string            `json:"os" example:"linux"`
	Tags     map[string]string `json:"tags"`
	LastSeen int64             `json:"last_seen" example:"1640995200"`
}

// HostListResponse 主机列表响应
type HostListResponse struct {
	Success bool               `json:"success" example:"true"`
	Data    []HostInfoResponse `json:"data"`
}

// PendingHostResponse 待准入主机响应
type PendingHostResponse struct {
	HostID    string            `json:"host_id" example:"agent-pending-001"`
	Hostname  string            `json:"hostname" example:"new-server-01"`
	IP        string            `json:"ip" example:"192.168.1.200"`
	OS        string            `json:"os" example:"linux"`
	Tags      map[string]string `json:"tags"`
	FirstSeen int64             `json:"first_seen" example:"1640995000"`
	LastSeen  int64             `json:"last_seen" example:"1640995200"`
}

// PendingHostListResponse 待准入主机列表响应
type PendingHostListResponse struct {
	Success bool                  `json:"success" example:"true"`
	Data    []PendingHostResponse `json:"data"`
}

// TaskResponse 任务响应
type TaskResponse struct {
	ID             uint   `json:"id" example:"1"`
	TaskID         string `json:"task_id" example:"task-001"`
	Name           string `json:"name" example:"部署应用"`
	Description    string `json:"description" example:"部署Web应用到生产环境"`
	Status         string `json:"status" example:"completed"`
	TotalHosts     int    `json:"total_hosts" example:"5"`
	CompletedHosts int    `json:"completed_hosts" example:"5"`
	FailedHosts    int    `json:"failed_hosts" example:"0"`
	CreatedBy      string `json:"created_by" example:"admin"`
	CreatedAt      string `json:"created_at" example:"2024-01-01T09:55:00Z"`
	UpdatedAt      string `json:"updated_at" example:"2024-01-01T10:05:00Z"`
}

// TaskListResponse 任务列表响应
type TaskListResponse struct {
	Success bool           `json:"success" example:"true"`
	Data    []TaskResponse `json:"data"`
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name        string            `json:"name" example:"执行脚本任务" binding:"required"`
	Description string            `json:"description" example:"在指定主机上执行部署脚本"`
	HostIDs     []string          `json:"host_ids" example:"agent-host-001,agent-host-002" binding:"required"`
	Command     string            `json:"command" example:"bash deploy.sh" binding:"required"`
	Timeout     int               `json:"timeout" example:"300"`
	Parameters  map[string]string `json:"parameters"`
}

// HostRegisterRequest 主机注册请求
type HostRegisterRequest struct {
	Hostname string            `json:"hostname" example:"web-server-01" binding:"required"`
	IP       string            `json:"ip" example:"192.168.1.100"`
	OS       string            `json:"os" example:"linux"`
	Tags     map[string]string `json:"tags"`
}
