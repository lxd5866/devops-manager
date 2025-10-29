package controller

import (
	"log"
	"time"

	"devops-manager/agent/pkg/service"
	"devops-manager/api/protobuf"
)

// TaskGRPCController 任务gRPC业务控制器
type TaskGRPCController struct {
	protobuf.UnimplementedCommandServiceServer
	taskService *service.TaskService
}

// NewTaskGRPCController 创建任务gRPC控制器
func NewTaskGRPCController() *TaskGRPCController {
	return &TaskGRPCController{
		taskService: service.NewTaskService(),
	}
}

// ConnectForCommands 处理命令连接请求（Agent作为服务端接收Server的命令）
func (tgc *TaskGRPCController) ConnectForCommands(stream protobuf.CommandService_ConnectForCommandsServer) error {
	LogGRPCRequest("ConnectForCommands", "Command stream established")

	log.Println("Command stream established with server")

	for {
		// 接收来自Server的命令
		msg, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving command: %v", err)
			return err
		}

		// 处理命令
		if commandContent := msg.GetCommandContent(); commandContent != nil {
			go tgc.handleCommand(stream, commandContent)
		}
	}
}

// handleCommand 处理单个命令
func (tgc *TaskGRPCController) handleCommand(stream protobuf.CommandService_ConnectForCommandsServer, cmd *protobuf.CommandContent) {
	LogGRPCRequest("HandleCommand", cmd.CommandId)

	log.Printf("Executing command: %s", cmd.Command)

	// 设置超时时间
	timeout := 30 * time.Second
	if cmd.Timeout != nil {
		timeout = cmd.Timeout.AsDuration()
	}

	// 执行命令
	result, err := tgc.taskService.ExecuteTask(cmd.CommandId, cmd.Command, timeout)

	// 构建响应
	var commandResult *protobuf.CommandResult
	if err != nil {
		commandResult = &protobuf.CommandResult{
			CommandId:    cmd.CommandId,
			HostId:       cmd.HostId,
			Stdout:       "",
			Stderr:       err.Error(),
			ExitCode:     -1,
			ErrorMessage: err.Error(),
		}
	} else {
		commandResult = &protobuf.CommandResult{
			CommandId:    cmd.CommandId,
			HostId:       cmd.HostId,
			Stdout:       result.Stdout,
			Stderr:       result.Stderr,
			ExitCode:     int32(result.ExitCode),
			ErrorMessage: result.Error,
		}
	}

	// 发送结果
	response := &protobuf.CommandMessage{
		CommandResult: commandResult,
	}

	if err := stream.Send(response); err != nil {
		log.Printf("Error sending command result: %v", err)
		return
	}

	LogGRPCResponse("HandleCommand", result.ExitCode == 0, "Command executed")
	log.Printf("Command %s completed with exit code: %d", cmd.CommandId, result.ExitCode)
}

// GetTaskStatus 获取任务状态（内部方法）
func (tgc *TaskGRPCController) GetTaskStatus(taskID string) (*service.TaskExecution, bool) {
	return tgc.taskService.GetTaskStatus(taskID)
}

// CancelTask 取消任务（内部方法）
func (tgc *TaskGRPCController) CancelTask(taskID string) error {
	return tgc.taskService.CancelTask(taskID)
}

// GetRunningTasks 获取运行中的任务列表（内部方法）
func (tgc *TaskGRPCController) GetRunningTasks() []string {
	return tgc.taskService.GetRunningTasks()
}

// CleanupTasks 清理已完成的任务（内部方法）
func (tgc *TaskGRPCController) CleanupTasks() {
	tgc.taskService.CleanupCompletedTasks(24 * time.Hour) // 清理24小时前的任务
}
