package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"devops-manager/api/models"
	"devops-manager/server/pkg/database"

	"github.com/redis/go-redis/v9"
)

// TaskCacheService 任务缓存服务
type TaskCacheService struct {
	redis *redis.Client
	ctx   context.Context
}

// NewTaskCacheService 创建任务缓存服务
func NewTaskCacheService() *TaskCacheService {
	return &TaskCacheService{
		redis: database.GetRedis(),
		ctx:   context.Background(),
	}
}

// 缓存键前缀
const (
	TaskStatusCachePrefix    = "task:status:"
	TaskProgressCachePrefix  = "task:progress:"
	TaskStatsCachePrefix     = "task:stats:"
	TaskListCachePrefix      = "task:list:"
	HostTasksCachePrefix     = "host:tasks:"
	TaskExecutionCachePrefix = "task:execution:"
)

// 缓存过期时间
const (
	TaskStatusCacheTTL    = 5 * time.Minute  // 任务状态缓存5分钟
	TaskProgressCacheTTL  = 2 * time.Minute  // 任务进度缓存2分钟
	TaskStatsCacheTTL     = 10 * time.Minute // 任务统计缓存10分钟
	TaskListCacheTTL      = 3 * time.Minute  // 任务列表缓存3分钟
	HostTasksCacheTTL     = 5 * time.Minute  // 主机任务缓存5分钟
	TaskExecutionCacheTTL = 1 * time.Minute  // 任务执行详情缓存1分钟
)

// CacheTaskStatus 缓存任务状态
func (tcs *TaskCacheService) CacheTaskStatus(taskID string, status map[string]interface{}) error {
	key := TaskStatusCachePrefix + taskID

	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal task status: %w", err)
	}

	err = tcs.redis.Set(tcs.ctx, key, data, TaskStatusCacheTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to cache task status: %w", err)
	}

	log.Printf("Cached task status: %s", taskID)
	return nil
}

// GetCachedTaskStatus 获取缓存的任务状态
func (tcs *TaskCacheService) GetCachedTaskStatus(taskID string) (map[string]interface{}, error) {
	key := TaskStatusCachePrefix + taskID

	data, err := tcs.redis.Get(tcs.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, fmt.Errorf("failed to get cached task status: %w", err)
	}

	var status map[string]interface{}
	err = json.Unmarshal([]byte(data), &status)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached task status: %w", err)
	}

	return status, nil
}

// CacheTaskProgress 缓存任务进度
func (tcs *TaskCacheService) CacheTaskProgress(taskID string, progress map[string]interface{}) error {
	key := TaskProgressCachePrefix + taskID

	data, err := json.Marshal(progress)
	if err != nil {
		return fmt.Errorf("failed to marshal task progress: %w", err)
	}

	err = tcs.redis.Set(tcs.ctx, key, data, TaskProgressCacheTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to cache task progress: %w", err)
	}

	log.Printf("Cached task progress: %s", taskID)
	return nil
}

// GetCachedTaskProgress 获取缓存的任务进度
func (tcs *TaskCacheService) GetCachedTaskProgress(taskID string) (map[string]interface{}, error) {
	key := TaskProgressCachePrefix + taskID

	data, err := tcs.redis.Get(tcs.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, fmt.Errorf("failed to get cached task progress: %w", err)
	}

	var progress map[string]interface{}
	err = json.Unmarshal([]byte(data), &progress)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached task progress: %w", err)
	}

	return progress, nil
}

// CacheTaskStatistics 缓存任务统计信息
func (tcs *TaskCacheService) CacheTaskStatistics(stats map[string]interface{}) error {
	key := TaskStatsCachePrefix + "global"

	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal task statistics: %w", err)
	}

	err = tcs.redis.Set(tcs.ctx, key, data, TaskStatsCacheTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to cache task statistics: %w", err)
	}

	log.Printf("Cached task statistics")
	return nil
}

// GetCachedTaskStatistics 获取缓存的任务统计信息
func (tcs *TaskCacheService) GetCachedTaskStatistics() (map[string]interface{}, error) {
	key := TaskStatsCachePrefix + "global"

	data, err := tcs.redis.Get(tcs.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, fmt.Errorf("failed to get cached task statistics: %w", err)
	}

	var stats map[string]interface{}
	err = json.Unmarshal([]byte(data), &stats)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached task statistics: %w", err)
	}

	return stats, nil
}

// CacheTaskList 缓存任务列表
func (tcs *TaskCacheService) CacheTaskList(cacheKey string, tasks []*models.Task, total int) error {
	key := TaskListCachePrefix + cacheKey

	listData := map[string]interface{}{
		"tasks":     tasks,
		"total":     total,
		"cached_at": time.Now(),
	}

	data, err := json.Marshal(listData)
	if err != nil {
		return fmt.Errorf("failed to marshal task list: %w", err)
	}

	err = tcs.redis.Set(tcs.ctx, key, data, TaskListCacheTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to cache task list: %w", err)
	}

	log.Printf("Cached task list: %s", cacheKey)
	return nil
}

// GetCachedTaskList 获取缓存的任务列表
func (tcs *TaskCacheService) GetCachedTaskList(cacheKey string) ([]*models.Task, int, error) {
	key := TaskListCachePrefix + cacheKey

	data, err := tcs.redis.Get(tcs.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, 0, nil // 缓存未命中
		}
		return nil, 0, fmt.Errorf("failed to get cached task list: %w", err)
	}

	var listData map[string]interface{}
	err = json.Unmarshal([]byte(data), &listData)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal cached task list: %w", err)
	}

	// 转换任务数据
	tasksData, ok := listData["tasks"].([]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid cached task list format")
	}

	var tasks []*models.Task
	for _, taskData := range tasksData {
		taskBytes, err := json.Marshal(taskData)
		if err != nil {
			continue
		}

		var task models.Task
		err = json.Unmarshal(taskBytes, &task)
		if err != nil {
			continue
		}

		tasks = append(tasks, &task)
	}

	total, ok := listData["total"].(float64)
	if !ok {
		total = 0
	}

	return tasks, int(total), nil
}

// CacheHostTasks 缓存主机任务列表
func (tcs *TaskCacheService) CacheHostTasks(hostID, cacheKey string, tasks []*models.Task, total int) error {
	key := HostTasksCachePrefix + hostID + ":" + cacheKey

	listData := map[string]interface{}{
		"tasks":     tasks,
		"total":     total,
		"cached_at": time.Now(),
	}

	data, err := json.Marshal(listData)
	if err != nil {
		return fmt.Errorf("failed to marshal host tasks: %w", err)
	}

	err = tcs.redis.Set(tcs.ctx, key, data, HostTasksCacheTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to cache host tasks: %w", err)
	}

	log.Printf("Cached host tasks: %s, key: %s", hostID, cacheKey)
	return nil
}

// GetCachedHostTasks 获取缓存的主机任务列表
func (tcs *TaskCacheService) GetCachedHostTasks(hostID, cacheKey string) ([]*models.Task, int, error) {
	key := HostTasksCachePrefix + hostID + ":" + cacheKey

	data, err := tcs.redis.Get(tcs.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, 0, nil // 缓存未命中
		}
		return nil, 0, fmt.Errorf("failed to get cached host tasks: %w", err)
	}

	var listData map[string]interface{}
	err = json.Unmarshal([]byte(data), &listData)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal cached host tasks: %w", err)
	}

	// 转换任务数据
	tasksData, ok := listData["tasks"].([]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid cached host tasks format")
	}

	var tasks []*models.Task
	for _, taskData := range tasksData {
		taskBytes, err := json.Marshal(taskData)
		if err != nil {
			continue
		}

		var task models.Task
		err = json.Unmarshal(taskBytes, &task)
		if err != nil {
			continue
		}

		tasks = append(tasks, &task)
	}

	total, ok := listData["total"].(float64)
	if !ok {
		total = 0
	}

	return tasks, int(total), nil
}

// CacheTaskExecution 缓存任务执行详情
func (tcs *TaskCacheService) CacheTaskExecution(taskID string, execution map[string]interface{}) error {
	key := TaskExecutionCachePrefix + taskID

	data, err := json.Marshal(execution)
	if err != nil {
		return fmt.Errorf("failed to marshal task execution: %w", err)
	}

	err = tcs.redis.Set(tcs.ctx, key, data, TaskExecutionCacheTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to cache task execution: %w", err)
	}

	log.Printf("Cached task execution: %s", taskID)
	return nil
}

// GetCachedTaskExecution 获取缓存的任务执行详情
func (tcs *TaskCacheService) GetCachedTaskExecution(taskID string) (map[string]interface{}, error) {
	key := TaskExecutionCachePrefix + taskID

	data, err := tcs.redis.Get(tcs.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, fmt.Errorf("failed to get cached task execution: %w", err)
	}

	var execution map[string]interface{}
	err = json.Unmarshal([]byte(data), &execution)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached task execution: %w", err)
	}

	return execution, nil
}

// InvalidateTaskCache 使任务相关缓存失效
func (tcs *TaskCacheService) InvalidateTaskCache(taskID string) error {
	keys := []string{
		TaskStatusCachePrefix + taskID,
		TaskProgressCachePrefix + taskID,
		TaskExecutionCachePrefix + taskID,
	}

	for _, key := range keys {
		err := tcs.redis.Del(tcs.ctx, key).Err()
		if err != nil {
			log.Printf("Failed to invalidate cache key %s: %v", key, err)
		}
	}

	// 使任务统计缓存失效
	err := tcs.InvalidateTaskStatistics()
	if err != nil {
		log.Printf("Failed to invalidate task statistics cache: %v", err)
	}

	log.Printf("Invalidated task cache: %s", taskID)
	return nil
}

// InvalidateTaskListCache 使任务列表缓存失效
func (tcs *TaskCacheService) InvalidateTaskListCache() error {
	// 使用模式匹配删除所有任务列表缓存
	keys, err := tcs.redis.Keys(tcs.ctx, TaskListCachePrefix+"*").Result()
	if err != nil {
		return fmt.Errorf("failed to get task list cache keys: %w", err)
	}

	if len(keys) > 0 {
		err = tcs.redis.Del(tcs.ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to invalidate task list cache: %w", err)
		}
	}

	log.Printf("Invalidated task list cache, %d keys deleted", len(keys))
	return nil
}

// InvalidateHostTasksCache 使主机任务缓存失效
func (tcs *TaskCacheService) InvalidateHostTasksCache(hostID string) error {
	// 使用模式匹配删除指定主机的任务缓存
	pattern := HostTasksCachePrefix + hostID + ":*"
	keys, err := tcs.redis.Keys(tcs.ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get host tasks cache keys: %w", err)
	}

	if len(keys) > 0 {
		err = tcs.redis.Del(tcs.ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to invalidate host tasks cache: %w", err)
		}
	}

	log.Printf("Invalidated host tasks cache for %s, %d keys deleted", hostID, len(keys))
	return nil
}

// InvalidateAllTaskCache 使所有任务缓存失效
func (tcs *TaskCacheService) InvalidateAllTaskCache() error {
	patterns := []string{
		TaskStatusCachePrefix + "*",
		TaskProgressCachePrefix + "*",
		TaskStatsCachePrefix + "*",
		TaskListCachePrefix + "*",
		HostTasksCachePrefix + "*",
		TaskExecutionCachePrefix + "*",
	}

	totalDeleted := 0
	for _, pattern := range patterns {
		keys, err := tcs.redis.Keys(tcs.ctx, pattern).Result()
		if err != nil {
			log.Printf("Failed to get cache keys for pattern %s: %v", pattern, err)
			continue
		}

		if len(keys) > 0 {
			err = tcs.redis.Del(tcs.ctx, keys...).Err()
			if err != nil {
				log.Printf("Failed to delete cache keys for pattern %s: %v", pattern, err)
			} else {
				totalDeleted += len(keys)
			}
		}
	}

	log.Printf("Invalidated all task cache, %d keys deleted", totalDeleted)
	return nil
}

// GetCacheStatistics 获取缓存统计信息
func (tcs *TaskCacheService) GetCacheStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 统计各类缓存的数量
	patterns := map[string]string{
		"task_status":    TaskStatusCachePrefix + "*",
		"task_progress":  TaskProgressCachePrefix + "*",
		"task_stats":     TaskStatsCachePrefix + "*",
		"task_list":      TaskListCachePrefix + "*",
		"host_tasks":     HostTasksCachePrefix + "*",
		"task_execution": TaskExecutionCachePrefix + "*",
	}

	cacheCounts := make(map[string]int)
	totalKeys := 0

	for cacheType, pattern := range patterns {
		keys, err := tcs.redis.Keys(tcs.ctx, pattern).Result()
		if err != nil {
			log.Printf("Failed to count cache keys for %s: %v", cacheType, err)
			cacheCounts[cacheType] = 0
		} else {
			cacheCounts[cacheType] = len(keys)
			totalKeys += len(keys)
		}
	}

	stats["cache_counts"] = cacheCounts
	stats["total_cache_keys"] = totalKeys

	// 获取 Redis 内存使用情况
	memInfo, err := tcs.redis.Info(tcs.ctx, "memory").Result()
	if err == nil {
		stats["redis_memory_info"] = memInfo
	}

	// 获取 Redis 连接信息
	clientInfo, err := tcs.redis.Info(tcs.ctx, "clients").Result()
	if err == nil {
		stats["redis_client_info"] = clientInfo
	}

	return stats, nil
}

// GenerateTaskListCacheKey 生成任务列表缓存键
func (tcs *TaskCacheService) GenerateTaskListCacheKey(page, size int, status, name string) string {
	return fmt.Sprintf("page:%d:size:%d:status:%s:name:%s", page, size, status, name)
}

// GenerateHostTasksCacheKey 生成主机任务缓存键
func (tcs *TaskCacheService) GenerateHostTasksCacheKey(page, size int, status string) string {
	return fmt.Sprintf("page:%d:size:%d:status:%s", page, size, status)
}

// WarmupCache 预热缓存
func (tcs *TaskCacheService) WarmupCache() error {
	log.Printf("Starting cache warmup...")

	// 这里可以预加载一些常用的缓存数据
	// 例如：最近的任务列表、统计信息等

	log.Printf("Cache warmup completed")
	return nil
}

// CleanupExpiredCache 清理过期缓存（可选，Redis 会自动清理）
func (tcs *TaskCacheService) CleanupExpiredCache() error {
	// Redis 会自动清理过期的键，这里主要用于手动清理或统计
	log.Printf("Cache cleanup completed")
	return nil
}

// InvalidateTaskStatistics 使任务统计缓存失效
func (tcs *TaskCacheService) InvalidateTaskStatistics() error {
	err := tcs.redis.Del(tcs.ctx, TaskStatsCachePrefix+"global").Err()
	if err != nil {
		return fmt.Errorf("failed to invalidate task statistics cache: %w", err)
	}

	log.Printf("Invalidated task statistics cache")
	return nil
}
