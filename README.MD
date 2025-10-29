# DevOps Manager - 分布式运维管理系统

📖 项目架构设计文档

## 1. 项目目标
- 支持 10000+ 客户端节点 并发连接与控制
- 实现 agent → server 长连接心跳，保证上下线可见
- 支持 批量下发任务（命令执行 / 文件分发 / 脚本运行）
- 支持 节点分组 / 标签管理
- 提供 RESTful API / Web 界面 供用户远程操作
- 集成 Swagger API 文档，支持在线测试和调试
- 高可用 & 可扩展
- server 前端使用 tailwindcss ，后端使用 golang gin，client 使用golang 

## 2. 整体架构图（逻辑）

```
+---------------+       +------------------+       +------------------+
|   agent 节点  | <---> |   Server         | <---> |   Web / API 接口  |
| (1w+ 机器)     |       | (接入 & 任务分发)  |       | (UI & 管理平台)   |
+---------------+       +------------------+       +------------------+
        |                        |
        v                        v
+----------------+       +------------------+
|   Agent 模块   |       |   Storage (DB)   |
| - 心跳         |       | - 节点信息       |
| - 命令执行     |        | - 任务记录       |
| - 文件传输     |        | - 日志存储       |
| - web 页面    |        |                 |
+----------------+       +------------------+
```

## 3. 项目结构
```
├── agent/                    # Agent客户端
│   ├── cmd/
│   │   └── main.go          # Agent启动入口
│   ├── config/              # 配置文件
│   │   ├── config.yaml      # 默认配置
│   │   └── *.yaml           # 其他环境配置
│   ├── pkg/
│   │   ├── controller/      # HTTP/gRPC控制器
│   │   │   ├── grpc_controller.go        # gRPC基础逻辑，不实现业务逻辑
│   │   │   ├── grpc_host_controller.go   # gRPC主机相关业务逻辑
│   │   │   ├── grpc_task_controller.go   # gRPC任务相关业务逻辑
│   │   │   ├── grpc_file_controller.go   # gRPC文件传输业务逻辑
│   │   │   ├── http_controller.go        # HTTP基础逻辑，不实现业务逻辑
│   │   │   ├── http_host_controller.go   # HTTP主机相关业务逻辑
│   │   │   ├── http_task_controller.go   # HTTP任务相关业务逻辑
│   │   │   └── web_controller.go         # Web页面控制器
│   │   ├── service/         # 业务服务层
│   │   │   ├── host_service.go           # 主机服务
│   │   │   ├── task_service.go           # 任务执行服务
│   │   │   ├── file_service.go           # 文件传输服务
│   │   │   └── connection_service.go     # 连接管理服务
│   │   ├── utils/           # 工具类
│   │   │   ├── system.go                 # 系统信息工具
│   │   │   ├── file.go                   # 文件操作工具
│   │   │   └── command.go                # 命令执行工具
│   │   └── config/          # 配置管理
│   │       └── config.go                 # 配置加载逻辑
│   └── web/                 # Agent Web页面
│       ├── static/          # 静态资源
│       └── templates/       # 页面模板
├── server/                   # Server服务端
│   ├── cmd/
│   │   └── main.go          # Server启动入口
│   ├── config/              # 配置文件
│   │   └── config.yaml      # 服务端配置
│   ├── pkg/
│   │   ├── controller/      # HTTP/gRPC控制器
│   │   │   ├── grpc_controller.go        # gRPC基础逻辑，不实现业务逻辑
│   │   │   ├── grpc_host_controller.go   # gRPC主机管理业务逻辑
│   │   │   ├── grpc_task_controller.go   # gRPC任务管理业务逻辑
│   │   │   ├── http_controller.go        # HTTP基础逻辑，不实现业务逻辑
│   │   │   ├── http_host_controller.go   # HTTP主机管理业务逻辑
│   │   │   ├── http_task_controller.go   # HTTP任务管理业务逻辑
│   │   │   └── web_controller.go         # Web管理界面控制器
│   │   ├── service/         # 业务服务层
│   │   │   ├── host_service.go           # 主机管理服务
│   │   │   ├── task_service.go           # 任务调度服务
│   │   │   └── user_service.go           # 用户管理服务
│   │   └── database/        # 数据库连接
│   │       ├── mysql.go                  # MySQL连接
│   │       └── redis.go                  # Redis连接
│   └── web/                 # Server Web管理界面
│       ├── frontend/        # 前端项目（TailwindCSS）
│       ├── static/          # 静态资源
│       └── templates/       # 页面模板
├── api/                      # 共享API定义
│   ├── models/              # 数据模型
│   │   ├── host.go                       # 主机模型
│   │   ├── task.go                       # 任务模型
│   │   └── user.go                       # 用户模型
│   ├── protobuf/            # 生成的protobuf代码
│   └── protobuf_base/       # protobuf定义文件
│       ├── host.proto                    # 主机相关协议
│       └── command.proto                 # 命令相关协议
├── docs/                    # 文档目录
│   ├── docs.go                          # Swagger文档入口
│   ├── swagger.json                     # Swagger JSON文档
│   ├── swagger.yaml                     # Swagger YAML文档
│   └── API_TESTING.md                   # API测试文档
├── scripts/                 # 脚本文件
│   ├── generate_proto.sh                # protobuf生成脚本
│   └── generate_swagger.sh              # Swagger文档生成脚本
└── examples/                # 示例代码
    └── command_client_example.go         # 客户端示例
```

## 4. 模块划分
### 4.1 Agent (客户端)

#### 部署在被控服务器，负责：
1. 长连接管理：使用 WebSocket 长连接 server 端，创建web terminal 
2. 心跳上报：定时上报状态（CPU、内存、磁盘、负载等可扩展）
3. Task 执行：接受 server 的任务并执行，返回结果
4. 文件传输：可支持 server 下发文件 / 上传结果文件


### 4.2 Server (控制端)
运行在中心机房或云上，负责：
- agent 管理：使用 grpc 长连接与 agent 连接，记录节点上下线、分组、标签、资源情况
- 任务调度：接收管理员下发的批量任务，分发给节点,统一收集 agent 执行结果，并记录


### 4.3 Web / API 管理端

#### 提供用户交互入口：
- 节点视图（在线 / 离线 / 标签 / 分组）
- 批量任务下发（命令 / 文件分发 / 脚本运行）
- 历史任务查询（成功率、执行时间、日志）
- 监控告警（心跳丢失报警）

### 4.4 数据存储层
- 数据库（MySQL）：存储节点信息、任务记录、日志
- 缓存（Redis）：存储节点状态、心跳数据、任务队列
#### 表设计（简化）：
- nodes（节点表：id、ip、hostname、status、last_seen、tags）
- tasks（任务表：id、command、status、create_time）
- task_results（任务结果表：task_id、node_id、output、status）


## 5. 快速开始

### 5.1 环境依赖

- Go 1.21+
- MySQL 实例（在 `config.yaml` 中配置 DSN）

### 5.2 运行步骤

1. 根据实际环境修改配置文件：
   - 服务端配置：`server/config/config.yaml`
   - 客户端配置：`agent/config/config.yaml`

2. 生成protobuf文件：
   ```bash
   bash scripts/generate_proto.sh
   ```

3. 生成Swagger文档：
   ```bash
   # 安装swag工具
   go install github.com/swaggo/swag/cmd/swag@latest
   
   # 生成文档
   swag init -g server/cmd/main.go -o docs
   ```

4. 启动服务端：
   ```bash
   go run ./server/cmd/main.go
   ```

5. 启动客户端：
   
   **简单模式（仅Agent客户端）：**
   ```bash
   go run ./agent/cmd/main.go
   ```
   
   **完整模式（Agent + Web界面）：**
   ```bash
   go run ./agent/cmd/main.go -web
   ```
   
   **自定义端口：**
   ```bash
   go run ./agent/cmd/main.go -web -web-port :8082 -grpc-port :50053
   ```

6. 访问Web界面：
   - 服务端管理界面：http://localhost:3000
   - 服务端API文档：http://localhost:8080/swagger/index.html
   - 客户端状态界面：http://localhost:8081 (启用-web参数时)

## 6. Agent详细说明

### 6.1 Agent架构
Agent采用模块化设计，主要包含以下组件：

#### 控制器层 (Controller)
- **grpc_controller.go**: gRPC基础框架，负责服务注册和路由
- **grpc_host_controller.go**: 主机相关gRPC业务逻辑
- **grpc_task_controller.go**: 任务执行gRPC业务逻辑  
- **grpc_file_controller.go**: 文件传输gRPC业务逻辑
- **http_controller.go**: HTTP基础框架，负责路由注册
- **http_host_controller.go**: 主机相关HTTP API
- **http_task_controller.go**: 任务管理HTTP API
- **http_file_controller.go**: 文件管理HTTP API
- **web_controller.go**: Web页面控制器

#### 服务层 (Service)
- **host_service.go**: 主机注册、状态上报、连接管理
- **task_service.go**: 任务执行、状态跟踪、结果收集
- **file_service.go**: 文件上传下载、完整性验证
- **connection_service.go**: gRPC连接管理、心跳维护

#### 工具层 (Utils)
- **system.go**: 系统信息收集（CPU、内存、磁盘等）
- **file.go**: 文件操作工具（复制、MD5校验等）
- **command.go**: 命令执行工具（跨平台shell执行）

### 6.2 Agent功能特性

#### 主机管理
- 自动注册到服务端
- 定期上报系统状态
- 支持标签和分组管理
- 连接状态监控

#### 任务执行
- 接收服务端下发的命令
- 支持超时控制
- 实时返回执行结果
- 任务状态跟踪

#### 文件传输
- 支持文件上传下载
- MD5完整性校验
- 文件信息查询
- 目录管理

#### Web界面
- 实时状态监控
- 任务执行历史
- 文件管理界面
- RESTful API接口

### 6.3 配置说明

Agent配置文件 `agent/config/config.yaml`：

```yaml
server:
  address: "localhost:50051"    # 服务端gRPC地址
  timeout: 10s                  # 请求超时时间
  retry_interval: 5s            # 重连间隔

agent:
  report_interval: 30s          # 状态上报间隔
  client_id: ""                 # 客户端ID（空则自动生成）
  tags:                         # 自定义标签
    role: "agent"
    env: "production"
    version: "1.0.0"

logging:
  level: "info"                 # 日志级别
  format: "text"                # 日志格式
```

## 7. Swagger API文档

### 7.1 Swagger UI 访问

系统集成了完整的Swagger API文档，提供在线测试和调试功能：

- **Swagger UI界面**: http://localhost:8080/swagger/index.html
- **JSON格式文档**: http://localhost:8080/swagger/doc.json
- **YAML格式文档**: http://localhost:8080/swagger/swagger.yaml

### 7.2 API接口概览

#### 主机管理 API
| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/hosts` | 获取所有已准入主机列表 |
| POST | `/api/v1/hosts/register` | 注册新主机到系统 |
| GET | `/api/v1/hosts/{id}` | 获取指定主机详细信息 |
| PUT | `/api/v1/hosts/{id}` | 更新主机信息 |
| DELETE | `/api/v1/hosts/{id}` | 删除主机 |
| POST | `/api/v1/hosts/{id}/status` | 上报主机状态 |
| GET | `/api/v1/hosts/{id}/status` | 获取主机状态 |

#### 待准入主机管理 API
| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/pending-hosts` | 获取待准入主机列表 |
| GET | `/api/v1/pending-hosts/count` | 获取待准入主机数量 |
| POST | `/api/v1/pending-hosts/{id}/approve` | 准入指定主机 |
| POST | `/api/v1/pending-hosts/{id}/reject` | 拒绝指定主机 |

#### 任务管理 API
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/tasks` | 创建新任务 |
| GET | `/api/v1/tasks` | 获取任务列表（支持分页筛选） |
| GET | `/api/v1/tasks/{id}` | 获取任务详细信息 |
| PUT | `/api/v1/tasks/{id}` | 更新任务信息 |
| GET | `/api/v1/tasks/{id}/commands` | 获取任务的命令执行记录 |

### 7.3 API请求示例

#### 注册主机
```bash
curl -X POST "http://localhost:8080/api/v1/hosts/register" \
     -H "Content-Type: application/json" \
     -d '{
       "hostname": "web-server-01",
       "ip": "192.168.1.100",
       "os": "linux",
       "tags": {
         "role": "web",
         "env": "production"
       }
     }'
```

#### 创建任务
```bash
curl -X POST "http://localhost:8080/api/v1/tasks" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "部署应用",
       "description": "在生产环境部署新版本",
       "host_ids": ["host-001", "host-002"],
       "command": "bash /opt/deploy.sh",
       "timeout": 300,
       "parameters": {
         "version": "1.2.3",
         "env": "production"
       }
     }'
```

#### 获取主机列表
```bash
curl -X GET "http://localhost:8080/api/v1/hosts" \
     -H "accept: application/json"
```

### 7.4 响应格式

#### 成功响应
```json
{
  "success": true,
  "data": {
    // 实际数据内容
  },
  "message": "操作成功"
}
```

#### 错误响应
```json
{
  "success": false,
  "error_message": "具体错误信息",
  "message": "请求处理失败"
}
```

### 7.5 Swagger文档生成

#### 安装swag工具
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

#### 生成文档
```bash
# 在项目根目录执行
swag init -g server/cmd/main.go -o docs

# 或使用脚本
bash scripts/generate_swagger.sh
```

#### 添加API注释
在控制器方法上添加Swagger注释：

```go
// CreateTask 创建任务
// @Summary      创建新的执行任务
// @Description  在指定主机上创建并执行任务
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        task  body      models.CreateTaskRequest  true  "任务信息"
// @Success      200   {object}  models.APIResponse{data=models.Task}
// @Failure      400   {object}  models.APIResponse
// @Failure      500   {object}  models.APIResponse
// @Router       /api/v1/tasks [post]
func (c *HTTPTaskController) CreateTask(ctx *gin.Context) {
    // 实现逻辑
}
```

### 7.6 在线测试

1. 启动服务端后，访问 Swagger UI：http://localhost:8080/swagger/index.html
2. 选择要测试的API接口
3. 点击 "Try it out" 按钮
4. 填写必要的请求参数
5. 点击 "Execute" 执行请求
6. 查看响应结果和状态码

### 7.7 开发注意事项

1. **文档同步**: 修改API后需要重新生成Swagger文档
2. **注释规范**: 严格按照swag注释格式编写
3. **模型定义**: 确保所有请求/响应模型都有完整的结构体定义
4. **错误处理**: 统一使用标准的错误响应格式
5. **版本管理**: API版本变更时需要更新文档版本信息

详细的API测试文档和示例请参考：[docs/API_TESTING.md](docs/API_TESTING.md)



        






