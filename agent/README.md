# DevOps Manager Agent

DevOps Manager Agent是一个轻量级的客户端程序，部署在需要管理的服务器上，负责与中心服务器通信，执行任务和传输文件。

## 功能特性

### 🔗 连接管理
- 自动连接到中心服务器
- 断线重连机制
- 心跳保活
- 连接状态监控

### 📊 系统监控
- CPU使用率监控
- 内存使用情况
- 磁盘空间统计
- 系统负载信息
- 自定义标签支持

### ⚡ 任务执行
- 接收服务端下发的命令
- 跨平台命令执行（Windows/Linux/macOS）
- 超时控制
- 实时结果返回
- 任务状态跟踪

### 📁 文件传输
- 文件上传下载
- MD5完整性校验
- 文件信息查询
- 批量文件操作

### 🌐 Web界面
- 实时状态监控
- 任务管理界面
- 文件管理功能
- RESTful API接口

## 快速开始

### 1. 配置文件

编辑 `config/config.yaml`：

```yaml
server:
  address: "your-server:50051"  # 服务端地址
  timeout: 10s
  retry_interval: 5s

agent:
  report_interval: 30s
  client_id: ""                 # 留空自动生成
  tags:
    role: "web-server"
    env: "production"
    datacenter: "us-west-1"

logging:
  level: "info"
  format: "text"
```

### 2. 启动Agent

```bash
# 简单模式（仅Agent客户端）
go run ./cmd/main.go

# 完整模式（Agent + Web界面）
go run ./cmd/main.go -web

# 指定配置文件
go run ./cmd/main.go -config /path/to/config.yaml

# 自定义Web端口
go run ./cmd/main.go -web -web-port :8082 -grpc-port :50053

# 查看版本信息
go run ./cmd/main.go -version

# 查看帮助信息
go run ./cmd/main.go -help
```

#### 启动模式说明

**简单模式**：
- 仅启动Agent客户端
- 连接到Server进行状态上报
- 不提供Web界面和本地gRPC服务
- 适合生产环境的轻量级部署

**完整模式**：
- 启动Agent客户端 + Web界面 + 本地gRPC服务
- 提供完整的管理功能
- 适合开发环境和需要本地管理的场景

### 3. 访问Web界面

Agent启动后，可以通过以下地址访问Web界面：

- 主页：http://localhost:8081/
- 状态监控：http://localhost:8081/status
- 任务管理：http://localhost:8081/tasks
- 文件管理：http://localhost:8081/files

## API接口

### 主机信息
- `GET /api/v1/host/info` - 获取主机信息
- `GET /api/v1/host/status` - 获取主机状态
- `POST /api/v1/host/update` - 更新主机信息

### 任务管理
- `POST /api/v1/task/execute` - 执行任务
- `GET /api/v1/task/status/:id` - 获取任务状态
- `POST /api/v1/task/cancel/:id` - 取消任务
- `GET /api/v1/task/list` - 获取任务列表

### 文件操作
- `POST /api/v1/file/upload` - 上传文件
- `GET /api/v1/file/download/:name` - 下载文件
- `GET /api/v1/file/list` - 列出文件
- `DELETE /api/v1/file/:name` - 删除文件
- `GET /api/v1/file/info/:name` - 获取文件信息

## 目录结构

```
agent/
├── cmd/                    # 启动入口
│   └── main.go
├── config/                 # 配置文件
│   └── config.yaml
├── pkg/                    # 核心代码
│   ├── controller/         # 控制器层
│   │   ├── grpc_*.go      # gRPC业务控制器
│   │   ├── http_*.go      # HTTP业务控制器
│   │   └── web_controller.go
│   ├── service/           # 服务层
│   │   ├── host_service.go
│   │   ├── task_service.go
│   │   ├── file_service.go
│   │   └── connection_service.go
│   ├── utils/             # 工具类
│   │   ├── system.go
│   │   ├── file.go
│   │   └── command.go
│   ├── config/            # 配置管理
│   │   └── config.go
│   └── grpc/              # gRPC客户端
│       └── client.go
└── web/                   # Web界面
    ├── static/            # 静态资源
    │   ├── css/
    │   ├── js/
    │   └── images/
    └── templates/         # HTML模板
        ├── index.html
        ├── status.html
        ├── tasks.html
        └── files.html
```

## 安全注意事项

1. **命令执行安全**：Agent会验证命令的安全性，拒绝执行危险命令
2. **文件传输安全**：所有文件传输都会进行MD5校验
3. **连接安全**：建议在生产环境中使用TLS加密连接
4. **权限控制**：Agent以当前用户权限运行，请合理配置用户权限

## 故障排除

### 连接问题
- 检查服务端地址和端口是否正确
- 确认网络连通性
- 查看防火墙设置

### 任务执行失败
- 检查命令语法是否正确
- 确认执行权限
- 查看超时设置

### 文件传输问题
- 检查磁盘空间
- 确认文件权限
- 验证文件路径

## 日志查看

Agent的日志输出包含以下信息：
- 连接状态变化
- 任务执行记录
- 文件传输日志
- 错误和警告信息

可以通过调整配置文件中的 `logging.level` 来控制日志详细程度。

## 性能优化

1. **合理设置上报间隔**：根据监控需求调整 `report_interval`
2. **控制并发任务数**：避免同时执行过多任务
3. **定期清理文件**：清理上传下载目录中的临时文件
4. **监控资源使用**：关注Agent本身的CPU和内存使用情况