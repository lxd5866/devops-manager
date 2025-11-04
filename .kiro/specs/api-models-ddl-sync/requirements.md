# 需求文档

## 介绍

DevOps Manager 系统已更新了任务管理、命令执行和主机关系的数据库架构。API 模型需要与新的 DDL 结构同步，以确保正确的数据映射和功能。这涉及更新 Task、Command、CommandResult 和相关模型以匹配当前的数据库架构。

## 术语表

- **API_Models**: `api/models/` 包中映射到数据库表的 Go 结构体定义
- **DDL**: 数据定义语言 - `server/sql/init.sql` 中的 SQL 架构定义
- **GORM**: 用于数据库操作的 Go 对象关系映射库
- **Command_Host_Association**: 将命令链接到特定主机并包含执行详情的 `commands_hosts` 表
- **Task_System**: 包括任务、命令及其关系的任务管理系统
- **JSON_Field**: 存储 JSON 数据并具有适当 Go 类型映射的数据库字段

## 需求

### 需求 1

**用户故事:** 作为开发者，我希望 API 模型准确反映当前的数据库架构，以便数据操作能够正确工作而不出现映射错误。

#### 验收标准

1. WHEN 系统从 `tasks` 表读取数据时，THE API_Models SHALL 正确映射所有字段，包括 `task_id`、`name`、`description`、`status`、`total_hosts`、`completed_hosts`、`failed_hosts`、`created_by`、`started_at`、`finished_at` 和时间戳字段
2. WHEN 系统从 `commands` 表读取数据时，THE API_Models SHALL 正确映射所有字段，包括 `command_id`、`task_id`、`host_id`、`command`、`parameters`、`timeout`、`stdout`、`stderr`、`exit_code`、`started_at`、`finished_at` 和 `error_message`
3. WHEN 系统从 `commands_hosts` 表读取数据时，THE API_Models SHALL 正确映射所有字段，包括 `command_id`、`host_id`、`status`、`stdout`、`stderr`、`exit_code`、`started_at`、`finished_at`、`error_message` 和 `execution_time`
4. THE API_Models SHALL 使用与数据库列名和约束匹配的正确 GORM 标签
5. THE API_Models SHALL 在适当的地方使用指针类型正确处理可空字段

### 需求 2

**用户故事:** 作为开发者，我希望命令-主机关系模型匹配新的 `commands_hosts` 表结构，以便命令执行跟踪能够在多个主机上正常工作。

#### 验收标准

1. THE API_Models SHALL 定义映射到 `commands_hosts` 表的 `CommandHost` 结构体
2. WHEN 命令在多个主机上执行时，THE Command_Host_Association SHALL 跟踪每个主机的单独执行结果
3. THE CommandHost 模型 SHALL 包含 `command_id`、`host_id`、`status`、`stdout`、`stderr`、`exit_code`、`started_at`、`finished_at`、`error_message` 和 `execution_time` 字段
4. THE CommandHost 模型 SHALL 使用适当的状态值，包括"待执行"、"运行中"、"下发失败"、"执行失败"、"执行超时"、"取消执行"
5. THE CommandHost 模型 SHALL 与 Command 和 Host 模型建立适当的外键关系

### 需求 3

**用户故事:** 作为开发者，我希望 Task 模型更新以反映当前架构，以便任务管理操作能够正确工作。

#### 验收标准

1. THE Task 模型 SHALL 移除引用 `task_hosts` 表的旧 `TaskHost` 关联
2. THE Task 模型 SHALL 通过 `task_id` 外键维护与 Command 模型的关系
3. THE Task 模型 SHALL 包含 `tasks` 表架构中存在的所有字段
4. THE Task 模型 SHALL 为索引和约束使用适当的 GORM 标签
5. THE Task 模型 SHALL 为现有业务逻辑方法维护向后兼容性

### 需求 4

**用户故事:** 作为开发者，我希望 Command 模型更新以与新架构配合工作，以便命令操作能够与主机关联系统正确集成。

#### 验收标准

1. THE Command 模型 SHALL 移除现在由 `commands_hosts` 表处理的字段
2. THE Command 模型 SHALL 维护核心命令定义字段，如 `command`、`parameters` 和 `timeout`
3. THE Command 模型 SHALL 与 CommandHost 模型建立适当的关系
4. THE Command 模型 SHALL 修复 protobuf 转换方法中的编译错误
5. THE Command 模型 SHALL 将 `parameters` 字段作为适当的文本字段而不是 JSON 处理

### 需求 5

**用户故事:** 作为开发者，我希望有适当的类型定义和错误处理，以便模型能够正确编译和运行。

#### 验收标准

1. THE API_Models SHALL 编译时不出现语法错误或类型不匹配
2. THE API_Models SHALL 使用适当的 Go 类型和数据库映射处理 JSON 字段
3. THE API_Models SHALL 为数据库列和 Go 字段使用一致的命名约定
4. THE API_Models SHALL 在适当的地方包含适当的验证标签
5. THE API_Models SHALL 在适用的地方维护现有的 protobuf 转换功能