package controller

import (
	"log"
	"sync"
	"time"

	"devops-manager/api/protobuf"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCTaskController 任务 GRPC 控制器
// 用于与Agent建立命令执行的双向流连接
type GRPCTaskController struct {
	protobuf.UnimplementedCommandServiceServer
	// 存储与各个Agent的连接
	agentConnections map[string]protobuf.CommandService_ConnectForCommandsServer
	connectionsMutex sync.RWMutex
}

// NewGRPCTaskController 创建新的任务 GRPC 控制器
func NewGRPCTaskController() *GRPCTaskController {
	return &GRPCTaskController{
		agentConnections: make(map[string]protobuf.CommandService_ConnectForCommandsServer),
	}
}

// RegisterTaskGRPCService 注册任务 GRPC 服务
func RegisterTaskGRPCService(s *grpc.Server) {
	controller := NewGRPCTaskController()
	protobuf.RegisterCommandServiceServer(s, controller)
	log.Println("Command GRPC service registered successfully")
}

// ConnectForCommands 处理Agent的命令连接请求
// Agent调用此方法与Server建立长连接，用于接收和执行命令
func (tc *GRPCTaskController) ConnectForCommands(stream protobuf.CommandService_ConnectForCommandsServer) error {
	LogGRPCRequest("ConnectForCommands", "New agent connection")

	// 获取客户端信息（这里简化处理，实际应该从认证信息中获取）
	agentID := tc.generateAgentID()

	// 存储连接
	tc.connectionsMutex.Lock()
	tc.agentConnections[agentID] = stream
	tc.connectionsMutex.Unlock()

	log.Printf("Agent %s connected for command execution", agentID)

	// 监听Agent的响应
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Printf("Agent %s disconnected: %v", agentID, err)
			// 清理连接
			tc.connectionsMutex.Lock()
			delete(tc.agentConnections, agentID)
			tc.connectionsMutex.Unlock()
			return err
		}

		// 处理Agent返回的命令执行结果
		if result := msg.GetCommandResult(); result != nil {
			tc.handleCommandResult(agentID, result)
		}
	}
}

// SendCommandToAgent 向指定Agent发送命令
// 这是一个内部方法，供其他服务调用
func (tc *GRPCTaskController) SendCommandToAgent(agentID, commandID, command string, timeout time.Duration) error {
	tc.connectionsMutex.RLock()
	stream, exists := tc.agentConnections[agentID]
	tc.connectionsMutex.RUnlock()

	if !exists {
		log.Printf("Agent %s not connected", agentID)
		return nil
	}

	// 构建命令消息
	commandMsg := &protobuf.CommandMessage{
		CommandContent: &protobuf.CommandContent{
			CommandId:  commandID,
			HostId:     agentID,
			Command:    command,
			Parameters: "",  // 现在是 string 类型
			Timeout:    nil, // 可以根据需要设置超时
			CreatedAt:  timestamppb.Now(),
		},
	}

	// 发送命令
	if err := stream.Send(commandMsg); err != nil {
		log.Printf("Failed to send command to agent %s: %v", agentID, err)
		// 清理失效连接
		tc.connectionsMutex.Lock()
		delete(tc.agentConnections, agentID)
		tc.connectionsMutex.Unlock()
		return err
	}

	LogGRPCRequest("SendCommand", commandID)
	log.Printf("Command %s sent to agent %s", commandID, agentID)
	return nil
}

// handleCommandResult 处理Agent返回的命令执行结果
func (tc *GRPCTaskController) handleCommandResult(agentID string, result *protobuf.CommandResult) {
	LogGRPCResponse("CommandResult", result.ExitCode == 0, result.CommandId)

	log.Printf("Received command result from agent %s: command=%s, exit_code=%d",
		agentID, result.CommandId, result.ExitCode)

	// 这里可以将结果存储到数据库或发送给其他服务
	// TODO: 实现结果处理逻辑
}

// generateAgentID 生成Agent ID（简化实现）
func (tc *GRPCTaskController) generateAgentID() string {
	// 实际实现中应该从认证信息或其他方式获取真实的Agent ID
	return "agent-" + time.Now().Format("20060102150405")
}
