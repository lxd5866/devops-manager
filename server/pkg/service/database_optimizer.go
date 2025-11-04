package service

import (
	"fmt"
	"log"
	"time"

	"devops-manager/api/models"

	"gorm.io/gorm"
)

// DatabaseOptimizer 数据库优化器
type DatabaseOptimizer struct {
	db *gorm.DB
}

// NewDatabaseOptimizer 创建数据库优化器
func NewDatabaseOptimizer(db *gorm.DB) *DatabaseOptimizer {
	return &DatabaseOptimizer{db: db}
}

// CreateOptimizedIndexes 创建优化索引
func (do *DatabaseOptimizer) CreateOptimizedIndexes() error {
	log.Println("Creating optimized database indexes...")

	// 任务表索引
	taskIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_created_by ON tasks(created_by)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_status_created_at ON tasks(status, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_created_by_status ON tasks(created_by, status)",
	}

	// 命令表索引
	commandIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_commands_task_id ON commands(task_id)",
		"CREATE INDEX IF NOT EXISTS idx_commands_host_id ON commands(host_id)",
		"CREATE INDEX IF NOT EXISTS idx_commands_status ON commands(status)",
		"CREATE INDEX IF NOT EXISTS idx_commands_created_at ON commands(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_commands_task_status ON commands(task_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_commands_host_status ON commands(host_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_commands_status_created_at ON commands(status, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_commands_started_at ON commands(started_at)",
		"CREATE INDEX IF NOT EXISTS idx_commands_finished_at ON commands(finished_at)",
	}

	// 命令主机表索引
	commandHostIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_commands_hosts_command_id ON commands_hosts(command_id)",
		"CREATE INDEX IF NOT EXISTS idx_commands_hosts_host_id ON commands_hosts(host_id)",
		"CREATE INDEX IF NOT EXISTS idx_commands_hosts_status ON commands_hosts(status)",
		"CREATE INDEX IF NOT EXISTS idx_commands_hosts_created_at ON commands_hosts(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_commands_hosts_command_status ON commands_hosts(command_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_commands_hosts_host_status ON commands_hosts(host_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_commands_hosts_status_created_at ON commands_hosts(status, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_commands_hosts_started_at ON commands_hosts(started_at)",
		"CREATE INDEX IF NOT EXISTS idx_commands_hosts_finished_at ON commands_hosts(finished_at)",
		"CREATE INDEX IF NOT EXISTS idx_commands_hosts_execution_time ON commands_hosts(execution_time)",
	}

	// 命令结果表索引
	commandResultIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_command_results_host_id ON command_results(host_id)",
		"CREATE INDEX IF NOT EXISTS idx_command_results_created_at ON command_results(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_command_results_exit_code ON command_results(exit_code)",
		"CREATE INDEX IF NOT EXISTS idx_command_results_host_created_at ON command_results(host_id, created_at)",
	}

	// 执行所有索引创建
	allIndexes := append(taskIndexes, commandIndexes...)
	allIndexes = append(allIndexes, commandHostIndexes...)
	allIndexes = append(allIndexes, commandResultIndexes...)

	for _, indexSQL := range allIndexes {
		if err := do.db.Exec(indexSQL).Error; err != nil {
			log.Printf("Failed to create index: %s, error: %v", indexSQL, err)
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	log.Printf("Successfully created %d optimized indexes", len(allIndexes))
	return nil
}

// BatchUpdateCommandHostStatus 批量更新 CommandHost 状态
func (do *DatabaseOptimizer) BatchUpdateCommandHostStatus(updates []CommandHostStatusUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	return do.db.Transaction(func(tx *gorm.DB) error {
		// 按状态分组批量更新
		statusGroups := make(map[string][]string)
		updateData := make(map[string]CommandHostStatusUpdate)

		for _, update := range updates {
			key := fmt.Sprintf("%s_%s", update.Status, update.ErrorMessage)
			statusGroups[key] = append(statusGroups[key], update.CommandID)
			updateData[key] = update
		}

		// 批量执行更新
		for key, commandIDs := range statusGroups {
			update := updateData[key]
			updateFields := map[string]interface{}{
				"status":     update.Status,
				"updated_at": time.Now(),
			}

			if update.FinishedAt != nil {
				updateFields["finished_at"] = update.FinishedAt
			}
			if update.ErrorMessage != "" {
				updateFields["error_message"] = update.ErrorMessage
			}
			if update.ExitCode != nil {
				updateFields["exit_code"] = *update.ExitCode
			}
			if update.ExecutionTime != nil {
				updateFields["execution_time"] = *update.ExecutionTime
			}

			err := tx.Model(&models.CommandHost{}).
				Where("command_id IN ?", commandIDs).
				Updates(updateFields).Error
			if err != nil {
				return fmt.Errorf("failed to batch update command host status: %w", err)
			}
		}

		log.Printf("Batch updated %d command host records", len(updates))
		return nil
	})
}

// BatchUpdateCommandStatus 批量更新 Command 状态
func (do *DatabaseOptimizer) BatchUpdateCommandStatus(updates []CommandStatusUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	return do.db.Transaction(func(tx *gorm.DB) error {
		// 按状态分组批量更新
		statusGroups := make(map[string][]string)
		updateData := make(map[string]CommandStatusUpdate)

		for _, update := range updates {
			key := fmt.Sprintf("%s_%s", update.Status, update.ErrorMsg)
			statusGroups[key] = append(statusGroups[key], update.CommandID)
			updateData[key] = update
		}

		// 批量执行更新
		for key, commandIDs := range statusGroups {
			update := updateData[key]
			updateFields := map[string]interface{}{
				"status":     update.Status,
				"updated_at": time.Now(),
			}

			if update.FinishedAt != nil {
				updateFields["finished_at"] = update.FinishedAt
			}
			if update.ErrorMsg != "" {
				updateFields["error_msg"] = update.ErrorMsg
			}
			if update.ExitCode != nil {
				updateFields["exit_code"] = *update.ExitCode
			}

			err := tx.Model(&models.Command{}).
				Where("command_id IN ?", commandIDs).
				Updates(updateFields).Error
			if err != nil {
				return fmt.Errorf("failed to batch update command status: %w", err)
			}
		}

		log.Printf("Batch updated %d command records", len(updates))
		return nil
	})
}

// CleanupOldRecords 清理旧记录
func (do *DatabaseOptimizer) CleanupOldRecords(retentionDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	return do.db.Transaction(func(tx *gorm.DB) error {
		// 清理旧的命令结果
		var deletedResults int64
		err := tx.Where("created_at < ?", cutoffDate).Delete(&models.CommandResult{}).Error
		if err != nil {
			return fmt.Errorf("failed to cleanup old command results: %w", err)
		}
		tx.Model(&models.CommandResult{}).Where("created_at < ?", cutoffDate).Count(&deletedResults)

		// 清理已完成的旧任务（保留失败的任务用于分析）
		var deletedTasks int64
		err = tx.Where("created_at < ? AND status IN ?", cutoffDate, []models.TaskStatus{
			models.TaskStatusCompleted,
			models.TaskStatusCanceled,
		}).Delete(&models.Task{}).Error
		if err != nil {
			return fmt.Errorf("failed to cleanup old tasks: %w", err)
		}
		tx.Model(&models.Task{}).Where("created_at < ? AND status IN ?", cutoffDate, []models.TaskStatus{
			models.TaskStatusCompleted,
			models.TaskStatusCanceled,
		}).Count(&deletedTasks)

		// 清理孤立的命令记录（没有关联任务的命令）
		var deletedCommands int64
		err = tx.Where("task_id NOT IN (SELECT task_id FROM tasks WHERE task_id IS NOT NULL)").Delete(&models.Command{}).Error
		if err != nil {
			return fmt.Errorf("failed to cleanup orphaned commands: %w", err)
		}
		tx.Model(&models.Command{}).Where("task_id NOT IN (SELECT task_id FROM tasks WHERE task_id IS NOT NULL)").Count(&deletedCommands)

		// 清理孤立的命令主机记录
		var deletedCommandHosts int64
		err = tx.Where("command_id NOT IN (SELECT command_id FROM commands WHERE command_id IS NOT NULL)").Delete(&models.CommandHost{}).Error
		if err != nil {
			return fmt.Errorf("failed to cleanup orphaned command hosts: %w", err)
		}
		tx.Model(&models.CommandHost{}).Where("command_id NOT IN (SELECT command_id FROM commands WHERE command_id IS NOT NULL)").Count(&deletedCommandHosts)

		log.Printf("Cleanup completed: deleted %d command results, %d tasks, %d commands, %d command hosts",
			deletedResults, deletedTasks, deletedCommands, deletedCommandHosts)
		return nil
	})
}

// AnalyzeTableSizes 分析表大小
func (do *DatabaseOptimizer) AnalyzeTableSizes() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 统计各表的记录数
	tables := []struct {
		Name  string
		Model interface{}
	}{
		{"tasks", &models.Task{}},
		{"commands", &models.Command{}},
		{"commands_hosts", &models.CommandHost{}},
		{"command_results", &models.CommandResult{}},
	}

	for _, table := range tables {
		var count int64
		err := do.db.Model(table.Model).Count(&count).Error
		if err != nil {
			return nil, fmt.Errorf("failed to count %s: %w", table.Name, err)
		}
		stats[table.Name+"_count"] = count
	}

	// 统计磁盘使用情况（MySQL特定）
	var diskUsage []struct {
		TableName string `gorm:"column:TABLE_NAME"`
		DataSize  int64  `gorm:"column:DATA_LENGTH"`
		IndexSize int64  `gorm:"column:INDEX_LENGTH"`
	}

	err := do.db.Raw(`
		SELECT TABLE_NAME, DATA_LENGTH, INDEX_LENGTH 
		FROM information_schema.TABLES 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND TABLE_NAME IN ('tasks', 'commands', 'commands_hosts', 'command_results')
	`).Scan(&diskUsage).Error

	if err == nil {
		stats["disk_usage"] = diskUsage
	}

	return stats, nil
}

// OptimizeTables 优化表结构
func (do *DatabaseOptimizer) OptimizeTables() error {
	tables := []string{"tasks", "commands", "commands_hosts", "command_results"}

	for _, table := range tables {
		// MySQL 表优化
		err := do.db.Exec(fmt.Sprintf("OPTIMIZE TABLE %s", table)).Error
		if err != nil {
			log.Printf("Failed to optimize table %s: %v", table, err)
			// 不返回错误，继续优化其他表
		} else {
			log.Printf("Optimized table %s", table)
		}
	}

	return nil
}

// GetSlowQueries 获取慢查询统计
func (do *DatabaseOptimizer) GetSlowQueries() ([]map[string]interface{}, error) {
	var slowQueries []map[string]interface{}

	// 这里可以添加慢查询分析逻辑
	// 由于不同数据库的慢查询日志格式不同，这里提供一个基础框架

	return slowQueries, nil
}

// CommandHostStatusUpdate 命令主机状态更新结构
type CommandHostStatusUpdate struct {
	CommandID     string
	Status        string
	FinishedAt    *time.Time
	ErrorMessage  string
	ExitCode      *int
	ExecutionTime *int64
}

// CommandStatusUpdate 命令状态更新结构
type CommandStatusUpdate struct {
	CommandID  string
	Status     models.CommandStatus
	FinishedAt *time.Time
	ErrorMsg   string
	ExitCode   *int32
}
