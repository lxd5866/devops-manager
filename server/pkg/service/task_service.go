package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"devops-manager/api/models"

	"github.com/google/uuid"
)

// TaskService 任务服务
type TaskService struct {
	tasks      map[string]*models.Task
	tasksMutex sync.RWMutex
}

var (
	taskServiceInstance *TaskService
	taskServiceOnce     sync.Once
)

// GetTaskService 获取任务服务单例
func GetTaskService() *TaskService {
	taskServiceOnce.Do(func() {
		taskServiceInstance = &TaskService{
			tasks: make(map[string]*models.Task),
		}
	})
	return taskServiceInstance
}

// CreateTask 创建任务
func (ts *TaskService) CreateTask(name, description string, hostIDs []string, command string, timeout int, parameters string, createdBy string) (*models.Task, error) {
	ts.tasksMutex.Lock()
	defer ts.tasksMutex.Unlock()


	// 生成任务ID
	taskID := "task-" + uuid.New().String()


	// 1.开启mysql 事务
	// 使用 command  timeout parameters  创建  

	// 创建任务
	task := &models.Task{
		TaskID:      taskID,
		Name:        name,
		Description: description,
		CreatedBy:   createdBy,
		Status:      models.TaskStatusPending,
		TotalHosts:  len(hostIDs),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 创建任务主机关联
	for _, hostID := range hostIDs {
		taskHost := models.TaskHost{
			TaskID: taskID,
			HostID: hostID,
			Status: models.TaskStatusPending,
		}
		task.TaskHosts = append(task.TaskHosts, taskHost)
	}

	// 存储任务到内存映射表中
	ts.tasks[taskID] = task

	log.Printf("Task created: %s with %d hosts", taskID, len(hostIDs))
	return task, nil
}

// GetTask 获取单个任务
func (ts *TaskService) GetTask(taskID string) (*models.Task, error) {
	ts.tasksMutex.RLock()
	defer ts.tasksMutex.RUnlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// GetTasks 获取任务列表
func (ts *TaskService) GetTasks(page, size int, status, name string) ([]*models.Task, int, error) {
	ts.tasksMutex.RLock()
	defer ts.tasksMutex.RUnlock()

	var filteredTasks []*models.Task

	// 过滤任务
	for _, task := range ts.tasks {
		// 状态过滤
		if status != "" && string(task.Status) != status {
			continue
		}
		// 名称过滤
		if name != "" && task.Name != name {
			continue
		}
		filteredTasks = append(filteredTasks, task)
	}

	total := len(filteredTasks)

	// 分页
	start := (page - 1) * size
	end := start + size

	if start >= total {
		return []*models.Task{}, total, nil
	}

	if end > total {
		end = total
	}

	return filteredTasks[start:end], total, nil
}

// UpdateTask 更新任务
func (ts *TaskService) UpdateTask(taskID string, updates map[string]interface{}) error {
	ts.tasksMutex.Lock()
	defer ts.tasksMutex.Unlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// 更新字段
	if name, ok := updates["name"].(string); ok {
		task.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		task.Description = description
	}

	task.UpdatedAt = time.Now()
	return nil
}

// DeleteTask 删除任务
func (ts *TaskService) DeleteTask(taskID string) error {
	ts.tasksMutex.Lock()
	defer ts.tasksMutex.Unlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// 检查任务状态
	if task.IsRunning() {
		return fmt.Errorf("cannot delete running task: %s", taskID)
	}

	delete(ts.tasks, taskID)
	log.Printf("Task deleted: %s", taskID)
	return nil
}

// StartTask 启动任务
func (ts *TaskService) StartTask(taskID string) error {
	ts.tasksMutex.Lock()
	defer ts.tasksMutex.Unlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if !task.IsPending() {
		return fmt.Errorf("task is not in pending status: %s", taskID)
	}

	// 更新任务状态
	task.Status = models.TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now
	task.UpdatedAt = now

	// TODO: 向所有目标主机发送命令
	// 这里需要通过其他方式与gRPC控制器通信，避免循环导入

	log.Printf("Task started: %s", taskID)
	return nil
}

// StopTask 停止任务
func (ts *TaskService) StopTask(taskID string) error {
	ts.tasksMutex.Lock()
	defer ts.tasksMutex.Unlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if !task.IsRunning() {
		return fmt.Errorf("task is not running: %s", taskID)
	}

	// 更新任务状态
	task.Status = models.TaskStatusCanceled
	now := time.Now()
	task.FinishedAt = &now
	task.UpdatedAt = now

	log.Printf("Task stopped: %s", taskID)
	return nil
}

// CancelTask 取消任务
func (ts *TaskService) CancelTask(taskID string) error {
	return ts.StopTask(taskID) // 取消和停止逻辑相同
}

// GetTaskStatus 获取任务状态
func (ts *TaskService) GetTaskStatus(taskID string) (map[string]interface{}, error) {
	ts.tasksMutex.RLock()
	defer ts.tasksMutex.RUnlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	status := map[string]interface{}{
		"task_id":         task.TaskID,
		"status":          task.Status,
		"total_hosts":     task.TotalHosts,
		"completed_hosts": task.CompletedHosts,
		"failed_hosts":    task.FailedHosts,
		"success_rate":    task.SuccessRate(),
		"started_at":      task.StartedAt,
		"finished_at":     task.FinishedAt,
	}

	if task.IsCompleted() {
		status["duration"] = task.Duration().Seconds()
	}

	return status, nil
}

// GetTaskProgress 获取任务进度
func (ts *TaskService) GetTaskProgress(taskID string) (map[string]interface{}, error) {
	status, err := ts.GetTaskStatus(taskID)
	if err != nil {
		return nil, err
	}

	// 添加进度信息
	task, _ := ts.tasks[taskID]
	progress := map[string]interface{}{
		"task_id":      taskID,
		"progress":     float64(task.CompletedHosts+task.FailedHosts) / float64(task.TotalHosts) * 100,
		"status":       status,
		"host_details": ts.getHostProgress(task),
	}

	return progress, nil
}

// getHostProgress 获取主机进度详情
func (ts *TaskService) getHostProgress(task *models.Task) []map[string]interface{} {
	var hostProgress []map[string]interface{}

	for _, taskHost := range task.TaskHosts {
		hostProgress = append(hostProgress, map[string]interface{}{
			"host_id": taskHost.HostID,
			"status":  taskHost.Status,
		})
	}

	return hostProgress
}

// AddTaskHosts 添加任务主机
func (ts *TaskService) AddTaskHosts(taskID string, hostIDs []string) error {
	ts.tasksMutex.Lock()
	defer ts.tasksMutex.Unlock()

	// 开启mysql 事物，
	//1. 数据库保存 task

	task, exists := ts.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.IsRunning() {
		return fmt.Errorf("cannot add hosts to running task: %s", taskID)
	}

	// 添加新主机
	for _, hostID := range hostIDs {
		taskHost := models.TaskHost{
			TaskID: taskID,
			HostID: hostID,
			Status: models.TaskStatusPending,
		}
		task.TaskHosts = append(task.TaskHosts, taskHost)
	}

	task.TotalHosts = len(task.TaskHosts)
	task.UpdatedAt = time.Now()

	log.Printf("Added %d hosts to task: %s", len(hostIDs), taskID)
	return nil
}

// RemoveTaskHost 移除任务主机
func (ts *TaskService) RemoveTaskHost(taskID, hostID string) error {
	ts.tasksMutex.Lock()
	defer ts.tasksMutex.Unlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.IsRunning() {
		return fmt.Errorf("cannot remove host from running task: %s", taskID)
	}

	// 移除主机
	var newTaskHosts []models.TaskHost
	for _, taskHost := range task.TaskHosts {
		if taskHost.HostID != hostID {
			newTaskHosts = append(newTaskHosts, taskHost)
		}
	}

	task.TaskHosts = newTaskHosts
	task.TotalHosts = len(task.TaskHosts)
	task.UpdatedAt = time.Now()

	log.Printf("Removed host %s from task: %s", hostID, taskID)
	return nil
}

// GetTaskHosts 获取任务主机列表
func (ts *TaskService) GetTaskHosts(taskID string) ([]models.TaskHost, error) {
	ts.tasksMutex.RLock()
	defer ts.tasksMutex.RUnlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task.TaskHosts, nil
}

// GetTaskCommands 获取任务命令列表
func (ts *TaskService) GetTaskCommands(taskID string) ([]models.Command, error) {
	ts.tasksMutex.RLock()
	defer ts.tasksMutex.RUnlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task.Commands, nil
}

// GetTaskLogs 获取任务日志
func (ts *TaskService) GetTaskLogs(taskID string) ([]string, error) {
	ts.tasksMutex.RLock()
	defer ts.tasksMutex.RUnlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// 简化实现，返回基本日志信息
	logs := []string{
		fmt.Sprintf("Task %s created at %s", taskID, task.CreatedAt.Format(time.RFC3339)),
	}

	if task.StartedAt != nil {
		logs = append(logs, fmt.Sprintf("Task %s started at %s", taskID, task.StartedAt.Format(time.RFC3339)))
	}

	if task.FinishedAt != nil {
		logs = append(logs, fmt.Sprintf("Task %s finished at %s", taskID, task.FinishedAt.Format(time.RFC3339)))
	}

	return logs, nil
}
