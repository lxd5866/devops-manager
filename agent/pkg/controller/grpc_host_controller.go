package controller

import (
	"context"
	"log"

	"devops-manager/agent/pkg/service"
	"devops-manager/api/protobuf"
)

// HostGRPCController 主机gRPC业务控制器
type HostGRPCController struct {
	protobuf.UnimplementedHostServiceServer
	hostService *service.HostAgent
}

// NewHostGRPCController 创建主机gRPC控制器
func NewHostGRPCController() *HostGRPCController {
	return &HostGRPCController{
		// hostService 将在启动时注入
	}
}

// SetHostService 设置主机服务
func (hgc *HostGRPCController) SetHostService(hostService *service.HostAgent) {
	hgc.hostService = hostService
}

// Register 处理主机注册请求
func (hgc *HostGRPCController) Register(ctx context.Context, req *protobuf.HostInfo) (*protobuf.RegisterResponse, error) {
	LogGRPCRequest("Register", req.Hostname)

	// 这里实际上是Agent作为客户端，不会接收注册请求
	// 但为了实现接口完整性，提供一个默认实现
	response := &protobuf.RegisterResponse{
		Success:      false,
		ErrorMessage: "Agent does not accept registration requests",
	}

	LogGRPCResponse("Register", false, "Agent does not accept registration requests")
	return response, nil
}

// ReportStatus 处理状态上报请求
func (hgc *HostGRPCController) ReportStatus(ctx context.Context, req *protobuf.HostStatus) (*protobuf.HostStatusResponse, error) {
	LogGRPCRequest("ReportStatus", req.HostId)

	// Agent作为客户端，不会接收状态上报请求
	response := &protobuf.HostStatusResponse{
		Success: false,
		Message: "Agent does not accept status reports",
	}

	LogGRPCResponse("ReportStatus", false, "Agent does not accept status reports")
	return response, nil
}

// GetHostInfo 获取本机信息（内部方法）
func (hgc *HostGRPCController) GetHostInfo() *protobuf.HostInfo {
	if hgc.hostService == nil {
		return nil
	}

	// 这里应该从hostService获取主机信息
	// 具体实现依赖于service层的接口
	log.Println("Getting host info from service")

	// 返回默认信息，实际实现需要从service获取
	return &protobuf.HostInfo{
		Id:       "agent-local",
		Hostname: "localhost",
		Ip:       "127.0.0.1",
		Os:       "linux",
		Tags:     make(map[string]string),
	}
}
