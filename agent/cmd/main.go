package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"devops-manager/agent/pkg/config"
	"devops-manager/agent/pkg/controller"
	"devops-manager/agent/pkg/service"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

var (
	configPath = flag.String("config", "", "Path to configuration file")
	version    = flag.Bool("version", false, "Show version information")
	enableWeb  = flag.Bool("web", false, "Enable web interface and gRPC server")
	webPort    = flag.String("web-port", ":8081", "Web interface port")
	grpcPort   = flag.String("grpc-port", ":50052", "gRPC server port for receiving commands")
)

const (
	AppName    = "DevOps Manager Agent"
	AppVersion = "1.0.0"
)

func main() {
	flag.Parse()

	if *version {
		log.Printf("%s v%s", AppName, AppVersion)
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建主机代理服务
	hostAgent := service.NewHostAgent(cfg)

	if *enableWeb {
		// 启动带Web界面的模式
		startWithWeb(hostAgent)
	} else {
		// 启动简单模式（仅Agent客户端）
		startSimpleMode(hostAgent)
	}
}

// startSimpleMode 启动简单模式（仅Agent客户端）
func startSimpleMode(hostAgent *service.HostAgent) {
	if err := hostAgent.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	// 等待信号
	waitForSignal(hostAgent)
}

// startWithWeb 启动带Web界面的模式
func startWithWeb(hostAgent *service.HostAgent) {
	var wg sync.WaitGroup

	// 启动主机代理（连接到server）
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := hostAgent.Start(); err != nil {
			log.Fatalf("Failed to start host agent: %v", err)
		}
		hostAgent.Wait()
	}()

	// 启动gRPC服务器（接收server的命令）
	wg.Add(1)
	go func() {
		defer wg.Done()
		startGRPCServer(*grpcPort)
	}()

	// 启动HTTP Web服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		startHTTPServer(*webPort)
	}()

	// 等待信号
	go func() {
		waitForSignal(hostAgent)
		// 收到信号后，不需要等待其他goroutine
		os.Exit(0)
	}()

	log.Printf("%s v%s started successfully with Web interface", AppName, AppVersion)
	log.Printf("Web interface: http://localhost%s", *webPort)
	log.Printf("gRPC server: localhost%s", *grpcPort)

	wg.Wait()
}

func startGRPCServer(port string) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", port, err)
	}

	s := grpc.NewServer()

	// 注册gRPC服务
	grpcController := controller.NewGRPCController()
	grpcController.RegisterServices()

	log.Printf("Agent gRPC server listening on %s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func startHTTPServer(port string) {
	// 设置gin模式
	gin.SetMode(gin.ReleaseMode)

	httpController := controller.NewHTTPController()
	httpController.RegisterRoutes()

	router := httpController.GetRouter()

	log.Printf("Agent HTTP server listening on %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to serve HTTP: %v", err)
	}
}

func waitForSignal(hostAgent *service.HostAgent) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("%s v%s started successfully", AppName, AppVersion)

	sig := <-sigChan
	log.Printf("Received signal: %v", sig)

	log.Println("Shutting down...")
	hostAgent.Stop()
	hostAgent.Wait()
	log.Println("Agent stopped")
}
