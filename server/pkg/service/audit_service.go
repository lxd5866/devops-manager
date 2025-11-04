package service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"devops-manager/api/models"
	"devops-manager/server/pkg/database"

	"gorm.io/gorm"
)

// AuditService 审计服务
type AuditService struct {
	db *gorm.DB
}

// NewAuditService 创建审计服务
func NewAuditService() *AuditService {
	return &AuditService{
		db: database.GetDB(),
	}
}

// AuditAction 审计操作类型
type AuditAction string

const (
	AuditActionTaskCreated    AuditAction = "task_created"
	AuditActionTaskStarted    AuditAction = "task_started"
	AuditActionTaskCompleted  AuditAction = "task_completed"
	AuditActionTaskFailed     AuditAction = "task_failed"
	AuditActionTaskCanceled   AuditAction = "task_canceled"
	AuditActionCommandSent    AuditAction = "command_sent"
	AuditActionCommandStarted AuditAction = "command_started"
	AuditActionCommandResult  AuditAction = "command_result"
	AuditActionCommandTimeout AuditAction = "command_timeout"
	AuditActionCommandError   AuditAction = "command_error"
	AuditActionHostConnected  AuditAction = "host_connected"
	AuditActionHostDisconnect AuditAction = "host_disconnected"
)

// AuditLog 审计日志模型
type AuditLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Action     string    `json:"action" gorm:"size:50;not null;comment:操作类型"`
	EntityID   string    `json:"entity_id" gorm:"size:255;comment:实体ID(任务ID/命令ID等)"`
	EntityType string    `json:"entity_type" gorm:"size:50;comment:实体类型"`
	HostID     string    `json:"host_id" gorm:"size:255;comment:主机ID"`
	UserID     string    `json:"user_id" gorm:"size:255;comment:用户ID"`
	Details    []byte    `json:"details" gorm:"type:json;comment:详细信息"`
	Timestamp  time.Time `json:"timestamp" gorm:"not null;comment:时间戳"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "audit_logs"
}

// TaskExecutionLog 任务执行日志模型
type TaskExecutionLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TaskID    string    `json:"task_id" gorm:"size:255;not null;comment:任务ID"`
	LogLevel  string    `json:"log_level" gorm:"size:20;default:INFO;comment:日志级别"`
	Message   string    `json:"message" gorm:"type:text;not null;comment:日志消息"`
	Details   []byte    `json:"details" gorm:"type:json;comment:详细信息"`
	HostID    string    `json:"host_id" gorm:"size:255;comment:主机ID"`
	CommandID string    `json:"command_id" gorm:"size:255;comment:命令ID"`
	Timestamp time.Time `json:"timestamp" gorm:"not null;comment:时间戳"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (TaskExecutionLog) TableName() string {
	return "task_execution_logs"
}

// ExecutionStatistics 执行统计信息模型
type ExecutionStatistics struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	Date               time.Time `json:"date" gorm:"uniqueIndex:idx_date_type;not null;comment:统计日期"`
	StatType           string    `json:"stat_type" gorm:"uniqueIndex:idx_date_type;size:50;not null;comment:统计类型"`
	TotalTasks         int64     `json:"total_tasks" gorm:"default:0;comment:总任务数"`
	CompletedTasks     int64     `json:"completed_tasks" gorm:"default:0;comment:完成任务数"`
	FailedTasks        int64     `json:"failed_tasks" gorm:"default:0;comment:失败任务数"`
	CanceledTasks      int64     `json:"canceled_tasks" gorm:"default:0;comment:取消任务数"`
	TotalCommands      int64     `json:"total_commands" gorm:"default:0;comment:总命令数"`
	SuccessfulCommands int64     `json:"successful_commands" gorm:"default:0;comment:成功命令数"`
	FailedCommands     int64     `json:"failed_commands" gorm:"default:0;comment:失败命令数"`
	TimeoutCommands    int64     `json:"timeout_commands" gorm:"default:0;comment:超时命令数"`
	AvgExecutionTime   float64   `json:"avg_execution_time" gorm:"default:0;comment:平均执行时间(秒)"`
	TotalExecutionTime int64     `json:"total_execution_time" gorm:"default:0;comment:总执行时间(毫秒)"`
	ActiveHosts        int64     `json:"active_hosts" gorm:"default:0;comment:活跃主机数"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// TableName 指定表名
func (ExecutionStatistics) TableName() string {
	return "execution_statistics"
}

// LogTaskAction 记录任务操作审计日志
func (as *AuditService) LogTaskAction(action AuditAction, taskID, userID string, details interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	auditLog := &AuditLog{
		Action:     string(action),
		EntityID:   taskID,
		EntityType: "task",
		UserID:     userID,
		Details:    detailsJSON,
		Timestamp:  time.Now(),
		CreatedAt:  time.Now(),
	}

	err = as.db.Create(auditLog).Error
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	log.Printf("Audit log created: action=%s, task_id=%s, user_id=%s", action, taskID, userID)
	return nil
}

// LogCommandAction 记录命令操作审计日志
func (as *AuditService) LogCommandAction(action AuditAction, commandID, hostID, userID string, details interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	auditLog := &AuditLog{
		Action:     string(action),
		EntityID:   commandID,
		EntityType: "command",
		HostID:     hostID,
		UserID:     userID,
		Details:    detailsJSON,
		Timestamp:  time.Now(),
		CreatedAt:  time.Now(),
	}

	err = as.db.Create(auditLog).Error
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	log.Printf("Audit log created: action=%s, command_id=%s, host_id=%s", action, commandID, hostID)
	return nil
}

// LogHostAction 记录主机操作审计日志
func (as *AuditService) LogHostAction(action AuditAction, hostID string, details interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	auditLog := &AuditLog{
		Action:     string(action),
		EntityID:   hostID,
		EntityType: "host",
		HostID:     hostID,
		Details:    detailsJSON,
		Timestamp:  time.Now(),
		CreatedAt:  time.Now(),
	}

	err = as.db.Create(auditLog).Error
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	log.Printf("Audit log created: action=%s, host_id=%s", action, hostID)
	return nil
}

// LogTaskExecution 记录任务执行日志
func (as *AuditService) LogTaskExecution(taskID, logLevel, message string, details interface{}, hostID, commandID string) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	execLog := &TaskExecutionLog{
		TaskID:    taskID,
		LogLevel:  logLevel,
		Message:   message,
		Details:   detailsJSON,
		HostID:    hostID,
		CommandID: commandID,
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	err = as.db.Create(execLog).Error
	if err != nil {
		return fmt.Errorf("failed to create execution log: %w", err)
	}

	return nil
}

// GetAuditLogs 获取审计日志
func (as *AuditService) GetAuditLogs(page, size int, action, entityType, entityID, hostID string, startTime, endTime *time.Time) ([]AuditLog, int, error) {
	var logs []AuditLog
	var total int64

	// 构建查询条件
	query := as.db.Model(&AuditLog{})

	if action != "" {
		query = query.Where("action = ?", action)
	}
	if entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}
	if entityID != "" {
		query = query.Where("entity_id = ?", entityID)
	}
	if hostID != "" {
		query = query.Where("host_id = ?", hostID)
	}
	if startTime != nil {
		query = query.Where("timestamp >= ?", startTime)
	}
	if endTime != nil {
		query = query.Where("timestamp <= ?", endTime)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("timestamp DESC").Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs: %w", err)
	}

	return logs, int(total), nil
}

// GetTaskExecutionLogs 获取任务执行日志
func (as *AuditService) GetTaskExecutionLogs(taskID string, page, size int, logLevel string, startTime, endTime *time.Time) ([]TaskExecutionLog, int, error) {
	var logs []TaskExecutionLog
	var total int64

	// 构建查询条件
	query := as.db.Model(&TaskExecutionLog{}).Where("task_id = ?", taskID)

	if logLevel != "" {
		query = query.Where("log_level = ?", logLevel)
	}
	if startTime != nil {
		query = query.Where("timestamp >= ?", startTime)
	}
	if endTime != nil {
		query = query.Where("timestamp <= ?", endTime)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count execution logs: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("timestamp ASC").Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get execution logs: %w", err)
	}

	return logs, int(total), nil
}

// GetCommandExecutionHistory 获取命令执行历史
func (as *AuditService) GetCommandExecutionHistory(commandID string) ([]AuditLog, error) {
	var logs []AuditLog

	err := as.db.Where("entity_id = ? AND entity_type = ?", commandID, "command").
		Order("timestamp ASC").
		Find(&logs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get command execution history: %w", err)
	}

	return logs, nil
}

// UpdateExecutionStatistics 更新执行统计信息
func (as *AuditService) UpdateExecutionStatistics(date time.Time, statType string) error {
	// 计算当日统计数据
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// 统计任务数据
	var taskStats struct {
		TotalTasks     int64
		CompletedTasks int64
		FailedTasks    int64
		CanceledTasks  int64
	}

	err := as.db.Model(&models.Task{}).
		Select(`
			COUNT(*) as total_tasks,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_tasks,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_tasks,
			SUM(CASE WHEN status = 'canceled' THEN 1 ELSE 0 END) as canceled_tasks
		`).
		Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
		Scan(&taskStats).Error
	if err != nil {
		return fmt.Errorf("failed to get task statistics: %w", err)
	}

	// 统计命令数据
	var commandStats struct {
		TotalCommands      int64
		SuccessfulCommands int64
		FailedCommands     int64
		TimeoutCommands    int64
		AvgExecutionTime   float64
		TotalExecutionTime int64
	}

	err = as.db.Model(&models.CommandHost{}).
		Select(`
			COUNT(*) as total_commands,
			SUM(CASE WHEN status = '执行完成' THEN 1 ELSE 0 END) as successful_commands,
			SUM(CASE WHEN status IN ('执行失败', '下发失败') THEN 1 ELSE 0 END) as failed_commands,
			SUM(CASE WHEN status = '执行超时' THEN 1 ELSE 0 END) as timeout_commands,
			AVG(CASE WHEN execution_time IS NOT NULL THEN execution_time/1000.0 ELSE NULL END) as avg_execution_time,
			SUM(CASE WHEN execution_time IS NOT NULL THEN execution_time ELSE 0 END) as total_execution_time
		`).
		Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
		Scan(&commandStats).Error
	if err != nil {
		return fmt.Errorf("failed to get command statistics: %w", err)
	}

	// 统计活跃主机数
	var activeHosts int64
	err = as.db.Model(&models.CommandHost{}).
		Select("COUNT(DISTINCT host_id)").
		Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
		Scan(&activeHosts).Error
	if err != nil {
		return fmt.Errorf("failed to get active hosts count: %w", err)
	}

	// 创建或更新统计记录
	stats := &ExecutionStatistics{
		Date:               startOfDay,
		StatType:           statType,
		TotalTasks:         taskStats.TotalTasks,
		CompletedTasks:     taskStats.CompletedTasks,
		FailedTasks:        taskStats.FailedTasks,
		CanceledTasks:      taskStats.CanceledTasks,
		TotalCommands:      commandStats.TotalCommands,
		SuccessfulCommands: commandStats.SuccessfulCommands,
		FailedCommands:     commandStats.FailedCommands,
		TimeoutCommands:    commandStats.TimeoutCommands,
		AvgExecutionTime:   commandStats.AvgExecutionTime,
		TotalExecutionTime: commandStats.TotalExecutionTime,
		ActiveHosts:        activeHosts,
		UpdatedAt:          time.Now(),
	}

	// 使用 ON DUPLICATE KEY UPDATE 或 UPSERT
	err = as.db.Where("date = ? AND stat_type = ?", startOfDay, statType).
		Assign(stats).
		FirstOrCreate(stats).Error
	if err != nil {
		return fmt.Errorf("failed to update execution statistics: %w", err)
	}

	log.Printf("Execution statistics updated for date=%s, type=%s", startOfDay.Format("2006-01-02"), statType)
	return nil
}

// GetExecutionStatistics 获取执行统计信息
func (as *AuditService) GetExecutionStatistics(startDate, endDate time.Time, statType string) ([]ExecutionStatistics, error) {
	var stats []ExecutionStatistics

	query := as.db.Model(&ExecutionStatistics{}).
		Where("date >= ? AND date <= ?", startDate, endDate)

	if statType != "" {
		query = query.Where("stat_type = ?", statType)
	}

	err := query.Order("date ASC").Find(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get execution statistics: %w", err)
	}

	return stats, nil
}

// GetAuditSummary 获取审计摘要
func (as *AuditService) GetAuditSummary(startTime, endTime time.Time) (map[string]interface{}, error) {
	summary := make(map[string]interface{})

	// 统计各类操作的数量
	var actionCounts []struct {
		Action string
		Count  int64
	}

	err := as.db.Model(&AuditLog{}).
		Select("action, COUNT(*) as count").
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Group("action").
		Order("count DESC").
		Scan(&actionCounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get action counts: %w", err)
	}

	summary["action_counts"] = actionCounts

	// 统计各实体类型的操作数量
	var entityTypeCounts []struct {
		EntityType string
		Count      int64
	}

	err = as.db.Model(&AuditLog{}).
		Select("entity_type, COUNT(*) as count").
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Group("entity_type").
		Order("count DESC").
		Scan(&entityTypeCounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get entity type counts: %w", err)
	}

	summary["entity_type_counts"] = entityTypeCounts

	// 统计最活跃的主机
	var activeHosts []struct {
		HostID string
		Count  int64
	}

	err = as.db.Model(&AuditLog{}).
		Select("host_id, COUNT(*) as count").
		Where("timestamp >= ? AND timestamp <= ? AND host_id != ''", startTime, endTime).
		Group("host_id").
		Order("count DESC").
		Limit(10).
		Scan(&activeHosts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get active hosts: %w", err)
	}

	summary["active_hosts"] = activeHosts

	// 统计总操作数
	var totalOperations int64
	err = as.db.Model(&AuditLog{}).
		Where("timestamp >= ? AND timestamp <= ?", startTime, endTime).
		Count(&totalOperations).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count total operations: %w", err)
	}

	summary["total_operations"] = totalOperations
	summary["start_time"] = startTime
	summary["end_time"] = endTime

	return summary, nil
}

// CleanupOldAuditLogs 清理旧的审计日志
func (as *AuditService) CleanupOldAuditLogs(retentionDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	// 删除旧的审计日志
	result := as.db.Where("timestamp < ?", cutoffDate).Delete(&AuditLog{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old audit logs: %w", result.Error)
	}

	// 删除旧的执行日志
	result = as.db.Where("timestamp < ?", cutoffDate).Delete(&TaskExecutionLog{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old execution logs: %w", result.Error)
	}

	log.Printf("Cleaned up audit logs older than %d days, deleted %d records", retentionDays, result.RowsAffected)
	return nil
}

// GetLogStatistics 获取日志统计信息
func (as *AuditService) GetLogStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 统计审计日志总数
	var auditLogCount int64
	err := as.db.Model(&AuditLog{}).Count(&auditLogCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count audit logs: %w", err)
	}
	stats["total_audit_logs"] = auditLogCount

	// 统计执行日志总数
	var execLogCount int64
	err = as.db.Model(&TaskExecutionLog{}).Count(&execLogCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count execution logs: %w", err)
	}
	stats["total_execution_logs"] = execLogCount

	// 统计今日日志数量
	today := time.Now().Truncate(24 * time.Hour)
	var todayAuditLogs int64
	err = as.db.Model(&AuditLog{}).Where("timestamp >= ?", today).Count(&todayAuditLogs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count today audit logs: %w", err)
	}
	stats["today_audit_logs"] = todayAuditLogs

	var todayExecLogs int64
	err = as.db.Model(&TaskExecutionLog{}).Where("timestamp >= ?", today).Count(&todayExecLogs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count today execution logs: %w", err)
	}
	stats["today_execution_logs"] = todayExecLogs

	// 统计最近7天的日志趋势
	var dailyTrend []struct {
		Date      string
		AuditLogs int64
		ExecLogs  int64
	}

	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Truncate(24 * time.Hour)
		nextDate := date.Add(24 * time.Hour)
		dateStr := date.Format("2006-01-02")

		var auditCount, execCount int64
		as.db.Model(&AuditLog{}).Where("timestamp >= ? AND timestamp < ?", date, nextDate).Count(&auditCount)
		as.db.Model(&TaskExecutionLog{}).Where("timestamp >= ? AND timestamp < ?", date, nextDate).Count(&execCount)

		dailyTrend = append(dailyTrend, struct {
			Date      string
			AuditLogs int64
			ExecLogs  int64
		}{
			Date:      dateStr,
			AuditLogs: auditCount,
			ExecLogs:  execCount,
		})
	}

	stats["daily_trend"] = dailyTrend

	return stats, nil
}
