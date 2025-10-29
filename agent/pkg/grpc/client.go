package grpc

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"devops-manager/api/protobuf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Agent struct {
	serverAddr    string
	timeout       time.Duration
	retryInterval time.Duration
	conn          *grpc.ClientConn
	client        protobuf.HostServiceClient
	mutex         sync.RWMutex
	connected     bool
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewAgent(serverAddr string, timeout, retryInterval time.Duration) *Agent {
	return &Agent{
		serverAddr:    serverAddr,
		timeout:       timeout,
		retryInterval: retryInterval,
		connected:     false,
	}
}

func (c *Agent) Start(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)

	// 启动连接管理器
	go c.connectionManager()

	return nil
}

func (c *Agent) Stop() {
	if c.cancel != nil {
		c.cancel()
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connected = false
}

func (c *Agent) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

func (c *Agent) Register(ctx context.Context, hostInfo *protobuf.HostInfo) (*protobuf.RegisterResponse, error) {
	c.mutex.RLock()
	client := c.client
	c.mutex.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := client.Register(ctx, hostInfo)
	if err != nil {
		// 检查是否是连接错误
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Unavailable, codes.DeadlineExceeded:
				c.markDisconnected()
			}
		}
		return nil, err
	}

	return response, nil
}

func (c *Agent) ReportStatus(ctx context.Context, hostStatus *protobuf.HostStatus) (*protobuf.HostStatusResponse, error) {
	c.mutex.RLock()
	client := c.client
	c.mutex.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := client.ReportStatus(ctx, hostStatus)
	if err != nil {
		// 检查是否是连接错误
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Unavailable, codes.DeadlineExceeded:
				c.markDisconnected()
			}
		}
		return nil, err
	}

	return response, nil
}

func (c *Agent) connectionManager() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if !c.IsConnected() {
				log.Println("Attempting to connect to server...")
				if err := c.connect(); err != nil {
					log.Printf("Failed to connect: %v, retrying in %v", err, c.retryInterval)
					time.Sleep(c.retryInterval)
					continue
				}
				log.Println("Successfully connected to server")
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func (c *Agent) connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 关闭旧连接
	if c.conn != nil {
		c.conn.Close()
	}
	// 创建新连接
	conn, err := grpc.Dial(c.serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}

	c.conn = conn
	c.client = protobuf.NewHostServiceClient(conn)
	c.connected = true

	return nil
}

func (c *Agent) markDisconnected() {
	c.mutex.Lock()
	c.connected = false
	c.mutex.Unlock()
}
