<template>
  <div class="job-history">
    <div class="page-header">
      <h1>执行历史</h1>
      <p>查看所有任务的执行历史记录</p>
    </div>

    <!-- 搜索和筛选 -->
    <a-card class="filter-card">
      <a-form :model="searchForm" layout="inline">
        <a-form-item label="任务名称">
          <a-input
            v-model:value="searchForm.name"
            placeholder="请输入任务名称"
            allow-clear
            style="width: 200px"
          />
        </a-form-item>
        
        <a-form-item label="任务状态">
          <a-select
            v-model:value="searchForm.status"
            placeholder="请选择状态"
            allow-clear
            style="width: 150px"
          >
            <a-select-option value="pending">待执行</a-select-option>
            <a-select-option value="running">执行中</a-select-option>
            <a-select-option value="completed">已完成</a-select-option>
            <a-select-option value="failed">执行失败</a-select-option>
            <a-select-option value="canceled">已取消</a-select-option>
          </a-select>
        </a-form-item>
        
        <a-form-item label="创建者">
          <a-input
            v-model:value="searchForm.createdBy"
            placeholder="请输入创建者"
            allow-clear
            style="width: 150px"
          />
        </a-form-item>
        
        <a-form-item label="创建时间">
          <a-range-picker
            v-model:value="searchForm.dateRange"
            show-time
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 350px"
          />
        </a-form-item>
        
        <a-form-item>
          <a-button type="primary" @click="handleSearch" :loading="loading">
            搜索
          </a-button>
          <a-button @click="handleReset" style="margin-left: 8px">重置</a-button>
          <a-button type="primary" @click="$router.push('/jobs/script-execution')" style="margin-left: 8px">
            新建任务
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 任务列表 -->
    <a-card class="table-card">
      <a-table
        :data-source="tasks"
        :loading="loading"
        :pagination="paginationConfig"
        @change="handleTableChange"
        row-key="task_id"
      >
        <a-table-column key="name" title="任务名称" data-index="name" width="200" :ellipsis="true" />
        
        <a-table-column key="status" title="任务状态" data-index="status" width="100" align="center">
          <template #default="{ record }">
            <a-tag :color="getStatusColor(record.status)">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
        </a-table-column>
        
        <a-table-column key="total_hosts" title="总主机数" data-index="total_hosts" width="100" align="center" />
        
        <a-table-column key="completed_hosts" title="已完成主机数" data-index="completed_hosts" width="120" align="center">
          <template #default="{ record }">
            <span :class="{ 'success-text': record.completed_hosts === record.total_hosts && record.total_hosts > 0 }">
              {{ record.completed_hosts }}
            </span>
          </template>
        </a-table-column>
        
        <a-table-column key="failed_hosts" title="失败主机数" data-index="failed_hosts" width="100" align="center">
          <template #default="{ record }">
            <span :class="{ 'error-text': record.failed_hosts > 0 }">
              {{ record.failed_hosts }}
            </span>
          </template>
        </a-table-column>
        
        <a-table-column key="started_at" title="开始时间" data-index="started_at" width="160" :sorter="true">
          <template #default="{ record }">
            {{ formatDateTime(record.started_at) }}
          </template>
        </a-table-column>
        
        <a-table-column key="finished_at" title="结束时间" data-index="finished_at" width="160" :sorter="true">
          <template #default="{ record }">
            {{ formatDateTime(record.finished_at) }}
          </template>
        </a-table-column>
        
        <a-table-column key="duration" title="总耗时" width="100" align="center">
          <template #default="{ record }">
            {{ calculateDuration(record.started_at, record.finished_at) }}
          </template>
        </a-table-column>
        
        <a-table-column key="created_by" title="创建者" data-index="created_by" width="120" :ellipsis="true" />
        
        <a-table-column key="description" title="任务描述" data-index="description" :ellipsis="true" />
        
        <a-table-column key="actions" title="操作" width="180" fixed="right">
          <template #default="{ record }">
            <a-button
              type="primary"
              size="small"
              @click="viewDetail(record)"
            >
              详情
            </a-button>
            
            <a-button
              v-if="record.status === 'running'"
              type="primary"
              danger
              size="small"
              @click="stopTask(record)"
              style="margin-left: 8px"
            >
              停止
            </a-button>
            
            <a-button
              v-if="record.status === 'pending'"
              danger
              size="small"
              @click="cancelTask(record)"
              style="margin-left: 8px"
            >
              取消
            </a-button>
            
            <a-dropdown v-if="record.status === 'completed' || record.status === 'failed'">
              <a-button size="small" style="margin-left: 8px">
                更多
                <template #icon><down-outlined /></template>
              </a-button>
              <template #overlay>
                <a-menu>
                  <a-menu-item @click="retryTask(record)">重新执行</a-menu-item>
                  <a-menu-item @click="exportResult(record)">导出结果</a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </template>
        </a-table-column>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import { DownOutlined } from '@ant-design/icons-vue'
import { tasksApi } from '@/api'

const router = useRouter()

// 搜索表单
const searchForm = reactive({
  name: '',
  status: '',
  createdBy: '',
  dateRange: []
})

// 分页信息
const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`
})

// 排序信息
const sortInfo = reactive({
  field: 'created_at',
  order: 'descend'
})

// 状态变量
const loading = ref(false)
const tasks = ref([])

// 分页配置
const paginationConfig = computed(() => ({
  ...pagination,
  onChange: (page, pageSize) => {
    pagination.current = page
    pagination.pageSize = pageSize
    loadTasks()
  },
  onShowSizeChange: (current, size) => {
    pagination.current = 1
    pagination.pageSize = size
    loadTasks()
  }
}))

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

// 格式化日期时间
const formatDateTime = (dateTime) => {
  if (!dateTime) return '-'
  return new Date(dateTime).toLocaleString('zh-CN')
}

// 计算耗时
const calculateDuration = (startTime, endTime) => {
  if (!startTime || !endTime) return '-'
  
  const start = new Date(startTime)
  const end = new Date(endTime)
  const duration = Math.floor((end - start) / 1000)
  
  if (duration < 60) return `${duration}秒`
  if (duration < 3600) return `${Math.floor(duration / 60)}分${duration % 60}秒`
  
  const hours = Math.floor(duration / 3600)
  const minutes = Math.floor((duration % 3600) / 60)
  const seconds = duration % 60
  
  return `${hours}时${minutes}分${seconds}秒`
}

// 加载任务列表
const loadTasks = async () => {
  try {
    loading.value = true
    
    const params = {
      page: pagination.current,
      size: pagination.pageSize,
      sort_by: sortInfo.field,
      sort_order: sortInfo.order === 'ascend' ? 'asc' : 'desc',
      ...searchForm
    }
    
    // 处理日期范围
    if (searchForm.dateRange && searchForm.dateRange.length === 2) {
      params.start_time = searchForm.dateRange[0]
      params.end_time = searchForm.dateRange[1]
    }
    
    const response = await tasksApi.getTasks(params)
    
    tasks.value = response?.items || []
    pagination.total = response?.total || 0
    
  } catch (error) {
    message.error('加载任务列表失败')
    console.error('Load tasks error:', error)
  } finally {
    loading.value = false
  }
}

// 表格变化处理
const handleTableChange = (pag, filters, sorter) => {
  if (sorter.field) {
    sortInfo.field = sorter.field
    sortInfo.order = sorter.order
  }
  loadTasks()
}

// 搜索处理
const handleSearch = () => {
  pagination.current = 1
  loadTasks()
}

// 重置搜索
const handleReset = () => {
  Object.assign(searchForm, {
    name: '',
    status: '',
    createdBy: '',
    dateRange: []
  })
  pagination.current = 1
  loadTasks()
}

// 查看详情
const viewDetail = (task) => {
  router.push(`/jobs/detail/${task.task_id}`)
}

// 停止任务
const stopTask = async (task) => {
  Modal.confirm({
    title: '停止任务',
    content: `确定要停止任务 "${task.name}" 吗？`,
    okText: '确定',
    cancelText: '取消',
    onOk: async () => {
      try {
        await tasksApi.stopTask(task.task_id)
        message.success('任务停止成功')
        loadTasks()
      } catch (error) {
        message.error('停止任务失败')
        console.error('Stop task error:', error)
      }
    }
  })
}

// 取消任务
const cancelTask = async (task) => {
  Modal.confirm({
    title: '取消任务',
    content: `确定要取消任务 "${task.name}" 吗？`,
    okText: '确定',
    cancelText: '取消',
    onOk: async () => {
      try {
        await tasksApi.cancelTask(task.task_id)
        message.success('任务取消成功')
        loadTasks()
      } catch (error) {
        message.error('取消任务失败')
        console.error('Cancel task error:', error)
      }
    }
  })
}

// 重新执行任务
const retryTask = async (task) => {
  Modal.confirm({
    title: '重新执行',
    content: `确定要重新执行任务 "${task.name}" 吗？`,
    okText: '确定',
    cancelText: '取消',
    onOk: async () => {
      try {
        await tasksApi.retryTask(task.task_id)
        message.success('任务重新执行成功')
        loadTasks()
      } catch (error) {
        message.error('重新执行任务失败')
        console.error('Retry task error:', error)
      }
    }
  })
}

// 导出结果
const exportResult = async (task) => {
  try {
    const response = await tasksApi.exportTaskResult(task.task_id)
    
    // 创建下载链接
    const blob = new Blob([response], { type: 'application/json' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `task_${task.task_id}_result.json`
    link.click()
    window.URL.revokeObjectURL(url)
    
    message.success('导出成功')
    
  } catch (error) {
    message.error('导出失败')
    console.error('Export result error:', error)
  }
}

// 初始化
onMounted(() => {
  loadTasks()
})
</script>

<style scoped>
.job-history {
  padding: 20px;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h1 {
  margin: 0 0 8px 0;
  font-size: 24px;
  color: #303133;
}

.page-header p {
  margin: 0;
  color: #909399;
}

.filter-card {
  margin-bottom: 20px;
}

.table-card {
  min-height: 400px;
}

.success-text {
  color: #52c41a;
  font-weight: bold;
}

.error-text {
  color: #ff4d4f;
  font-weight: bold;
}
</style>