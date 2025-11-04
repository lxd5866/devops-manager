package controller

import (
	"context"
	"log"

	"devops-manager/api/protobuf"
	"devops-manager/server/pkg/service"

	"google.golang.org/grpc"
)

// GRPCController GRPC 控制器基础结构
type GRPCController struct {
	protobuf.UnimplementedHostServiceServer
	hostService *service.HostService
}

// NewGRPCController 创建新的 GRPC 控制器
func NewGRPCController() *GRPCController {
	return &GRPCController{
		hostService: service.GetHostService(),
	}
}

// RegisterGRPCServices 注册所有 GRPC 服务
func RegisterGRPCServices(s *grpc.Server) *GRPCTaskController {
	// 注册主机服务
	RegisterHostGRPCService(s)

	// 注册任务服务 - 需要传入任务服务实例
	taskService := service.GetTaskService()
	taskController := RegisterTaskGRPCService(s, taskService)

	// 注意：CommandService 现在由 GRPCTaskController 实现，不需要单独注册

	return taskController
}

// SetupTaskDispatcher 设置任务分发器，建立 TaskService 和 gRPC 控制器的连接
func SetupTaskDispatcher(taskController *GRPCTaskController) {
	// 将 gRPC 任务控制器设置为任务分发器
	service.SetTaskDispatcher(taskController)
	log.Println("Task dispatcher setup completed")
}

// 注意：RegisterCommandGRPCService 已被移除
// CommandService 现在由 GRPCTaskController 实现

// LogGRPCRequest 记录 GRPC 请求日志
func LogGRPCRequest(method string, details string) {
	log.Printf("GRPC Request - Method: %s, Details: %s", method, details)
}

// LogGRPCResponse 记录 GRPC 响应日志
func LogGRPCResponse(method string, success bool, message string) {
	log.Printf("GRPC Response - Method: %s, Success: %t, Message: %s", method, success, message)
}

// ValidateGRPCRequest 验证 GRPC 请求
func ValidateGRPCRequest(req interface{}) error {
	// TODO: 实现通用请求验证逻辑
	return nil
}

// 注意：旧的 CommandService 实现已被移除
// 现在使用 GRPCTaskController 来处理命令服务

// Register 实现HostServiceServer接口
func (gc *GRPCController) Register(ctx context.Context, req *protobuf.HostInfo) (*protobuf.RegisterResponse, error) {
	LogGRPCRequest("Register", req.Hostname)

	err := gc.hostService.RegisterHost(req)
	if err != nil {
		LogGRPCResponse("Register", false, err.Error())
		return &protobuf.RegisterResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	LogGRPCResponse("Register", true, "Host registered successfully")
	return &protobuf.RegisterResponse{
		Success:    true,
		AssignedId: req.Id,
	}, nil
}

// ReportStatus 实现HostServiceServer接口
func (gc *GRPCController) ReportStatus(ctx context.Context, req *protobuf.HostStatus) (*protobuf.HostStatusResponse, error) {
	LogGRPCRequest("ReportStatus", req.HostId)

	err := gc.hostService.ReportHostStatus(req)
	if err != nil {
		LogGRPCResponse("ReportStatus", false, err.Error())
		return &protobuf.HostStatusResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	LogGRPCResponse("ReportStatus", true, "Status reported successfully")
	return &protobuf.HostStatusResponse{
		Success: true,
		Message: "Status reported successfully",
	}, nil
}
