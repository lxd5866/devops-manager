package service

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"devops-manager/agent/pkg/config"
	"devops-manager/agent/pkg/grpc"
	"devops-manager/agent/pkg/utils"
	"devops-manager/api/protobuf"
)

type HostAgent struct {
	config       *config.Config
	grpcAgent    *grpc.Agent
	hostInfo     *protobuf.HostInfo
	ctx          context.Context
	cancel       context.CancelFunc
	mutex        sync.RWMutex
	startTime    time.Time
	isRegistered bool
	lastRegister time.Time
}

func NewHostAgent(cfg *config.Config) *HostAgent {
	ctx, cancel := context.WithCancel(context.Background())

	hostInfo := &protobuf.HostInfo{
		Id:       generateAgentID(cfg.Agent.AgentID),
		Hostname: utils.GetHostname(),
		Ip:       utils.GetLocalIP(),
		Os:       runtime.GOOS,
		Tags:     make(map[string]string),
	}

	// 复制配置中的标签
	for k, v := range cfg.Agent.Tags {
		hostInfo.Tags[k] = v
	}

	grpcAgent := grpc.NewAgent(cfg.Server.Address, cfg.Server.Timeout, cfg.Server.RetryInterval)

	return &HostAgent{
		config:    cfg,
		grpcAgent: grpcAgent,
		hostInfo:  hostInfo,
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}
}

func (ha *HostAgent) Start() error {
	log.Printf("Starting host agent for %s (ID: %s)", ha.hostInfo.Hostname, ha.hostInfo.Id)

	// 启动 gRPC 客户端
	if err := ha.grpcAgent.Start(ha.ctx); err != nil {
		return fmt.Errorf("failed to start grpc agent: %w", err)
	}

	// 启动状态上报器
	go ha.statusReporter()

	return nil
}

func (ha *HostAgent) Stop() {
	log.Println("Stopping host agent...")
	ha.cancel()
	ha.grpcAgent.Stop()
}

func (ha *HostAgent) Wait() {
	<-ha.ctx.Done()
}

func (ha *HostAgent) statusReporter() {
	// 首次连接时尝试注册
	registerTicker := time.NewTicker(30 * time.Second) // 每30秒检查一次注册状态
	reportTicker := time.NewTicker(ha.config.Agent.ReportInterval)
	defer registerTicker.Stop()
	defer reportTicker.Stop()

	for {
		select {
		case <-ha.ctx.Done():
			return
		case <-registerTicker.C:
			if ha.grpcAgent.IsConnected() && !ha.isRegistered {
				if err := ha.tryRegister(); err != nil {
					log.Printf("Failed to register: %v", err)
				}
			}
		case <-reportTicker.C:
			if ha.grpcAgent.IsConnected() && ha.isRegistered {
				if err := ha.reportStatus(); err != nil {
					log.Printf("Failed to report status: %v", err)
					// 如果状态上报失败，可能是主机未准入，重置注册状态
					if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not approved") {
						ha.mutex.Lock()
						ha.isRegistered = false
						ha.mutex.Unlock()
						log.Println("Host not approved, will retry registration")
					}
				}
			}
		}
	}
}

func (ha *HostAgent) tryRegister() error {
	// 检查是否需要重新注册（距离上次注册超过1小时或从未注册）
	ha.mutex.RLock()
	needRegister := !ha.isRegistered || time.Since(ha.lastRegister) > time.Hour
	ha.mutex.RUnlock()

	if !needRegister {
		return nil
	}

	// 更新主机信息
	ha.updateHostInfo()

	response, err := ha.grpcAgent.Register(ha.ctx, ha.hostInfo)
	if err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("registration failed: %s", response.ErrorMessage)
	}

	ha.mutex.Lock()
	ha.isRegistered = true
	ha.lastRegister = time.Now()
	ha.mutex.Unlock()

	log.Printf("Host registered successfully (ID: %s)", response.AssignedId)
	return nil
}

func (ha *HostAgent) reportStatus() error {
	// 获取系统状态信息
	status := utils.GetSystemStatus()
	status.HostId = ha.hostInfo.Id

	// 添加自定义标签
	for k, v := range ha.config.Agent.Tags {
		status.CustomTags[k] = v
	}

	response, err := ha.grpcAgent.ReportStatus(ha.ctx, status)
	if err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("status report failed: %s", response.Message)
	}

	log.Printf("Status reported successfully for host: %s", status.HostId)
	return nil
}

func (ha *HostAgent) updateHostInfo() {
	ha.mutex.Lock()
	defer ha.mutex.Unlock()

	ha.hostInfo.LastSeen = time.Now().Unix()

	// 更新系统信息
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	ha.hostInfo.Tags["goroutines"] = fmt.Sprintf("%d", runtime.NumGoroutine())
	ha.hostInfo.Tags["memory_mb"] = fmt.Sprintf("%.2f", float64(m.Alloc)/1024/1024)
	ha.hostInfo.Tags["uptime"] = time.Since(ha.startTime).String()
	ha.hostInfo.Tags["cpu_count"] = fmt.Sprintf("%d", runtime.NumCPU())
}

func generateAgentID(configID string) string {
	if configID != "" {
		return configID
	}
	hostname := utils.GetHostname()
	return fmt.Sprintf("agent-%s-%d", hostname, time.Now().Unix())
}
