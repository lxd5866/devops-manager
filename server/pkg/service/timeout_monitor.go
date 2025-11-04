package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"devops-manager/api/models"

	"gorm.io/gorm"
)

// TimeoutMonitor 超时监控器
type TimeoutMonitor struct {
	db            *gorm.DB
	taskService   *TaskService
	checkInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	running       bool
	mutex         sync.RWMutex
}

// NewTimeoutMonitor 创建新的超时监控器
func NewTimeoutMonitor(db *gorm.DB, taskService *TaskService) *TimeoutMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &TimeoutMonitor{
		db:            db,
		taskService:   taskService,
		checkInterval: 30 * time.Second, // 每30秒检查一次
		ctx:           ctx,
		cancel:        cancel,
		running:       false,
	}
}

// Start 启动超时监控
func (tm *TimeoutMonitor) Start() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if tm.running {
		log.Println("Timeout monitor is already running")
		return
	}

	tm.running = true
	tm.wg.Add(1)

	go func() {
		defer tm.wg.Done()
		tm.monitorLoop()
	}()

	log.Println("Timeout monitor started")
}

// Stop 停止超时监控
func (tm *TimeoutMonitor) Stop() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if !tm.running {
		return
	}

	tm.cancel()
	tm.wg.Wait()
	tm.running = false

	log.Println("Timeout monitor stopped")
}

// IsRunning 检查监控器是否正在运行
func (tm *TimeoutMonitor) IsRunning() bool {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	return tm.running
}

// monitorLoop 监控循环
func (tm *TimeoutMonitor) monitorLoop() {
	ticker := time.NewTicker(tm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tm.ctx.Done():
			log.Println("Timeout monitor loop stopped")
			return
		case <-ticker.C:
			tm.checkTimeouts()
		}
	}
}

// checkTimeouts 检查超时的命令
func (tm *TimeoutMonitor) checkTimeouts() {
	// 查找所有运行中的命令
	var runningCommands []models.Command
	err := tm.db.Where("status = ?", models.CommandStatusRunning).Find(&runningCommands).Error
	if err != nil {
		log.Printf("Failed to query running commands: %v", err)
		return
	}

	now := time.Now()
	var timeoutCommands []models.Command

	for _, cmd := range runningCommands {
		// 检查命令是否超时
		if tm.isCommandTimeout(cmd, now) {
			timeoutCommands = append(timeoutCommands, cmd)
		}
	}

	if len(timeoutCommands) > 0 {
		log.Printf("Found %d timeout commands", len(timeoutCommands))
		tm.handleTimeoutCommands(timeoutCommands)
	}
}

// isCommandTimeout 检查命令是否超时
func (tm *TimeoutMonitor) isCommandTimeout(cmd models.Command, now time.Time) bool {
	// 如果没有设置超时时间，不处理超时
	if cmd.Timeout <= 0 {
		return false
	}

	// 如果命令还没有开始执行，不处理超时
	if cmd.StartedAt == nil {
		return false
	}

	// 计算执行时长
	executionDuration := now.Sub(*cmd.StartedAt)
	timeoutDuration := time.Duration(cmd.Timeout) * time.Second

	return executionDuration > timeoutDuration
}

// handleTimeoutCommands 处理超时的命令
func (tm *TimeoutMonitor) handleTimeoutCommands(timeoutCommands []models.Command) {
	for _, cmd := range timeoutCommands {
		err := tm.handleSingleTimeoutCommand(cmd)
		if err != nil {
			log.Printf("Failed to handle timeout command %s: %v", cmd.CommandID, err)
		}
	}
}

// handleSingleTimeoutCommand 处理单个超时命令
func (tm *TimeoutMonitor) handleSingleTimeoutCommand(cmd models.Command) error {
	return tm.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 更新命令状态为超时
		cmdUpdates := map[string]interface{}{
			"status":      models.CommandStatusTimeout,
			"finished_at": now,
			"error_msg":   "Command execution timeout",
			"updated_at":  now,
		}

		err := tx.Model(&models.Command{}).Where("command_id = ?", cmd.CommandID).Updates(cmdUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to update timeout command: %w", err)
		}

		// 更新 CommandHost 状态为超时
		hostUpdates := map[string]interface{}{
			"status":        string(models.CommandHostStatusTimeout),
			"finished_at":   now,
			"error_message": "Command execution timeout",
			"updated_at":    now,
		}

		err = tx.Model(&models.CommandHost{}).Where("command_id = ?", cmd.CommandID).Updates(hostUpdates).Error
		if err != nil {
			return fmt.Errorf("failed to update timeout command host: %w", err)
		}

		// 更新任务进度
		if cmd.TaskID != nil {
			err = tm.taskService.updateTaskProgressInTransaction(tx, *cmd.TaskID)
			if err != nil {
				return fmt.Errorf("failed to update task progress: %w", err)
			}
		}

		log.Printf("Command %s marked as timeout for host %s", cmd.CommandID, cmd.HostID)
		return nil
	})
}

// CheckCommandTimeout 手动检查特定命令的超时状态
func (tm *TimeoutMonitor) CheckCommandTimeout(commandID string) error {
	var cmd models.Command
	err := tm.db.Where("command_id = ? AND status = ?", commandID, models.CommandStatusRunning).First(&cmd).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // 命令不存在或不在运行状态
		}
		return fmt.Errorf("failed to get command: %w", err)
	}

	if tm.isCommandTimeout(cmd, time.Now()) {
		return tm.handleSingleTimeoutCommand(cmd)
	}

	return nil
}

// GetTimeoutStatistics 获取超时统计信息
func (tm *TimeoutMonitor) GetTimeoutStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 统计超时命令数量
	var timeoutCount int64
	err := tm.db.Model(&models.CommandHost{}).
		Where("status = ?", string(models.CommandHostStatusTimeout)).
		Count(&timeoutCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count timeout commands: %w", err)
	}

	stats["timeout_commands"] = timeoutCount

	// 统计各主机的超时次数
	var hostTimeoutCounts []struct {
		HostID string
		Count  int64
	}

	err = tm.db.Model(&models.CommandHost{}).
		Select("host_id, COUNT(*) as count").
		Where("status = ?", string(models.CommandHostStatusTimeout)).
		Group("host_id").
		Order("count DESC").
		Limit(10).
		Scan(&hostTimeoutCounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get host timeout counts: %w", err)
	}

	stats["host_timeout_counts"] = hostTimeoutCounts

	// 统计平均超时时长
	var avgTimeoutDuration struct {
		AvgDuration float64
	}

	err = tm.db.Raw(`
		SELECT AVG(execution_time) as avg_duration
		FROM commands_hosts 
		WHERE status = ? AND execution_time IS NOT NULL
	`, string(models.CommandHostStatusTimeout)).Scan(&avgTimeoutDuration).Error
	if err != nil {
		return nil, fmt.Errorf("failed to calculate average timeout duration: %w", err)
	}

	stats["average_timeout_duration_ms"] = avgTimeoutDuration.AvgDuration

	return stats, nil
}
