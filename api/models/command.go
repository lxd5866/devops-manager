package models

import (
	"devops-manager/api/protobuf"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// CommandStatus 命令状态枚举
type CommandStatus string

const (
	CommandStatusPending   CommandStatus = "pending"   // 待执行
	CommandStatusRunning   CommandStatus = "running"   // 执行中
	CommandStatusCompleted CommandStatus = "completed" // 已完成
	CommandStatusFailed    CommandStatus = "failed"    // 执行失败
	CommandStatusTimeout   CommandStatus = "timeout"   // 超时
	CommandStatusCanceled  CommandStatus = "canceled"  // 已取消
)

// Command 命令模型
type Command struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	CommandID  string         `json:"command_id" gorm:"uniqueIndex;size:255;not null;comment:命令唯一标识"`
	TaskID     *string        `json:"task_id" gorm:"size:255;comment:所属任务ID"`
	HostID     string         `json:"host_id" gorm:"size:255;not null;comment:目标主机ID"`
	Command    string         `json:"command" gorm:"type:text;not null;comment:命令内容"`
	Parameters string         `json:"parameters" gorm:"type:text;comment:命令参数"`
	Timeout    int64          `json:"timeout" gorm:"comment:超时时间(秒)"`
	Status     CommandStatus  `json:"status" gorm:"size:20;default:pending;comment:命令状态"`
	Stdout     string         `json:"stdout" gorm:"type:longtext;comment:标准输出"`
	Stderr     string         `json:"stderr" gorm:"type:longtext;comment:错误输出"`
	ExitCode   *int32         `json:"exit_code" gorm:"comment:退出码"`
	StartedAt  *time.Time     `json:"started_at" gorm:"comment:开始执行时间"`
	FinishedAt *time.Time     `json:"finished_at" gorm:"comment:完成时间"`
	ErrorMsg   string         `json:"error_message" gorm:"type:text;comment:执行错误信息"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Task          *Task          `json:"task,omitempty" gorm:"foreignKey:TaskID;references:TaskID"`
	Host          *Host          `json:"host,omitempty" gorm:"foreignKey:HostID;references:HostID"`
	CommandResult *CommandResult `json:"command_result,omitempty" gorm:"foreignKey:CommandID;references:CommandID"`
	CommandHosts  []CommandHost  `json:"command_hosts,omitempty" gorm:"foreignKey:CommandID;references:CommandID"`
}

// TableName 指定表名
func (Command) TableName() string {
	return "commands"
}

// ToProtobufContent 转换为 protobuf CommandContent 格式
func (c *Command) ToProtobufContent() *protobuf.CommandContent {
	// 转换超时时间
	var timeout *durationpb.Duration
	if c.Timeout > 0 {
		timeout = durationpb.New(time.Duration(c.Timeout) * time.Second)
	}

	return &protobuf.CommandContent{
		CommandId:  c.CommandID,
		HostId:     c.HostID,
		Command:    c.Command,
		Parameters: c.Parameters, // 现在直接使用 string 类型
		Timeout:    timeout,
		CreatedAt:  timestamppb.New(c.CreatedAt),
	}
}

// ToProtobufResult 转换为 protobuf CommandResult 格式
func (c *Command) ToProtobufResult() *protobuf.CommandResult {
	result := &protobuf.CommandResult{
		CommandId:    c.CommandID,
		HostId:       c.HostID,
		Stdout:       c.Stdout,
		Stderr:       c.Stderr,
		ErrorMessage: c.ErrorMsg,
	}

	if c.ExitCode != nil {
		result.ExitCode = *c.ExitCode
	}

	if c.StartedAt != nil {
		result.StartedAt = timestamppb.New(*c.StartedAt)
	}

	if c.FinishedAt != nil {
		result.FinishedAt = timestamppb.New(*c.FinishedAt)
	}

	return result
}

// FromProtobufContent 从 protobuf CommandContent 创建命令
func (c *Command) FromProtobufContent(content *protobuf.CommandContent) {
	c.CommandID = content.CommandId
	c.HostID = content.HostId
	c.Command = content.Command
	c.Status = CommandStatusPending

	// 直接使用 string 类型的参数
	c.Parameters = content.Parameters

	// 转换超时时间
	if content.Timeout != nil {
		c.Timeout = int64(content.Timeout.AsDuration().Seconds())
	}

	// 转换创建时间
	if content.CreatedAt != nil {
		c.CreatedAt = content.CreatedAt.AsTime()
	}
}

// UpdateFromProtobufResult 从 protobuf CommandResult 更新命令结果
func (c *Command) UpdateFromProtobufResult(result *protobuf.CommandResult) {
	c.Stdout = result.Stdout
	c.Stderr = result.Stderr
	c.ErrorMsg = result.ErrorMessage
	c.ExitCode = &result.ExitCode

	if result.StartedAt != nil {
		startedAt := result.StartedAt.AsTime()
		c.StartedAt = &startedAt
		c.Status = CommandStatusRunning
	}

	if result.FinishedAt != nil {
		finishedAt := result.FinishedAt.AsTime()
		c.FinishedAt = &finishedAt

		// 根据退出码设置状态
		if result.ExitCode == 0 {
			c.Status = CommandStatusCompleted
		} else {
			c.Status = CommandStatusFailed
		}
	}
}

// IsCompleted 检查命令是否已完成
func (c *Command) IsCompleted() bool {
	return c.Status == CommandStatusCompleted ||
		c.Status == CommandStatusFailed ||
		c.Status == CommandStatusTimeout ||
		c.Status == CommandStatusCanceled
}

// IsRunning 检查命令是否正在运行
func (c *Command) IsRunning() bool {
	return c.Status == CommandStatusRunning
}

// IsPending 检查命令是否待执行
func (c *Command) IsPending() bool {
	return c.Status == CommandStatusPending
}

// Duration 获取命令执行时长
func (c *Command) Duration() time.Duration {
	if c.StartedAt == nil || c.FinishedAt == nil {
		return 0
	}
	return c.FinishedAt.Sub(*c.StartedAt)
}

// CommandHistory 命令历史记录（用于审计）
type CommandHistory struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CommandID string    `json:"command_id" gorm:"size:255;not null;comment:命令ID"`
	HostID    string    `json:"host_id" gorm:"size:255;not null;comment:主机ID"`
	Action    string    `json:"action" gorm:"size:50;not null;comment:操作类型"`
	Details   JSON      `json:"details" gorm:"type:json;comment:操作详情"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (CommandHistory) TableName() string {
	return "command_histories"
}

// CreateCommandFromProtobuf 从 protobuf CommandContent 创建新命令
func CreateCommandFromProtobuf(content *protobuf.CommandContent) *Command {
	cmd := &Command{}
	cmd.FromProtobufContent(content)
	return cmd
}
