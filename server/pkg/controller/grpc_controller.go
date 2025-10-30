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
func RegisterGRPCServices(s *grpc.Server) {
	// 注册主机服务
	RegisterHostGRPCService(s)

	// 注册任务服务
	RegisterTaskGRPCService(s)

	// 注册命令服务
	RegisterCommandGRPCService(s)
}

// RegisterCommandGRPCService 注册命令 GRPC 服务
func RegisterCommandGRPCService(s *grpc.Server) {
	commandService := NewCommandService()
	protobuf.RegisterCommandServiceServer(s, commandService)
	log.Println("Command GRPC service registered successfully")
}

// NewCommandService 创建命令服务实例
func NewCommandService() *CommandService {
	return &CommandService{}
}

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

// CommandService 命令服务实现
type CommandService struct {
	protobuf.UnimplementedCommandServiceServer
	// 存储连接的 Agent 流
	agentStreams map[string]protobuf.CommandService_ConnectForCommandsServer
	// 待发送的命令队列
	commandQueue map[string][]*protobuf.CommandContent
}

// ConnectForCommands Agent 连接到 Server 接收命令
func (cs *CommandService) ConnectForCommands(stream protobuf.CommandService_ConnectForCommandsServer) error {
	log.Println("Agent connected for command reception")

	var hostID string

	// 初始化存储
	if cs.agentStreams == nil {
		cs.agentStreams = make(map[string]protobuf.CommandService_ConnectForCommandsServer)
	}
	if cs.commandQueue == nil {
		cs.commandQueue = make(map[string][]*protobuf.CommandContent)
	}

	for {
		// 接收 Agent 消息
		msg, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				log.Printf("Agent %s disconnected", hostID)
				// 清理连接
				if hostID != "" {
					delete(cs.agentStreams, hostID)
				}
				return nil
			}
			log.Printf("Error receiving message from agent: %v", err)
			return err
		}

		// 处理不同类型的消息
		if msg.GetCommandResult() != nil {
			// Agent 返回的命令执行结果
			result := msg.GetCommandResult()
			hostID = result.HostId

			// 注册 Agent 连接
			cs.agentStreams[hostID] = stream

			log.Printf("Received command result from agent %s: command_id=%s, exit_code=%d",
				result.HostId, result.CommandId, result.ExitCode)

			// TODO: 将结果存储到数据库
			// 这里可以调用 service 层保存命令执行结果

		} else if msg.GetCommandContent() != nil {
			// Agent 发送的心跳或注册信息
			content := msg.GetCommandContent()
			hostID = content.HostId

			// 注册 Agent 连接
			cs.agentStreams[hostID] = stream
			log.Printf("Agent %s registered for command reception", hostID)

			// 检查是否有待发送的命令
			if commands, exists := cs.commandQueue[hostID]; exists && len(commands) > 0 {
				// 发送队列中的命令
				for _, cmd := range commands {
					cmdMsg := &protobuf.CommandMessage{
						CommandContent: cmd,
					}
					if err := stream.Send(cmdMsg); err != nil {
						log.Printf("Error sending queued command to agent %s: %v", hostID, err)
						return err
					}
					log.Printf("Sent queued command %s to agent %s", cmd.CommandId, hostID)
				}
				// 清空队列
				delete(cs.commandQueue, hostID)
			}
		}
	}
}

// SendCommandToAgent Server 向指定 Agent 发送命令
func (cs *CommandService) SendCommandToAgent(hostID string, command *protobuf.CommandContent) error {
	if cs.agentStreams == nil {
		cs.agentStreams = make(map[string]protobuf.CommandService_ConnectForCommandsServer)
	}
	if cs.commandQueue == nil {
		cs.commandQueue = make(map[string][]*protobuf.CommandContent)
	}

	// 检查 Agent 是否在线
	if stream, exists := cs.agentStreams[hostID]; exists {
		// Agent 在线，直接发送命令
		msg := &protobuf.CommandMessage{
			CommandContent: command,
		}

		if err := stream.Send(msg); err != nil {
			log.Printf("Error sending command to agent %s: %v", hostID, err)
			// 发送失败，可能连接已断开，移除连接并加入队列
			delete(cs.agentStreams, hostID)
			cs.commandQueue[hostID] = append(cs.commandQueue[hostID], command)
			return err
		}

		log.Printf("Sent command %s to agent %s", command.CommandId, hostID)
		return nil
	} else {
		// Agent 不在线，加入队列等待连接
		cs.commandQueue[hostID] = append(cs.commandQueue[hostID], command)
		log.Printf("Agent %s not online, command %s queued", hostID, command.CommandId)
		return nil
	}
}


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
