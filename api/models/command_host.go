package models

import (
	"time"
)

// CommandHostStatus 命令主机执行状态
type CommandHostStatus string

const (
	CommandHostStatusPending    CommandHostStatus = "待执行"
	CommandHostStatusRunning    CommandHostStatus = "运行中"
	CommandHostStatusFailed     CommandHostStatus = "下发失败"
	CommandHostStatusExecFailed CommandHostStatus = "执行失败"
	CommandHostStatusTimeout    CommandHostStatus = "执行超时"
	CommandHostStatusCanceled   CommandHostStatus = "取消执行"
	CommandHostStatusCompleted  CommandHostStatus = "执行完成"
)

// CommandHost 命令主机关联模型，映射到 commands_hosts 表
type CommandHost struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	CommandID     string     `json:"command_id" gorm:"size:255;not null;comment:命令ID"`
	HostID        string     `json:"host_id" gorm:"size:255;not null;comment:主机ID"`
	Status        string     `json:"status" gorm:"size:20;default:待执行;comment:命令状态"`
	Stdout        string     `json:"stdout" gorm:"type:longtext;comment:标准输出"`
	Stderr        string     `json:"stderr" gorm:"type:longtext;comment:错误输出"`
	ExitCode      int        `json:"exit_code" gorm:"default:0;comment:退出码"`
	StartedAt     *time.Time `json:"started_at" gorm:"comment:开始执行时间"`
	FinishedAt    *time.Time `json:"finished_at" gorm:"comment:完成时间"`
	ErrorMessage  string     `json:"error_message" gorm:"type:text;comment:执行错误信息"`
	ExecutionTime *int64     `json:"execution_time" gorm:"comment:执行时长(毫秒)"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// 关联关系
	Command *Command `json:"command,omitempty" gorm:"foreignKey:CommandID;references:CommandID"`
	Host    *Host    `json:"host,omitempty" gorm:"foreignKey:HostID;references:HostID"`
}

// TableName 指定表名
func (CommandHost) TableName() string {
	return "commands_hosts"
}

// IsCompleted 检查命令是否已完成
func (ch *CommandHost) IsCompleted() bool {
	return ch.Status == string(CommandHostStatusCompleted) ||
		ch.Status == string(CommandHostStatusFailed) ||
		ch.Status == string(CommandHostStatusExecFailed) ||
		ch.Status == string(CommandHostStatusTimeout) ||
		ch.Status == string(CommandHostStatusCanceled)
}

// IsRunning 检查命令是否正在运行
func (ch *CommandHost) IsRunning() bool {
	return ch.Status == string(CommandHostStatusRunning)
}

// IsPending 检查命令是否待执行
func (ch *CommandHost) IsPending() bool {
	return ch.Status == string(CommandHostStatusPending)
}

// Duration 获取命令执行时长
func (ch *CommandHost) Duration() time.Duration {
	if ch.StartedAt == nil || ch.FinishedAt == nil {
		return 0
	}
	return ch.FinishedAt.Sub(*ch.StartedAt)
}

// IsSuccess 检查命令是否执行成功
func (ch *CommandHost) IsSuccess() bool {
	return ch.Status == string(CommandHostStatusCompleted) && ch.ExitCode == 0
}

// UpdateExecutionTime 更新执行时长
func (ch *CommandHost) UpdateExecutionTime() {
	if ch.StartedAt != nil && ch.FinishedAt != nil {
		duration := ch.FinishedAt.Sub(*ch.StartedAt)
		executionTime := duration.Milliseconds()
		ch.ExecutionTime = &executionTime
	}
}
