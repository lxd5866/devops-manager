package controller

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"devops-manager/api/models"
	"devops-manager/api/protobuf"

	"google.golang.org/grpc"
)

// AgentConnection Agent连接信息
type AgentConnection struct {
	Stream      protobuf.CommandService_ConnectForCommandsServer
	ConnectedAt time.Time
	LastPing    time.Time
	IsActive    bool
	Context     context.Context
	Cancel      context.CancelFunc
}

// ConnectionPool 连接池管理
type ConnectionPool struct {
	connections map[string]*AgentConnection
	mutex       sync.RWMutex
	// 心跳检测配置
	heartbeatInterval time.Duration
	connectionTimeout time.Duration
	// 停止心跳检测的通道
	stopHeartbeat chan struct{}
}

// GRPCTaskController 任务 GRPC 控制器
// 用于与Agent建立命令执行的双向流连接
type GRPCTaskController struct {
	protobuf.UnimplementedCommandServiceServer
	// 连接池管理
	connectionPool *ConnectionPool
	// 任务服务引用，用于处理命令结果
	taskService TaskServiceInterface
}

// TaskServiceInterface 任务服务接口，避免循环导入
type TaskServiceInterface interface {
	HandleCommandResult(result *models.CommandResult) error
	HandleHostConnectionChange(hostID string, connected bool) error
}

// AddConnection 添加Agent连接到连接池
func (cp *ConnectionPool) AddConnection(agentID string, stream protobuf.CommandService_ConnectForCommandsServer) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	// 如果已存在连接，先清理旧连接
	if oldConn, exists := cp.connections[agentID]; exists {
		if oldConn.Cancel != nil {
			oldConn.Cancel()
		}
	}

	// 创建新的连接上下文
	ctx, cancel := context.WithCancel(context.Background())

	cp.connections[agentID] = &AgentConnection{
		Stream:      stream,
		ConnectedAt: time.Now(),
		LastPing:    time.Now(),
		IsActive:    true,
		Context:     ctx,
		Cancel:      cancel,
	}

	log.Printf("Agent %s added to connection pool", agentID)
}

// RemoveConnection 从连接池移除Agent连接
func (cp *ConnectionPool) RemoveConnection(agentID string) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if conn, exists := cp.connections[agentID]; exists {
		if conn.Cancel != nil {
			conn.Cancel()
		}
		delete(cp.connections, agentID)
		log.Printf("Agent %s removed from connection pool", agentID)
	}
}

// GetConnection 获取Agent连接
func (cp *ConnectionPool) GetConnection(agentID string) (*AgentConnection, bool) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	conn, exists := cp.connections[agentID]
	if exists && conn.IsActive {
		return conn, true
	}
	return nil, false
}

// UpdateLastPing 更新Agent最后心跳时间
func (cp *ConnectionPool) UpdateLastPing(agentID string) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if conn, exists := cp.connections[agentID]; exists {
		conn.LastPing = time.Now()
		conn.IsActive = true
	}
}

// GetActiveConnections 获取所有活跃连接
func (cp *ConnectionPool) GetActiveConnections() map[string]*AgentConnection {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	activeConns := make(map[string]*AgentConnection)
	for agentID, conn := range cp.connections {
		if conn.IsActive {
			activeConns[agentID] = conn
		}
	}
	return activeConns
}

// startHeartbeatMonitor 启动心跳监控
func (cp *ConnectionPool) startHeartbeatMonitor() {
	ticker := time.NewTicker(cp.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cp.checkConnectionHealth()
		case <-cp.stopHeartbeat:
			log.Println("Heartbeat monitor stopped")
			return
		}
	}
}

// checkConnectionHealth 检查连接健康状态
func (cp *ConnectionPool) checkConnectionHealth() {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	now := time.Now()
	var disconnectedAgents []string

	for agentID, conn := range cp.connections {
		// 检查连接是否超时
		if now.Sub(conn.LastPing) > cp.connectionTimeout {
			log.Printf("Agent %s connection timeout, marking as inactive", agentID)
			conn.IsActive = false
			disconnectedAgents = append(disconnectedAgents, agentID)
		}
	}

	// 清理超时的连接
	for _, agentID := range disconnectedAgents {
		if conn, exists := cp.connections[agentID]; exists {
			if conn.Cancel != nil {
				conn.Cancel()
			}
			delete(cp.connections, agentID)
		}
	}
}

// Stop 停止连接池
func (cp *ConnectionPool) Stop() {
	close(cp.stopHeartbeat)

	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	// 清理所有连接
	for agentID, conn := range cp.connections {
		if conn.Cancel != nil {
			conn.Cancel()
		}
		log.Printf("Connection closed for agent %s", agentID)
	}
	cp.connections = make(map[string]*AgentConnection)
}

// NewConnectionPool 创建新的连接池
func NewConnectionPool() *ConnectionPool {
	pool := &ConnectionPool{
		connections:       make(map[string]*AgentConnection),
		heartbeatInterval: 30 * time.Second, // 30秒心跳间隔
		connectionTimeout: 90 * time.Second, // 90秒连接超时
		stopHeartbeat:     make(chan struct{}),
	}

	// 启动心跳检测
	go pool.startHeartbeatMonitor()

	return pool
}

// NewGRPCTaskController 创建新的任务 GRPC 控制器
func NewGRPCTaskController(taskService TaskServiceInterface) *GRPCTaskController {
	return &GRPCTaskController{
		connectionPool: NewConnectionPool(),
		taskService:    taskService,
	}
}

// RegisterTaskGRPCService 注册任务 GRPC 服务
func RegisterTaskGRPCService(s *grpc.Server, taskService TaskServiceInterface) *GRPCTaskController {
	controller := NewGRPCTaskController(taskService)
	protobuf.RegisterCommandServiceServer(s, controller)
	log.Println("Command GRPC service registered successfully")
	return controller
}

// ConnectForCommands 处理Agent的命令连接请求
// Agent调用此方法与Server建立长连接，用于接收和执行命令
func (tc *GRPCTaskController) ConnectForCommands(stream protobuf.CommandService_ConnectForCommandsServer) error {
	LogGRPCRequest("ConnectForCommands", "New agent connection")

	var agentID string
	var isRegistered bool

	// 监听Agent的消息
	for {
		msg, err := stream.Recv()
		if err != nil {
			if agentID != "" {
				log.Printf("Agent %s disconnected: %v", agentID, err)
				// 从连接池移除连接
				tc.connectionPool.RemoveConnection(agentID)

				// 通知任务服务主机连接断开
				if tc.taskService != nil {
					tc.taskService.HandleHostConnectionChange(agentID, false)
				}
			} else {
				log.Printf("Unknown agent disconnected: %v", err)
			}
			return err
		}

		// 处理Agent返回的命令执行结果
		if result := msg.GetCommandResult(); result != nil {
			// 如果还没有注册，从结果中获取 Agent ID
			if !isRegistered {
				agentID = result.HostId
				tc.registerAgent(agentID, stream)
				isRegistered = true
			}
			// 更新心跳时间
			tc.connectionPool.UpdateLastPing(agentID)
			// 处理命令结果
			tc.handleCommandResult(agentID, result)
		}

		// 处理Agent发送的心跳或注册信息
		if content := msg.GetCommandContent(); content != nil {
			// 如果还没有注册，从内容中获取 Agent ID
			if !isRegistered {
				agentID = content.HostId
				tc.registerAgent(agentID, stream)
				isRegistered = true
			}
			// 更新心跳时间
			tc.connectionPool.UpdateLastPing(agentID)
			log.Printf("Received heartbeat from agent %s", agentID)
		}
	}
}

// registerAgent 注册 Agent 连接
func (tc *GRPCTaskController) registerAgent(agentID string, stream protobuf.CommandService_ConnectForCommandsServer) {
	// 添加到连接池
	tc.connectionPool.AddConnection(agentID, stream)

	log.Printf("Agent %s registered for command execution", agentID)

	// 通知任务服务主机连接建立
	if tc.taskService != nil {
		tc.taskService.HandleHostConnectionChange(agentID, true)
	}
}

// SendCommandToAgent 实现 TaskDispatcher 接口 - 向指定Agent发送命令
func (tc *GRPCTaskController) SendCommandToAgent(hostID string, command *models.Command) error {
	// 从连接池获取连接
	conn, exists := tc.connectionPool.GetConnection(hostID)
	if !exists {
		return fmt.Errorf("agent %s not connected or inactive", hostID)
	}

	// 将 Command 模型转换为 protobuf 格式
	commandContent := command.ToProtobufContent()

	// 构建命令消息
	commandMsg := &protobuf.CommandMessage{
		CommandContent: commandContent,
	}

	// 发送命令
	if err := conn.Stream.Send(commandMsg); err != nil {
		log.Printf("Failed to send command to agent %s: %v", hostID, err)
		// 从连接池移除失效连接
		tc.connectionPool.RemoveConnection(hostID)

		// 通知任务服务主机连接断开
		if tc.taskService != nil {
			tc.taskService.HandleHostConnectionChange(hostID, false)
		}

		return err
	}

	LogGRPCRequest("SendCommand", command.CommandID)
	log.Printf("Command %s sent to agent %s", command.CommandID, hostID)
	return nil
}

// handleCommandResult 处理Agent返回的命令执行结果
func (tc *GRPCTaskController) handleCommandResult(agentID string, result *protobuf.CommandResult) {
	LogGRPCResponse("CommandResult", result.ExitCode == 0, result.CommandId)

	log.Printf("Received command result from agent %s: command=%s, exit_code=%d, started_at=%v, finished_at=%v",
		agentID, result.CommandId, result.ExitCode, result.StartedAt, result.FinishedAt)

	// 验证结果数据完整性
	if result.CommandId == "" {
		log.Printf("Warning: Received command result with empty command_id from agent %s", agentID)
		return
	}

	if result.HostId == "" {
		log.Printf("Warning: Received command result with empty host_id from agent %s", agentID)
		return
	}

	// 将 protobuf 结果转换为模型
	commandResult := models.CreateCommandResultFromProtobuf(result)

	// 记录详细的执行信息
	if commandResult.StartedAt != nil && commandResult.FinishedAt != nil {
		duration := commandResult.FinishedAt.Sub(*commandResult.StartedAt)
		log.Printf("Command %s execution completed: duration=%v, exit_code=%d, stdout_size=%d, stderr_size=%d",
			result.CommandId, duration, result.ExitCode, len(result.Stdout), len(result.Stderr))
	} else if commandResult.StartedAt != nil {
		log.Printf("Command %s execution started at %v", result.CommandId, commandResult.StartedAt)
	}

	// 通过任务服务处理命令结果
	if tc.taskService != nil {
		err := tc.taskService.HandleCommandResult(commandResult)
		if err != nil {
			log.Printf("Failed to handle command result for command %s from agent %s: %v",
				result.CommandId, agentID, err)

			// 记录处理失败的详细信息
			log.Printf("Command result details: exit_code=%d, stdout_length=%d, stderr_length=%d, error_message=%s",
				result.ExitCode, len(result.Stdout), len(result.Stderr), result.ErrorMessage)
		} else {
			log.Printf("Successfully processed command result for command %s from agent %s",
				result.CommandId, agentID)
		}
	} else {
		log.Printf("Warning: TaskService not set, command result not processed for command %s from agent %s",
			result.CommandId, agentID)
	}
}

// GetConnectedAgents 获取所有已连接的Agent列表
func (tc *GRPCTaskController) GetConnectedAgents() []string {
	activeConns := tc.connectionPool.GetActiveConnections()
	agents := make([]string, 0, len(activeConns))

	for agentID := range activeConns {
		agents = append(agents, agentID)
	}

	return agents
}

// GetConnectionInfo 获取连接信息
func (tc *GRPCTaskController) GetConnectionInfo(agentID string) map[string]interface{} {
	conn, exists := tc.connectionPool.GetConnection(agentID)
	if !exists {
		return map[string]interface{}{
			"connected": false,
			"error":     "Agent not connected",
		}
	}

	return map[string]interface{}{
		"connected":    true,
		"connected_at": conn.ConnectedAt,
		"last_ping":    conn.LastPing,
		"is_active":    conn.IsActive,
		"uptime":       time.Since(conn.ConnectedAt).Seconds(),
	}
}

// GetConnectionStatistics 获取连接统计信息
func (tc *GRPCTaskController) GetConnectionStatistics() map[string]interface{} {
	activeConns := tc.connectionPool.GetActiveConnections()

	stats := map[string]interface{}{
		"total_connections":  len(activeConns),
		"active_agents":      make([]string, 0, len(activeConns)),
		"connection_details": make([]map[string]interface{}, 0, len(activeConns)),
	}

	for agentID, conn := range activeConns {
		stats["active_agents"] = append(stats["active_agents"].([]string), agentID)
		stats["connection_details"] = append(stats["connection_details"].([]map[string]interface{}), map[string]interface{}{
			"agent_id":     agentID,
			"connected_at": conn.ConnectedAt,
			"last_ping":    conn.LastPing,
			"uptime":       time.Since(conn.ConnectedAt).Seconds(),
		})
	}

	return stats
}

// SendHeartbeatToAgent 向Agent发送心跳检测
func (tc *GRPCTaskController) SendHeartbeatToAgent(agentID string) error {
	conn, exists := tc.connectionPool.GetConnection(agentID)
	if !exists {
		return fmt.Errorf("agent %s not connected", agentID)
	}

	// 构建心跳消息（使用空的CommandContent作为心跳）
	heartbeatMsg := &protobuf.CommandMessage{
		CommandContent: &protobuf.CommandContent{
			CommandId: "heartbeat-" + time.Now().Format("20060102150405"),
			HostId:    agentID,
			Command:   "ping",
		},
	}

	if err := conn.Stream.Send(heartbeatMsg); err != nil {
		log.Printf("Failed to send heartbeat to agent %s: %v", agentID, err)
		tc.connectionPool.RemoveConnection(agentID)
		return err
	}

	log.Printf("Heartbeat sent to agent %s", agentID)
	return nil
}

// BroadcastToAllAgents 向所有连接的Agent广播消息
func (tc *GRPCTaskController) BroadcastToAllAgents(message *protobuf.CommandMessage) map[string]error {
	activeConns := tc.connectionPool.GetActiveConnections()
	results := make(map[string]error)

	for agentID, conn := range activeConns {
		if err := conn.Stream.Send(message); err != nil {
			log.Printf("Failed to broadcast message to agent %s: %v", agentID, err)
			tc.connectionPool.RemoveConnection(agentID)
			results[agentID] = err
		} else {
			results[agentID] = nil
		}
	}

	return results
}

// DisconnectAgent 主动断开Agent连接
func (tc *GRPCTaskController) DisconnectAgent(agentID string) error {
	conn, exists := tc.connectionPool.GetConnection(agentID)
	if !exists {
		return fmt.Errorf("agent %s not connected", agentID)
	}

	// 取消连接上下文
	if conn.Cancel != nil {
		conn.Cancel()
	}

	// 从连接池移除
	tc.connectionPool.RemoveConnection(agentID)

	// 通知任务服务
	if tc.taskService != nil {
		tc.taskService.HandleHostConnectionChange(agentID, false)
	}

	log.Printf("Agent %s disconnected by server", agentID)
	return nil
}

// Shutdown 关闭控制器，清理所有连接
func (tc *GRPCTaskController) Shutdown() {
	log.Println("Shutting down gRPC task controller...")

	// 停止连接池
	tc.connectionPool.Stop()

	log.Println("gRPC task controller shutdown completed")
}

// generateAgentID 生成Agent ID（简化实现）
func (tc *GRPCTaskController) generateAgentID() string {
	// 实际实现中应该从认证信息或其他方式获取真实的Agent ID
	return "agent-" + time.Now().Format("20060102150405")
}
