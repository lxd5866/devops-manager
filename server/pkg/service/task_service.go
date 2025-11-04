package service

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"devops-manager/api/models"
	"devops-manager/server/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskService 任务服务
type TaskService struct {
	db               *gorm.DB
	timeoutMonitor   *TimeoutMonitor
	dbOptimizer      *DatabaseOptimizer
	cacheService     *TaskCacheService
	auditService     *AuditService
	batchUpdateQueue chan BatchUpdate
	batchSize        int
	batchTimeout     time.Duration
	queueManager     *TaskQueueManager
	loadMonitor      *SystemLoadMonitor
}

var (
	taskServiceInstance *TaskService
	taskServiceOnce     sync.Once
)

// GetTaskService 获取任务服务单例
func GetTaskService() *TaskService {
	taskServiceOnce.Do(func() {
		taskServiceInstance = &TaskService{
			db:               database.GetDB(),
			batchUpdateQueue: make(chan BatchUpdate, 1000),
			batchSize:        50,              // 批量处理大小
			batchTimeout:     5 * time.Second, // 批量处理超时
		}
		// 初始化缓存服务
		taskServiceInstance.cacheService = NewTaskCacheService()
		// 初始化审计服务
		taskServiceInstance.auditService = NewAuditService()
		// 创建审计相关的数据库表
		if err := taskServiceInstance.db.AutoMigrate(
			&AuditLog{},
			&TaskExecutionLog{},
			&ExecutionStatistics{},
		); err != nil {
			log.Printf("Failed to migrate audit tables: %v", err)
		}
		// 初始化数据库优化器
		taskServiceInstance.dbOptimizer = NewDatabaseOptimizer(taskServiceInstance.db)
		// 创建优化索引
		if err := taskServiceInstance.dbOptimizer.CreateOptimizedIndexes(); err != nil {
			log.Printf("Failed to create optimized indexes: %v", err)
		}
		// 初始化超时监控器
		taskServiceInstance.timeoutMonitor = NewTimeoutMonitor(taskServiceInstance.db, taskServiceInstance)
		// 启动超时监控
		taskServiceInstance.timeoutMonitor.Start()
		// 初始化系统负载监控器
		taskServiceInstance.loadMonitor = NewSystemLoadMonitor(10 * time.Second)
		// 初始化任务队列管理器
		queueConfig := TaskQueueConfig{
			MaxConcurrentTasks:     20,
			MaxTasksPerHost:        5,
			QueueCapacity:          1000,
			WorkerCount:            10,
			LoadBalanceStrategy:    ResourceBased,
			AdaptiveThrottling:     true,
			SystemLoadThreshold:    80.0,
			HostLoadUpdateInterval: 30 * time.Second,
		}
		taskServiceInstance.queueManager = NewTaskQueueManager(taskServiceInstance, queueConfig)
		// 启动批量更新处理器
		go taskServiceInstance.startBatchUpdateProcessor()
		// 预热缓存
		go func() {
			if err := taskServiceInstance.cacheService.WarmupCache(); err != nil {
				log.Printf("Failed to warmup cache: %v", err)
			}
		}()
		// 启动定期缓存清理任务
		go taskServiceInstance.startCacheCleanupTask()
		// 启动定期统计更新任务
		go taskServiceInstance.startStatisticsUpdateTask()
	})
	return taskServiceInstance
}

// CreateTask 创建任务
func (ts *TaskService) CreateTask(name, description string, hostIDs []string, command string, timeout int, parameters string, createdBy string) (*models.Task, error) {
	// 生成任务ID
	taskID := "task-" + uuid.New().String()

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

	// 使用数据库事务确保数据一致性
	err := ts.db.Transaction(func(tx *gorm.DB) error {
		// 1. 保存任务到数据库
		if err := tx.Create(task).Error; err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		// 2. 为每个目标主机创建对应的 Command 和 CommandHost 记录
		for _, hostID := range hostIDs {
			// 生成命令ID
			commandID := "cmd-" + uuid.New().String()

			// 创建命令记录
			cmd := &models.Command{
				CommandID:  commandID,
				TaskID:     &taskID,
				HostID:     hostID,
				Command:    command,
				Parameters: parameters,
				Timeout:    int64(timeout),
				Status:     models.CommandStatusPending,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			if err := tx.Create(cmd).Error; err != nil {
				return fmt.Errorf("failed to create command for host %s: %w", hostID, err)
			}

			// 创建命令主机关联记录
			cmdHost := &models.CommandHost{
				CommandID: commandID,
				HostID:    hostID,
				Status:    string(models.CommandHostStatusPending),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := tx.Create(cmdHost).Error; err != nil {
				return fmt.Errorf("failed to create command host for host %s: %w", hostID, err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 异步使任务列表缓存失效
	go func() {
		if ts.cacheService != nil {
			if err := ts.cacheService.InvalidateTaskListCache(); err != nil {
				log.Printf("Failed to invalidate task list cache: %v", err)
			}
			if err := ts.cacheService.InvalidateTaskStatistics(); err != nil {
				log.Printf("Failed to invalidate task statistics cache: %v", err)
			}
		}
	}()

	// 记录任务创建审计日志
	go func() {
		details := map[string]interface{}{
			"task_name":   name,
			"description": description,
			"host_count":  len(hostIDs),
			"host_ids":    hostIDs,
			"command":     command,
			"timeout":     timeout,
			"parameters":  parameters,
		}
		if err := ts.auditService.LogTaskAction(AuditActionTaskCreated, taskID, createdBy, details); err != nil {
			log.Printf("Failed to log task creation audit: %v", err)
		}

		// 记录任务执行日志
		if err := ts.auditService.LogTaskExecution(taskID, "INFO", fmt.Sprintf("Task '%s' created with %d hosts", name, len(hostIDs)), details, "", ""); err != nil {
			log.Printf("Failed to log task execution: %v", err)
		}
	}()

	log.Printf("Task created: %s with %d hosts", taskID, len(hostIDs))
	return task, nil
}

// GetTask 获取单个任务
func (ts *TaskService) GetTask(taskID string) (*models.Task, error) {
	var task models.Task

	// 从数据库查询任务，包含关联的命令信息
	err := ts.db.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// 加载关联的命令和命令主机信息
	var commands []models.Command
	err = ts.db.Where("task_id = ?", taskID).Find(&commands).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load task commands: %w", err)
	}

	// 为每个命令加载关联的 CommandHost 信息
	for i := range commands {
		var commandHosts []models.CommandHost
		err = ts.db.Where("command_id = ?", commands[i].CommandID).Find(&commandHosts).Error
		if err != nil {
			return nil, fmt.Errorf("failed to load command hosts: %w", err)
		}
		commands[i].CommandHosts = commandHosts
	}

	task.Commands = commands
	return &task, nil
}

// GetTasks 获取任务列表
func (ts *TaskService) GetTasks(page, size int, status, name string) ([]*models.Task, int, error) {
	// 生成缓存键
	cacheKey := ts.cacheService.GenerateTaskListCacheKey(page, size, status, name)

	// 尝试从缓存获取
	if cachedTasks, cachedTotal, err := ts.cacheService.GetCachedTaskList(cacheKey); err == nil && cachedTasks != nil {
		log.Printf("Task list cache hit: %s", cacheKey)
		return cachedTasks, cachedTotal, nil
	}

	var tasks []models.Task
	var total int64

	// 构建查询条件
	query := ts.db.Model(&models.Task{})

	// 状态过滤
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 名称过滤
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get tasks: %w", err)
	}

	// 转换为指针切片
	result := make([]*models.Task, len(tasks))
	for i := range tasks {
		result[i] = &tasks[i]
	}

	// 异步缓存结果
	go func() {
		if err := ts.cacheService.CacheTaskList(cacheKey, result, int(total)); err != nil {
			log.Printf("Failed to cache task list: %v", err)
		}
	}()

	return result, int(total), nil
}

// UpdateTask 更新任务
func (ts *TaskService) UpdateTask(taskID string, updates map[string]interface{}) error {
	// 添加更新时间
	updates["updated_at"] = time.Now()

	// 更新数据库记录
	result := ts.db.Model(&models.Task{}).Where("task_id = ?", taskID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update task: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found: %s", taskID)
	}

	return nil
}

// DeleteTask 删除任务
func (ts *TaskService) DeleteTask(taskID string) error {
	// 先检查任务是否存在和状态
	var task models.Task
	err := ts.db.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("task not found: %s", taskID)
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 检查任务状态
	if task.IsRunning() {
		return fmt.Errorf("cannot delete running task: %s", taskID)
	}

	// 使用事务删除任务及其关联记录
	err = ts.db.Transaction(func(tx *gorm.DB) error {
		// 删除 CommandHost 记录
		if err := tx.Where("command_id IN (SELECT command_id FROM commands WHERE task_id = ?)", taskID).Delete(&models.CommandHost{}).Error; err != nil {
			return fmt.Errorf("failed to delete command hosts: %w", err)
		}

		// 删除 Command 记录
		if err := tx.Where("task_id = ?", taskID).Delete(&models.Command{}).Error; err != nil {
			return fmt.Errorf("failed to delete commands: %w", err)
		}

		// 删除 Task 记录
		if err := tx.Where("task_id = ?", taskID).Delete(&models.Task{}).Error; err != nil {
			return fmt.Errorf("failed to delete task: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("Task deleted: %s", taskID)
	return nil
}

// TaskDispatcher 任务分发器接口，用于与 gRPC 控制器通信
type TaskDispatcher interface {
	SendCommandToAgent(hostID string, command *models.Command) error
}

// taskDispatcher 全局任务分发器实例
var taskDispatcher TaskDispatcher

// SetTaskDispatcher 设置任务分发器
func SetTaskDispatcher(dispatcher TaskDispatcher) {
	taskDispatcher = dispatcher
}

// StartTask 启动任务 - 实现真正的任务下发逻辑
func (ts *TaskService) StartTask(taskID string) error {
	// 使用事务确保数据一致性
	return ts.db.Transaction(func(tx *gorm.DB) error {
		// 1. 检查任务状态
		var task models.Task
		err := tx.Where("task_id = ?", taskID).First(&task).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("task not found: %s", taskID)
			}
			return fmt.Errorf("failed to get task: %w", err)
		}

		if !task.IsPending() {
			return fmt.Errorf("task is not in pending status: %s", taskID)
		}

		// 2. 更新任务状态为运行中
		now := time.Now()
		taskUpdates := map[string]interface{}{
			"status":     models.TaskStatusRunning,
			"started_at": now,
			"updated_at": now,
		}

		err = tx.Model(&models.Task{}).Where("task_id = ?", taskID).Updates(taskUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to update task status: %w", err)
		}

		// 3. 获取任务的所有命令
		var commands []models.Command
		err = tx.Where("task_id = ?", taskID).Find(&commands).Error
		if err != nil {
			return fmt.Errorf("failed to get task commands: %w", err)
		}

		// 4. 向所有目标主机下发命令
		for _, cmd := range commands {
			// 更新命令状态为待下发
			cmdUpdates := map[string]interface{}{
				"status":     models.CommandStatusPending,
				"updated_at": now,
			}
			err = tx.Model(&models.Command{}).Where("command_id = ?", cmd.CommandID).Updates(cmdUpdates).Error
			if err != nil {
				return fmt.Errorf("failed to update command status: %w", err)
			}

			// 更新 CommandHost 状态为待下发
			hostUpdates := map[string]interface{}{
				"status":     string(models.CommandHostStatusPending),
				"updated_at": now,
			}
			err = tx.Model(&models.CommandHost{}).Where("command_id = ?", cmd.CommandID).Updates(hostUpdates).Error
			if err != nil {
				return fmt.Errorf("failed to update command host status: %w", err)
			}

			// 通过 gRPC 控制器向 Agent 发送命令
			if taskDispatcher != nil {
				// 异步发送命令，避免阻塞事务
				go func(command models.Command) {
					err := taskDispatcher.SendCommandToAgent(command.HostID, &command)
					if err != nil {
						log.Printf("Failed to send command %s to agent %s: %v", command.CommandID, command.HostID, err)
						// 更新命令状态为下发失败
						ts.updateCommandDispatchFailed(command.CommandID, err.Error())
					} else {
						log.Printf("Command %s sent to agent %s successfully", command.CommandID, command.HostID)
					}
				}(cmd)
			} else {
				log.Printf("Warning: TaskDispatcher not set, command %s not sent to agent %s", cmd.CommandID, cmd.HostID)
			}
		}

		log.Printf("Task started: %s with %d commands", taskID, len(commands))

		// 异步记录审计日志和使缓存失效
		go func() {
			// 记录任务启动审计日志
			details := map[string]interface{}{
				"task_name":     task.Name,
				"command_count": len(commands),
				"host_ids": func() []string {
					hostIDs := make([]string, len(commands))
					for i, cmd := range commands {
						hostIDs[i] = cmd.HostID
					}
					return hostIDs
				}(),
			}
			if err := ts.auditService.LogTaskAction(AuditActionTaskStarted, taskID, task.CreatedBy, details); err != nil {
				log.Printf("Failed to log task start audit: %v", err)
			}

			// 记录任务执行日志
			if err := ts.auditService.LogTaskExecution(taskID, "INFO", fmt.Sprintf("Task '%s' started with %d commands", task.Name, len(commands)), details, "", ""); err != nil {
				log.Printf("Failed to log task execution: %v", err)
			}

			// 为每个命令记录下发日志
			for _, cmd := range commands {
				cmdDetails := map[string]interface{}{
					"command":    cmd.Command,
					"parameters": cmd.Parameters,
					"timeout":    cmd.Timeout,
				}
				if err := ts.auditService.LogCommandAction(AuditActionCommandSent, cmd.CommandID, cmd.HostID, task.CreatedBy, cmdDetails); err != nil {
					log.Printf("Failed to log command send audit: %v", err)
				}

				if err := ts.auditService.LogTaskExecution(taskID, "INFO", fmt.Sprintf("Command sent to host %s", cmd.HostID), cmdDetails, cmd.HostID, cmd.CommandID); err != nil {
					log.Printf("Failed to log command execution: %v", err)
				}
			}

			// 使相关缓存失效
			if err := ts.cacheService.InvalidateTaskCache(taskID); err != nil {
				log.Printf("Failed to invalidate task cache: %v", err)
			}
			if err := ts.cacheService.InvalidateTaskListCache(); err != nil {
				log.Printf("Failed to invalidate task list cache: %v", err)
			}
		}()

		return nil
	})
}

// StartTaskWithQueue 通过队列启动任务
func (ts *TaskService) StartTaskWithQueue(taskID string, priority TaskPriority) error {
	// 获取任务信息
	task, err := ts.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if !task.IsPending() {
		return fmt.Errorf("task is not in pending status: %s", taskID)
	}

	// 提取主机ID列表
	hostIDs := make([]string, 0)
	for _, cmd := range task.Commands {
		hostIDs = append(hostIDs, cmd.HostID)
	}

	// 将任务加入队列
	err = ts.queueManager.EnqueueTask(taskID, priority, hostIDs)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.Printf("Task %s enqueued with priority %d", taskID, priority)
	return nil
}

// updateCommandDispatchFailed 更新命令下发失败状态
func (ts *TaskService) updateCommandDispatchFailed(commandID, errorMsg string) {
	now := time.Now()

	// 更新命令状态
	cmdUpdates := map[string]interface{}{
		"status":     models.CommandStatusFailed,
		"error_msg":  errorMsg,
		"updated_at": now,
	}
	ts.db.Model(&models.Command{}).Where("command_id = ?", commandID).Updates(cmdUpdates)

	// 更新 CommandHost 状态
	hostUpdates := map[string]interface{}{
		"status":        string(models.CommandHostStatusFailed),
		"error_message": errorMsg,
		"updated_at":    now,
	}
	ts.db.Model(&models.CommandHost{}).Where("command_id = ?", commandID).Updates(hostUpdates)
}

// StopTask 停止任务
func (ts *TaskService) StopTask(taskID string) error {
	// 先检查任务状态
	var task models.Task
	err := ts.db.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("task not found: %s", taskID)
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	if !task.IsRunning() {
		return fmt.Errorf("task is not running: %s", taskID)
	}

	// 更新任务状态
	now := time.Now()
	updates := map[string]interface{}{
		"status":      models.TaskStatusCanceled,
		"finished_at": now,
		"updated_at":  now,
	}

	err = ts.db.Model(&models.Task{}).Where("task_id = ?", taskID).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	log.Printf("Task stopped: %s", taskID)
	return nil
}

// CancelTask 取消任务 - 实现真正的任务取消逻辑
func (ts *TaskService) CancelTask(taskID string) error {
	// 使用事务确保数据一致性
	return ts.db.Transaction(func(tx *gorm.DB) error {
		// 1. 检查任务状态
		var task models.Task
		err := tx.Where("task_id = ?", taskID).First(&task).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("task not found: %s", taskID)
			}
			return fmt.Errorf("failed to get task: %w", err)
		}

		if task.IsCompleted() {
			return fmt.Errorf("task is already completed: %s", taskID)
		}

		// 2. 更新任务状态为已取消
		now := time.Now()
		taskUpdates := map[string]interface{}{
			"status":      models.TaskStatusCanceled,
			"finished_at": now,
			"updated_at":  now,
		}

		err = tx.Model(&models.Task{}).Where("task_id = ?", taskID).Updates(taskUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to update task status: %w", err)
		}

		// 3. 取消所有未完成的命令
		err = tx.Model(&models.Command{}).
			Where("task_id = ? AND status IN (?)", taskID, []models.CommandStatus{
				models.CommandStatusPending,
				models.CommandStatusRunning,
			}).
			Updates(map[string]interface{}{
				"status":      models.CommandStatusCanceled,
				"finished_at": now,
				"updated_at":  now,
			}).Error
		if err != nil {
			return fmt.Errorf("failed to cancel commands: %w", err)
		}

		// 4. 取消所有未完成的 CommandHost
		err = tx.Model(&models.CommandHost{}).
			Where("command_id IN (SELECT command_id FROM commands WHERE task_id = ?) AND status IN (?)",
				taskID, []string{
					string(models.CommandHostStatusPending),
					string(models.CommandHostStatusRunning),
				}).
			Updates(map[string]interface{}{
				"status":      string(models.CommandHostStatusCanceled),
				"finished_at": now,
				"updated_at":  now,
			}).Error
		if err != nil {
			return fmt.Errorf("failed to cancel command hosts: %w", err)
		}

		// 5. 通知所有相关的 Agent 取消命令执行
		if taskDispatcher != nil {
			// 获取所有需要取消的命令
			var commands []models.Command
			err = tx.Where("task_id = ? AND status = ?", taskID, models.CommandStatusCanceled).Find(&commands).Error
			if err != nil {
				log.Printf("Failed to get canceled commands for task %s: %v", taskID, err)
			} else {
				// 异步通知 Agent 取消命令
				for _, cmd := range commands {
					go func(command models.Command) {
						err := ts.sendCancelCommandToAgent(command)
						if err != nil {
							log.Printf("Failed to send cancel command to agent %s: %v", command.HostID, err)
						} else {
							log.Printf("Cancel command sent to agent %s for command %s", command.HostID, command.CommandID)
						}
					}(cmd)
				}
			}
		}

		log.Printf("Task canceled: %s", taskID)

		// 异步使相关缓存失效
		go func() {
			if err := ts.cacheService.InvalidateTaskCache(taskID); err != nil {
				log.Printf("Failed to invalidate task cache: %v", err)
			}
			if err := ts.cacheService.InvalidateTaskListCache(); err != nil {
				log.Printf("Failed to invalidate task list cache: %v", err)
			}
		}()

		return nil
	})
}

// GetTaskStatus 获取任务状态
func (ts *TaskService) GetTaskStatus(taskID string) (map[string]interface{}, error) {
	// 尝试从缓存获取
	if cachedStatus, err := ts.cacheService.GetCachedTaskStatus(taskID); err == nil && cachedStatus != nil {
		log.Printf("Task status cache hit: %s", taskID)
		return cachedStatus, nil
	}

	// 使用事务确保数据一致性
	var status map[string]interface{}
	err := ts.db.Transaction(func(tx *gorm.DB) error {
		// 获取任务基本信息
		var task models.Task
		err := tx.Where("task_id = ?", taskID).First(&task).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("task not found: %s", taskID)
			}
			return fmt.Errorf("failed to get task: %w", err)
		}

		// 统计 CommandHost 状态
		var statusCounts []struct {
			Status string
			Count  int64
		}

		err = tx.Model(&models.CommandHost{}).
			Select("status, COUNT(*) as count").
			Where("command_id IN (SELECT command_id FROM commands WHERE task_id = ?)", taskID).
			Group("status").
			Scan(&statusCounts).Error
		if err != nil {
			return fmt.Errorf("failed to count command host status: %w", err)
		}

		// 计算各状态的主机数量
		completedCount := int64(0)
		failedCount := int64(0)
		runningCount := int64(0)
		pendingCount := int64(0)
		canceledCount := int64(0)

		for _, sc := range statusCounts {
			switch sc.Status {
			case string(models.CommandHostStatusCompleted):
				completedCount = sc.Count
			case string(models.CommandHostStatusFailed),
				string(models.CommandHostStatusExecFailed),
				string(models.CommandHostStatusTimeout):
				failedCount = sc.Count
			case string(models.CommandHostStatusRunning):
				runningCount = sc.Count
			case string(models.CommandHostStatusPending):
				pendingCount = sc.Count
			case string(models.CommandHostStatusCanceled):
				canceledCount = sc.Count
			}
		}

		// 计算成功率
		totalProcessed := completedCount + failedCount
		successRate := float64(0)
		if totalProcessed > 0 {
			successRate = float64(completedCount) / float64(totalProcessed) * 100
		}

		// 计算进度百分比
		progressPercent := float64(0)
		if task.TotalHosts > 0 {
			progressPercent = float64(completedCount+failedCount+canceledCount) / float64(task.TotalHosts) * 100
		}

		// 更新任务状态
		now := time.Now()
		taskUpdates := map[string]interface{}{
			"completed_hosts": completedCount,
			"failed_hosts":    failedCount,
			"updated_at":      now,
		}

		// 判断任务整体状态
		totalFinished := completedCount + failedCount + canceledCount
		if totalFinished == int64(task.TotalHosts) {
			// 所有主机都完成了
			if canceledCount > 0 {
				taskUpdates["status"] = models.TaskStatusCanceled
			} else if failedCount == 0 {
				taskUpdates["status"] = models.TaskStatusCompleted
			} else {
				taskUpdates["status"] = models.TaskStatusFailed
			}
			if task.FinishedAt == nil {
				taskUpdates["finished_at"] = now
			}
		} else if runningCount > 0 || completedCount > 0 {
			// 有主机在运行或已完成
			taskUpdates["status"] = models.TaskStatusRunning
			if task.StartedAt == nil {
				taskUpdates["started_at"] = now
			}
		}

		// 更新任务记录
		err = tx.Model(&models.Task{}).Where("task_id = ?", taskID).Updates(taskUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		// 重新获取更新后的任务信息
		err = tx.Where("task_id = ?", taskID).First(&task).Error
		if err != nil {
			return fmt.Errorf("failed to get updated task: %w", err)
		}

		// 构建状态响应
		status = map[string]interface{}{
			"task_id":         task.TaskID,
			"name":            task.Name,
			"description":     task.Description,
			"status":          task.Status,
			"total_hosts":     task.TotalHosts,
			"completed_hosts": completedCount,
			"failed_hosts":    failedCount,
			"running_hosts":   runningCount,
			"pending_hosts":   pendingCount,
			"canceled_hosts":  canceledCount,
			"success_rate":    successRate,
			"progress":        progressPercent,
			"created_by":      task.CreatedBy,
			"created_at":      task.CreatedAt,
			"started_at":      task.StartedAt,
			"finished_at":     task.FinishedAt,
			"updated_at":      task.UpdatedAt,
		}

		// 计算执行时长
		if task.StartedAt != nil {
			if task.FinishedAt != nil {
				duration := task.FinishedAt.Sub(*task.StartedAt)
				status["duration_seconds"] = duration.Seconds()
			} else {
				duration := time.Since(*task.StartedAt)
				status["duration_seconds"] = duration.Seconds()
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 异步缓存结果
	go func() {
		if err := ts.cacheService.CacheTaskStatus(taskID, status); err != nil {
			log.Printf("Failed to cache task status: %v", err)
		}
	}()

	return status, nil
}

// GetTaskProgress 获取任务进度
func (ts *TaskService) GetTaskProgress(taskID string) (map[string]interface{}, error) {
	// 尝试从缓存获取
	if cachedProgress, err := ts.cacheService.GetCachedTaskProgress(taskID); err == nil && cachedProgress != nil {
		log.Printf("Task progress cache hit: %s", taskID)
		return cachedProgress, nil
	}

	// 获取任务状态
	status, err := ts.GetTaskStatus(taskID)
	if err != nil {
		return nil, err
	}

	// 获取详细的主机进度信息
	hostDetails, err := ts.getHostProgressDetails(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get host progress details: %w", err)
	}

	progress := map[string]interface{}{
		"task_id":      taskID,
		"progress":     status["progress"],
		"status":       status,
		"host_details": hostDetails,
	}

	// 异步缓存结果
	go func() {
		if err := ts.cacheService.CacheTaskProgress(taskID, progress); err != nil {
			log.Printf("Failed to cache task progress: %v", err)
		}
	}()

	return progress, nil
}

// getHostProgressDetails 获取主机进度详情
func (ts *TaskService) getHostProgressDetails(taskID string) ([]map[string]interface{}, error) {
	var hostDetails []map[string]interface{}

	// 查询任务的所有 CommandHost 记录，包含详细信息
	var commandHosts []models.CommandHost
	err := ts.db.Where("command_id IN (SELECT command_id FROM commands WHERE task_id = ?)", taskID).
		Order("created_at ASC").
		Find(&commandHosts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get command hosts: %w", err)
	}

	// 构建主机进度详情
	for _, cmdHost := range commandHosts {
		detail := map[string]interface{}{
			"host_id":        cmdHost.HostID,
			"command_id":     cmdHost.CommandID,
			"status":         cmdHost.Status,
			"exit_code":      cmdHost.ExitCode,
			"started_at":     cmdHost.StartedAt,
			"finished_at":    cmdHost.FinishedAt,
			"execution_time": cmdHost.ExecutionTime,
			"error_message":  cmdHost.ErrorMessage,
			"created_at":     cmdHost.CreatedAt,
			"updated_at":     cmdHost.UpdatedAt,
		}

		// 计算执行时长（如果有）
		if cmdHost.StartedAt != nil && cmdHost.FinishedAt != nil {
			duration := cmdHost.FinishedAt.Sub(*cmdHost.StartedAt)
			detail["duration_seconds"] = duration.Seconds()
		} else if cmdHost.StartedAt != nil {
			duration := time.Since(*cmdHost.StartedAt)
			detail["duration_seconds"] = duration.Seconds()
		}

		// 添加输出信息（截断长输出）
		if len(cmdHost.Stdout) > 0 {
			if len(cmdHost.Stdout) > 500 {
				detail["stdout_preview"] = cmdHost.Stdout[:500] + "..."
				detail["stdout_length"] = len(cmdHost.Stdout)
			} else {
				detail["stdout_preview"] = cmdHost.Stdout
				detail["stdout_length"] = len(cmdHost.Stdout)
			}
		}

		if len(cmdHost.Stderr) > 0 {
			if len(cmdHost.Stderr) > 500 {
				detail["stderr_preview"] = cmdHost.Stderr[:500] + "..."
				detail["stderr_length"] = len(cmdHost.Stderr)
			} else {
				detail["stderr_preview"] = cmdHost.Stderr
				detail["stderr_length"] = len(cmdHost.Stderr)
			}
		}

		hostDetails = append(hostDetails, detail)
	}

	return hostDetails, nil
}

// AddTaskHosts 添加任务主机
func (ts *TaskService) AddTaskHosts(taskID string, hostIDs []string) error {
	// 先检查任务状态
	var task models.Task
	err := ts.db.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("task not found: %s", taskID)
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.IsRunning() {
		return fmt.Errorf("cannot add hosts to running task: %s", taskID)
	}

	// 获取任务的命令信息
	var existingCommand models.Command
	err = ts.db.Where("task_id = ?", taskID).First(&existingCommand).Error
	if err != nil {
		return fmt.Errorf("failed to get task command: %w", err)
	}

	// 使用事务添加新主机
	err = ts.db.Transaction(func(tx *gorm.DB) error {
		// 为每个新主机创建 Command 和 CommandHost 记录
		for _, hostID := range hostIDs {
			// 生成命令ID
			commandID := "cmd-" + uuid.New().String()

			// 创建命令记录
			cmd := &models.Command{
				CommandID:  commandID,
				TaskID:     &taskID,
				HostID:     hostID,
				Command:    existingCommand.Command,
				Parameters: existingCommand.Parameters,
				Timeout:    existingCommand.Timeout,
				Status:     models.CommandStatusPending,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			if err := tx.Create(cmd).Error; err != nil {
				return fmt.Errorf("failed to create command for host %s: %w", hostID, err)
			}

			// 创建命令主机关联记录
			cmdHost := &models.CommandHost{
				CommandID: commandID,
				HostID:    hostID,
				Status:    string(models.CommandHostStatusPending),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := tx.Create(cmdHost).Error; err != nil {
				return fmt.Errorf("failed to create command host for host %s: %w", hostID, err)
			}
		}

		// 更新任务的主机总数
		updates := map[string]interface{}{
			"total_hosts": gorm.Expr("total_hosts + ?", len(hostIDs)),
			"updated_at":  time.Now(),
		}
		if err := tx.Model(&models.Task{}).Where("task_id = ?", taskID).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update task host count: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("Added %d hosts to task: %s", len(hostIDs), taskID)
	return nil
}

// RemoveTaskHost 移除任务主机
func (ts *TaskService) RemoveTaskHost(taskID, hostID string) error {
	// 先检查任务状态
	var task models.Task
	err := ts.db.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("task not found: %s", taskID)
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.IsRunning() {
		return fmt.Errorf("cannot remove host from running task: %s", taskID)
	}

	// 使用事务移除主机
	err = ts.db.Transaction(func(tx *gorm.DB) error {
		// 删除该主机的 CommandHost 记录
		if err := tx.Where("host_id = ? AND command_id IN (SELECT command_id FROM commands WHERE task_id = ?)", hostID, taskID).Delete(&models.CommandHost{}).Error; err != nil {
			return fmt.Errorf("failed to delete command host: %w", err)
		}

		// 删除该主机的 Command 记录
		if err := tx.Where("host_id = ? AND task_id = ?", hostID, taskID).Delete(&models.Command{}).Error; err != nil {
			return fmt.Errorf("failed to delete command: %w", err)
		}

		// 更新任务的主机总数
		updates := map[string]interface{}{
			"total_hosts": gorm.Expr("GREATEST(total_hosts - 1, 0)"),
			"updated_at":  time.Now(),
		}
		if err := tx.Model(&models.Task{}).Where("task_id = ?", taskID).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update task host count: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("Removed host %s from task: %s", hostID, taskID)
	return nil
}

// GetTaskHosts 获取任务主机列表（通过 CommandHost 获取）
func (ts *TaskService) GetTaskHosts(taskID string) ([]models.CommandHost, error) {
	// 先检查任务是否存在
	var task models.Task
	err := ts.db.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// 查询任务的所有 CommandHost 记录
	var commandHosts []models.CommandHost
	err = ts.db.Where("command_id IN (SELECT command_id FROM commands WHERE task_id = ?)", taskID).Find(&commandHosts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get task hosts: %w", err)
	}

	return commandHosts, nil
}

// GetTaskCommands 获取任务命令列表
func (ts *TaskService) GetTaskCommands(taskID string) ([]models.Command, error) {
	// 先检查任务是否存在
	var task models.Task
	err := ts.db.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// 查询任务的所有命令
	var commands []models.Command
	err = ts.db.Where("task_id = ?", taskID).Find(&commands).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get task commands: %w", err)
	}

	return commands, nil
}

// GetTaskLogs 获取任务日志
func (ts *TaskService) GetTaskLogs(taskID string) (map[string]interface{}, error) {
	// 获取任务信息
	task, err := ts.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	// 获取任务执行日志
	execLogs, _, err := ts.auditService.GetTaskExecutionLogs(taskID, 1, 1000, "", nil, nil)
	if err != nil {
		log.Printf("Failed to get task execution logs: %v", err)
		execLogs = []TaskExecutionLog{} // 使用空切片作为默认值
	}

	// 获取审计日志
	auditLogs, _, err := ts.auditService.GetAuditLogs(1, 1000, "", "task", taskID, "", nil, nil)
	if err != nil {
		log.Printf("Failed to get audit logs: %v", err)
		auditLogs = []AuditLog{} // 使用空切片作为默认值
	}

	// 构建基本日志信息
	basicLogs := []string{
		fmt.Sprintf("Task %s (%s) created at %s by %s", taskID, task.Name, task.CreatedAt.Format(time.RFC3339), task.CreatedBy),
	}

	if task.StartedAt != nil {
		basicLogs = append(basicLogs, fmt.Sprintf("Task %s started at %s", taskID, task.StartedAt.Format(time.RFC3339)))
	}

	if task.FinishedAt != nil {
		duration := ""
		if task.StartedAt != nil {
			duration = fmt.Sprintf(" (duration: %s)", task.FinishedAt.Sub(*task.StartedAt).String())
		}
		basicLogs = append(basicLogs, fmt.Sprintf("Task %s finished at %s with status %s%s", taskID, task.FinishedAt.Format(time.RFC3339), task.Status, duration))
	}

	// 添加命令执行摘要
	commandSummary := make([]map[string]interface{}, 0)
	for _, cmd := range task.Commands {
		for _, cmdHost := range cmd.CommandHosts {
			summary := map[string]interface{}{
				"command_id":  cmd.CommandID,
				"host_id":     cmdHost.HostID,
				"command":     cmd.Command,
				"status":      cmdHost.Status,
				"exit_code":   cmdHost.ExitCode,
				"created_at":  cmdHost.CreatedAt,
				"started_at":  cmdHost.StartedAt,
				"finished_at": cmdHost.FinishedAt,
			}

			if cmdHost.ExecutionTime != nil {
				summary["execution_time_ms"] = *cmdHost.ExecutionTime
			}

			if cmdHost.ErrorMessage != "" {
				summary["error_message"] = cmdHost.ErrorMessage
			}

			// 添加输出预览（截断长输出）
			if len(cmdHost.Stdout) > 0 {
				if len(cmdHost.Stdout) > 200 {
					summary["stdout_preview"] = cmdHost.Stdout[:200] + "..."
				} else {
					summary["stdout_preview"] = cmdHost.Stdout
				}
				summary["stdout_length"] = len(cmdHost.Stdout)
			}

			if len(cmdHost.Stderr) > 0 {
				if len(cmdHost.Stderr) > 200 {
					summary["stderr_preview"] = cmdHost.Stderr[:200] + "..."
				} else {
					summary["stderr_preview"] = cmdHost.Stderr
				}
				summary["stderr_length"] = len(cmdHost.Stderr)
			}

			commandSummary = append(commandSummary, summary)
		}
	}

	// 构建完整的日志响应
	logResponse := map[string]interface{}{
		"task_id":         taskID,
		"task_name":       task.Name,
		"task_status":     task.Status,
		"basic_logs":      basicLogs,
		"execution_logs":  execLogs,
		"audit_logs":      auditLogs,
		"command_summary": commandSummary,
		"log_statistics": map[string]interface{}{
			"total_execution_logs": len(execLogs),
			"total_audit_logs":     len(auditLogs),
			"total_commands":       len(commandSummary),
		},
	}

	return logResponse, nil
}

// HandleCommandResult 处理命令执行结果并更新任务状态
func (ts *TaskService) HandleCommandResult(result *models.CommandResult) error {
	// 使用事务更新命令结果和任务状态
	return ts.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 计算执行时长（如果有开始和结束时间）
		if result.StartedAt != nil && result.FinishedAt != nil {
			duration := result.FinishedAt.Sub(*result.StartedAt)
			executionTime := duration.Milliseconds()
			result.ExecutionTime = &executionTime
		}

		// 1. 更新 CommandHost 记录
		hostUpdates := map[string]interface{}{
			"stdout":         result.Stdout,
			"stderr":         result.Stderr,
			"exit_code":      result.ExitCode,
			"started_at":     result.StartedAt,
			"finished_at":    result.FinishedAt,
			"error_message":  result.ErrorMessage,
			"execution_time": result.ExecutionTime,
			"updated_at":     now,
		}

		// 根据执行结果设置 CommandHost 状态
		if result.FinishedAt != nil {
			if result.ExitCode == 0 {
				hostUpdates["status"] = string(models.CommandHostStatusCompleted)
			} else {
				hostUpdates["status"] = string(models.CommandHostStatusExecFailed)
			}
		} else if result.StartedAt != nil {
			hostUpdates["status"] = string(models.CommandHostStatusRunning)
		}

		err := tx.Model(&models.CommandHost{}).Where("command_id = ? AND host_id = ?", result.CommandID, result.HostID).Updates(hostUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to update command host: %w", err)
		}

		// 2. 更新 Command 记录
		cmdUpdates := map[string]interface{}{
			"stdout":      result.Stdout,
			"stderr":      result.Stderr,
			"exit_code":   result.ExitCode,
			"started_at":  result.StartedAt,
			"finished_at": result.FinishedAt,
			"error_msg":   result.ErrorMessage,
			"updated_at":  now,
		}

		// 设置命令状态
		if result.FinishedAt != nil {
			if result.ExitCode == 0 {
				cmdUpdates["status"] = models.CommandStatusCompleted
			} else {
				cmdUpdates["status"] = models.CommandStatusFailed
			}
		} else if result.StartedAt != nil {
			cmdUpdates["status"] = models.CommandStatusRunning
		}

		err = tx.Model(&models.Command{}).Where("command_id = ?", result.CommandID).Updates(cmdUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to update command: %w", err)
		}

		// 3. 保存命令结果记录（避免重复插入）
		result.CreatedAt = now
		result.UpdatedAt = now

		// 使用 ON DUPLICATE KEY UPDATE 或者先检查是否存在
		var existingResult models.CommandResult
		err = tx.Where("command_id = ? AND host_id = ?", result.CommandID, result.HostID).First(&existingResult).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// 记录不存在，创建新记录
				err = tx.Create(result).Error
				if err != nil {
					return fmt.Errorf("failed to create command result: %w", err)
				}
			} else {
				return fmt.Errorf("failed to check existing command result: %w", err)
			}
		} else {
			// 记录已存在，更新现有记录
			err = tx.Model(&existingResult).Updates(map[string]interface{}{
				"stdout":         result.Stdout,
				"stderr":         result.Stderr,
				"exit_code":      result.ExitCode,
				"started_at":     result.StartedAt,
				"finished_at":    result.FinishedAt,
				"error_message":  result.ErrorMessage,
				"execution_time": result.ExecutionTime,
				"updated_at":     now,
			}).Error
			if err != nil {
				return fmt.Errorf("failed to update command result: %w", err)
			}
		}

		// 4. 获取命令所属的任务ID并更新任务状态
		var command models.Command
		err = tx.Where("command_id = ?", result.CommandID).First(&command).Error
		if err != nil {
			return fmt.Errorf("failed to get command: %w", err)
		}

		if command.TaskID != nil {
			// 更新任务进度和状态
			err = ts.updateTaskProgressInTransaction(tx, *command.TaskID)
			if err != nil {
				return fmt.Errorf("failed to update task progress: %w", err)
			}
		}

		log.Printf("Command result processed: command_id=%s, host_id=%s, exit_code=%d, execution_time=%v",
			result.CommandID, result.HostID, result.ExitCode, result.ExecutionTime)

		// 异步记录审计日志和使缓存失效
		go func() {
			// 记录命令执行结果审计日志
			details := map[string]interface{}{
				"exit_code":      result.ExitCode,
				"execution_time": result.ExecutionTime,
				"stdout_length":  len(result.Stdout),
				"stderr_length":  len(result.Stderr),
				"started_at":     result.StartedAt,
				"finished_at":    result.FinishedAt,
				"error_message":  result.ErrorMessage,
			}

			// 根据执行结果选择审计动作
			var auditAction AuditAction
			var logLevel string
			var logMessage string

			if result.FinishedAt != nil {
				if result.ExitCode == 0 {
					auditAction = AuditActionCommandResult
					logLevel = "INFO"
					logMessage = fmt.Sprintf("Command completed successfully on host %s", result.HostID)
				} else {
					auditAction = AuditActionCommandError
					logLevel = "ERROR"
					logMessage = fmt.Sprintf("Command failed on host %s with exit code %d", result.HostID, result.ExitCode)
				}
			} else if result.StartedAt != nil {
				auditAction = AuditActionCommandStarted
				logLevel = "INFO"
				logMessage = fmt.Sprintf("Command started on host %s", result.HostID)
			}

			if err := ts.auditService.LogCommandAction(auditAction, result.CommandID, result.HostID, "", details); err != nil {
				log.Printf("Failed to log command result audit: %v", err)
			}

			// 记录任务执行日志
			if command.TaskID != nil {
				if err := ts.auditService.LogTaskExecution(*command.TaskID, logLevel, logMessage, details, result.HostID, result.CommandID); err != nil {
					log.Printf("Failed to log task execution: %v", err)
				}

				// 使任务相关缓存失效
				if err := ts.cacheService.InvalidateTaskCache(*command.TaskID); err != nil {
					log.Printf("Failed to invalidate task cache: %v", err)
				}
			}

			// 使主机任务缓存失效
			if err := ts.cacheService.InvalidateHostTasksCache(result.HostID); err != nil {
				log.Printf("Failed to invalidate host tasks cache: %v", err)
			}
		}()

		return nil
	})
}

// updateTaskProgressInTransaction 在事务中更新任务进度
func (ts *TaskService) updateTaskProgressInTransaction(tx *gorm.DB, taskID string) error {
	// 获取任务信息
	var task models.Task
	err := tx.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 统计任务中所有 CommandHost 的状态
	var statusCounts []struct {
		Status string
		Count  int64
	}

	err = tx.Model(&models.CommandHost{}).
		Select("status, COUNT(*) as count").
		Where("command_id IN (SELECT command_id FROM commands WHERE task_id = ?)", taskID).
		Group("status").
		Scan(&statusCounts).Error
	if err != nil {
		return fmt.Errorf("failed to count command host status: %w", err)
	}

	// 计算各状态的主机数量
	completedCount := int64(0)
	failedCount := int64(0)
	runningCount := int64(0)
	pendingCount := int64(0)
	canceledCount := int64(0)

	for _, sc := range statusCounts {
		switch sc.Status {
		case string(models.CommandHostStatusCompleted):
			completedCount = sc.Count
		case string(models.CommandHostStatusFailed),
			string(models.CommandHostStatusExecFailed),
			string(models.CommandHostStatusTimeout):
			failedCount = sc.Count
		case string(models.CommandHostStatusRunning):
			runningCount = sc.Count
		case string(models.CommandHostStatusPending):
			pendingCount = sc.Count
		case string(models.CommandHostStatusCanceled):
			canceledCount = sc.Count
		}
	}

	// 计算成功率
	totalProcessed := completedCount + failedCount
	successRate := float64(0)
	if totalProcessed > 0 {
		successRate = float64(completedCount) / float64(totalProcessed) * 100
	}

	// 更新任务状态
	now := time.Now()
	taskUpdates := map[string]interface{}{
		"completed_hosts": completedCount,
		"failed_hosts":    failedCount,
		"updated_at":      now,
	}

	// 判断任务整体状态
	totalFinished := completedCount + failedCount + canceledCount
	if totalFinished == int64(task.TotalHosts) {
		// 所有主机都完成了
		if canceledCount > 0 {
			taskUpdates["status"] = models.TaskStatusCanceled
		} else if failedCount == 0 {
			taskUpdates["status"] = models.TaskStatusCompleted
		} else {
			taskUpdates["status"] = models.TaskStatusFailed
		}
		taskUpdates["finished_at"] = now
	} else if runningCount > 0 || completedCount > 0 {
		// 有主机在运行或已完成
		taskUpdates["status"] = models.TaskStatusRunning
		if task.StartedAt == nil {
			taskUpdates["started_at"] = now
		}
	}

	// 检查任务状态是否发生变化
	oldStatus := task.Status
	var newStatus models.TaskStatus
	if statusInterface, exists := taskUpdates["status"]; exists {
		newStatus = statusInterface.(models.TaskStatus)
	} else {
		newStatus = oldStatus
	}

	// 更新任务记录
	err = tx.Model(&models.Task{}).Where("task_id = ?", taskID).Updates(taskUpdates).Error
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 如果任务状态发生变化，记录审计日志
	if oldStatus != newStatus {
		go func() {
			var auditAction AuditAction
			var logLevel string
			var logMessage string

			switch newStatus {
			case models.TaskStatusCompleted:
				auditAction = AuditActionTaskCompleted
				logLevel = "INFO"
				logMessage = fmt.Sprintf("Task '%s' completed successfully", task.Name)
			case models.TaskStatusFailed:
				auditAction = AuditActionTaskFailed
				logLevel = "ERROR"
				logMessage = fmt.Sprintf("Task '%s' failed", task.Name)
			case models.TaskStatusCanceled:
				auditAction = AuditActionTaskCanceled
				logLevel = "WARN"
				logMessage = fmt.Sprintf("Task '%s' was canceled", task.Name)
			}

			if auditAction != "" {
				details := map[string]interface{}{
					"old_status":      oldStatus,
					"new_status":      newStatus,
					"completed_hosts": completedCount,
					"failed_hosts":    failedCount,
					"running_hosts":   runningCount,
					"pending_hosts":   pendingCount,
					"canceled_hosts":  canceledCount,
					"total_hosts":     task.TotalHosts,
					"success_rate":    successRate,
				}

				if err := ts.auditService.LogTaskAction(auditAction, taskID, task.CreatedBy, details); err != nil {
					log.Printf("Failed to log task status change audit: %v", err)
				}

				if err := ts.auditService.LogTaskExecution(taskID, logLevel, logMessage, details, "", ""); err != nil {
					log.Printf("Failed to log task execution: %v", err)
				}
			}
		}()
	}

	log.Printf("Task progress updated: task_id=%s, completed=%d, failed=%d, running=%d, pending=%d, canceled=%d, total=%d, success_rate=%.2f%%",
		taskID, completedCount, failedCount, runningCount, pendingCount, canceledCount, task.TotalHosts, successRate)
	return nil
}

// HandleHostConnectionChange 处理主机连接状态变化
func (ts *TaskService) HandleHostConnectionChange(hostID string, connected bool) error {
	if !connected {
		// 主机断开连接，标记相关的运行中命令为失败
		updates := map[string]interface{}{
			"status":        string(models.CommandHostStatusFailed),
			"error_message": "Host connection lost",
			"updated_at":    time.Now(),
		}

		err := ts.db.Model(&models.CommandHost{}).
			Where("host_id = ? AND status = ?", hostID, string(models.CommandHostStatusRunning)).
			Updates(updates).Error
		if err != nil {
			return fmt.Errorf("failed to update disconnected host commands: %w", err)
		}

		// 同时更新 Command 记录
		cmdUpdates := map[string]interface{}{
			"status":     models.CommandStatusFailed,
			"error_msg":  "Host connection lost",
			"updated_at": time.Now(),
		}

		err = ts.db.Model(&models.Command{}).
			Where("host_id = ? AND status = ?", hostID, models.CommandStatusRunning).
			Updates(cmdUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to update disconnected host commands: %w", err)
		}

		// 记录主机断开连接的审计日志
		go func() {
			details := map[string]interface{}{
				"connection_status": "disconnected",
				"reason":            "Host connection lost",
			}
			if err := ts.auditService.LogHostAction(AuditActionHostDisconnect, hostID, details); err != nil {
				log.Printf("Failed to log host disconnection audit: %v", err)
			}
		}()

		log.Printf("Marked running commands as failed for disconnected host: %s", hostID)
	} else {
		// 记录主机连接的审计日志
		go func() {
			details := map[string]interface{}{
				"connection_status": "connected",
			}
			if err := ts.auditService.LogHostAction(AuditActionHostConnected, hostID, details); err != nil {
				log.Printf("Failed to log host connection audit: %v", err)
			}
		}()
	}

	return nil
}

// GetPendingCommands 获取待执行的命令列表
func (ts *TaskService) GetPendingCommands(hostID string) ([]models.Command, error) {
	var commands []models.Command

	err := ts.db.Where("host_id = ? AND status = ?", hostID, models.CommandStatusPending).
		Order("created_at ASC").
		Find(&commands).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get pending commands: %w", err)
	}

	return commands, nil
}

// UpdateCommandStatus 更新命令状态
func (ts *TaskService) UpdateCommandStatus(commandID string, status models.CommandStatus) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	// 如果是开始执行，设置开始时间
	if status == models.CommandStatusRunning {
		updates["started_at"] = time.Now()
	}

	err := ts.db.Model(&models.Command{}).Where("command_id = ?", commandID).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("failed to update command status: %w", err)
	}

	// 同时更新 CommandHost 状态
	hostStatus := string(models.CommandHostStatusPending)
	switch status {
	case models.CommandStatusRunning:
		hostStatus = string(models.CommandHostStatusRunning)
	case models.CommandStatusCompleted:
		hostStatus = string(models.CommandHostStatusCompleted)
	case models.CommandStatusFailed:
		hostStatus = string(models.CommandHostStatusExecFailed)
	case models.CommandStatusTimeout:
		hostStatus = string(models.CommandHostStatusTimeout)
	case models.CommandStatusCanceled:
		hostStatus = string(models.CommandHostStatusCanceled)
	}

	hostUpdates := map[string]interface{}{
		"status":     hostStatus,
		"updated_at": time.Now(),
	}

	if status == models.CommandStatusRunning {
		hostUpdates["started_at"] = time.Now()
	}

	err = ts.db.Model(&models.CommandHost{}).Where("command_id = ?", commandID).Updates(hostUpdates).Error
	if err != nil {
		return fmt.Errorf("failed to update command host status: %w", err)
	}

	return nil
}

// GetRunningTasks 获取正在运行的任务列表
func (ts *TaskService) GetRunningTasks() ([]models.Task, error) {
	var tasks []models.Task

	err := ts.db.Where("status = ?", models.TaskStatusRunning).Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get running tasks: %w", err)
	}

	return tasks, nil
}

// GetTaskStatistics 获取任务统计信息
func (ts *TaskService) GetTaskStatistics() (map[string]interface{}, error) {
	// 尝试从缓存获取
	if cachedStats, err := ts.cacheService.GetCachedTaskStatistics(); err == nil && cachedStats != nil {
		log.Printf("Task statistics cache hit")
		return cachedStats, nil
	}

	stats := make(map[string]interface{})

	// 统计各状态的任务数量
	var statusCounts []struct {
		Status string
		Count  int64
	}

	err := ts.db.Model(&models.Task{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get task status counts: %w", err)
	}

	statusMap := make(map[string]int64)
	for _, sc := range statusCounts {
		statusMap[sc.Status] = sc.Count
	}

	stats["task_status_counts"] = statusMap

	// 统计总任务数
	var totalTasks int64
	err = ts.db.Model(&models.Task{}).Count(&totalTasks).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count total tasks: %w", err)
	}
	stats["total_tasks"] = totalTasks

	// 统计今日创建的任务数
	today := time.Now().Truncate(24 * time.Hour)
	var todayTasks int64
	err = ts.db.Model(&models.Task{}).Where("created_at >= ?", today).Count(&todayTasks).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count today tasks: %w", err)
	}
	stats["today_tasks"] = todayTasks

	// 统计本周创建的任务数
	weekStart := time.Now().AddDate(0, 0, -int(time.Now().Weekday()))
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
	var weekTasks int64
	err = ts.db.Model(&models.Task{}).Where("created_at >= ?", weekStart).Count(&weekTasks).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count week tasks: %w", err)
	}
	stats["week_tasks"] = weekTasks

	// 统计本月创建的任务数
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Now().Location())
	var monthTasks int64
	err = ts.db.Model(&models.Task{}).Where("created_at >= ?", monthStart).Count(&monthTasks).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count month tasks: %w", err)
	}
	stats["month_tasks"] = monthTasks

	// 统计命令执行统计信息
	commandStats, err := ts.getCommandExecutionStatistics()
	if err != nil {
		log.Printf("Failed to get command execution statistics: %v", err)
	} else {
		stats["command_statistics"] = commandStats
	}

	// 统计主机执行统计信息
	hostStats, err := ts.getHostExecutionStatistics()
	if err != nil {
		log.Printf("Failed to get host execution statistics: %v", err)
	} else {
		stats["host_statistics"] = hostStats
	}

	// 异步缓存结果
	go func() {
		if err := ts.cacheService.CacheTaskStatistics(stats); err != nil {
			log.Printf("Failed to cache task statistics: %v", err)
		}
	}()

	return stats, nil
}

// GetTasksByHost 按主机筛选任务
func (ts *TaskService) GetTasksByHost(hostID string, page, size int, status string) ([]*models.Task, int, error) {
	// 生成缓存键
	cacheKey := ts.cacheService.GenerateHostTasksCacheKey(page, size, status)

	// 尝试从缓存获取
	if cachedTasks, cachedTotal, err := ts.cacheService.GetCachedHostTasks(hostID, cacheKey); err == nil && cachedTasks != nil {
		log.Printf("Host tasks cache hit: %s, key: %s", hostID, cacheKey)
		return cachedTasks, cachedTotal, nil
	}

	var tasks []models.Task
	var total int64

	// 构建查询条件
	query := ts.db.Model(&models.Task{}).
		Where("task_id IN (SELECT DISTINCT task_id FROM commands WHERE host_id = ?)", hostID)

	// 状态过滤
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks by host: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get tasks by host: %w", err)
	}

	// 转换为指针切片
	result := make([]*models.Task, len(tasks))
	for i := range tasks {
		result[i] = &tasks[i]
	}

	// 异步缓存结果
	go func() {
		if err := ts.cacheService.CacheHostTasks(hostID, cacheKey, result, int(total)); err != nil {
			log.Printf("Failed to cache host tasks: %v", err)
		}
	}()

	return result, int(total), nil
}

// GetTasksByStatus 按状态筛选任务
func (ts *TaskService) GetTasksByStatus(status string, page, size int) ([]*models.Task, int, error) {
	var tasks []models.Task
	var total int64

	// 构建查询条件
	query := ts.db.Model(&models.Task{}).Where("status = ?", status)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks by status: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get tasks by status: %w", err)
	}

	// 转换为指针切片
	result := make([]*models.Task, len(tasks))
	for i := range tasks {
		result[i] = &tasks[i]
	}

	return result, int(total), nil
}

// GetTasksByDateRange 按日期范围筛选任务
func (ts *TaskService) GetTasksByDateRange(startDate, endDate time.Time, page, size int, status string) ([]*models.Task, int, error) {
	var tasks []models.Task
	var total int64

	// 构建查询条件
	query := ts.db.Model(&models.Task{}).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate)

	// 状态过滤
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks by date range: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get tasks by date range: %w", err)
	}

	// 转换为指针切片
	result := make([]*models.Task, len(tasks))
	for i := range tasks {
		result[i] = &tasks[i]
	}

	return result, int(total), nil
}

// getHostExecutionStatistics 获取主机执行统计信息
func (ts *TaskService) getHostExecutionStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 统计各主机的执行次数
	var hostCounts []struct {
		HostID string
		Count  int64
	}

	err := ts.db.Model(&models.CommandHost{}).
		Select("host_id, COUNT(*) as count").
		Group("host_id").
		Order("count DESC").
		Limit(10). // 只取前10个最活跃的主机
		Scan(&hostCounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get host execution counts: %w", err)
	}

	stats["top_hosts"] = hostCounts

	// 统计各主机的成功率
	var hostSuccessRates []struct {
		HostID       string
		TotalCount   int64
		SuccessCount int64
		SuccessRate  float64
	}

	err = ts.db.Raw(`
		SELECT 
			host_id,
			COUNT(*) as total_count,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as success_count,
			ROUND(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 2) as success_rate
		FROM commands_hosts 
		WHERE status IN (?, ?, ?, ?)
		GROUP BY host_id 
		HAVING COUNT(*) >= 5
		ORDER BY success_rate DESC
		LIMIT 10
	`, string(models.CommandHostStatusCompleted), string(models.CommandHostStatusCompleted),
		string(models.CommandHostStatusCompleted), string(models.CommandHostStatusExecFailed),
		string(models.CommandHostStatusTimeout), string(models.CommandHostStatusFailed)).
		Scan(&hostSuccessRates).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get host success rates: %w", err)
	}

	stats["host_success_rates"] = hostSuccessRates

	return stats, nil
}

// getCommandExecutionStatistics 获取命令执行统计信息
func (ts *TaskService) getCommandExecutionStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 统计命令执行状态
	var cmdStatusCounts []struct {
		Status string
		Count  int64
	}

	err := ts.db.Model(&models.CommandHost{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&cmdStatusCounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get command status counts: %w", err)
	}

	cmdStatusMap := make(map[string]int64)
	for _, sc := range cmdStatusCounts {
		cmdStatusMap[sc.Status] = sc.Count
	}
	stats["status_counts"] = cmdStatusMap

	// 计算平均执行时长
	var avgExecutionTime struct {
		AvgTime float64
	}

	err = ts.db.Model(&models.CommandHost{}).
		Select("AVG(execution_time) as avg_time").
		Where("execution_time IS NOT NULL AND execution_time > 0").
		Scan(&avgExecutionTime).Error
	if err != nil {
		return nil, fmt.Errorf("failed to calculate average execution time: %w", err)
	}
	stats["average_execution_time_ms"] = avgExecutionTime.AvgTime

	// 统计成功率
	var successRate struct {
		TotalCommands      int64
		SuccessfulCommands int64
	}

	err = ts.db.Model(&models.CommandHost{}).
		Select("COUNT(*) as total_commands, SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as successful_commands",
			string(models.CommandHostStatusCompleted)).
		Where("status IN (?)", []string{
			string(models.CommandHostStatusCompleted),
			string(models.CommandHostStatusExecFailed),
			string(models.CommandHostStatusTimeout),
		}).
		Scan(&successRate).Error
	if err != nil {
		return nil, fmt.Errorf("failed to calculate success rate: %w", err)
	}

	if successRate.TotalCommands > 0 {
		stats["success_rate"] = float64(successRate.SuccessfulCommands) / float64(successRate.TotalCommands) * 100
	} else {
		stats["success_rate"] = 0.0
	}

	stats["total_executed_commands"] = successRate.TotalCommands
	stats["successful_commands"] = successRate.SuccessfulCommands

	return stats, nil
}

// GetTaskExecutionSummary 获取任务执行摘要
func (ts *TaskService) GetTaskExecutionSummary(taskID string) (map[string]interface{}, error) {
	// 尝试从缓存获取
	if cachedExecution, err := ts.cacheService.GetCachedTaskExecution(taskID); err == nil && cachedExecution != nil {
		log.Printf("Task execution cache hit: %s", taskID)
		return cachedExecution, nil
	}

	// 获取任务信息
	task, err := ts.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	summary := make(map[string]interface{})
	summary["task_id"] = task.TaskID
	summary["task_name"] = task.Name
	summary["status"] = task.Status
	summary["total_hosts"] = task.TotalHosts
	summary["completed_hosts"] = task.CompletedHosts
	summary["failed_hosts"] = task.FailedHosts
	summary["success_rate"] = task.SuccessRate()

	// 计算执行时长
	if task.StartedAt != nil {
		if task.FinishedAt != nil {
			summary["duration_seconds"] = task.Duration().Seconds()
		} else {
			summary["duration_seconds"] = time.Since(*task.StartedAt).Seconds()
		}
	}

	// 统计各主机的执行详情
	var hostDetails []map[string]interface{}
	for _, cmd := range task.Commands {
		for _, cmdHost := range cmd.CommandHosts {
			detail := map[string]interface{}{
				"host_id":        cmdHost.HostID,
				"command_id":     cmdHost.CommandID,
				"status":         cmdHost.Status,
				"exit_code":      cmdHost.ExitCode,
				"started_at":     cmdHost.StartedAt,
				"finished_at":    cmdHost.FinishedAt,
				"execution_time": cmdHost.ExecutionTime,
				"error_message":  cmdHost.ErrorMessage,
			}

			// 计算执行时长（如果有）
			if cmdHost.StartedAt != nil && cmdHost.FinishedAt != nil {
				detail["duration_seconds"] = cmdHost.FinishedAt.Sub(*cmdHost.StartedAt).Seconds()
			}

			hostDetails = append(hostDetails, detail)
		}
	}
	summary["host_details"] = hostDetails

	// 统计错误信息
	var errorSummary []map[string]interface{}
	errorCounts := make(map[string]int)
	for _, cmd := range task.Commands {
		for _, cmdHost := range cmd.CommandHosts {
			if cmdHost.ErrorMessage != "" {
				errorCounts[cmdHost.ErrorMessage]++
			}
		}
	}

	for errorMsg, count := range errorCounts {
		errorSummary = append(errorSummary, map[string]interface{}{
			"error_message": errorMsg,
			"count":         count,
		})
	}
	summary["error_summary"] = errorSummary

	// 异步缓存结果
	go func() {
		if err := ts.cacheService.CacheTaskExecution(taskID, summary); err != nil {
			log.Printf("Failed to cache task execution: %v", err)
		}
	}()

	return summary, nil
}

// sendCancelCommandToAgent 向 Agent 发送取消命令
func (ts *TaskService) sendCancelCommandToAgent(command models.Command) error {
	if taskDispatcher == nil {
		return fmt.Errorf("task dispatcher not available")
	}

	// 创建取消命令
	cancelCommand := &models.Command{
		CommandID:  "cancel-" + command.CommandID,
		HostID:     command.HostID,
		Command:    "cancel",
		Parameters: command.CommandID, // 传递要取消的命令ID
		Timeout:    30,                // 取消命令的超时时间
		Status:     models.CommandStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 发送取消命令到 Agent
	return taskDispatcher.SendCommandToAgent(command.HostID, cancelCommand)
}

// HandleAgentDisconnection 处理 Agent 断开连接
func (ts *TaskService) HandleAgentDisconnection(hostID string) error {
	return ts.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 标记该主机上所有运行中的命令为失败
		cmdUpdates := map[string]interface{}{
			"status":      models.CommandStatusFailed,
			"finished_at": now,
			"error_msg":   "Agent disconnected",
			"updated_at":  now,
		}

		err := tx.Model(&models.Command{}).
			Where("host_id = ? AND status IN (?)", hostID, []models.CommandStatus{
				models.CommandStatusPending,
				models.CommandStatusRunning,
			}).
			Updates(cmdUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to update commands for disconnected agent: %w", err)
		}

		// 更新 CommandHost 状态
		hostUpdates := map[string]interface{}{
			"status":        string(models.CommandHostStatusFailed),
			"finished_at":   now,
			"error_message": "Agent disconnected",
			"updated_at":    now,
		}

		err = tx.Model(&models.CommandHost{}).
			Where("host_id = ? AND status IN (?)", hostID, []string{
				string(models.CommandHostStatusPending),
				string(models.CommandHostStatusRunning),
			}).
			Updates(hostUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to update command hosts for disconnected agent: %w", err)
		}

		// 获取受影响的任务并更新进度
		var affectedTaskIDs []string
		err = tx.Model(&models.Command{}).
			Select("DISTINCT task_id").
			Where("host_id = ? AND status = ?", hostID, models.CommandStatusFailed).
			Pluck("task_id", &affectedTaskIDs).Error
		if err != nil {
			return fmt.Errorf("failed to get affected tasks: %w", err)
		}

		// 更新所有受影响任务的进度
		for _, taskID := range affectedTaskIDs {
			if taskID != "" {
				err = ts.updateTaskProgressInTransaction(tx, taskID)
				if err != nil {
					log.Printf("Failed to update task progress for task %s: %v", taskID, err)
				}
			}
		}

		log.Printf("Handled agent disconnection for host %s, affected %d tasks", hostID, len(affectedTaskIDs))
		return nil
	})
}

// HandleCommandExecutionError 处理命令执行错误
func (ts *TaskService) HandleCommandExecutionError(commandID, hostID, errorMessage string) error {
	return ts.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 更新命令状态
		cmdUpdates := map[string]interface{}{
			"status":      models.CommandStatusFailed,
			"finished_at": now,
			"error_msg":   errorMessage,
			"updated_at":  now,
		}

		err := tx.Model(&models.Command{}).Where("command_id = ?", commandID).Updates(cmdUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to update command error: %w", err)
		}

		// 更新 CommandHost 状态
		hostUpdates := map[string]interface{}{
			"status":        string(models.CommandHostStatusExecFailed),
			"finished_at":   now,
			"error_message": errorMessage,
			"updated_at":    now,
		}

		err = tx.Model(&models.CommandHost{}).Where("command_id = ? AND host_id = ?", commandID, hostID).Updates(hostUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to update command host error: %w", err)
		}

		// 获取命令所属的任务并更新进度
		var command models.Command
		err = tx.Where("command_id = ?", commandID).First(&command).Error
		if err != nil {
			return fmt.Errorf("failed to get command: %w", err)
		}

		if command.TaskID != nil {
			err = ts.updateTaskProgressInTransaction(tx, *command.TaskID)
			if err != nil {
				return fmt.Errorf("failed to update task progress: %w", err)
			}
		}

		log.Printf("Handled command execution error: command_id=%s, host_id=%s, error=%s", commandID, hostID, errorMessage)
		return nil
	})
}

// RetryFailedCommand 重试失败的命令
func (ts *TaskService) RetryFailedCommand(commandID string) error {
	return ts.db.Transaction(func(tx *gorm.DB) error {
		// 获取失败的命令
		var command models.Command
		err := tx.Where("command_id = ? AND status IN (?)", commandID, []models.CommandStatus{
			models.CommandStatusFailed,
			models.CommandStatusTimeout,
		}).First(&command).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("command not found or not in failed status: %s", commandID)
			}
			return fmt.Errorf("failed to get command: %w", err)
		}

		// 重置命令状态
		now := time.Now()
		cmdUpdates := map[string]interface{}{
			"status":      models.CommandStatusPending,
			"started_at":  nil,
			"finished_at": nil,
			"error_msg":   "",
			"stdout":      "",
			"stderr":      "",
			"exit_code":   nil,
			"updated_at":  now,
		}

		err = tx.Model(&models.Command{}).Where("command_id = ?", commandID).Updates(cmdUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to reset command status: %w", err)
		}

		// 重置 CommandHost 状态
		hostUpdates := map[string]interface{}{
			"status":         string(models.CommandHostStatusPending),
			"started_at":     nil,
			"finished_at":    nil,
			"error_message":  "",
			"stdout":         "",
			"stderr":         "",
			"exit_code":      0,
			"execution_time": nil,
			"updated_at":     now,
		}

		err = tx.Model(&models.CommandHost{}).Where("command_id = ?", commandID).Updates(hostUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to reset command host status: %w", err)
		}

		// 重新发送命令到 Agent
		if taskDispatcher != nil {
			// 重新加载命令信息
			err = tx.Where("command_id = ?", commandID).First(&command).Error
			if err != nil {
				return fmt.Errorf("failed to reload command: %w", err)
			}

			// 异步发送命令
			go func() {
				err := taskDispatcher.SendCommandToAgent(command.HostID, &command)
				if err != nil {
					log.Printf("Failed to resend command %s to agent %s: %v", commandID, command.HostID, err)
					// 标记命令为下发失败
					ts.updateCommandDispatchFailed(commandID, err.Error())
				} else {
					log.Printf("Command %s resent to agent %s successfully", commandID, command.HostID)
				}
			}()
		}

		// 更新任务进度
		if command.TaskID != nil {
			err = ts.updateTaskProgressInTransaction(tx, *command.TaskID)
			if err != nil {
				return fmt.Errorf("failed to update task progress: %w", err)
			}
		}

		log.Printf("Command %s retry initiated for host %s", commandID, command.HostID)
		return nil
	})
}

// GetFailedCommands 获取失败的命令列表
func (ts *TaskService) GetFailedCommands(page, size int, hostID string) ([]models.Command, int, error) {
	var commands []models.Command
	var total int64

	// 构建查询条件
	query := ts.db.Model(&models.Command{}).Where("status IN (?)", []models.CommandStatus{
		models.CommandStatusFailed,
		models.CommandStatusTimeout,
	})

	// 主机过滤
	if hostID != "" {
		query = query.Where("host_id = ?", hostID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count failed commands: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("updated_at DESC").Find(&commands).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get failed commands: %w", err)
	}

	return commands, int(total), nil
}

// GetTimeoutStatistics 获取超时统计信息
func (ts *TaskService) GetTimeoutStatistics() (map[string]interface{}, error) {
	if ts.timeoutMonitor == nil {
		return nil, fmt.Errorf("timeout monitor not initialized")
	}
	return ts.timeoutMonitor.GetTimeoutStatistics()
}

// CheckCommandTimeout 手动检查命令超时
func (ts *TaskService) CheckCommandTimeout(commandID string) error {
	if ts.timeoutMonitor == nil {
		return fmt.Errorf("timeout monitor not initialized")
	}
	return ts.timeoutMonitor.CheckCommandTimeout(commandID)
}

// GetErrorStatistics 获取错误统计信息
func (ts *TaskService) GetErrorStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 统计各种错误类型的数量
	var errorCounts []struct {
		Status string
		Count  int64
	}

	err := ts.db.Model(&models.CommandHost{}).
		Select("status, COUNT(*) as count").
		Where("status IN (?)", []string{
			string(models.CommandHostStatusFailed),
			string(models.CommandHostStatusExecFailed),
			string(models.CommandHostStatusTimeout),
		}).
		Group("status").
		Scan(&errorCounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get error counts: %w", err)
	}

	errorMap := make(map[string]int64)
	for _, ec := range errorCounts {
		errorMap[ec.Status] = ec.Count
	}
	stats["error_type_counts"] = errorMap

	// 统计最常见的错误信息
	var commonErrors []struct {
		ErrorMessage string
		Count        int64
	}

	err = ts.db.Model(&models.CommandHost{}).
		Select("error_message, COUNT(*) as count").
		Where("error_message != '' AND error_message IS NOT NULL").
		Group("error_message").
		Order("count DESC").
		Limit(10).
		Scan(&commonErrors).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get common errors: %w", err)
	}

	stats["common_errors"] = commonErrors

	// 统计各主机的错误率
	var hostErrorRates []struct {
		HostID    string
		Total     int64
		Errors    int64
		ErrorRate float64
	}

	err = ts.db.Raw(`
		SELECT 
			host_id,
			COUNT(*) as total,
			SUM(CASE WHEN status IN (?, ?, ?) THEN 1 ELSE 0 END) as errors,
			ROUND(SUM(CASE WHEN status IN (?, ?, ?) THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 2) as error_rate
		FROM commands_hosts 
		GROUP BY host_id 
		HAVING COUNT(*) >= 5
		ORDER BY error_rate DESC
		LIMIT 10
	`, string(models.CommandHostStatusFailed), string(models.CommandHostStatusExecFailed), string(models.CommandHostStatusTimeout),
		string(models.CommandHostStatusFailed), string(models.CommandHostStatusExecFailed), string(models.CommandHostStatusTimeout)).
		Scan(&hostErrorRates).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get host error rates: %w", err)
	}

	stats["host_error_rates"] = hostErrorRates

	return stats, nil
}

// BatchUpdate 批量更新结构
type BatchUpdate struct {
	Type string
	Data interface{}
}

// BatchCommandHostUpdate 批量命令主机更新
type BatchCommandHostUpdate struct {
	CommandID     string
	HostID        string
	Status        string
	FinishedAt    *time.Time
	ErrorMessage  string
	ExitCode      *int
	ExecutionTime *int64
}

// BatchCommandUpdate 批量命令更新
type BatchCommandUpdate struct {
	CommandID  string
	Status     models.CommandStatus
	FinishedAt *time.Time
	ErrorMsg   string
	ExitCode   *int32
}

// startBatchUpdateProcessor 启动批量更新处理器
func (ts *TaskService) startBatchUpdateProcessor() {
	commandHostUpdates := make([]CommandHostStatusUpdate, 0, ts.batchSize)
	commandUpdates := make([]CommandStatusUpdate, 0, ts.batchSize)

	ticker := time.NewTicker(ts.batchTimeout)
	defer ticker.Stop()

	for {
		select {
		case update := <-ts.batchUpdateQueue:
			switch update.Type {
			case "command_host":
				if data, ok := update.Data.(BatchCommandHostUpdate); ok {
					commandHostUpdates = append(commandHostUpdates, CommandHostStatusUpdate{
						CommandID:     data.CommandID,
						Status:        data.Status,
						FinishedAt:    data.FinishedAt,
						ErrorMessage:  data.ErrorMessage,
						ExitCode:      data.ExitCode,
						ExecutionTime: data.ExecutionTime,
					})
				}
			case "command":
				if data, ok := update.Data.(BatchCommandUpdate); ok {
					commandUpdates = append(commandUpdates, CommandStatusUpdate{
						CommandID:  data.CommandID,
						Status:     data.Status,
						FinishedAt: data.FinishedAt,
						ErrorMsg:   data.ErrorMsg,
						ExitCode:   data.ExitCode,
					})
				}
			}

			// 检查是否达到批量大小
			if len(commandHostUpdates) >= ts.batchSize {
				ts.processBatchCommandHostUpdates(commandHostUpdates)
				commandHostUpdates = commandHostUpdates[:0]
			}
			if len(commandUpdates) >= ts.batchSize {
				ts.processBatchCommandUpdates(commandUpdates)
				commandUpdates = commandUpdates[:0]
			}

		case <-ticker.C:
			// 定时处理剩余的更新
			if len(commandHostUpdates) > 0 {
				ts.processBatchCommandHostUpdates(commandHostUpdates)
				commandHostUpdates = commandHostUpdates[:0]
			}
			if len(commandUpdates) > 0 {
				ts.processBatchCommandUpdates(commandUpdates)
				commandUpdates = commandUpdates[:0]
			}
		}
	}
}

// processBatchCommandHostUpdates 处理批量命令主机更新
func (ts *TaskService) processBatchCommandHostUpdates(updates []CommandHostStatusUpdate) {
	if len(updates) == 0 {
		return
	}

	err := ts.dbOptimizer.BatchUpdateCommandHostStatus(updates)
	if err != nil {
		log.Printf("Failed to process batch command host updates: %v", err)
	}
}

// processBatchCommandUpdates 处理批量命令更新
func (ts *TaskService) processBatchCommandUpdates(updates []CommandStatusUpdate) {
	if len(updates) == 0 {
		return
	}

	err := ts.dbOptimizer.BatchUpdateCommandStatus(updates)
	if err != nil {
		log.Printf("Failed to process batch command updates: %v", err)
	}
}

// QueueBatchUpdate 队列批量更新
func (ts *TaskService) QueueBatchUpdate(updateType string, data interface{}) {
	select {
	case ts.batchUpdateQueue <- BatchUpdate{Type: updateType, Data: data}:
		// 成功加入队列
	default:
		// 队列满了，直接处理
		log.Printf("Batch update queue is full, processing immediately")
		switch updateType {
		case "command_host":
			if update, ok := data.(BatchCommandHostUpdate); ok {
				ts.processBatchCommandHostUpdates([]CommandHostStatusUpdate{{
					CommandID:     update.CommandID,
					Status:        update.Status,
					FinishedAt:    update.FinishedAt,
					ErrorMessage:  update.ErrorMessage,
					ExitCode:      update.ExitCode,
					ExecutionTime: update.ExecutionTime,
				}})
			}
		case "command":
			if update, ok := data.(BatchCommandUpdate); ok {
				ts.processBatchCommandUpdates([]CommandStatusUpdate{{
					CommandID:  update.CommandID,
					Status:     update.Status,
					FinishedAt: update.FinishedAt,
					ErrorMsg:   update.ErrorMsg,
					ExitCode:   update.ExitCode,
				}})
			}
		}
	}
}

// OptimizedHandleCommandResult 优化的命令结果处理（使用批量更新）
func (ts *TaskService) OptimizedHandleCommandResult(result *models.CommandResult) error {
	// 计算执行时长
	if result.StartedAt != nil && result.FinishedAt != nil {
		duration := result.FinishedAt.Sub(*result.StartedAt)
		executionTime := duration.Milliseconds()
		result.ExecutionTime = &executionTime
	}

	// 确定状态
	var status string
	var commandStatus models.CommandStatus
	if result.FinishedAt != nil {
		if result.ExitCode == 0 {
			status = string(models.CommandHostStatusCompleted)
			commandStatus = models.CommandStatusCompleted
		} else {
			status = string(models.CommandHostStatusExecFailed)
			commandStatus = models.CommandStatusFailed
		}
	} else if result.StartedAt != nil {
		status = string(models.CommandHostStatusRunning)
		commandStatus = models.CommandStatusRunning
	}

	// 队列批量更新 CommandHost
	exitCode := int(result.ExitCode)
	ts.QueueBatchUpdate("command_host", BatchCommandHostUpdate{
		CommandID:     result.CommandID,
		HostID:        result.HostID,
		Status:        status,
		FinishedAt:    result.FinishedAt,
		ErrorMessage:  result.ErrorMessage,
		ExitCode:      &exitCode,
		ExecutionTime: result.ExecutionTime,
	})

	// 队列批量更新 Command
	exitCode32 := result.ExitCode
	ts.QueueBatchUpdate("command", BatchCommandUpdate{
		CommandID:  result.CommandID,
		Status:     commandStatus,
		FinishedAt: result.FinishedAt,
		ErrorMsg:   result.ErrorMessage,
		ExitCode:   &exitCode32,
	})

	// 保存命令结果记录（避免重复插入）
	err := ts.db.Transaction(func(tx *gorm.DB) error {
		var existingResult models.CommandResult
		err := tx.Where("command_id = ? AND host_id = ?", result.CommandID, result.HostID).First(&existingResult).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// 记录不存在，创建新记录
				result.CreatedAt = time.Now()
				result.UpdatedAt = time.Now()
				err = tx.Create(result).Error
				if err != nil {
					return fmt.Errorf("failed to create command result: %w", err)
				}
			} else {
				return fmt.Errorf("failed to check existing command result: %w", err)
			}
		} else {
			// 记录已存在，更新现有记录
			err = tx.Model(&existingResult).Updates(map[string]interface{}{
				"stdout":         result.Stdout,
				"stderr":         result.Stderr,
				"exit_code":      result.ExitCode,
				"started_at":     result.StartedAt,
				"finished_at":    result.FinishedAt,
				"error_message":  result.ErrorMessage,
				"execution_time": result.ExecutionTime,
				"updated_at":     time.Now(),
			}).Error
			if err != nil {
				return fmt.Errorf("failed to update command result: %w", err)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	// 异步更新任务进度（避免阻塞）
	go func() {
		var command models.Command
		err := ts.db.Where("command_id = ?", result.CommandID).First(&command).Error
		if err == nil && command.TaskID != nil {
			err = ts.db.Transaction(func(tx *gorm.DB) error {
				return ts.updateTaskProgressInTransaction(tx, *command.TaskID)
			})
			if err != nil {
				log.Printf("Failed to update task progress: %v", err)
			}
		}
	}()

	log.Printf("Optimized command result processed: command_id=%s, host_id=%s, exit_code=%d",
		result.CommandID, result.HostID, result.ExitCode)
	return nil
}

// CleanupOldRecords 清理旧记录
func (ts *TaskService) CleanupOldRecords(retentionDays int) error {
	return ts.dbOptimizer.CleanupOldRecords(retentionDays)
}

// AnalyzeTableSizes 分析表大小
func (ts *TaskService) AnalyzeTableSizes() (map[string]interface{}, error) {
	return ts.dbOptimizer.AnalyzeTableSizes()
}

// OptimizeTables 优化表结构
func (ts *TaskService) OptimizeTables() error {
	return ts.dbOptimizer.OptimizeTables()
}

// GetDatabaseStatistics 获取数据库统计信息
func (ts *TaskService) GetDatabaseStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 获取表大小分析
	tableSizes, err := ts.AnalyzeTableSizes()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze table sizes: %w", err)
	}
	stats["table_sizes"] = tableSizes

	// 获取索引使用情况（MySQL特定）
	var indexUsage []struct {
		TableName   string `gorm:"column:TABLE_NAME"`
		IndexName   string `gorm:"column:INDEX_NAME"`
		Cardinality int64  `gorm:"column:CARDINALITY"`
	}

	err = ts.db.Raw(`
		SELECT TABLE_NAME, INDEX_NAME, CARDINALITY 
		FROM information_schema.STATISTICS 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND TABLE_NAME IN ('tasks', 'commands', 'commands_hosts', 'command_results')
		ORDER BY TABLE_NAME, CARDINALITY DESC
	`).Scan(&indexUsage).Error

	if err == nil {
		stats["index_usage"] = indexUsage
	}

	// 获取连接池状态
	sqlDB, err := ts.db.DB()
	if err == nil {
		dbStats := sqlDB.Stats()
		stats["connection_pool"] = map[string]interface{}{
			"open_connections":     dbStats.OpenConnections,
			"in_use":               dbStats.InUse,
			"idle":                 dbStats.Idle,
			"wait_count":           dbStats.WaitCount,
			"wait_duration":        dbStats.WaitDuration.Seconds(),
			"max_idle_closed":      dbStats.MaxIdleClosed,
			"max_idle_time_closed": dbStats.MaxIdleTimeClosed,
			"max_lifetime_closed":  dbStats.MaxLifetimeClosed,
		}
	}

	return stats, nil
}

// GetCacheStatistics 获取缓存统计信息
func (ts *TaskService) GetCacheStatistics() (map[string]interface{}, error) {
	if ts.cacheService == nil {
		return nil, fmt.Errorf("cache service not initialized")
	}
	return ts.cacheService.GetCacheStatistics()
}

// InvalidateAllCache 使所有缓存失效
func (ts *TaskService) InvalidateAllCache() error {
	if ts.cacheService == nil {
		return fmt.Errorf("cache service not initialized")
	}
	return ts.cacheService.InvalidateAllTaskCache()
}

// WarmupCache 预热缓存
func (ts *TaskService) WarmupCache() error {
	if ts.cacheService == nil {
		return fmt.Errorf("cache service not initialized")
	}
	return ts.cacheService.WarmupCache()
}

// CleanupCache 清理过期缓存
func (ts *TaskService) CleanupCache() error {
	if ts.cacheService == nil {
		return fmt.Errorf("cache service not initialized")
	}
	return ts.cacheService.CleanupExpiredCache()
}

// startCacheCleanupTask 启动定期缓存清理任务
func (ts *TaskService) startCacheCleanupTask() {
	ticker := time.NewTicker(30 * time.Minute) // 每30分钟清理一次
	defer ticker.Stop()

	for range ticker.C {
		if err := ts.CleanupCache(); err != nil {
			log.Printf("Failed to cleanup cache: %v", err)
		}

		// 获取缓存统计信息并记录
		if stats, err := ts.GetCacheStatistics(); err == nil {
			if totalKeys, ok := stats["total_cache_keys"].(int); ok {
				log.Printf("Cache cleanup completed, total cache keys: %d", totalKeys)
			}
		}
	}
}

// startStatisticsUpdateTask 启动定期统计更新任务
func (ts *TaskService) startStatisticsUpdateTask() {
	// 每天凌晨1点更新统计信息
	ticker := time.NewTicker(1 * time.Hour) // 每小时检查一次
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		// 检查是否是凌晨1点
		if now.Hour() == 1 && now.Minute() < 5 {
			// 更新昨天的统计信息
			yesterday := now.AddDate(0, 0, -1)
			if err := ts.auditService.UpdateExecutionStatistics(yesterday, "daily"); err != nil {
				log.Printf("Failed to update daily statistics: %v", err)
			} else {
				log.Printf("Daily statistics updated for date: %s", yesterday.Format("2006-01-02"))
			}

			// 清理30天前的旧日志
			if err := ts.auditService.CleanupOldAuditLogs(30); err != nil {
				log.Printf("Failed to cleanup old audit logs: %v", err)
			} else {
				log.Printf("Old audit logs cleaned up (retention: 30 days)")
			}
		}
	}
}

// GetQueueStatus 获取队列状态
func (ts *TaskService) GetQueueStatus() map[string]interface{} {
	if ts.queueManager == nil {
		return map[string]interface{}{
			"error": "queue manager not initialized",
		}
	}
	return ts.queueManager.GetQueueStatus()
}

// GetSystemLoadStatus 获取系统负载状态
func (ts *TaskService) GetSystemLoadStatus() map[string]interface{} {
	if ts.loadMonitor == nil {
		return map[string]interface{}{
			"error": "load monitor not initialized",
		}
	}
	return ts.loadMonitor.GetSystemHealth()
}

// GetLoadStatistics 获取负载统计
func (ts *TaskService) GetLoadStatistics(duration time.Duration) map[string]interface{} {
	if ts.loadMonitor == nil {
		return map[string]interface{}{
			"error": "load monitor not initialized",
		}
	}
	return ts.loadMonitor.GetLoadStatistics(duration)
}

// UpdateHostLoad 更新主机负载信息
func (ts *TaskService) UpdateHostLoad(hostID string, cpuUsage, memoryUsage float64, available bool) {
	if ts.queueManager != nil {
		ts.queueManager.UpdateHostLoad(hostID, cpuUsage, memoryUsage, available)
	}
}

// CancelQueuedTask 取消队列中的任务
func (ts *TaskService) CancelQueuedTask(taskID string) error {
	if ts.queueManager == nil {
		return fmt.Errorf("queue manager not initialized")
	}
	return ts.queueManager.CancelTask(taskID)
}

// GetTaskQueuePosition 获取任务在队列中的位置
func (ts *TaskService) GetTaskQueuePosition(taskID string) (int, error) {
	if ts.queueManager == nil {
		return -1, fmt.Errorf("queue manager not initialized")
	}
	return ts.queueManager.GetTaskPosition(taskID)
}

// GetRecommendedConcurrency 获取推荐的并发数
func (ts *TaskService) GetRecommendedConcurrency(maxConcurrency int) int {
	if ts.loadMonitor == nil {
		return maxConcurrency
	}
	return ts.loadMonitor.GetRecommendedConcurrency(maxConcurrency)
}

// IsSystemOverloaded 检查系统是否过载
func (ts *TaskService) IsSystemOverloaded() bool {
	if ts.loadMonitor == nil {
		return false
	}
	return ts.loadMonitor.IsSystemOverloaded()
}

// GetQueueManagerConfig 获取队列管理器配置
func (ts *TaskService) GetQueueManagerConfig() map[string]interface{} {
	if ts.queueManager == nil {
		return map[string]interface{}{
			"error": "queue manager not initialized",
		}
	}

	status := ts.queueManager.GetQueueStatus()
	return map[string]interface{}{
		"max_concurrent_tasks":  status["max_concurrent_tasks"],
		"worker_count":          status["worker_count"],
		"load_balance_strategy": status["load_balance_strategy"],
		"adaptive_throttling":   status["adaptive_throttling"],
		"system_load":           status["system_load"],
	}
}

// UpdateQueueConfig 更新队列配置
func (ts *TaskService) UpdateQueueConfig(config map[string]interface{}) error {
	// 这里可以实现动态配置更新
	// 目前只是记录日志
	log.Printf("Queue config update requested: %+v", config)
	return nil
}

// GetPerformanceMetrics 获取性能指标
func (ts *TaskService) GetPerformanceMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// 队列指标
	if ts.queueManager != nil {
		queueStatus := ts.queueManager.GetQueueStatus()
		metrics["queue"] = queueStatus
	}

	// 系统负载指标
	if ts.loadMonitor != nil {
		loadStatus := ts.loadMonitor.GetSystemHealth()
		metrics["system_load"] = loadStatus

		// 内存统计
		memStats := ts.loadMonitor.GetMemoryStats()
		metrics["memory"] = memStats
	}

	// 数据库统计
	if ts.dbOptimizer != nil {
		dbStats, err := ts.GetDatabaseStatistics()
		if err == nil {
			metrics["database"] = dbStats
		}
	}

	// 缓存统计
	if ts.cacheService != nil {
		cacheStats, err := ts.GetCacheStatistics()
		if err == nil {
			metrics["cache"] = cacheStats
		}
	}

	return metrics
}

// OptimizePerformance 性能优化
func (ts *TaskService) OptimizePerformance() error {
	log.Println("Starting performance optimization...")

	// 强制垃圾回收
	if ts.loadMonitor != nil {
		ts.loadMonitor.ForceGC()
	}

	// 优化数据库表
	if ts.dbOptimizer != nil {
		err := ts.dbOptimizer.OptimizeTables()
		if err != nil {
			log.Printf("Failed to optimize database tables: %v", err)
		}
	}

	// 清理缓存
	if ts.cacheService != nil {
		err := ts.CleanupCache()
		if err != nil {
			log.Printf("Failed to cleanup cache: %v", err)
		}
	}

	log.Println("Performance optimization completed")
	return nil
}

// GetDetailedTaskLogs 获取详细的任务日志（包含完整输出）
func (ts *TaskService) GetDetailedTaskLogs(taskID, commandID, hostID string) (map[string]interface{}, error) {
	// 获取任务信息
	task, err := ts.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	response := map[string]interface{}{
		"task_id":   taskID,
		"task_name": task.Name,
	}

	// 如果指定了命令ID和主机ID，返回特定命令的详细日志
	if commandID != "" && hostID != "" {
		var cmdHost models.CommandHost
		err := ts.db.Where("command_id = ? AND host_id = ?", commandID, hostID).First(&cmdHost).Error
		if err != nil {
			return nil, fmt.Errorf("command host not found: %w", err)
		}

		var cmd models.Command
		err = ts.db.Where("command_id = ?", commandID).First(&cmd).Error
		if err != nil {
			return nil, fmt.Errorf("command not found: %w", err)
		}

		response["command_details"] = map[string]interface{}{
			"command_id":     commandID,
			"host_id":        hostID,
			"command":        cmd.Command,
			"parameters":     cmd.Parameters,
			"timeout":        cmd.Timeout,
			"status":         cmdHost.Status,
			"exit_code":      cmdHost.ExitCode,
			"stdout":         cmdHost.Stdout,
			"stderr":         cmdHost.Stderr,
			"error_message":  cmdHost.ErrorMessage,
			"execution_time": cmdHost.ExecutionTime,
			"started_at":     cmdHost.StartedAt,
			"finished_at":    cmdHost.FinishedAt,
			"created_at":     cmdHost.CreatedAt,
			"updated_at":     cmdHost.UpdatedAt,
		}

		// 获取该命令的执行历史
		history, err := ts.auditService.GetCommandExecutionHistory(commandID)
		if err != nil {
			log.Printf("Failed to get command execution history: %v", err)
		} else {
			response["execution_history"] = history
		}
	}

	return response, nil
}

// GetTaskAuditTrail 获取任务审计追踪
func (ts *TaskService) GetTaskAuditTrail(taskID string, page, size int) (map[string]interface{}, error) {
	// 获取任务相关的审计日志
	auditLogs, total, err := ts.auditService.GetAuditLogs(page, size, "", "", taskID, "", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}

	// 获取任务执行日志
	execLogs, execTotal, err := ts.auditService.GetTaskExecutionLogs(taskID, page, size, "", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution logs: %w", err)
	}

	return map[string]interface{}{
		"task_id":        taskID,
		"audit_logs":     auditLogs,
		"execution_logs": execLogs,
		"pagination": map[string]interface{}{
			"page":             page,
			"size":             size,
			"total_audit_logs": total,
			"total_exec_logs":  execTotal,
		},
	}, nil
}

// GetExecutionStatistics 获取执行统计信息
func (ts *TaskService) GetExecutionStatistics(startDate, endDate time.Time, statType string) ([]ExecutionStatistics, error) {
	return ts.auditService.GetExecutionStatistics(startDate, endDate, statType)
}

// UpdateDailyStatistics 更新每日统计信息
func (ts *TaskService) UpdateDailyStatistics() error {
	today := time.Now().Truncate(24 * time.Hour)
	return ts.auditService.UpdateExecutionStatistics(today, "daily")
}

// GetAuditSummary 获取审计摘要
func (ts *TaskService) GetAuditSummary(days int) (map[string]interface{}, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)
	return ts.auditService.GetAuditSummary(startTime, endTime)
}

// GetLogStatistics 获取日志统计信息
func (ts *TaskService) GetLogStatistics() (map[string]interface{}, error) {
	return ts.auditService.GetLogStatistics()
}

// CleanupOldLogs 清理旧日志
func (ts *TaskService) CleanupOldLogs(retentionDays int) error {
	return ts.auditService.CleanupOldAuditLogs(retentionDays)
}

// SearchLogs 搜索日志
func (ts *TaskService) SearchLogs(keyword string, logType string, startTime, endTime *time.Time, page, size int) (map[string]interface{}, error) {
	results := make(map[string]interface{})

	if logType == "" || logType == "audit" {
		// 搜索审计日志
		auditLogs, auditTotal, err := ts.auditService.GetAuditLogs(page, size, "", "", "", "", startTime, endTime)
		if err != nil {
			return nil, fmt.Errorf("failed to search audit logs: %w", err)
		}

		// 过滤包含关键词的日志
		filteredAuditLogs := make([]AuditLog, 0)
		for _, log := range auditLogs {
			if keyword == "" ||
				strings.Contains(log.Action, keyword) ||
				strings.Contains(log.EntityID, keyword) ||
				strings.Contains(log.HostID, keyword) {
				filteredAuditLogs = append(filteredAuditLogs, log)
			}
		}

		results["audit_logs"] = filteredAuditLogs
		results["audit_total"] = auditTotal
	}

	if logType == "" || logType == "execution" {
		// 搜索执行日志 - 这里需要修改 GetTaskExecutionLogs 方法来支持全局搜索
		// 暂时返回空结果
		results["execution_logs"] = []TaskExecutionLog{}
		results["execution_total"] = 0
	}

	results["keyword"] = keyword
	results["log_type"] = logType
	results["page"] = page
	results["size"] = size

	return results, nil
}

// GetTaskExecutionTimeline 获取任务执行时间线
func (ts *TaskService) GetTaskExecutionTimeline(taskID string) ([]map[string]interface{}, error) {
	// 获取任务相关的所有审计日志，按时间排序
	auditLogs, _, err := ts.auditService.GetAuditLogs(1, 1000, "", "", taskID, "", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}

	// 获取任务执行日志
	execLogs, _, err := ts.auditService.GetTaskExecutionLogs(taskID, 1, 1000, "", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution logs: %w", err)
	}

	// 合并并排序时间线事件
	timeline := make([]map[string]interface{}, 0)

	// 添加审计日志事件
	for _, log := range auditLogs {
		event := map[string]interface{}{
			"timestamp":   log.Timestamp,
			"type":        "audit",
			"action":      log.Action,
			"entity_id":   log.EntityID,
			"entity_type": log.EntityType,
			"host_id":     log.HostID,
			"user_id":     log.UserID,
			"details":     log.Details,
		}
		timeline = append(timeline, event)
	}

	// 添加执行日志事件
	for _, log := range execLogs {
		event := map[string]interface{}{
			"timestamp":  log.Timestamp,
			"type":       "execution",
			"log_level":  log.LogLevel,
			"message":    log.Message,
			"host_id":    log.HostID,
			"command_id": log.CommandID,
			"details":    log.Details,
		}
		timeline = append(timeline, event)
	}

	// 按时间戳排序
	sort.Slice(timeline, func(i, j int) bool {
		timeI := timeline[i]["timestamp"].(time.Time)
		timeJ := timeline[j]["timestamp"].(time.Time)
		return timeI.Before(timeJ)
	})

	return timeline, nil
}

// Shutdown 关闭任务服务
func (ts *TaskService) Shutdown() {
	log.Println("Shutting down task service...")

	if ts.timeoutMonitor != nil {
		ts.timeoutMonitor.Stop()
	}

	if ts.queueManager != nil {
		ts.queueManager.Shutdown()
	}

	if ts.loadMonitor != nil {
		ts.loadMonitor.Shutdown()
	}

	// 关闭批量更新队列
	close(ts.batchUpdateQueue)

	log.Println("Task service shutdown completed")
}
