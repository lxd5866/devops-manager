package controller

import (
	"log"

	"devops-manager/api/protobuf"

	"google.golang.org/grpc"
)

// GRPCController gRPC基础控制器，不实现具体业务逻辑
type GRPCController struct {
	server *grpc.Server
}

// NewGRPCController 创建gRPC控制器
func NewGRPCController() *GRPCController {
	return &GRPCController{
		server: grpc.NewServer(),
	}
}

// RegisterServices 注册所有gRPC服务
func (gc *GRPCController) RegisterServices() {
	// 注册主机服务
	RegisterHostGRPCService(gc.server)

	// 注册任务服务
	RegisterTaskGRPCService(gc.server)

	// 注册文件服务
	RegisterFileGRPCService(gc.server)

	log.Println("All gRPC services registered successfully")
}

// GetServer 获取gRPC服务器实例
func (gc *GRPCController) GetServer() *grpc.Server {
	return gc.server
}

// RegisterHostGRPCService 注册主机gRPC服务
func RegisterHostGRPCService(s *grpc.Server) {
	hostController := NewHostGRPCController()
	protobuf.RegisterHostServiceServer(s, hostController)
	log.Println("Host gRPC service registered")
}

// RegisterTaskGRPCService 注册任务gRPC服务
func RegisterTaskGRPCService(s *grpc.Server) {
	taskController := NewTaskGRPCController()
	protobuf.RegisterCommandServiceServer(s, taskController)
	log.Println("Task gRPC service registered")
}

// RegisterFileGRPCService 注册文件gRPC服务
func RegisterFileGRPCService(s *grpc.Server) {
	// 文件服务暂时通过HTTP实现
	log.Println("File gRPC service registered (placeholder)")
}

// LogGRPCRequest 记录gRPC请求日志
func LogGRPCRequest(method string, details string) {
	log.Printf("gRPC Request - Method: %s, Details: %s", method, details)
}

// LogGRPCResponse 记录gRPC响应日志
func LogGRPCResponse(method string, success bool, message string) {
	log.Printf("gRPC Response - Method: %s, Success: %t, Message: %s", method, success, message)
}
