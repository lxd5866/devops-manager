package main

import (
	"log"
	"net"
	"sync"

	"devops-manager/server/pkg/config"
	"devops-manager/server/pkg/controller"
	"devops-manager/server/pkg/database"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"

	// Swagger imports
	_ "devops-manager/docs"
)

// @title           DevOps Manager API
// @version         1.0
// @description     DevOps Manager 分布式运维管理系统API文档，支持主机管理、任务调度、文件传输等功能
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.basic  BasicAuth

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库连接
	if err := database.InitMySQL(&cfg.MySQL); err != nil {
		log.Fatalf("Failed to initialize MySQL: %v", err)
	}
	defer database.CloseMySQL()

	// 初始化 Redis 连接
	if err := database.InitRedis(&cfg.Redis); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer database.CloseRedis()

	var wg sync.WaitGroup

	// 启动 gRPC 服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		startGRPCServer(cfg)
	}()

	// 启动 HTTP 服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		startHTTPServer(cfg)
	}()

	wg.Wait()
}

func startGRPCServer(cfg *config.Config) {
	lis, err := net.Listen("tcp", cfg.GRPC.Address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", cfg.GRPC.Address, err)
	}

	s := grpc.NewServer()

	// 注册所有 gRPC 服务并获取任务控制器
	taskController := controller.RegisterGRPCServices(s)

	// 设置任务分发器，建立 TaskService 和 gRPC 控制器的连接
	controller.SetupTaskDispatcher(taskController)

	log.Printf("gRPC server listening on %s", cfg.GRPC.Address)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func startHTTPServer(cfg *config.Config) {
	r := gin.Default()

	// 注册 API 路由
	controller.RegisterHTTPRoutes(r)

	// 注册 Web 路由
	controller.RegisterWebRoutes(r)

	// 注册 Swagger 路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Printf("HTTP server listening on %s", cfg.HTTP.Address)
	log.Printf("Swagger UI available at: http://localhost%s/swagger/index.html", cfg.HTTP.Address)
	if err := r.Run(cfg.HTTP.Address); err != nil {
		log.Fatalf("Failed to serve HTTP: %v", err)
	}
}
