# 项目结构与组织

## 根目录组织

```
├── agent/          # Agent 客户端应用
├── server/         # Server 控制平面
├── api/            # 共享 API 定义和模型
├── docs/           # 文档和 Swagger 规范
├── scripts/        # 构建和工具脚本
└── .kiro/          # Kiro IDE 配置
```

## 架构模式

### 分层架构
Agent 和 Server 都遵循一致的 3 层模式：
- **控制器层**: HTTP/gRPC 请求处理 (`pkg/controller/`)
- **服务层**: 业务逻辑实现 (`pkg/service/`)
- **工具层**: 通用函数和辅助工具 (`pkg/utils/`)

### 控制器命名约定
控制器按协议和职责分离：
- `*_controller.go`: 基础框架设置，不包含业务逻辑
- `*_host_controller.go`: 主机管理业务逻辑
- `*_task_controller.go`: 任务执行业务逻辑
- `*_file_controller.go`: 文件传输业务逻辑
- `web_controller.go`: Web 界面控制器

## 关键目录

### Agent 结构
```
agent/
├── cmd/main.go                    # 入口点，包含 CLI 参数
├── config/*.yaml                  # 环境配置文件
├── pkg/
│   ├── controller/               # HTTP/gRPC 处理器
│   ├── service/                  # 核心 Agent 服务
│   ├── utils/                    # 系统工具
│   └── config/                   # 配置管理
└── web/                          # Agent Web 界面
    ├── static/                   # CSS/JS 资源
    └── templates/                # HTML 模板
```

### Server 结构
```
server/
├── cmd/main.go                   # 入口点，包含 Swagger 设置
├── config/config.yaml            # 服务端配置
├── pkg/
│   ├── controller/              # API 控制器
│   ├── service/                 # 业务服务
│   ├── database/                # 数据库连接
│   └── models/                  # 响应模型
├── sql/init.sql                 # 数据库 schema
└── web/
    ├── frontend/                # Vue.js 应用
    ├── static/                  # 静态资源
    └── templates/               # 服务端模板
```

### 共享 API 结构
```
api/
├── models/                      # Go 数据模型
├── protobuf/                    # 生成的 protobuf 代码
└── protobuf_base/               # Proto 定义文件
```

## 文件命名约定

- **Go 文件**: snake_case 配合描述性后缀
- **配置文件**: 环境特定的 YAML 文件
- **Proto 文件**: 小写配合 .proto 扩展名
- **SQL 文件**: 描述性名称 (init.sql, migrations)
- **前端**: 组件使用 kebab-case，JS 使用 camelCase

## 配置管理

- **Server**: 单个 `config.yaml` 包含环境配置段
- **Agent**: 多个环境特定配置文件
- **Frontend**: Vite 配置配合环境变量
- **Database**: Schema 位于 `server/sql/init.sql`

## 代码组织规则

1. **关注点分离**: 控制器处理请求，服务包含业务逻辑
2. **共享模型**: 通用数据结构位于 `api/models/`
3. **Protocol Buffers**: 定义在 `api/protobuf_base/`，生成代码在 `api/protobuf/`
4. **Web 资源**: 静态文件与模板分离
5. **文档**: 控制器中的 Swagger 注解，主要目录中的 README 文件