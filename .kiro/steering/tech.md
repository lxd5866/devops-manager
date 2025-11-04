# 技术栈与构建系统

## 后端技术栈

- **编程语言**: Go 1.24+
- **Web 框架**: Gin (HTTP 服务器)
- **RPC 框架**: gRPC 配合 Protocol Buffers
- **数据库**: MySQL 配合 GORM ORM
- **缓存**: Redis
- **文档**: Swagger/OpenAPI 配合 swaggo

## 前端技术栈

- **框架**: Vue 3 配合 Composition API
- **构建工具**: Vite
- **UI 组件库**: Ant Design Vue 4.0
- **状态管理**: Pinia
- **HTTP 客户端**: Axios
- **样式**: TailwindCSS
- **日期处理**: Day.js

## 开发依赖

- **Protocol Buffers**: protoc 编译器配合 Go 插件
- **代码生成**: swag 用于 Swagger 文档生成
- **代码检查**: 前端使用 ESLint，Go 使用标准工具

## 常用命令

### 后端开发
```bash
# 生成 Protocol Buffer 文件
bash scripts/generate_proto.sh

# 生成 Swagger 文档
swag init -g server/cmd/main.go -o docs

# 运行服务端
go run ./server/cmd/main.go

# 运行 Agent（简单模式）
go run ./agent/cmd/main.go

# 运行 Agent（带 Web 界面）
go run ./agent/cmd/main.go -web -web-port :8082 -grpc-port :50053
```

### 前端开发
```bash
# 安装依赖
cd server/web/frontend && npm install

# 开发服务器
npm run dev

# 生产构建
npm run build

# 预览生产构建
npm run preview

# 代码检查和修复
npm run lint
```

### 数据库设置
```bash
# 使用 schema 初始化数据库
mysql -u root -p devops_manager < server/sql/init.sql
```

## 配置文件

- 服务端配置: `server/config/config.yaml`
- Agent 配置: `agent/config/*.yaml`
- 前端配置: `server/web/frontend/vite.config.js`

## 构建要求

- Go 1.24+ 并启用模块支持
- Node.js 16+ 用于前端开发
- Protocol Buffers 编译器 (protoc)
- MySQL 8.0+
- Redis 6.0+