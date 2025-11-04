package examples

import (
	"fmt"
	"log"
	"time"

	"devops-manager/api/models"
	"devops-manager/server/pkg/service"
)

// ExampleTaskDispatcher 示例任务分发器实现
type ExampleTaskDispatcher struct{}

func (e *ExampleTaskDispatcher) SendCommandToAgent(hostID string, command *models.Command) error {
	fmt.Printf("📤 发送命令到 Agent %s:\n", hostID)
	fmt.Printf("   命令ID: %s\n", command.CommandID)
	fmt.Printf("   命令内容: %s\n", command.Command)
	fmt.Printf("   参数: %s\n", command.Parameters)
	fmt.Printf("   超时: %d 秒\n", command.Timeout)

	// 模拟异步命令执行
	go func() {
		// 模拟命令执行延迟
		time.Sleep(2 * time.Second)

		// 模拟命令执行结果
		now := time.Now()
		startTime := now.Add(-2 * time.Second)

		result := &models.CommandResult{
			CommandID:     command.CommandID,
			HostID:        hostID,
			Stdout:        fmt.Sprintf("命令 %s 在主机 %s 上执行成功", command.Command, hostID),
			Stderr:        "",
			ExitCode:      0,
			StartedAt:     &startTime,
			FinishedAt:    &now,
			ErrorMessage:  "",
			ExecutionTime: func() *int64 { t := int64(2000); return &t }(),
		}

		// 处理命令结果
		taskService := service.GetTaskService()
		err := taskService.HandleCommandResult(result)
		if err != nil {
			log.Printf("处理命令结果失败: %v", err)
		} else {
			fmt.Printf("✅ 命令 %s 在主机 %s 上执行完成\n", command.CommandID, hostID)
		}
	}()

	return nil
}

func RunTaskDispatchExample() {
	fmt.Println("🚀 任务下发执行系统示例")
	fmt.Println("========================")

	// 注意：这个示例需要数据库连接才能运行
	// 在实际使用中，需要先初始化数据库连接

	// 设置示例任务分发器
	dispatcher := &ExampleTaskDispatcher{}
	service.SetTaskDispatcher(dispatcher)

	// 获取任务服务
	taskService := service.GetTaskService()

	// 创建示例任务
	hostIDs := []string{"web-server-01", "web-server-02", "db-server-01"}

	fmt.Printf("📋 创建任务，目标主机: %v\n", hostIDs)

	task, err := taskService.CreateTask(
		"系统更新任务",
		"更新所有服务器的系统包",
		hostIDs,
		"sudo apt update && sudo apt upgrade -y",
		300, // 5分钟超时
		"",
		"admin",
	)

	if err != nil {
		log.Fatalf("创建任务失败: %v", err)
	}

	fmt.Printf("✅ 任务创建成功: %s\n", task.TaskID)
	fmt.Printf("   任务名称: %s\n", task.Name)
	fmt.Printf("   目标主机数: %d\n", task.TotalHosts)
	fmt.Printf("   任务状态: %s\n", task.Status)

	// 启动任务
	fmt.Printf("\n🎯 启动任务下发...\n")
	err = taskService.StartTask(task.TaskID)
	if err != nil {
		log.Fatalf("启动任务失败: %v", err)
	}

	// 监控任务进度
	fmt.Printf("\n📊 监控任务执行进度...\n")
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)

		status, err := taskService.GetTaskStatus(task.TaskID)
		if err != nil {
			log.Printf("获取任务状态失败: %v", err)
			continue
		}

		fmt.Printf("⏱️  [%ds] 任务状态: %s, 完成: %v/%v, 失败: %v, 成功率: %.1f%%\n",
			i+1,
			status["status"],
			status["completed_hosts"],
			status["total_hosts"],
			status["failed_hosts"],
			status["success_rate"],
		)

		// 检查任务是否完成
		if status["status"] == string(models.TaskStatusCompleted) ||
			status["status"] == string(models.TaskStatusFailed) ||
			status["status"] == string(models.TaskStatusCanceled) {
			fmt.Printf("\n🎉 任务执行完成！\n")
			break
		}
	}

	// 获取最终任务详情
	finalTask, err := taskService.GetTask(task.TaskID)
	if err != nil {
		log.Printf("获取最终任务状态失败: %v", err)
		return
	}

	fmt.Printf("\n📈 任务执行总结:\n")
	fmt.Printf("   任务ID: %s\n", finalTask.TaskID)
	fmt.Printf("   最终状态: %s\n", finalTask.Status)
	fmt.Printf("   总主机数: %d\n", finalTask.TotalHosts)
	fmt.Printf("   成功主机数: %d\n", finalTask.CompletedHosts)
	fmt.Printf("   失败主机数: %d\n", finalTask.FailedHosts)
	fmt.Printf("   成功率: %.1f%%\n", finalTask.SuccessRate())

	if finalTask.StartedAt != nil && finalTask.FinishedAt != nil {
		duration := finalTask.Duration()
		fmt.Printf("   执行时长: %v\n", duration)
	}

	fmt.Printf("\n✨ 示例完成！\n")
}
