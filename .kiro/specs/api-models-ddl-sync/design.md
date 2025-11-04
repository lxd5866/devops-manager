# 设计文档

## 概述

本设计文档描述了如何更新 DevOps Manager 系统中的 API 模型，使其与更新后的数据库 DDL 保持同步。主要涉及重构任务管理、命令执行和主机关联的数据模型，确保 Go 结构体与数据库表结构完全匹配。

## 架构

### 当前架构问题

1. **表结构不匹配**: 现有的 `TaskHost` 模型引用了不存在的 `task_hosts` 表
2. **字段映射错误**: Command 模型中的某些字段与实际数据库列不匹配
3. **关系定义错误**: 命令与主机的关系应该通过 `commands_hosts` 表管理
4. **类型错误**: 某些字段的 Go 类型与数据库类型不匹配
5. **数据重复**: `commands` 表和 `commands_hosts` 表都包含执行结果字段
6. **DDL 不一致**: `commands` 表的索引引用了不存在的 `status` 字段

### 目标架构

根据实际 DDL，数据库设计采用了双重存储模式：

```
Task (tasks 表)
├── 基本信息: task_id, name, description, status
├── 统计信息: total_hosts, completed_hosts, failed_hosts
├── 时间信息: started_at, finished_at, created_at, updated_at
└── 关系: 一对多 Commands

Command (commands 表) - 包含完整的命令和执行信息
├── 基本信息: command_id, task_id, host_id, command, parameters
├── 执行配置: timeout
├── 执行结果: stdout, stderr, exit_code, started_at, finished_at, error_message
├── 时间信息: created_at, updated_at
└── 关系: 一对多 CommandHosts

CommandHost (commands_hosts 表) - 命令在特定主机上的执行记录
├── 关联信息: command_id, host_id
├── 执行结果: status, stdout, stderr, exit_code
├── 时间信息: started_at, finished_at, execution_time
├── 错误信息: error_message
└── 时间戳: created_at, updated_at

Host (hosts 表)
├── 基本信息: host_id, hostname, ip, os
├── 状态信息: status, last_seen
└── 标签信息: tags (JSON)
```

**注意**: DDL 显示 `commands` 表和 `commands_hosts` 表都包含执行结果字段，这表明系统设计为：
- `commands` 表存储命令定义和可能的执行结果
- `commands_hosts` 表存储命令在每个主机上的具体执行记录

## 组件和接口

### 1. Task 模型重构

**文件**: `api/models/task.go`

**主要变更**:
- 移除 `TaskHost` 结构体和相关关系
- 保持与 `Command` 的一对多关系
- 更新业务逻辑方法以适应新的数据结构

**新的关系定义**:
```go
// 关联关系
Commands []Command `json:"commands" gorm:"foreignKey:TaskID;references:TaskID"`
```

### 2. Command 模型重构

**文件**: `api/models/command.go`

**主要变更**:
- 保持所有 DDL 中定义的字段，包括执行结果字段
- 修复 protobuf 转换方法中的类型错误
- 修复 Parameters 字段的类型（从 JSON 改为 string）
- 添加与 `CommandHost` 的一对多关系
- 添加缺失的 Status 字段

**字段调整**:
```go
type Command struct {
    ID         uint           `json:"id" gorm:"primaryKey"`
    CommandID  string         `json:"command_id" gorm:"uniqueIndex;size:255;not null;comment:命令唯一标识"`
    TaskID     *string        `json:"task_id" gorm:"size:255;comment:所属任务ID"`
    HostID     string         `json:"host_id" gorm:"size:255;not null;comment:目标主机ID"`
    Command    string         `json:"command" gorm:"type:text;not null;comment:命令内容"`
    Parameters string         `json:"parameters" gorm:"type:text;comment:命令参数"`  // 修复类型
    Timeout    int64          `json:"timeout" gorm:"comment:超时时间(秒)"`
    
    // 保留执行结果字段（与 DDL 匹配）
    Stdout     string         `json:"stdout" gorm:"type:longtext;comment:标准输出"`
    Stderr     string         `json:"stderr" gorm:"type:longtext;comment:错误输出"`
    ExitCode   *int           `json:"exit_code" gorm:"comment:退出码"`
    StartedAt  *time.Time     `json:"started_at" gorm:"comment:开始执行时间"`
    FinishedAt *time.Time     `json:"finished_at" gorm:"comment:完成时间"`
    ErrorMsg   string         `json:"error_message" gorm:"type:text;comment:执行错误信息"`
    
    // 时间戳字段
    CreatedAt  time.Time      `json:"created_at"`
    UpdatedAt  time.Time      `json:"updated_at"`
    DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
    
    // 关系
    CommandHosts []CommandHost `json:"command_hosts" gorm:"foreignKey:CommandID;references:CommandID"`
}
```

**注意**: DDL 中存在索引 `idx_commands_status` 但 `commands` 表中没有定义 `status` 字段。这需要在实施过程中决定是添加字段还是移除索引。

### 3. CommandHost 模型创建

**文件**: `api/models/command_host.go` (新文件)

**结构定义**:
```go
type CommandHost struct {
    ID            uint       `json:"id" gorm:"primaryKey"`
    CommandID     string     `json:"command_id" gorm:"size:255;not null;comment:命令ID"`
    HostID        string     `json:"host_id" gorm:"size:255;not null;comment:主机ID"`
    Status        string     `json:"status" gorm:"size:20;default:待执行;comment:命令状态"`
    Stdout        string     `json:"stdout" gorm:"type:longtext;comment:标准输出"`
    Stderr        string     `json:"stderr" gorm:"type:longtext;comment:错误输出"`
    ExitCode      int        `json:"exit_code" gorm:"default:0;comment:退出码"`
    StartedAt     *time.Time `json:"started_at" gorm:"comment:开始执行时间"`
    FinishedAt    *time.Time `json:"finished_at" gorm:"comment:完成时间"`
    ErrorMessage  string     `json:"error_message" gorm:"type:text;comment:执行错误信息"`
    ExecutionTime *int64     `json:"execution_time" gorm:"comment:执行时长(毫秒)"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
}
```

### 4. CommandResult 模型更新

**文件**: `api/models/command_result.go`

**主要变更**:
- 保持现有结构，但确保与 `CommandHost` 的兼容性
- 更新 protobuf 转换方法
- 添加与新模型的转换方法

## 数据模型

### 状态枚举定义

```go
// CommandHostStatus 命令主机执行状态
type CommandHostStatus string

const (
    CommandHostStatusPending    CommandHostStatus = "待执行"
    CommandHostStatusRunning    CommandHostStatus = "运行中"
    CommandHostStatusFailed     CommandHostStatus = "下发失败"
    CommandHostStatusExecFailed CommandHostStatus = "执行失败"
    CommandHostStatusTimeout    CommandHostStatus = "执行超时"
    CommandHostStatusCanceled   CommandHostStatus = "取消执行"
    CommandHostStatusCompleted  CommandHostStatus = "执行完成"
)
```

### 关系映射

1. **Task → Command**: 一对多关系，通过 `task_id` 关联
2. **Command → CommandHost**: 一对多关系，通过 `command_id` 关联
3. **Host → CommandHost**: 一对多关系，通过 `host_id` 关联

## 错误处理

### 1. 类型转换错误处理

- 修复 `Parameters` 字段的类型错误，从 JSON 改为 string
- 处理可空字段的指针类型转换
- 确保时间字段的正确处理

### 2. 关系查询错误处理

- 添加适当的 GORM 预加载配置
- 处理外键约束错误
- 添加数据一致性检查

### 3. Protobuf 转换错误处理

- 修复参数映射中的类型错误
- 处理空值和默认值
- 确保时间戳转换的正确性

## 迁移策略

### 1. 代码迁移步骤

1. 创建新的 `CommandHost` 模型
2. 更新 `Command` 模型，修复字段类型和添加缺失字段
3. 更新 `Task` 模型，移除 `TaskHost` 关系
4. 修复所有编译错误
5. 更新相关的服务层代码

### 2. 数据库迁移

- 确保 DDL 已正确应用
- 验证表结构与模型定义匹配
- 测试外键约束和索引

### 3. API 兼容性

- 保持现有 API 端点的响应格式
- 添加适配层处理数据结构变更
- 更新 API 文档和 Swagger 定义