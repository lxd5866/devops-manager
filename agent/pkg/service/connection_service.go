package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"devops-manager/agent/pkg/config"
	"devops-manager/agent/pkg/grpc"
	"devops-manager/api/protobuf"
)

// ConnectionService 连接管理服务
type ConnectionService struct {
	config      *config.Config
	grpcClient  *grpc.Agent
	isConnected bool
	lastPing    time.Time
	mutex       sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc

	// 回调函数
	onConnected    func()
	onDisconnected func()
	onMessage      func(*protobuf.CommandMessage)
}

// NewConnectionService 创建连接服务
func NewConnectionService(cfg *config.Config) *ConnectionService {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConnectionService{
		config:     cfg,
		grpcClient: grpc.NewAgent(cfg.Server.Address, cfg.Server.Timeout, cfg.Server.RetryInterval),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动连接服务
func (cs *ConnectionService) Start() error {
	log.Println("Starting connection service...")

	// 启动gRPC客户端
	if err := cs.grpcClient.Start(cs.ctx); err != nil {
		return err
	}

	// 启动连接监控
	go cs.connectionMonitor()

	// 启动心跳
	go cs.heartbeatLoop()

	return nil
}

// Stop 停止连接服务
func (cs *ConnectionService) Stop() {
	log.Println("Stopping connection service...")
	cs.cancel()
	if cs.grpcClient != nil {
		cs.grpcClient.Stop()
	}
}

// IsConnected 检查是否已连接
func (cs *ConnectionService) IsConnected() bool {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	return cs.isConnected
}

// GetLastPing 获取最后ping时间
func (cs *ConnectionService) GetLastPing() time.Time {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	return cs.lastPing
}

// SetCallbacks 设置回调函数
func (cs *ConnectionService) SetCallbacks(onConnected, onDisconnected func(), onMessage func(*protobuf.CommandMessage)) {
	cs.onConnected = onConnected
	cs.onDisconnected = onDisconnected
	cs.onMessage = onMessage
}

// SendMessage 发送消息
func (cs *ConnectionService) SendMessage(msg *protobuf.CommandMessage) error {
	if !cs.IsConnected() {
		return fmt.Errorf("not connected to server")
	}

	// TODO: 实现命令消息发送逻辑
	return fmt.Errorf("SendMessage not implemented yet")
}

// Register 注册到服务器
func (cs *ConnectionService) Register(hostInfo *protobuf.HostInfo) (*protobuf.RegisterResponse, error) {
	return cs.grpcClient.Register(cs.ctx, hostInfo)
}

// ReportStatus 上报状态
func (cs *ConnectionService) ReportStatus(status *protobuf.HostStatus) (*protobuf.HostStatusResponse, error) {
	return cs.grpcClient.ReportStatus(cs.ctx, status)
}

// connectionMonitor 连接监控
func (cs *ConnectionService) connectionMonitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-ticker.C:
			connected := cs.grpcClient.IsConnected()

			cs.mutex.Lock()
			wasConnected := cs.isConnected
			cs.isConnected = connected
			cs.mutex.Unlock()

			// 连接状态变化时触发回调
			if connected && !wasConnected {
				log.Println("Connected to server")
				if cs.onConnected != nil {
					cs.onConnected()
				}
			} else if !connected && wasConnected {
				log.Println("Disconnected from server")
				if cs.onDisconnected != nil {
					cs.onDisconnected()
				}
			}
		}
	}
}

// heartbeatLoop 心跳循环
func (cs *ConnectionService) heartbeatLoop() {
	ticker := time.NewTicker(cs.config.Agent.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-ticker.C:
			if cs.IsConnected() {
				cs.mutex.Lock()
				cs.lastPing = time.Now()
				cs.mutex.Unlock()

				// 这里可以发送心跳消息
				// 具体实现依赖于业务需求
			}
		}
	}
}

// GetConnectionStats 获取连接统计信息
func (cs *ConnectionService) GetConnectionStats() map[string]interface{} {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	return map[string]interface{}{
		"connected":   cs.isConnected,
		"last_ping":   cs.lastPing.Unix(),
		"server_addr": cs.config.Server.Address,
	}
}
