package examples

import (
	"fmt"
	"log"
	"time"

	"devops-manager/server/pkg/service"
)

// ExampleTaskService 示例任务服务
type ExampleTaskService struct{}

func (e *ExampleTaskService) StartTask(taskID string) error {
	log.Printf("Executing task: %s", taskID)
	// 模拟任务执行时间
	time.Sleep(time.Duration(100+taskID[len(taskID)-1]) * time.Millisecond)
	log.Printf("Task completed: %s", taskID)
	return nil
}

func RunTaskQueueExample() {
	log.Println("=== 任务队列和并发控制示例 ===")

	// 创建示例任务服务
	taskService := &ExampleTaskService{}

	// 配置任务队列管理器
	config := service.TaskQueueConfig{
		MaxConcurrentTasks:     3,  // 最大并发任务数
		MaxTasksPerHost:        2,  // 每个主机最大任务数
		QueueCapacity:          50, // 队列容量
		WorkerCount:            5,  // 工作协程数
		LoadBalanceStrategy:    service.ResourceBased,
		AdaptiveThrottling:     true,
		SystemLoadThreshold:    75.0,
		HostLoadUpdateInterval: 10 * time.Second,
	}

	// 创建任务队列管理器
	queueManager := service.NewTaskQueueManager(taskService, config)
	defer queueManager.Shutdown()

	// 创建系统负载监控器
	loadMonitor := service.NewSystemLoadMonitor(2 * time.Second)
	defer loadMonitor.Shutdown()

	log.Println("\n1. 添加不同优先级的任务到队列")

	// 添加不同优先级的任务
	tasks := []struct {
		taskID   string
		priority service.TaskPriority
		hostIDs  []string
	}{
		{"task-urgent-1", service.PriorityUrgent, []string{"host-1"}},
		{"task-normal-1", service.PriorityNormal, []string{"host-2"}},
		{"task-high-1", service.PriorityHigh, []string{"host-3"}},
		{"task-low-1", service.PriorityLow, []string{"host-4"}},
		{"task-normal-2", service.PriorityNormal, []string{"host-1", "host-2"}},
		{"task-urgent-2", service.PriorityUrgent, []string{"host-5"}},
	}

	for _, task := range tasks {
		err := queueManager.EnqueueTask(task.taskID, task.priority, task.hostIDs)
		if err != nil {
			log.Printf("Failed to enqueue task %s: %v", task.taskID, err)
		} else {
			log.Printf("Enqueued task %s with priority %d", task.taskID, task.priority)
		}
	}

	log.Println("\n2. 查看队列状态")
	status := queueManager.GetQueueStatus()
	fmt.Printf("队列长度: %d\n", status["queue_length"])
	fmt.Printf("运行中任务: %d\n", status["running_tasks"])
	fmt.Printf("最大并发数: %d\n", status["max_concurrent_tasks"])
	fmt.Printf("工作协程数: %d\n", status["worker_count"])
	fmt.Printf("负载均衡策略: %d\n", status["load_balance_strategy"])
	fmt.Printf("自适应调节: %t\n", status["adaptive_throttling"])

	// 显示优先级分布
	if priorityDist, ok := status["priority_distribution"].(map[service.TaskPriority]int); ok {
		fmt.Println("优先级分布:")
		for priority, count := range priorityDist {
			fmt.Printf("  优先级 %d: %d 个任务\n", priority, count)
		}
	}

	log.Println("\n3. 更新主机负载信息")
	queueManager.UpdateHostLoad("host-1", 45.0, 60.0, true)
	queueManager.UpdateHostLoad("host-2", 80.0, 85.0, true) // 高负载
	queueManager.UpdateHostLoad("host-3", 30.0, 40.0, true)
	queueManager.UpdateHostLoad("host-4", 60.0, 70.0, false) // 不可用
	queueManager.UpdateHostLoad("host-5", 25.0, 35.0, true)

	log.Println("\n4. 查看主机负载状态")
	status = queueManager.GetQueueStatus()
	if hostLoads, ok := status["host_loads"].(map[string]interface{}); ok {
		fmt.Println("主机负载状态:")
		for hostID, load := range hostLoads {
			if hostLoad, ok := load.(map[string]interface{}); ok {
				fmt.Printf("  %s: CPU=%.1f%%, Memory=%.1f%%, Available=%t, RunningTasks=%d\n",
					hostID,
					hostLoad["cpu_usage"],
					hostLoad["memory_usage"],
					hostLoad["available"],
					hostLoad["running_tasks"])
			}
		}
	}

	log.Println("\n5. 等待任务执行")
	time.Sleep(5 * time.Second)

	log.Println("\n6. 查看系统负载状态")
	health := loadMonitor.GetSystemHealth()
	fmt.Printf("系统状态: %s\n", health["status"])
	fmt.Printf("系统负载: %.2f%%\n", health["system_load"])
	fmt.Printf("CPU使用率: %.2f%%\n", health["cpu_usage"])
	fmt.Printf("内存使用率: %.2f%%\n", health["memory_usage"])
	fmt.Printf("协程数量: %d\n", health["goroutine_count"])

	log.Println("\n7. 获取负载统计信息")
	stats := loadMonitor.GetLoadStatistics(1 * time.Minute)
	if sampleCount, ok := stats["sample_count"].(int); ok && sampleCount > 0 {
		fmt.Printf("样本数量: %d\n", sampleCount)

		if cpuStats, ok := stats["cpu"].(map[string]interface{}); ok {
			fmt.Printf("CPU统计: 平均=%.2f%%, 最大=%.2f%%, 最小=%.2f%%\n",
				cpuStats["average"], cpuStats["max"], cpuStats["min"])
		}

		if memStats, ok := stats["memory"].(map[string]interface{}); ok {
			fmt.Printf("内存统计: 平均=%.2f%%, 最大=%.2f%%, 最小=%.2f%%\n",
				memStats["average"], memStats["max"], memStats["min"])
		}

		if loadStats, ok := stats["system_load"].(map[string]interface{}); ok {
			fmt.Printf("负载统计: 平均=%.2f%%, 最大=%.2f%%, 最小=%.2f%%\n",
				loadStats["average"], loadStats["max"], loadStats["min"])
		}
	}

	log.Println("\n8. 测试任务取消")
	// 添加一个新任务
	err := queueManager.EnqueueTask("task-to-cancel", service.PriorityNormal, []string{"host-1"})
	if err != nil {
		log.Printf("Failed to enqueue task: %v", err)
	} else {
		log.Println("Added task-to-cancel to queue")

		// 获取任务位置
		position, err := queueManager.GetTaskPosition("task-to-cancel")
		if err == nil {
			fmt.Printf("Task position in queue: %d\n", position)
		}

		// 取消任务
		err = queueManager.CancelTask("task-to-cancel")
		if err != nil {
			log.Printf("Failed to cancel task: %v", err)
		} else {
			log.Println("Task cancelled successfully")
		}
	}

	log.Println("\n9. 获取推荐并发数")
	recommended := loadMonitor.GetRecommendedConcurrency(10)
	fmt.Printf("推荐并发数: %d (最大: 10)\n", recommended)

	log.Println("\n10. 检查系统是否过载")
	overloaded := loadMonitor.IsSystemOverloaded()
	fmt.Printf("系统过载: %t\n", overloaded)

	log.Println("\n11. 最终队列状态")
	finalStatus := queueManager.GetQueueStatus()
	fmt.Printf("最终队列长度: %d\n", finalStatus["queue_length"])
	fmt.Printf("最终运行中任务: %d\n", finalStatus["running_tasks"])

	log.Println("\n=== 示例完成 ===")
}
