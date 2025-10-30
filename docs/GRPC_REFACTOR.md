# Server端gRPC控制器重构说明

## 重构目标
梳理server/pkg/controller下的gRPC控制器，删除多余方法，只保留与Agent通信相关的有效方法。

## 重构内容

### 1. grpc_host_controller.go
**保留的方法（与Agent通信相关）：**
- `Register()` - 处理Agent的主机注册请求
- `ReportStatus()` - 处理Agent的主机状态上报

**删除的方法（非Agent通信相关）：**
- `GetHost()` - 获取主机信息（内部查询方法，应由HTTP API提供）
- `GetAllHosts()` - 获取所有主机信息（内部查询方法，应由HTTP API提供）

**设计说明：**
- Agent作为客户端调用Server的gRPC服务
- Server接收Agent的注册和状态上报请求
- 查询类操作通过HTTP API提供，不需要gRPC接口

### 2. grpc_task_controller.go
**重构前问题：**
- 包含大量TODO方法，没有实际实现
- 方法定义不符合protobuf接口规范
- 缺少与Agent的实际通信逻辑

**重构后保留的方法：**
- `ConnectForCommands()` - 处理Agent的命令连接请求（双向流）
- `SendCommandToAgent()` - 向指定Agent发送命令（内部方法）

**删除的方法：**
- 所有TODO方法（CreateTask, GetTask, UpdateTask等）
- 这些方法应该在HTTP控制器中实现，用于Web界面和API调用

**设计说明：**
- 基于protobuf中的CommandService定义
- 实现Agent与Server的双向流通信
- Agent连接后保持长连接，接收Server发送的命令
- Server可以向单个或所有Agent发送命令执行请求

## 架构设计原则

### gRPC vs HTTP API的职责分工
- **gRPC**: 用于Agent与Server的长连接通信
  - 主机注册和状态上报
  - 命令执行的双向流通信
  - 实时性要求高的操作

- **HTTP API**: 用于Web界面和外部系统调用
  - 主机管理（查询、更新、删除）
  - 任务管理（创建、查询、更新）
  - 批量操作和复杂查询

### 通信流程
1. **Agent启动** → 调用`Register()`注册到Server
2. **Agent定期** → 调用`ReportStatus()`上报状态
3. **Agent连接** → 调用`ConnectForCommands()`建立命令执行长连接
4. **Server下发** → 通过`SendCommandToAgent()`发送命令给Agent
5. **Agent执行** → 返回命令执行结果给Server

## 文件变更总结

### server/pkg/controller/grpc_host_controller.go
- ✅ 保留 `Register()` - Agent主机注册
- ✅ 保留 `ReportStatus()` - Agent状态上报  
- ❌ 删除 `GetHost()` - 移至HTTP API
- ❌ 删除 `GetAllHosts()` - 移至HTTP API

### server/pkg/controller/grpc_task_controller.go
- ✅ 新增 `ConnectForCommands()` - Agent命令连接
- ✅ 新增 `SendCommandToAgent()` - 发送命令给Agent
- ❌ 删除所有TODO方法 - 移至HTTP API

## 后续工作
1. 确保HTTP控制器实现了被删除的查询和管理方法
2. 完善Agent连接管理和认证机制
3. 实现命令执行结果的持久化存储
4. 添加连接监控和故障恢复机制