package models

import (
	"devops-manager/api/protobuf"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// CommandResult 命令执行结果模型
type CommandResult struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	CommandID     string     `json:"command_id" gorm:"uniqueIndex;size:255;not null;comment:命令ID"`
	HostID        string     `json:"host_id" gorm:"size:255;not null;comment:执行主机ID"`
	Stdout        string     `json:"stdout" gorm:"type:longtext;comment:标准输出"`
	Stderr        string     `json:"stderr" gorm:"type:longtext;comment:错误输出"`
	ExitCode      int32      `json:"exit_code" gorm:"default:0;comment:退出码"`
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
func (CommandResult) TableName() string {
	return "command_results"
}

// ToProtobuf 转换为 protobuf CommandResult 格式
func (cr *CommandResult) ToProtobuf() *protobuf.CommandResult {
	result := &protobuf.CommandResult{
		CommandId:    cr.CommandID,
		HostId:       cr.HostID,
		Stdout:       cr.Stdout,
		Stderr:       cr.Stderr,
		ExitCode:     cr.ExitCode,
		ErrorMessage: cr.ErrorMessage,
	}

	if cr.StartedAt != nil {
		result.StartedAt = timestamppb.New(*cr.StartedAt)
	}

	if cr.FinishedAt != nil {
		result.FinishedAt = timestamppb.New(*cr.FinishedAt)
	}

	return result
}

// FromProtobuf 从 protobuf CommandResult 创建
func (cr *CommandResult) FromProtobuf(result *protobuf.CommandResult) {
	cr.CommandID = result.CommandId
	cr.HostID = result.HostId
	cr.Stdout = result.Stdout
	cr.Stderr = result.Stderr
	cr.ExitCode = result.ExitCode
	cr.ErrorMessage = result.ErrorMessage

	if result.StartedAt != nil {
		startedAt := result.StartedAt.AsTime()
		cr.StartedAt = &startedAt
	}

	if result.FinishedAt != nil {
		finishedAt := result.FinishedAt.AsTime()
		cr.FinishedAt = &finishedAt

		// 计算执行时长
		if cr.StartedAt != nil {
			duration := finishedAt.Sub(*cr.StartedAt)
			executionTime := duration.Milliseconds()
			cr.ExecutionTime = &executionTime
		}
	}
}

// Duration 获取执行时长
func (cr *CommandResult) Duration() time.Duration {
	if cr.StartedAt == nil || cr.FinishedAt == nil {
		return 0
	}
	return cr.FinishedAt.Sub(*cr.StartedAt)
}

// IsSuccess 检查命令是否执行成功
func (cr *CommandResult) IsSuccess() bool {
	return cr.ExitCode == 0
}

// CreateCommandResultFromProtobuf 从 protobuf CommandResult 创建新的命令结果
func CreateCommandResultFromProtobuf(result *protobuf.CommandResult) *CommandResult {
	cr := &CommandResult{}
	cr.FromProtobuf(result)
	return cr
}
