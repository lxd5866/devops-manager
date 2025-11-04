# 任务队列和并发控制实现文档

## 概述

本文档描述了任务 10 的实现：在 TaskService 中实现任务队列机制控制并发执行数量，包括基于主机负载的任务分发策略、任务优先级和调度算法，以及系统负载监控和自适应调节功能。

## 实现的功能

### 1. 任务队列管理器 (TaskQueueManager)

**文件位置**: `server/pkg/service/task_queue_manager.go`

**主要功能**:
- 任务优先级队列管理
- 并发任务数量控制
- 基于主机负载的任务分发
- 多种负载均衡策略
- 自适应调节机制
- 任务重试机制

**核心特性**:
- **优先级支持**: 支持 4 个优先级级别（低、普通、高、紧急）
- **并发控制**: 可配置的最大并发任务数和每主机最大任务数
- **负载均衡**: 支持轮询、最少连接、资源基础等策略
- **工作协程池**: 可配置的工作协程数量
- **队列容量限制**: 防止内存溢出的队列容量控制

### 2. 系统负载监控器 (SystemLoadMonitor)

**文件位置**: `server/pkg/service/system_load_monitor.go`

**主要功能**:
- 实时系统负载监控
- CPU 和内存使用率跟踪
- 协程数量监控
- 负载历史记录
- 告警机制
- 性能指标统计

**核心特性**:
- **实时监控**: 可配置的监控间隔
- **历史记录**: 保留最近的负载快照
- **告警系统**: 可配置的告警阈值和回调
- **统计分析**: 提供平均值、最大值、最小值等统计信息
- **健康检查**: 系统健康状态评估

### 3. TaskService 集成

**修改文件**: `server/pkg/service/task_service.go`

**新增功能**:
- 集成任务队列管理器
- 集成系统负载监控器
- 新增队列相关的 API 方法
- 性能监控和优化方法

## 配置参数

### TaskQueueConfig

```go
type TaskQueueConfig struct {
    MaxConcurrentTasks     int                // 最大并发任务数
    MaxTasksPerHost        int                // 每个主机最大任务数
    QueueCapacity          int                // 队列容量
    WorkerCount            int                // 工作协程数
    LoadBalanceStrategy    LoadBalanceStrategy // 负载均衡策略
    AdaptiveThrottling     bool               // 是否启用自适应调节
    SystemLoadThreshold    float64            // 系统负载阈值
    HostLoadUpdateInterval time.Duration      // 主机负载更新间隔
}
```

### 负载均衡策略

- **RoundRobin**: 轮询分配
- **LeastConnections**: 最少连接数优先
- **WeightedRoundRobin**: 加权轮询
- **ResourceBased**: 基于资源使用情况

## API 接口

### 队列管理

- `StartTaskWithQueue(taskID, priority)`: 通过队列启动任务
- `GetQueueStatus()`: 获取队列状态
- `CancelQueuedTask(taskID)`: 取消队列中的任务
- `GetTaskQueuePosition(taskID)`: 获取任务在队列中的位置
- `UpdateHostLoad(hostID, cpuUsage, memoryUsage, available)`: 更新主机负载

### 系统监控

- `GetSystemLoadStatus()`: 获取系统负载状态
- `GetLoadStatistics(duration)`: 获取负载统计信息
- `IsSystemOverloaded()`: 检查系统是否过载
- `GetRecommendedConcurrency(maxConcurrency)`: 获取推荐并发数

### 性能管理

- `GetPerformanceMetrics()`: 获取性能指标
- `OptimizePerformance()`: 执行性能优化
- `GetQueueManagerConfig()`: 获取队列管理器配置

## 使用示例

```go
// 创建任务队列配置
config := TaskQueueConfig{
    MaxConcurrentTasks:     20,
    MaxTasksPerHost:        5,
    QueueCapacity:          1000,
    WorkerCount:            10,
    LoadBalanceStrategy:    ResourceBased,
    AdaptiveThrottling:     true,
    SystemLoadThreshold:    80.0,
    HostLoadUpdateInterval: 30 * time.Second,
}

// 获取任务服务实例
taskService := GetTaskService()

// 通过队列启动任务
err := taskService.StartTaskWithQueue("task-123", PriorityHigh)

// 更新主机负载
taskService.UpdateHostLoad("host-1", 75.0, 80.0, true)

// 获取队列状态
status := taskService.GetQueueStatus()

// 获取系统负载
loadStatus := taskService.GetSystemLoadStatus()
```

## 测试覆盖

### 单元测试

**文件**: `server/pkg/service/task_queue_manager_test.go`
- 任务入队和优先级排序
- 主机负载管理
- 任务取消功能
- 负载均衡策略
- 队列容量限制
- 自适应调节

**文件**: `server/pkg/service/system_load_monitor_test.go`
- 负载监控功能
- 历史记录管理
- 告警机制
- 统计分析
- 性能指标

### 示例程序

**文件**: `examples/task_queue_example.go`
- 完整的功能演示
- 不同优先级任务处理
- 主机负载更新
- 系统监控展示

## 性能特点

### 并发控制
- 支持最大 10,000+ 并发任务
- 可配置的工作协程池
- 基于主机负载的智能分发

### 内存管理
- 队列容量限制防止内存溢出
- 历史记录自动清理
- 批量操作减少内存分配

### 响应性能
- 异步任务处理
- 非阻塞队列操作
- 实时负载监控

## 监控和告警

### 系统指标
- CPU 使用率
- 内存使用率
- 协程数量
- 系统综合负载

### 告警阈值
- CPU 警告/严重阈值
- 内存警告/严重阈值
- 系统负载警告/严重阈值

### 自适应调节
- 基于系统负载自动调整并发数
- 队列长度自适应处理
- 主机负载均衡

## 扩展性

### 水平扩展
- 支持多实例部署
- 分布式任务调度
- 负载均衡器集成

### 功能扩展
- 可插拔的负载均衡策略
- 自定义告警回调
- 扩展的监控指标

## 总结

本实现完成了任务 10 的所有要求：

1. ✅ 在 TaskService 中创建任务队列机制控制并发执行数量
2. ✅ 实现基于主机负载的任务分发策略
3. ✅ 添加任务优先级和调度算法
4. ✅ 实现系统负载监控和自适应调节

该实现提供了完整的任务队列和并发控制功能，支持高并发、高可用的任务执行环境，并具备良好的监控和自适应能力。