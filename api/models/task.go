package models

import (
	"time"

	"gorm.io/gorm"
)

// TaskStatus 任务状态枚举
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 待执行
	TaskStatusRunning   TaskStatus = "running"   // 执行中
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 执行失败
	TaskStatusCanceled  TaskStatus = "canceled"  // 已取消
)

// Task 任务模型
type Task struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	TaskID         string         `json:"task_id" gorm:"uniqueIndex;size:255;not null;comment:任务唯一标识"`
	Name           string         `json:"name" gorm:"size:255;not null;comment:任务名称"`
	Description    string         `json:"description" gorm:"type:text;comment:任务描述"`
	Status         TaskStatus     `json:"status" gorm:"size:20;default:pending;comment:任务状态"`
	TotalHosts     int            `json:"total_hosts" gorm:"default:0;comment:总主机数"`
	CompletedHosts int            `json:"completed_hosts" gorm:"default:0;comment:已完成主机数"`
	FailedHosts    int            `json:"failed_hosts" gorm:"default:0;comment:失败主机数"`
	CreatedBy      string         `json:"created_by" gorm:"size:255;comment:创建者"`
	StartedAt      *time.Time     `json:"started_at" gorm:"comment:开始时间"`
	FinishedAt     *time.Time     `json:"finished_at" gorm:"comment:完成时间"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	TaskHosts []TaskHost `json:"task_hosts" gorm:"foreignKey:TaskID;references:TaskID"`
	Commands  []Command  `json:"commands" gorm:"foreignKey:TaskID;references:TaskID"`
}

// TableName 指定表名
func (Task) TableName() string {
	return "tasks"
}

// TaskHost 任务主机关联模型
type TaskHost struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	TaskID    string     `json:"task_id" gorm:"size:255;not null;comment:任务ID"`
	HostID    string     `json:"host_id" gorm:"size:255;not null;comment:主机ID"`
	Status    TaskStatus `json:"status" gorm:"size:20;default:pending;comment:该主机在任务中的状态"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	// 关联关系
	Task *Task `json:"task,omitempty" gorm:"foreignKey:TaskID;references:TaskID"`
	Host *Host `json:"host,omitempty" gorm:"foreignKey:HostID;references:HostID"`
}

// TableName 指定表名
func (TaskHost) TableName() string {
	return "task_hosts"
}

// TaskBuilder 任务构建器
type TaskBuilder struct {
	task *Task
}

// NewTaskBuilder 创建新的任务构建器
func NewTaskBuilder() *TaskBuilder {
	return &TaskBuilder{
		task: &Task{
			Status: TaskStatusPending,
		},
	}
}

// WithName 设置任务名称
func (tb *TaskBuilder) WithName(name string) *TaskBuilder {
	tb.task.Name = name
	return tb
}

// WithDescription 设置任务描述
func (tb *TaskBuilder) WithDescription(desc string) *TaskBuilder {
	tb.task.Description = desc
	return tb
}

// WithCreatedBy 设置创建者
func (tb *TaskBuilder) WithCreatedBy(creator string) *TaskBuilder {
	tb.task.CreatedBy = creator
	return tb
}

// WithHosts 添加主机列表
func (tb *TaskBuilder) WithHosts(hostIDs []string) *TaskBuilder {
	tb.task.TotalHosts = len(hostIDs)
	tb.task.TaskHosts = make([]TaskHost, len(hostIDs))

	for i, hostID := range hostIDs {
		tb.task.TaskHosts[i] = TaskHost{
			TaskID: tb.task.TaskID,
			HostID: hostID,
			Status: TaskStatusPending,
		}
	}
	return tb
}

// Build 构建任务
func (tb *TaskBuilder) Build() *Task {
	tb.task.CreatedAt = time.Now()
	return tb.task
}

// Task 方法扩展

// IsCompleted 检查任务是否已完成
func (t *Task) IsCompleted() bool {
	return t.Status == TaskStatusCompleted ||
		t.Status == TaskStatusFailed ||
		t.Status == TaskStatusCanceled
}

// IsRunning 检查任务是否正在运行
func (t *Task) IsRunning() bool {
	return t.Status == TaskStatusRunning
}

// IsPending 检查任务是否待执行
func (t *Task) IsPending() bool {
	return t.Status == TaskStatusPending
}

// Duration 获取任务执行时长
func (t *Task) Duration() time.Duration {
	if t.StartedAt == nil || t.FinishedAt == nil {
		return 0
	}
	return t.FinishedAt.Sub(*t.StartedAt)
}

// SuccessRate 获取任务成功率
func (t *Task) SuccessRate() float64 {
	if t.TotalHosts == 0 {
		return 0
	}
	return float64(t.CompletedHosts) / float64(t.TotalHosts) * 100
}

// UpdateProgress 更新任务进度
func (t *Task) UpdateProgress() {
	completedCount := 0
	failedCount := 0
	runningCount := 0

	for _, th := range t.TaskHosts {
		switch th.Status {
		case TaskStatusCompleted:
			completedCount++
		case TaskStatusFailed:
			failedCount++
		case TaskStatusRunning:
			runningCount++
		}
	}

	t.CompletedHosts = completedCount
	t.FailedHosts = failedCount

	// 更新任务状态
	if completedCount+failedCount == t.TotalHosts {
		// 所有主机都完成了
		if failedCount == 0 {
			t.Status = TaskStatusCompleted
		} else {
			t.Status = TaskStatusFailed
		}
		if t.FinishedAt == nil {
			now := time.Now()
			t.FinishedAt = &now
		}
	} else if runningCount > 0 || completedCount > 0 {
		// 有主机在运行或已完成
		t.Status = TaskStatusRunning
		if t.StartedAt == nil {
			now := time.Now()
			t.StartedAt = &now
		}
	}
}
