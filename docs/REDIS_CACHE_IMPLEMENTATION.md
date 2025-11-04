# Redis 缓存实现文档

## 概述

本文档描述了任务下发执行系统中 Redis 缓存的实现，用于提升频繁查询的任务状态信息的性能。

## 实现的功能

### 1. 缓存的数据类型

- **任务状态缓存** (`task:status:*`): 缓存任务的详细状态信息，包括进度、主机统计等
- **任务进度缓存** (`task:progress:*`): 缓存任务执行进度和主机详情
- **任务统计缓存** (`task:stats:*`): 缓存全局任务统计信息
- **任务列表缓存** (`task:list:*`): 缓存分页的任务列表查询结果
- **主机任务缓存** (`host:tasks:*`): 缓存按主机筛选的任务列表
- **任务执行详情缓存** (`task:execution:*`): 缓存任务执行摘要信息

### 2. 缓存过期时间

- 任务状态缓存: 5分钟
- 任务进度缓存: 2分钟
- 任务统计缓存: 10分钟
- 任务列表缓存: 3分钟
- 主机任务缓存: 5分钟
- 任务执行详情缓存: 1分钟

### 3. 缓存失效策略

#### 自动失效
- 任务状态变化时自动使相关缓存失效
- 命令执行结果更新时使任务缓存失效
- 任务创建、启动、取消时使列表缓存失效

#### 手动失效
- 提供 API 接口手动清理所有缓存
- 支持按任务ID或主机ID清理特定缓存

## 核心组件

### TaskCacheService

负责所有缓存操作的核心服务类：

```go
type TaskCacheService struct {
    redis *redis.Client
    ctx   context.Context
}
```

#### 主要方法

- `CacheTaskStatus(taskID, status)`: 缓存任务状态
- `GetCachedTaskStatus(taskID)`: 获取缓存的任务状态
- `CacheTaskProgress(taskID, progress)`: 缓存任务进度
- `GetCachedTaskProgress(taskID)`: 获取缓存的任务进度
- `CacheTaskStatistics(stats)`: 缓存任务统计信息
- `GetCachedTaskStatistics()`: 获取缓存的任务统计信息
- `InvalidateTaskCache(taskID)`: 使任务相关缓存失效
- `InvalidateAllTaskCache()`: 使所有任务缓存失效

### TaskService 集成

TaskService 已集成缓存功能，主要变化：

1. **初始化时创建缓存服务实例**
2. **查询方法优先从缓存获取数据**
3. **数据变更时自动使相关缓存失效**
4. **异步缓存查询结果**

## 性能优化特性

### 1. 异步缓存
- 查询结果异步写入缓存，不阻塞主要业务流程
- 缓存失效操作异步执行

### 2. 智能缓存键生成
- 根据查询参数生成唯一缓存键
- 支持分页、筛选条件的缓存

### 3. 批量操作优化
- 支持批量缓存失效操作
- 模式匹配删除相关缓存

### 4. 缓存统计
- 提供缓存使用统计信息
- 监控缓存命中率和性能指标

## 使用示例

### 基本使用

```go
// 获取任务服务实例（已集成缓存）
taskService := GetTaskService()

// 获取任务状态（自动使用缓存）
status, err := taskService.GetTaskStatus("task-123")

// 获取任务列表（自动使用缓存）
tasks, total, err := taskService.GetTasks(1, 10, "running", "")

// 获取缓存统计信息
stats, err := taskService.GetCacheStatistics()
```

### 缓存管理

```go
// 手动清理所有缓存
err := taskService.InvalidateAllCache()

// 预热缓存
err := taskService.WarmupCache()

// 清理过期缓存
err := taskService.CleanupCache()
```

## 配置

### Redis 配置

在 `server/config/config.yaml` 中配置 Redis 连接：

```yaml
redis:
  host: "127.0.0.1"
  port: 6380
  password: ""
  db: 0
```

### 缓存配置

缓存相关配置在 `TaskCacheService` 中定义：

```go
const (
    TaskStatusCacheTTL    = 5 * time.Minute
    TaskProgressCacheTTL  = 2 * time.Minute
    TaskStatsCacheTTL     = 10 * time.Minute
    // ...
)
```

## 监控和维护

### 1. 缓存统计

通过 `GetCacheStatistics()` 方法获取：
- 各类缓存的数量
- Redis 内存使用情况
- 连接池状态

### 2. 自动清理

系统每30分钟自动执行缓存清理任务：
- 清理过期缓存
- 记录缓存统计信息

### 3. 日志监控

系统记录以下缓存相关日志：
- 缓存命中/未命中
- 缓存失效操作
- 缓存错误信息

## 性能影响

### 预期性能提升

- **任务状态查询**: 减少 80% 数据库查询
- **任务列表查询**: 减少 70% 数据库查询
- **统计信息查询**: 减少 90% 数据库查询
- **响应时间**: 平均减少 60-80%

### 内存使用

- 每个任务状态缓存: ~1-2KB
- 每个任务列表缓存: ~10-50KB
- 统计信息缓存: ~5-10KB
- 预计总内存使用: 100MB (10,000个活跃任务)

## 故障处理

### Redis 连接失败

- 系统自动降级到直接数据库查询
- 记录错误日志但不影响核心功能
- 连接恢复后自动重新启用缓存

### 缓存数据不一致

- 提供手动清理缓存的 API 接口
- 缓存有过期时间，自动更新
- 关键操作后自动使相关缓存失效

## 测试

### 单元测试

- `TestTaskCacheService_*`: 测试缓存服务的各项功能
- 覆盖缓存、获取、失效等核心操作

### 集成测试

- `TestTaskService_CacheIntegration_*`: 测试 TaskService 与缓存的集成
- 验证缓存命中、失效等场景

### 运行测试

```bash
# 运行缓存相关测试
go test ./server/pkg/service -run "TestTaskService_CacheIntegration|TestTaskCacheService" -v
```

## 注意事项

1. **Redis 依赖**: 确保 Redis 服务正常运行
2. **内存管理**: 监控 Redis 内存使用，适当调整缓存过期时间
3. **数据一致性**: 关键操作后及时使缓存失效
4. **网络延迟**: Redis 网络延迟可能影响缓存性能
5. **并发安全**: 缓存操作是并发安全的，但要注意业务逻辑的并发控制

## 未来优化

1. **缓存预热**: 系统启动时预加载热点数据
2. **智能过期**: 根据数据访问频率动态调整过期时间
3. **分布式缓存**: 支持 Redis 集群部署
4. **缓存压缩**: 对大数据进行压缩存储
5. **缓存分层**: 实现多级缓存策略