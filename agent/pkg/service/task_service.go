package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"devops-manager/agent/pkg/utils"
	"devops-manager/api/protobuf"
)

// TaskService 任务执行服务
type TaskService struct {
	runningTasks map[string]*TaskExecution
	mutex        sync.RWMutex
}

// TaskExecution 任务执行状态
type TaskExecution struct {
	TaskID    string
	Command   string
	Status    string // running, completed, failed, canceled
	StartTime time.Time
	EndTime   *time.Time
	Result    *utils.CommandResult
	Cancel    context.CancelFunc
}

// NewTaskService 创建任务服务
func NewTaskService() *TaskService {
	return &TaskService{
		runningTasks: make(map[string]*TaskExecution),
	}
}

// ExecuteTask 执行任务
func (ts *TaskService) ExecuteTask(taskID, command string, timeout time.Duration) (*utils.CommandResult, error) {
	// 验证命令安全性
	if err := utils.ValidateCommand(command); err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	// 检查任务是否已在执行
	ts.mutex.Lock()
	if _, exists := ts.runningTasks[taskID]; exists {
		ts.mutex.Unlock()
		return nil, fmt.Errorf("task %s is already running", taskID)
	}

	// 创建任务执行记录
	ctx, cancel := context.WithCancel(context.Background())
	execution := &TaskExecution{
		TaskID:    taskID,
		Command:   command,
		Status:    "running",
		StartTime: time.Now(),
		Cancel:    cancel,
	}
	ts.runningTasks[taskID] = execution
	ts.mutex.Unlock()

	log.Printf("Starting task execution: %s, command: %s", taskID, command)

	// 异步执行命令
	go func() {
		defer func() {
			ts.mutex.Lock()
			execution.Status = "completed"
			now := time.Now()
			execution.EndTime = &now
			ts.mutex.Unlock()
		}()

		result := utils.ExecuteCommand(command, timeout)

		ts.mutex.Lock()
		execution.Result = result
		if result.ExitCode != 0 {
			execution.Status = "failed"
		}
		ts.mutex.Unlock()

		log.Printf("Task %s completed with exit code: %d", taskID, result.ExitCode)
	}()

	// 等待任务完成或超时
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("task canceled")
	case <-time.After(timeout + time.Second): // 给一点额外时间
		ts.mutex.RLock()
		result := execution.Result
		ts.mutex.RUnlock()

		if result != nil {
			return result, nil
		}
		return nil, fmt.Errorf("task timeout")
	}
}

// GetTaskStatus 获取任务状态
func (ts *TaskService) GetTaskStatus(taskID string) (*TaskExecution, bool) {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	execution, exists := ts.runningTasks[taskID]
	return execution, exists
}

// CancelTask 取消任务
func (ts *TaskService) CancelTask(taskID string) error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	execution, exists := ts.runningTasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if execution.Status == "running" {
		execution.Cancel()
		execution.Status = "canceled"
		now := time.Now()
		execution.EndTime = &now
		log.Printf("Task %s canceled", taskID)
	}

	return nil
}

// GetRunningTasks 获取正在运行的任务列表
func (ts *TaskService) GetRunningTasks() []string {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	var tasks []string
	for taskID, execution := range ts.runningTasks {
		if execution.Status == "running" {
			tasks = append(tasks, taskID)
		}
	}
	return tasks
}

// CleanupCompletedTasks 清理已完成的任务
func (ts *TaskService) CleanupCompletedTasks(maxAge time.Duration) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for taskID, execution := range ts.runningTasks {
		if execution.Status != "running" && execution.EndTime != nil && execution.EndTime.Before(cutoff) {
			delete(ts.runningTasks, taskID)
			log.Printf("Cleaned up completed task: %s", taskID)
		}
	}
}

// ConvertToProtobuf 转换为protobuf格式
func (te *TaskExecution) ConvertToProtobuf() *protobuf.CommandResult {
	result := &protobuf.CommandResult{
		CommandId: te.TaskID,
	}

	if te.Result != nil {
		result.Stdout = te.Result.Stdout
		result.Stderr = te.Result.Stderr
		result.ExitCode = int32(te.Result.ExitCode)
		result.ErrorMessage = te.Result.Error
	}

	return result
}
