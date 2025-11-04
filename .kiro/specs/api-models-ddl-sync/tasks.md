# 实施计划

- [x] 1. 创建新的 CommandHost 模型
  - 创建 `api/models/command_host.go` 文件
  - 定义 CommandHost 结构体，映射到 `commands_hosts` 表
  - 添加状态枚举和常量定义
  - 实现表名映射和 GORM 标签
  - 添加与 Command 和 Host 的关联关系
  - _需求: 2.1, 2.3, 2.5_

- [x] 2. 重构 Command 模型
  - [x] 2.1 移除执行结果相关字段
    - 从 Command 结构体中移除 stdout, stderr, exit_code, started_at, finished_at, error_message 字段
    - 保留核心命令定义字段 (command_id, task_id, host_id, command, parameters, timeout)
    - _需求: 4.1, 4.2_

  - [x] 2.2 修复 Parameters 字段类型
    - 将 Parameters 字段从 JSON 类型改为 string 类型
    - 更新 GORM 标签为 `type:text`
    - _需求: 4.5_

  - [x] 2.3 修复 protobuf 转换方法
    - 修复 ToProtobufContent 方法中的类型错误
    - 修复 FromProtobufContent 方法中的参数处理
    - 移除与执行结果相关的 protobuf 转换代码
    - _需求: 4.4, 5.1_

  - [x] 2.4 添加 CommandHost 关系
    - 在 Command 结构体中添加 CommandHosts 关联字段
    - 配置正确的外键关系
    - _需求: 4.3_

- [x] 3. 重构 Task 模型
  - [x] 3.1 移除 TaskHost 相关代码
    - 删除 TaskHost 结构体定义
    - 从 Task 结构体中移除 TaskHosts 关联字段
    - 移除 TaskHost 相关的构建器方法
    - _需求: 3.1_

  - [x] 3.2 保持 Command 关系
    - 确保 Task 与 Command 的一对多关系正确配置
    - 验证外键关系通过 task_id 字段
    - _需求: 3.2_

  - [x] 3.3 更新业务逻辑方法
    - 修改 UpdateProgress 方法以使用新的数据结构
    - 更新任务状态计算逻辑
    - 确保向后兼容性
    - _需求: 3.5_

- [x] 4. 更新 CommandResult 模型
  - 验证 CommandResult 模型与数据库表的匹配性
  - 确保与新的 CommandHost 模型的兼容性
  - 更新 protobuf 转换方法
  - _需求: 5.5_

- [x] 5. 验证和修复编译错误
  - 运行 Go 编译检查所有模型文件
  - 修复任何类型不匹配或语法错误
  - 确保所有导入和依赖正确
  - _需求: 5.1, 5.2_

- [x] 6. 更新相关工具方法
  - 检查并更新 `api/models/command_utils.go` 中的相关方法
  - 确保工具方法与新的模型结构兼容
  - _需求: 5.3, 5.4_