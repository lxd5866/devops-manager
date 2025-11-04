# 任务下发执行系统性能优化建议

## 性能测试结果

基于基准测试的结果，以下是系统的性能表现和优化建议：

### 1. 任务创建性能

**当前性能：**
- 任务创建：~30,590 ns/op (约 0.03ms)
- 内存分配：10,728 B/op，124 allocs/op

**优化建议：**
- 使用对象池减少内存分配
- 批量创建任务以减少数据库往返
- 预编译 SQL 语句

### 2. 任务查询性能

**当前性能：**
- 任务查询：~105,850 ns/op (约 0.1ms)
- 内存分配：24,129 B/op，659 allocs/op

**优化建议：**
- 添加数据库索引优化查询
- 实现查询结果缓存
- 使用分页查询减少内存使用

### 3. 数据库优化

**索引建议：**
```sql
-- 任务表索引
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_created_by ON tasks(created_by);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);

-- 命令表索引
CREATE INDEX idx_commands_task_id ON commands(task_id);
CREATE INDEX idx_commands_host_id ON commands(host_id);
CREATE INDEX idx_commands_status ON commands(status);

-- 命令主机表索引
CREATE INDEX idx_command_hosts_status ON commands_hosts(status);
CREATE INDEX idx_command_hosts_host_id ON commands_hosts(host_id);
CREATE INDEX idx_command_hosts_created_at ON commands_hosts(created_at);

-- 复合索引
CREATE INDEX idx_commands_task_status ON commands(task_id, status);
CREATE INDEX idx_command_hosts_command_status ON commands_hosts(command_id, status);
```

### 4. 缓存策略

**Redis 缓存配置：**
- 任务状态缓存：TTL 30秒
- 任务列表缓存：TTL 60秒
- 统计信息缓存：TTL 300秒

**缓存键设计：**
```
task:status:{task_id}
task:list:{page}:{size}:{status}:{name}
task:stats:global
host:tasks:{host_id}:{page}:{size}
```

### 5. 连接池优化

**数据库连接池配置：**
```go
// 最大空闲连接数
sqlDB.SetMaxIdleConns(20)
// 最大打开连接数
sqlDB.SetMaxOpenConns(100)
// 连接最大生存时间
sqlDB.SetConnMaxLifetime(time.Hour)
// 连接最大空闲时间
sqlDB.SetConnMaxIdleTime(30 * time.Minute)
```

### 6. 批量操作优化

**批量插入配置：**
- 批量大小：100-500 条记录
- 使用事务包装批量操作
- 实现批量更新机制

**示例代码：**
```go
// 批量创建任务
func (ts *TaskService) CreateTasksBatch(tasks []models.Task) error {
    return ts.db.CreateInBatches(tasks, 100).Error
}

// 批量更新状态
func (ts *TaskService) UpdateCommandStatusBatch(updates []CommandStatusUpdate) error {
    return ts.db.Transaction(func(tx *gorm.DB) error {
        for _, update := range updates {
            err := tx.Model(&models.Command{}).
                Where("command_id = ?", update.CommandID).
                Updates(update.Fields).Error
            if err != nil {
                return err
            }
        }
        return nil
    })
}
```

### 7. 内存优化

**内存使用优化：**
- 实现对象池减少 GC 压力
- 限制查询结果集大小
- 定期清理过期数据

**对象池示例：**
```go
var taskPool = sync.Pool{
    New: func() interface{} {
        return &models.Task{}
    },
}

func GetTask() *models.Task {
    return taskPool.Get().(*models.Task)
}

func PutTask(task *models.Task) {
    // 重置对象状态
    *task = models.Task{}
    taskPool.Put(task)
}
```

### 8. 并发控制

**并发限制配置：**
- 最大并发任务数：100
- 每个主机最大并发数：5
- 队列容量：1000

**负载均衡策略：**
- 基于主机负载的任务分发
- 自适应并发控制
- 系统负载监控

### 9. 监控和指标

**关键性能指标：**
- 任务创建延迟：< 50ms (P95)
- 任务查询延迟：< 100ms (P95)
- 数据库连接使用率：< 80%
- 内存使用增长：< 10MB/hour
- CPU 使用率：< 70%

**监控实现：**
```go
// 性能指标收集
type PerformanceMetrics struct {
    TaskCreationLatency   time.Duration
    TaskQueryLatency      time.Duration
    DatabaseConnections   int
    MemoryUsage          int64
    CPUUsage             float64
}

func (ts *TaskService) CollectMetrics() *PerformanceMetrics {
    return &PerformanceMetrics{
        TaskCreationLatency: ts.getAvgTaskCreationLatency(),
        TaskQueryLatency:    ts.getAvgTaskQueryLatency(),
        DatabaseConnections: ts.getDatabaseConnections(),
        MemoryUsage:        ts.getMemoryUsage(),
        CPUUsage:           ts.getCPUUsage(),
    }
}
```

### 10. 系统调优

**操作系统级优化：**
- 增加文件描述符限制：`ulimit -n 65536`
- 调整 TCP 参数优化网络性能
- 配置适当的内存和 CPU 资源

**Go 运行时优化：**
```bash
# 设置 GOMAXPROCS
export GOMAXPROCS=8

# 调整 GC 目标百分比
export GOGC=100

# 启用内存分析
export GODEBUG=gctrace=1
```

### 11. 压力测试建议

**测试场景：**
1. 单机 1000 并发任务创建
2. 10000 个 Agent 同时心跳上报
3. 100 个任务同时执行，每个任务 100 个主机
4. 长时间运行测试（24小时）

**测试工具：**
- Go 内置 benchmark
- Apache Bench (ab)
- wrk 压力测试工具
- 自定义压力测试脚本

### 12. 部署优化

**容器化配置：**
```dockerfile
# 资源限制
FROM golang:1.21-alpine
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY . .
RUN go build -o main ./cmd/main.go

# 运行时配置
ENV GOMAXPROCS=4
ENV GOGC=100
EXPOSE 8080 50051

CMD ["./main"]
```

**Kubernetes 配置：**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: devops-manager
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: devops-manager
        image: devops-manager:latest
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        env:
        - name: GOMAXPROCS
          value: "2"
```

## 总结

通过实施以上优化建议，系统性能预期可以提升：
- 任务创建性能提升 50%
- 查询性能提升 70%
- 内存使用减少 30%
- 支持更大规模的并发连接

建议按优先级逐步实施这些优化措施，并持续监控系统性能指标。