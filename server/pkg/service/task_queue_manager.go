package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// TaskPriority 任务优先级
type TaskPriority int

const (
	PriorityLow    TaskPriority = 1
	PriorityNormal TaskPriority = 2
	PriorityHigh   TaskPriority = 3
	PriorityUrgent TaskPriority = 4
)

// QueuedTask 队列中的任务
type QueuedTask struct {
	TaskID     string
	Priority   TaskPriority
	CreatedAt  time.Time
	HostIDs    []string
	Retries    int
	MaxRetries int
}

// HostLoad 主机负载信息
type HostLoad struct {
	HostID             string
	RunningTasks       int
	MaxConcurrentTasks int
	CPUUsage           float64
	MemoryUsage        float64
	LastUpdated        time.Time
	Available          bool
}

// TaskQueueManager 任务队列管理器
type TaskQueueManager struct {
	mu                     sync.RWMutex
	taskQueue              []*QueuedTask
	runningTasks           map[string]*QueuedTask
	hostLoads              map[string]*HostLoad
	maxConcurrentTasks     int
	maxTasksPerHost        int
	queueCapacity          int
	workerCount            int
	workers                []chan *QueuedTask
	ctx                    context.Context
	cancel                 context.CancelFunc
	taskService            TaskExecutor
	loadBalanceStrategy    LoadBalanceStrategy
	adaptiveThrottling     bool
	systemLoadThreshold    float64
	hostLoadUpdateInterval time.Duration
}

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy int

const (
	RoundRobin LoadBalanceStrategy = iota
	LeastConnections
	WeightedRoundRobin
	ResourceBased
)

// TaskExecutor 任务执行器接口
type TaskExecutor interface {
	StartTask(taskID string) error
}

// NewTaskQueueManager 创建任务队列管理器
func NewTaskQueueManager(taskService TaskExecutor, config TaskQueueConfig) *TaskQueueManager {
	ctx, cancel := context.WithCancel(context.Background())

	tqm := &TaskQueueManager{
		taskQueue:              make([]*QueuedTask, 0),
		runningTasks:           make(map[string]*QueuedTask),
		hostLoads:              make(map[string]*HostLoad),
		maxConcurrentTasks:     config.MaxConcurrentTasks,
		maxTasksPerHost:        config.MaxTasksPerHost,
		queueCapacity:          config.QueueCapacity,
		workerCount:            config.WorkerCount,
		workers:                make([]chan *QueuedTask, config.WorkerCount),
		ctx:                    ctx,
		cancel:                 cancel,
		taskService:            taskService,
		loadBalanceStrategy:    config.LoadBalanceStrategy,
		adaptiveThrottling:     config.AdaptiveThrottling,
		systemLoadThreshold:    config.SystemLoadThreshold,
		hostLoadUpdateInterval: config.HostLoadUpdateInterval,
	}

	// 初始化工作协程
	for i := 0; i < tqm.workerCount; i++ {
		tqm.workers[i] = make(chan *QueuedTask, 10)
		go tqm.worker(i, tqm.workers[i])
	}

	// 启动队列处理器
	go tqm.queueProcessor()

	// 启动主机负载监控
	go tqm.hostLoadMonitor()

	// 启动自适应调节器
	if tqm.adaptiveThrottling {
		go tqm.adaptiveThrottler()
	}

	log.Printf("Task queue manager initialized with %d workers, max concurrent tasks: %d",
		tqm.workerCount, tqm.maxConcurrentTasks)

	return tqm
}

// TaskQueueConfig 任务队列配置
type TaskQueueConfig struct {
	MaxConcurrentTasks     int
	MaxTasksPerHost        int
	QueueCapacity          int
	WorkerCount            int
	LoadBalanceStrategy    LoadBalanceStrategy
	AdaptiveThrottling     bool
	SystemLoadThreshold    float64
	HostLoadUpdateInterval time.Duration
}

// EnqueueTask 将任务加入队列
func (tqm *TaskQueueManager) EnqueueTask(taskID string, priority TaskPriority, hostIDs []string) error {
	tqm.mu.Lock()
	defer tqm.mu.Unlock()

	// 检查队列容量
	if len(tqm.taskQueue) >= tqm.queueCapacity {
		return fmt.Errorf("task queue is full, capacity: %d", tqm.queueCapacity)
	}

	// 检查任务是否已在队列或运行中
	if _, exists := tqm.runningTasks[taskID]; exists {
		return fmt.Errorf("task %s is already running", taskID)
	}

	for _, queuedTask := range tqm.taskQueue {
		if queuedTask.TaskID == taskID {
			return fmt.Errorf("task %s is already in queue", taskID)
		}
	}

	// 创建队列任务
	queuedTask := &QueuedTask{
		TaskID:     taskID,
		Priority:   priority,
		CreatedAt:  time.Now(),
		HostIDs:    hostIDs,
		Retries:    0,
		MaxRetries: 3,
	}

	// 插入到队列中（按优先级排序）
	tqm.insertTaskByPriority(queuedTask)

	log.Printf("Task %s enqueued with priority %d, queue size: %d", taskID, priority, len(tqm.taskQueue))
	return nil
}

// insertTaskByPriority 按优先级插入任务
func (tqm *TaskQueueManager) insertTaskByPriority(task *QueuedTask) {
	// 找到插入位置（优先级高的在前面，同优先级按时间排序）
	insertIndex := len(tqm.taskQueue)
	for i, queuedTask := range tqm.taskQueue {
		if task.Priority > queuedTask.Priority ||
			(task.Priority == queuedTask.Priority && task.CreatedAt.Before(queuedTask.CreatedAt)) {
			insertIndex = i
			break
		}
	}

	// 插入任务
	tqm.taskQueue = append(tqm.taskQueue, nil)
	copy(tqm.taskQueue[insertIndex+1:], tqm.taskQueue[insertIndex:])
	tqm.taskQueue[insertIndex] = task
}

// queueProcessor 队列处理器
func (tqm *TaskQueueManager) queueProcessor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tqm.ctx.Done():
			return
		case <-ticker.C:
			tqm.processQueue()
		}
	}
}

// processQueue 处理队列
func (tqm *TaskQueueManager) processQueue() {
	tqm.mu.Lock()
	defer tqm.mu.Unlock()

	// 检查是否可以处理更多任务
	if len(tqm.runningTasks) >= tqm.maxConcurrentTasks {
		return
	}

	// 检查系统负载
	if tqm.adaptiveThrottling && tqm.getSystemLoad() > tqm.systemLoadThreshold {
		log.Printf("System load too high, throttling task execution")
		return
	}

	// 处理队列中的任务
	for i := 0; i < len(tqm.taskQueue) && len(tqm.runningTasks) < tqm.maxConcurrentTasks; i++ {
		task := tqm.taskQueue[i]

		// 检查主机负载是否允许执行
		if tqm.canExecuteTask(task) {
			// 移除任务从队列
			tqm.taskQueue = append(tqm.taskQueue[:i], tqm.taskQueue[i+1:]...)
			i-- // 调整索引

			// 添加到运行中任务
			tqm.runningTasks[task.TaskID] = task

			// 更新主机负载
			tqm.updateHostLoadForTask(task, true)

			// 分配给工作协程
			workerIndex := tqm.selectWorker(task)
			select {
			case tqm.workers[workerIndex] <- task:
				log.Printf("Task %s assigned to worker %d", task.TaskID, workerIndex)
			default:
				// 工作协程忙，重新放回队列
				tqm.taskQueue = append([]*QueuedTask{task}, tqm.taskQueue...)
				delete(tqm.runningTasks, task.TaskID)
				tqm.updateHostLoadForTask(task, false)
				break
			}
		}
	}
}

// canExecuteTask 检查是否可以执行任务
func (tqm *TaskQueueManager) canExecuteTask(task *QueuedTask) bool {
	// 检查每个主机的负载
	for _, hostID := range task.HostIDs {
		hostLoad, exists := tqm.hostLoads[hostID]
		if !exists {
			// 主机负载信息不存在，创建默认值
			tqm.hostLoads[hostID] = &HostLoad{
				HostID:             hostID,
				RunningTasks:       0,
				MaxConcurrentTasks: tqm.maxTasksPerHost,
				Available:          true,
				LastUpdated:        time.Now(),
			}
			continue
		}

		// 检查主机是否可用
		if !hostLoad.Available {
			return false
		}

		// 检查主机并发任务数
		if hostLoad.RunningTasks >= hostLoad.MaxConcurrentTasks {
			return false
		}

		// 检查主机资源使用率
		if hostLoad.CPUUsage > 80.0 || hostLoad.MemoryUsage > 80.0 {
			return false
		}
	}

	return true
}

// updateHostLoadForTask 更新任务相关主机的负载
func (tqm *TaskQueueManager) updateHostLoadForTask(task *QueuedTask, increment bool) {
	for _, hostID := range task.HostIDs {
		hostLoad, exists := tqm.hostLoads[hostID]
		if !exists {
			hostLoad = &HostLoad{
				HostID:             hostID,
				RunningTasks:       0,
				MaxConcurrentTasks: tqm.maxTasksPerHost,
				Available:          true,
				LastUpdated:        time.Now(),
			}
			tqm.hostLoads[hostID] = hostLoad
		}

		if increment {
			hostLoad.RunningTasks++
		} else {
			hostLoad.RunningTasks--
			if hostLoad.RunningTasks < 0 {
				hostLoad.RunningTasks = 0
			}
		}
		hostLoad.LastUpdated = time.Now()
	}
}

// selectWorker 选择工作协程
func (tqm *TaskQueueManager) selectWorker(task *QueuedTask) int {
	switch tqm.loadBalanceStrategy {
	case RoundRobin:
		return int(time.Now().UnixNano()) % tqm.workerCount
	case LeastConnections:
		return tqm.selectLeastBusyWorker()
	case ResourceBased:
		return tqm.selectResourceBasedWorker(task)
	default:
		return int(time.Now().UnixNano()) % tqm.workerCount
	}
}

// selectLeastBusyWorker 选择最不忙的工作协程
func (tqm *TaskQueueManager) selectLeastBusyWorker() int {
	minLoad := len(tqm.workers[0])
	selectedWorker := 0

	for i, worker := range tqm.workers {
		if len(worker) < minLoad {
			minLoad = len(worker)
			selectedWorker = i
		}
	}

	return selectedWorker
}

// selectResourceBasedWorker 基于资源选择工作协程
func (tqm *TaskQueueManager) selectResourceBasedWorker(task *QueuedTask) int {
	// 简单实现：选择负责主机负载最低的工作协程
	bestWorker := 0
	bestScore := float64(1000000)

	for i := range tqm.workers {
		score := tqm.calculateWorkerScore(i, task)
		if score < bestScore {
			bestScore = score
			bestWorker = i
		}
	}

	return bestWorker
}

// calculateWorkerScore 计算工作协程得分
func (tqm *TaskQueueManager) calculateWorkerScore(workerIndex int, task *QueuedTask) float64 {
	score := float64(len(tqm.workers[workerIndex])) * 10 // 队列长度权重

	// 计算任务相关主机的平均负载
	totalLoad := 0.0
	hostCount := 0
	for _, hostID := range task.HostIDs {
		if hostLoad, exists := tqm.hostLoads[hostID]; exists {
			totalLoad += hostLoad.CPUUsage + hostLoad.MemoryUsage
			hostCount++
		}
	}

	if hostCount > 0 {
		avgLoad := totalLoad / float64(hostCount)
		score += avgLoad
	}

	return score
}

// worker 工作协程
func (tqm *TaskQueueManager) worker(workerID int, taskChan chan *QueuedTask) {
	log.Printf("Worker %d started", workerID)

	for {
		select {
		case <-tqm.ctx.Done():
			log.Printf("Worker %d stopped", workerID)
			return
		case task := <-taskChan:
			tqm.executeTask(workerID, task)
		}
	}
}

// executeTask 执行任务
func (tqm *TaskQueueManager) executeTask(workerID int, task *QueuedTask) {
	log.Printf("Worker %d executing task %s", workerID, task.TaskID)

	startTime := time.Now()

	// 执行任务
	err := tqm.taskService.StartTask(task.TaskID)

	tqm.mu.Lock()
	defer tqm.mu.Unlock()

	if err != nil {
		log.Printf("Worker %d failed to execute task %s: %v", workerID, task.TaskID, err)

		// 检查是否需要重试
		if task.Retries < task.MaxRetries {
			task.Retries++
			log.Printf("Retrying task %s, attempt %d/%d", task.TaskID, task.Retries, task.MaxRetries)

			// 延迟重试
			go func() {
				time.Sleep(time.Duration(task.Retries) * 30 * time.Second)
				tqm.mu.Lock()
				tqm.insertTaskByPriority(task)
				tqm.mu.Unlock()
			}()
		} else {
			log.Printf("Task %s failed after %d retries", task.TaskID, task.MaxRetries)
		}
	} else {
		log.Printf("Worker %d successfully started task %s in %v",
			workerID, task.TaskID, time.Since(startTime))
	}

	// 从运行中任务移除
	delete(tqm.runningTasks, task.TaskID)

	// 更新主机负载
	tqm.updateHostLoadForTask(task, false)
}

// hostLoadMonitor 主机负载监控
func (tqm *TaskQueueManager) hostLoadMonitor() {
	ticker := time.NewTicker(tqm.hostLoadUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tqm.ctx.Done():
			return
		case <-ticker.C:
			tqm.updateHostLoads()
		}
	}
}

// updateHostLoads 更新主机负载信息
func (tqm *TaskQueueManager) updateHostLoads() {
	tqm.mu.Lock()
	defer tqm.mu.Unlock()

	// 这里应该从实际的监控系统获取主机负载信息
	// 目前使用模拟数据
	for hostID, hostLoad := range tqm.hostLoads {
		// 模拟负载更新
		if time.Since(hostLoad.LastUpdated) > 5*time.Minute {
			// 主机长时间未更新，标记为不可用
			hostLoad.Available = false
			log.Printf("Host %s marked as unavailable due to no recent updates", hostID)
		} else {
			hostLoad.Available = true
		}

		// 这里可以集成实际的监控数据
		// hostLoad.CPUUsage = getHostCPUUsage(hostID)
		// hostLoad.MemoryUsage = getHostMemoryUsage(hostID)
	}
}

// adaptiveThrottler 自适应调节器
func (tqm *TaskQueueManager) adaptiveThrottler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tqm.ctx.Done():
			return
		case <-ticker.C:
			tqm.adjustConcurrency()
		}
	}
}

// adjustConcurrency 调整并发数
func (tqm *TaskQueueManager) adjustConcurrency() {
	tqm.mu.Lock()
	defer tqm.mu.Unlock()

	systemLoad := tqm.getSystemLoad()
	queueLength := len(tqm.taskQueue)
	runningTasks := len(tqm.runningTasks)

	// 根据系统负载和队列长度调整并发数
	if systemLoad < 50.0 && queueLength > 10 && runningTasks < tqm.maxConcurrentTasks {
		// 系统负载低，队列较长，可以增加并发
		newMax := min(tqm.maxConcurrentTasks+2, 100)
		if newMax != tqm.maxConcurrentTasks {
			tqm.maxConcurrentTasks = newMax
			log.Printf("Increased max concurrent tasks to %d", tqm.maxConcurrentTasks)
		}
	} else if systemLoad > 80.0 && tqm.maxConcurrentTasks > 5 {
		// 系统负载高，减少并发
		newMax := max(tqm.maxConcurrentTasks-2, 5)
		if newMax != tqm.maxConcurrentTasks {
			tqm.maxConcurrentTasks = newMax
			log.Printf("Decreased max concurrent tasks to %d", tqm.maxConcurrentTasks)
		}
	}
}

// getSystemLoad 获取系统负载
func (tqm *TaskQueueManager) getSystemLoad() float64 {
	// 这里应该获取实际的系统负载
	// 目前返回模拟值
	tqm.mu.RLock()
	defer tqm.mu.RUnlock()

	// 基于运行任务数计算负载
	loadPercentage := float64(len(tqm.runningTasks)) / float64(tqm.maxConcurrentTasks) * 100
	return loadPercentage
}

// GetQueueStatus 获取队列状态
func (tqm *TaskQueueManager) GetQueueStatus() map[string]interface{} {
	tqm.mu.RLock()
	defer tqm.mu.RUnlock()

	// 统计优先级分布
	priorityCount := make(map[TaskPriority]int)
	for _, task := range tqm.taskQueue {
		priorityCount[task.Priority]++
	}

	// 统计主机负载
	hostLoadSummary := make(map[string]interface{})
	for hostID, hostLoad := range tqm.hostLoads {
		hostLoadSummary[hostID] = map[string]interface{}{
			"running_tasks":        hostLoad.RunningTasks,
			"max_concurrent_tasks": hostLoad.MaxConcurrentTasks,
			"cpu_usage":            hostLoad.CPUUsage,
			"memory_usage":         hostLoad.MemoryUsage,
			"available":            hostLoad.Available,
			"last_updated":         hostLoad.LastUpdated,
		}
	}

	return map[string]interface{}{
		"queue_length":          len(tqm.taskQueue),
		"running_tasks":         len(tqm.runningTasks),
		"max_concurrent_tasks":  tqm.maxConcurrentTasks,
		"worker_count":          tqm.workerCount,
		"priority_distribution": priorityCount,
		"host_loads":            hostLoadSummary,
		"system_load":           tqm.getSystemLoad(),
		"load_balance_strategy": tqm.loadBalanceStrategy,
		"adaptive_throttling":   tqm.adaptiveThrottling,
	}
}

// UpdateHostLoad 更新主机负载信息
func (tqm *TaskQueueManager) UpdateHostLoad(hostID string, cpuUsage, memoryUsage float64, available bool) {
	tqm.mu.Lock()
	defer tqm.mu.Unlock()

	hostLoad, exists := tqm.hostLoads[hostID]
	if !exists {
		hostLoad = &HostLoad{
			HostID:             hostID,
			RunningTasks:       0,
			MaxConcurrentTasks: tqm.maxTasksPerHost,
		}
		tqm.hostLoads[hostID] = hostLoad
	}

	hostLoad.CPUUsage = cpuUsage
	hostLoad.MemoryUsage = memoryUsage
	hostLoad.Available = available
	hostLoad.LastUpdated = time.Now()
}

// CancelTask 取消队列中的任务
func (tqm *TaskQueueManager) CancelTask(taskID string) error {
	tqm.mu.Lock()
	defer tqm.mu.Unlock()

	// 从队列中移除
	for i, task := range tqm.taskQueue {
		if task.TaskID == taskID {
			tqm.taskQueue = append(tqm.taskQueue[:i], tqm.taskQueue[i+1:]...)
			log.Printf("Task %s removed from queue", taskID)
			return nil
		}
	}

	// 检查是否在运行中
	if task, exists := tqm.runningTasks[taskID]; exists {
		delete(tqm.runningTasks, taskID)
		tqm.updateHostLoadForTask(task, false)
		log.Printf("Task %s removed from running tasks", taskID)
		return nil
	}

	return fmt.Errorf("task %s not found in queue or running tasks", taskID)
}

// GetTaskPosition 获取任务在队列中的位置
func (tqm *TaskQueueManager) GetTaskPosition(taskID string) (int, error) {
	tqm.mu.RLock()
	defer tqm.mu.RUnlock()

	for i, task := range tqm.taskQueue {
		if task.TaskID == taskID {
			return i + 1, nil // 返回1基础的位置
		}
	}

	return -1, fmt.Errorf("task %s not found in queue", taskID)
}

// Shutdown 关闭队列管理器
func (tqm *TaskQueueManager) Shutdown() {
	log.Println("Shutting down task queue manager...")

	tqm.cancel()

	// 等待所有工作协程完成
	for i, worker := range tqm.workers {
		close(worker)
		log.Printf("Worker %d stopped", i)
	}

	log.Println("Task queue manager shutdown completed")
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
