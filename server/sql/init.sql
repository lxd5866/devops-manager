CREATE TABLE `hosts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `host_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '主机唯一标识',
  `hostname` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '主机名',
  `ip` varchar(45) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'IP地址',
  `os` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '操作系统',
  `tags` json DEFAULT NULL COMMENT '标签信息',
  `last_seen` datetime(3) DEFAULT NULL COMMENT '最后上报时间',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `status` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT 'pending' COMMENT '主机状态',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_hosts_host_id` (`host_id`),
  KEY `idx_hosts_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

# Tasks 表 - 任务管理
CREATE TABLE `tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `task_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '任务唯一标识',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '任务名称',
  `description` text COLLATE utf8mb4_unicode_ci COMMENT '任务描述',
  `status` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT 'pending' COMMENT '任务状态: pending, running, completed, failed, canceled',
  `total_hosts` int DEFAULT 0 COMMENT '总主机数',
  `completed_hosts` int DEFAULT 0 COMMENT '已完成主机数',
  `failed_hosts` int DEFAULT 0 COMMENT '失败主机数',
  `created_by` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '创建者',
  `started_at` datetime(3) DEFAULT NULL COMMENT '开始时间',
  `finished_at` datetime(3) DEFAULT NULL COMMENT '完成时间',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_tasks_task_id` (`task_id`),
  KEY `idx_tasks_deleted_at` (`deleted_at`),
  KEY `idx_tasks_status` (`status`),
  KEY `idx_tasks_created_by` (`created_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


# Commands 表 - 命令内容
CREATE TABLE `commands` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `command_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '命令唯一标识',
  `task_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '所属任务ID',
  `host_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '目标主机ID',
  `command` text COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '命令内容',
  `parameters` text  COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '命令参数',
  `timeout` bigint DEFAULT NULL COMMENT '超时时间(秒)',
  `stdout` longtext COLLATE utf8mb4_unicode_ci COMMENT '标准输出',
  `stderr` longtext COLLATE utf8mb4_unicode_ci COMMENT '错误输出',
  `exit_code` int DEFAULT NULL COMMENT '退出码',
  `started_at` datetime(3) DEFAULT NULL COMMENT '开始执行时间',
  `finished_at` datetime(3) DEFAULT NULL COMMENT '完成时间',
  `error_message` text COLLATE utf8mb4_unicode_ci COMMENT '执行错误信息',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_commands_command_id` (`command_id`),
  KEY `idx_commands_deleted_at` (`deleted_at`),
  KEY `idx_commands_task_id` (`task_id`),
  KEY `idx_commands_host_id` (`host_id`),
  KEY `idx_commands_status` (`status`),
  KEY `idx_commands_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


# Task Hosts 关联表 - 任务与主机的关联
CREATE TABLE `commands_hosts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `command_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '命令ID',
  `host_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '主机ID',
  `status` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT '待执行' COMMENT '命令状态: 待执行,运行中,下发失败, 执行失败, 执行超时, 取消执行',
  `stdout` longtext COLLATE utf8mb4_unicode_ci COMMENT '标准输出',
  `stderr` longtext COLLATE utf8mb4_unicode_ci COMMENT '错误输出',
  `exit_code` int DEFAULT 0 COMMENT '退出码',
  `started_at` datetime(3) DEFAULT NULL COMMENT '开始执行时间',
  `finished_at` datetime(3) DEFAULT NULL COMMENT '完成时间',
  `error_message` text COLLATE utf8mb4_unicode_ci COMMENT '执行错误信息',
  `execution_time` bigint DEFAULT NULL COMMENT '执行时长(毫秒)',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_task_hosts_task_host` (`command_id`, `host_id`),
  KEY `idx_task_hosts_task_id` (`command_id`),
  KEY `idx_task_hosts_host_id` (`host_id`),
  KEY `idx_task_hosts_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
