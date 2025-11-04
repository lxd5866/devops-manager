package service

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"
)

// SystemLoadMonitor 系统负载监控器
type SystemLoadMonitor struct {
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	cpuUsage        float64
	memoryUsage     float64
	goroutineCount  int
	systemLoad      float64
	lastUpdate      time.Time
	updateInterval  time.Duration
	loadHistory     []LoadSnapshot
	maxHistorySize  int
	alertThresholds AlertThresholds
	alertCallbacks  []AlertCallback
	metrics         SystemMetrics
}

// LoadSnapshot 负载快照
type LoadSnapshot struct {
	Timestamp      time.Time
	CPUUsage       float64
	MemoryUsage    float64
	GoroutineCount int
	SystemLoad     float64
}

// AlertThresholds 告警阈值
type AlertThresholds struct {
	CPUWarning     float64
	CPUCritical    float64
	MemoryWarning  float64
	MemoryCritical float64
	LoadWarning    float64
	LoadCritical   float64
}

// AlertCallback 告警回调函数
type AlertCallback func(alertType string, level string, value float64, threshold float64)

// SystemMetrics 系统指标
type SystemMetrics struct {
	TotalRequests       int64
	ActiveConnections   int64
	QueuedTasks         int64
	ProcessedTasks      int64
	ErrorCount          int64
	AverageResponseTime float64
	ThroughputPerSecond float64
}

// NewSystemLoadMonitor 创建系统负载监控器
func NewSystemLoadMonitor(updateInterval time.Duration) *SystemLoadMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := &SystemLoadMonitor{
		ctx:            ctx,
		cancel:         cancel,
		updateInterval: updateInterval,
		loadHistory:    make([]LoadSnapshot, 0),
		maxHistorySize: 1000, // 保留最近1000个快照
		alertThresholds: AlertThresholds{
			CPUWarning:     70.0,
			CPUCritical:    90.0,
			MemoryWarning:  80.0,
			MemoryCritical: 95.0,
			LoadWarning:    75.0,
			LoadCritical:   90.0,
		},
		alertCallbacks: make([]AlertCallback, 0),
		lastUpdate:     time.Now(),
	}

	// 启动监控
	go monitor.startMonitoring()

	log.Printf("System load monitor started with update interval: %v", updateInterval)
	return monitor
}

// startMonitoring 开始监控
func (slm *SystemLoadMonitor) startMonitoring() {
	ticker := time.NewTicker(slm.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-slm.ctx.Done():
			log.Println("System load monitor stopped")
			return
		case <-ticker.C:
			slm.updateSystemLoad()
		}
	}
}

// updateSystemLoad 更新系统负载
func (slm *SystemLoadMonitor) updateSystemLoad() {
	slm.mu.Lock()
	defer slm.mu.Unlock()

	// 获取内存统计
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 计算内存使用率
	slm.memoryUsage = float64(memStats.Alloc) / float64(memStats.Sys) * 100

	// 获取协程数量
	slm.goroutineCount = runtime.NumGoroutine()

	// 计算CPU使用率（简化实现）
	slm.cpuUsage = slm.calculateCPUUsage()

	// 计算综合系统负载
	slm.systemLoad = slm.calculateSystemLoad()

	slm.lastUpdate = time.Now()

	// 创建快照
	snapshot := LoadSnapshot{
		Timestamp:      slm.lastUpdate,
		CPUUsage:       slm.cpuUsage,
		MemoryUsage:    slm.memoryUsage,
		GoroutineCount: slm.goroutineCount,
		SystemLoad:     slm.systemLoad,
	}

	// 添加到历史记录
	slm.addToHistory(snapshot)

	// 检查告警
	slm.checkAlerts()

	// 更新指标
	slm.updateMetrics()
}

// calculateCPUUsage 计算CPU使用率
func (slm *SystemLoadMonitor) calculateCPUUsage() float64 {
	// 这里是简化的CPU使用率计算
	// 在实际实现中，应该使用更精确的方法

	// 基于协程数量和内存使用情况估算CPU使用率
	goroutineFactor := float64(slm.goroutineCount) / 1000.0 * 20 // 每1000个协程贡献20%
	memoryFactor := slm.memoryUsage * 0.3                        // 内存使用率的30%

	cpuUsage := goroutineFactor + memoryFactor
	if cpuUsage > 100.0 {
		cpuUsage = 100.0
	}

	return cpuUsage
}

// calculateSystemLoad 计算系统负载
func (slm *SystemLoadMonitor) calculateSystemLoad() float64 {
	// 综合CPU、内存和协程数量计算系统负载
	cpuWeight := 0.4
	memoryWeight := 0.4
	goroutineWeight := 0.2

	// 协程数量标准化（假设1000个协程为100%负载）
	goroutineLoad := float64(slm.goroutineCount) / 1000.0 * 100
	if goroutineLoad > 100.0 {
		goroutineLoad = 100.0
	}

	systemLoad := slm.cpuUsage*cpuWeight + slm.memoryUsage*memoryWeight + goroutineLoad*goroutineWeight

	if systemLoad > 100.0 {
		systemLoad = 100.0
	}

	return systemLoad
}

// addToHistory 添加到历史记录
func (slm *SystemLoadMonitor) addToHistory(snapshot LoadSnapshot) {
	slm.loadHistory = append(slm.loadHistory, snapshot)

	// 保持历史记录大小
	if len(slm.loadHistory) > slm.maxHistorySize {
		slm.loadHistory = slm.loadHistory[1:]
	}
}

// checkAlerts 检查告警
func (slm *SystemLoadMonitor) checkAlerts() {
	// CPU告警
	if slm.cpuUsage >= slm.alertThresholds.CPUCritical {
		slm.triggerAlert("cpu", "critical", slm.cpuUsage, slm.alertThresholds.CPUCritical)
	} else if slm.cpuUsage >= slm.alertThresholds.CPUWarning {
		slm.triggerAlert("cpu", "warning", slm.cpuUsage, slm.alertThresholds.CPUWarning)
	}

	// 内存告警
	if slm.memoryUsage >= slm.alertThresholds.MemoryCritical {
		slm.triggerAlert("memory", "critical", slm.memoryUsage, slm.alertThresholds.MemoryCritical)
	} else if slm.memoryUsage >= slm.alertThresholds.MemoryWarning {
		slm.triggerAlert("memory", "warning", slm.memoryUsage, slm.alertThresholds.MemoryWarning)
	}

	// 系统负载告警
	if slm.systemLoad >= slm.alertThresholds.LoadCritical {
		slm.triggerAlert("system_load", "critical", slm.systemLoad, slm.alertThresholds.LoadCritical)
	} else if slm.systemLoad >= slm.alertThresholds.LoadWarning {
		slm.triggerAlert("system_load", "warning", slm.systemLoad, slm.alertThresholds.LoadWarning)
	}
}

// triggerAlert 触发告警
func (slm *SystemLoadMonitor) triggerAlert(alertType, level string, value, threshold float64) {
	log.Printf("ALERT [%s]: %s usage %.2f%% exceeds %s threshold %.2f%%",
		level, alertType, value, level, threshold)

	// 调用告警回调
	for _, callback := range slm.alertCallbacks {
		go callback(alertType, level, value, threshold)
	}
}

// updateMetrics 更新指标
func (slm *SystemLoadMonitor) updateMetrics() {
	// 这里可以更新各种业务指标
	// 目前只是占位符实现
}

// GetCurrentLoad 获取当前负载
func (slm *SystemLoadMonitor) GetCurrentLoad() LoadSnapshot {
	slm.mu.RLock()
	defer slm.mu.RUnlock()

	return LoadSnapshot{
		Timestamp:      slm.lastUpdate,
		CPUUsage:       slm.cpuUsage,
		MemoryUsage:    slm.memoryUsage,
		GoroutineCount: slm.goroutineCount,
		SystemLoad:     slm.systemLoad,
	}
}

// GetLoadHistory 获取负载历史
func (slm *SystemLoadMonitor) GetLoadHistory(duration time.Duration) []LoadSnapshot {
	slm.mu.RLock()
	defer slm.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	var result []LoadSnapshot

	for _, snapshot := range slm.loadHistory {
		if snapshot.Timestamp.After(cutoff) {
			result = append(result, snapshot)
		}
	}

	return result
}

// GetLoadStatistics 获取负载统计
func (slm *SystemLoadMonitor) GetLoadStatistics(duration time.Duration) map[string]interface{} {
	history := slm.GetLoadHistory(duration)

	if len(history) == 0 {
		return map[string]interface{}{
			"error": "no data available for the specified duration",
		}
	}

	// 计算统计信息
	var totalCPU, totalMemory, totalLoad float64
	var maxCPU, maxMemory, maxLoad float64
	var minCPU, minMemory, minLoad float64
	maxGoroutines := 0
	minGoroutines := history[0].GoroutineCount

	for i, snapshot := range history {
		totalCPU += snapshot.CPUUsage
		totalMemory += snapshot.MemoryUsage
		totalLoad += snapshot.SystemLoad

		if i == 0 {
			maxCPU = snapshot.CPUUsage
			minCPU = snapshot.CPUUsage
			maxMemory = snapshot.MemoryUsage
			minMemory = snapshot.MemoryUsage
			maxLoad = snapshot.SystemLoad
			minLoad = snapshot.SystemLoad
		} else {
			if snapshot.CPUUsage > maxCPU {
				maxCPU = snapshot.CPUUsage
			}
			if snapshot.CPUUsage < minCPU {
				minCPU = snapshot.CPUUsage
			}
			if snapshot.MemoryUsage > maxMemory {
				maxMemory = snapshot.MemoryUsage
			}
			if snapshot.MemoryUsage < minMemory {
				minMemory = snapshot.MemoryUsage
			}
			if snapshot.SystemLoad > maxLoad {
				maxLoad = snapshot.SystemLoad
			}
			if snapshot.SystemLoad < minLoad {
				minLoad = snapshot.SystemLoad
			}
		}

		if snapshot.GoroutineCount > maxGoroutines {
			maxGoroutines = snapshot.GoroutineCount
		}
		if snapshot.GoroutineCount < minGoroutines {
			minGoroutines = snapshot.GoroutineCount
		}
	}

	count := float64(len(history))

	return map[string]interface{}{
		"duration_minutes": duration.Minutes(),
		"sample_count":     len(history),
		"cpu": map[string]interface{}{
			"average": totalCPU / count,
			"max":     maxCPU,
			"min":     minCPU,
		},
		"memory": map[string]interface{}{
			"average": totalMemory / count,
			"max":     maxMemory,
			"min":     minMemory,
		},
		"system_load": map[string]interface{}{
			"average": totalLoad / count,
			"max":     maxLoad,
			"min":     minLoad,
		},
		"goroutines": map[string]interface{}{
			"max": maxGoroutines,
			"min": minGoroutines,
		},
		"current": slm.GetCurrentLoad(),
	}
}

// SetAlertThresholds 设置告警阈值
func (slm *SystemLoadMonitor) SetAlertThresholds(thresholds AlertThresholds) {
	slm.mu.Lock()
	defer slm.mu.Unlock()

	slm.alertThresholds = thresholds
	log.Printf("Alert thresholds updated: %+v", thresholds)
}

// AddAlertCallback 添加告警回调
func (slm *SystemLoadMonitor) AddAlertCallback(callback AlertCallback) {
	slm.mu.Lock()
	defer slm.mu.Unlock()

	slm.alertCallbacks = append(slm.alertCallbacks, callback)
}

// UpdateMetrics 更新业务指标
func (slm *SystemLoadMonitor) UpdateMetrics(metrics SystemMetrics) {
	slm.mu.Lock()
	defer slm.mu.Unlock()

	slm.metrics = metrics
}

// GetMetrics 获取业务指标
func (slm *SystemLoadMonitor) GetMetrics() SystemMetrics {
	slm.mu.RLock()
	defer slm.mu.RUnlock()

	return slm.metrics
}

// IsSystemOverloaded 检查系统是否过载
func (slm *SystemLoadMonitor) IsSystemOverloaded() bool {
	slm.mu.RLock()
	defer slm.mu.RUnlock()

	return slm.systemLoad >= slm.alertThresholds.LoadWarning
}

// GetRecommendedConcurrency 获取推荐的并发数
func (slm *SystemLoadMonitor) GetRecommendedConcurrency(maxConcurrency int) int {
	slm.mu.RLock()
	defer slm.mu.RUnlock()

	// 基于系统负载计算推荐的并发数
	loadFactor := (100.0 - slm.systemLoad) / 100.0
	if loadFactor < 0.1 {
		loadFactor = 0.1 // 最少保持10%的并发能力
	}

	recommended := int(float64(maxConcurrency) * loadFactor)
	if recommended < 1 {
		recommended = 1
	}

	return recommended
}

// GetSystemHealth 获取系统健康状态
func (slm *SystemLoadMonitor) GetSystemHealth() map[string]interface{} {
	slm.mu.RLock()
	defer slm.mu.RUnlock()

	health := "healthy"
	if slm.systemLoad >= slm.alertThresholds.LoadCritical {
		health = "critical"
	} else if slm.systemLoad >= slm.alertThresholds.LoadWarning {
		health = "warning"
	}

	return map[string]interface{}{
		"status":           health,
		"system_load":      slm.systemLoad,
		"cpu_usage":        slm.cpuUsage,
		"memory_usage":     slm.memoryUsage,
		"goroutine_count":  slm.goroutineCount,
		"last_update":      slm.lastUpdate,
		"alert_thresholds": slm.alertThresholds,
		"uptime_seconds":   time.Since(slm.lastUpdate).Seconds(),
	}
}

// Shutdown 关闭监控器
func (slm *SystemLoadMonitor) Shutdown() {
	log.Println("Shutting down system load monitor...")
	slm.cancel()
	log.Println("System load monitor shutdown completed")
}

// ForceGC 强制垃圾回收
func (slm *SystemLoadMonitor) ForceGC() {
	log.Println("Forcing garbage collection...")
	runtime.GC()
	log.Println("Garbage collection completed")
}

// GetMemoryStats 获取详细内存统计
func (slm *SystemLoadMonitor) GetMemoryStats() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"alloc_bytes":         memStats.Alloc,
		"total_alloc_bytes":   memStats.TotalAlloc,
		"sys_bytes":           memStats.Sys,
		"num_gc":              memStats.NumGC,
		"gc_cpu_fraction":     memStats.GCCPUFraction,
		"heap_alloc_bytes":    memStats.HeapAlloc,
		"heap_sys_bytes":      memStats.HeapSys,
		"heap_idle_bytes":     memStats.HeapIdle,
		"heap_inuse_bytes":    memStats.HeapInuse,
		"heap_released_bytes": memStats.HeapReleased,
		"heap_objects":        memStats.HeapObjects,
		"stack_inuse_bytes":   memStats.StackInuse,
		"stack_sys_bytes":     memStats.StackSys,
		"next_gc_bytes":       memStats.NextGC,
		"last_gc_time":        time.Unix(0, int64(memStats.LastGC)),
	}
}
