<template>
  <div class="job-detail">
    <div class="page-header">
      <a-button @click="$router.back()" style="margin-right: 16px">
        <template #icon><arrow-left-outlined /></template>
        返回
      </a-button>
      <h1>任务详情</h1>
    </div>

    <a-spin :spinning="loading">
      <!-- 任务基础信息 -->
      <a-card class="task-info-card" title="任务基础信息">
        <template #extra>
          <a-tag :color="getStatusColor(task.status)" style="margin-right: 8px">
            {{ getStatusText(task.status) }}
          </a-tag>
        </template>

        <a-descriptions :column="3" bordered>
          <a-descriptions-item label="任务名称">
            {{ task.name }}
          </a-descriptions-item>
          <a-descriptions-item label="任务ID">
            {{ task.task_id }}
          </a-descriptions-item>
          <a-descriptions-item label="创建者">
            {{ task.created_by }}
          </a-descriptions-item>
          <a-descriptions-item label="总主机数">
            {{ task.total_hosts }}
          </a-descriptions-item>
          <a-descriptions-item label="已完成主机数">
            {{ task.completed_hosts }}
          </a-descriptions-item>
          <a-descriptions-item label="失败主机数">
            {{ task.failed_hosts }}
          </a-descriptions-item>
          <a-descriptions-item label="任务描述" :span="3">
            {{ task.description }}
          </a-descriptions-item>
        </a-descriptions>
      </a-card>
    </a-spin>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { ArrowLeftOutlined } from '@ant-design/icons-vue'
import { tasksApi } from '@/api'

const route = useRoute()

// 状态变量
const loading = ref(false)
const task = ref({})

// 状态映射
const statusMap = {
  pending: { text: '待执行', color: 'default' },
  running: { text: '执行中', color: 'processing' },
  completed: { text: '已完成', color: 'success' },
  failed: { text: '执行失败', color: 'error' },
  canceled: { text: '已取消', color: 'default' }
}

// 获取状态文本
const getStatusText = (status) => {
  return statusMap[status]?.text || status
}

// 获取状态颜色
const getStatusColor = (status) => {
  return statusMap[status]?.color || 'default'
}

// 加载任务详情
const loadTaskDetail = async () => {
  try {
    loading.value = true
    const response = await tasksApi.getTask(route.params.taskId)
    task.value = response || {}
  } catch (error) {
    message.error('加载任务详情失败')
    console.error('Load task detail error:', error)
  } finally {
    loading.value = false
  }
}

// 初始化
onMounted(() => {
  loadTaskDetail()
})
</script>

<style scoped>
.job-detail {
  padding: 20px;
}

.page-header {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h1 {
  margin: 0;
  font-size: 24px;
  color: #303133;
}

.task-info-card {
  margin-bottom: 20px;
}
</style>