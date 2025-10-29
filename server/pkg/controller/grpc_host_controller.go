package controller

import (
	"context"
	"log"

	"devops-manager/api/protobuf"
	"devops-manager/server/pkg/service"

	"google.golang.org/grpc"
)

// GRPCHostController 主机 GRPC 控制器
type GRPCHostController struct {
	protobuf.UnimplementedHostServiceServer
	hostService *service.HostService
}

// NewGRPCHostController 创建新的主机 GRPC 控制器
func NewGRPCHostController() *GRPCHostController {
	return &GRPCHostController{
		hostService: service.GetHostService(),
	}
}

// RegisterHostGRPCService 注册主机 GRPC 服务
func RegisterHostGRPCService(s *grpc.Server) {
	controller := NewGRPCHostController()
	protobuf.RegisterHostServiceServer(s, controller)
	log.Println("Host GRPC service registered successfully")
}

// Register 主机注册
func (gc *GRPCHostController) Register(ctx context.Context, req *protobuf.HostInfo) (*protobuf.RegisterResponse, error) {
	LogGRPCRequest("Register", req.Hostname)

	// 验证请求
	if req.Hostname == "" {
		LogGRPCResponse("Register", false, "Hostname is required")
		return &protobuf.RegisterResponse{
			Success:      false,
			ErrorMessage: "Hostname is required",
		}, nil
	}

	// 注册主机
	err := gc.hostService.RegisterHost(req)
	if err != nil {
		LogGRPCResponse("Register", false, err.Error())
		return &protobuf.RegisterResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	LogGRPCResponse("Register", true, "Host registered successfully: "+req.Id)

	return &protobuf.RegisterResponse{
		Success:    true,
		AssignedId: req.Id,
	}, nil
}

// ReportStatus 处理主机状态上报
func (gc *GRPCHostController) ReportStatus(ctx context.Context, req *protobuf.HostStatus) (*protobuf.HostStatusResponse, error) {
	LogGRPCRequest("ReportStatus", req.HostId)

	// 验证请求
	if req.HostId == "" {
		LogGRPCResponse("ReportStatus", false, "Host ID is required")
		return &protobuf.HostStatusResponse{
			Success: false,
			Message: "Host ID is required",
		}, nil
	}

	// 处理状态上报
	err := gc.hostService.ReportHostStatus(req)
	if err != nil {
		LogGRPCResponse("ReportStatus", false, err.Error())
		return &protobuf.HostStatusResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	LogGRPCResponse("ReportStatus", true, "Status report processed successfully")

	return &protobuf.HostStatusResponse{
		Success: true,
		Message: "Status report received successfully",
	}, nil
}

// GetHost 获取主机信息（辅助方法）
func (gc *GRPCHostController) GetHost(id string) (*protobuf.HostInfo, bool) {
	return gc.hostService.GetHost(id)
}

// GetAllHosts 获取所有主机信息（辅助方法）
func (gc *GRPCHostController) GetAllHosts() []*protobuf.HostInfo {
	return gc.hostService.GetAllHosts()
}
