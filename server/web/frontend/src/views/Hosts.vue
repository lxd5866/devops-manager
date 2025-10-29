<template>
  <div>
    <!-- 统计卡片 -->
    <a-row :gutter="16" style="margin-bottom: 24px">
      <a-col :span="6">
        <a-card>
          <a-statistic
            title="总主机数"
            :value="hostsStore.totalHosts"
            :value-style="{ color: '#1890ff' }"
          >
            <template #prefix>
              <desktop-outlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      
      <a-col :span="6">
        <a-card>
          <a-statistic
            title="在线主机"
            :value="hostsStore.onlineHosts"
            :value-style="{ color: '#52c41a' }"
          >
            <template #prefix>
              <check-circle-outlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      
      <a-col :span="6">
        <a-card>
          <a-statistic
            title="离线主机"
            :value="hostsStore.offlineHosts"
            :value-style="{ color: '#ff4d4f' }"
          >
            <template #prefix>
              <close-circle-outlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>

      <a-col :span="6">
        <a-card>
          <a-statistic
            title="待准入节点"
            :value="pendingCount"
            :value-style="{ color: '#fa8c16' }"
          >
            <template #prefix>
              <clock-circle-outlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 主机列表 -->
    <a-card title="主机列表">
      <template #extra>
        <a-space>
          <a-button type="primary" @click="refreshHosts" :loading="hostsStore.loading">
            <template #icon>
              <reload-outlined />
            </template>
            刷新
          </a-button>
          <a-button type="default" @click="showPendingHosts" :disabled="pendingCount === 0">
            <template #icon>
              <clock-circle-outlined />
            </template>
            待准入节点 ({{ pendingCount }})
          </a-button>
        </a-space>
      </template>

      <a-table
        :columns="columns"
        :data-source="hostsStore.hostsWithStatus"
        :loading="hostsStore.loading"
        :pagination="{ pageSize: 10, showSizeChanger: true, showQuickJumper: true }"
        row-key="id"
      >
        <!-- 主机信息列 -->
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'hostInfo'">
            <div style="display: flex; align-items: center">
              <a-avatar style="background-color: #f56a00; margin-right: 12px">
                <template #icon>
                  <desktop-outlined />
                </template>
              </a-avatar>
              <div>
                <div style="font-weight: 500">{{ record.hostname }}</div>
                <div style="color: #666; font-size: 12px">{{ record.id }}</div>
              </div>
            </div>
          </template>

          <!-- 状态列 -->
          <template v-else-if="column.key === 'status'">
            <a-tag :color="record.status === 'online' ? 'green' : 'red'">
              {{ record.status === 'online' ? '在线' : '离线' }}
            </a-tag>
          </template>

          <!-- 标签列 -->
          <template v-else-if="column.key === 'tags'">
            <div>
              <a-tag
                v-for="(value, key) in getDisplayTags(record.tags)"
                :key="key"
                color="blue"
                style="margin-bottom: 4px"
              >
                {{ key }}: {{ value }}
              </a-tag>
              <a-tag v-if="getTagsCount(record.tags) > 3" color="default">
                +{{ getTagsCount(record.tags) - 3 }}
              </a-tag>
            </div>
          </template>

          <!-- 操作列 -->
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="viewHost(record)">
                查看
              </a-button>
              <a-popconfirm
                title="确定要删除这台主机吗？"
                ok-text="确定"
                cancel-text="取消"
                @confirm="deleteHost(record.id)"
              >
                <a-button type="link" size="small" danger>
                  删除
                </a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 主机详情模态框 -->
    <a-modal
      v-model:open="detailModalVisible"
      title="主机详情"
      :footer="null"
      width="600px"
    >
      <div v-if="selectedHost">
        <a-descriptions :column="2" bordered>
          <a-descriptions-item label="主机ID" :span="2">
            {{ selectedHost.id }}
          </a-descriptions-item>
          <a-descriptions-item label="主机名">
            {{ selectedHost.hostname }}
          </a-descriptions-item>
          <a-descriptions-item label="IP地址">
            {{ selectedHost.ip }}
          </a-descriptions-item>
          <a-descriptions-item label="操作系统">
            {{ selectedHost.os }}
          </a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="selectedHost.status === 'online' ? 'green' : 'red'">
              {{ selectedHost.status === 'online' ? '在线' : '离线' }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="最后上报时间" :span="2">
            {{ selectedHost.lastSeenText }}
          </a-descriptions-item>
          <a-descriptions-item label="标签" :span="2">
            <div>
              <a-tag
                v-for="(value, key) in selectedHost.tags"
                :key="key"
                color="blue"
                style="margin-bottom: 4px"
              >
                {{ key }}: {{ value }}
              </a-tag>
            </div>
          </a-descriptions-item>
        </a-descriptions>
      </div>
    </a-modal>

    <!-- 待准入节点模态框 -->
    <a-modal
      v-model:open="pendingModalVisible"
      title="待准入节点"
      :footer="null"
      width="800px"
    >
      <a-table
        :columns="pendingColumns"
        :data-source="pendingHosts"
        :pagination="false"
        row-key="host_id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'hostInfo'">
            <div style="display: flex; align-items: center">
              <a-avatar style="background-color: #fa8c16; margin-right: 12px">
                <template #icon>
                  <desktop-outlined />
                </template>
              </a-avatar>
              <div>
                <div style="font-weight: 500">{{ record.hostname }}</div>
                <div style="color: #666; font-size: 12px">{{ record.host_id }}</div>
              </div>
            </div>
          </template>

          <template v-else-if="column.key === 'tags'">
            <div>
              <a-tag
                v-for="(value, key) in getDisplayTags(record.tags)"
                :key="key"
                color="blue"
                style="margin-bottom: 4px"
              >
                {{ key }}: {{ value }}
              </a-tag>
              <a-tag v-if="getTagsCount(record.tags) > 2" color="default">
                +{{ getTagsCount(record.tags) - 2 }}
              </a-tag>
            </div>
          </template>

          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="primary" size="small" @click="approveHost(record.host_id)">
                准入
              </a-button>
              <a-popconfirm
                title="确定要拒绝这台主机吗？"
                ok-text="确定"
                cancel-text="取消"
                @confirm="rejectHost(record.host_id)"
              >
                <a-button type="default" size="small" danger>
                  拒绝
                </a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useHostsStore } from '@/stores/hosts'
import {
  DesktopOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ReloadOutlined,
  ClockCircleOutlined
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

const hostsStore = useHostsStore()

const detailModalVisible = ref(false)
const selectedHost = ref(null)
const pendingModalVisible = ref(false)
const pendingHosts = ref([])
const pendingCount = ref(0)
let refreshTimer = null

// 表格列定义
const columns = [
  {
    title: '主机信息',
    key: 'hostInfo',
    width: 200
  },
  {
    title: '状态',
    key: 'status',
    width: 80
  },
  {
    title: 'IP地址',
    dataIndex: 'ip',
    key: 'ip',
    width: 120
  },
  {
    title: '操作系统',
    dataIndex: 'os',
    key: 'os',
    width: 100
  },
  {
    title: '标签',
    key: 'tags',
    width: 200
  },
  {
    title: '最后上报',
    dataIndex: 'lastSeenText',
    key: 'lastSeen',
    width: 150
  },
  {
    title: '操作',
    key: 'action',
    width: 120,
    fixed: 'right'
  }
]

// 待准入节点表格列定义
const pendingColumns = [
  {
    title: '主机信息',
    key: 'hostInfo',
    width: 200
  },
  {
    title: 'IP地址',
    dataIndex: 'ip',
    key: 'ip',
    width: 120
  },
  {
    title: '操作系统',
    dataIndex: 'os',
    key: 'os',
    width: 100
  },
  {
    title: '标签',
    key: 'tags',
    width: 200
  },
  {
    title: '首次注册',
    dataIndex: 'first_seen',
    key: 'firstSeen',
    width: 150,
    customRender: ({ text }) => new Date(text * 1000).toLocaleString('zh-CN')
  },
  {
    title: '操作',
    key: 'action',
    width: 150,
    fixed: 'right'
  }
]

// 方法
const refreshHosts = async () => {
  await hostsStore.fetchHosts()
}

const viewHost = (host) => {
  selectedHost.value = host
  detailModalVisible.value = true
}

const deleteHost = async (id) => {
  try {
    await hostsStore.deleteHost(id)
  } catch (error) {
    console.error('Delete host failed:', error)
  }
}

const getDisplayTags = (tags) => {
  if (!tags) return {}
  const entries = Object.entries(tags)
  return Object.fromEntries(entries.slice(0, 3))
}

const getTagsCount = (tags) => {
  return tags ? Object.keys(tags).length : 0
}

const fetchPendingCount = async () => {
  try {
    const response = await fetch('/api/v1/pending-hosts/count')
    const data = await response.json()
    if (data.success) {
      pendingCount.value = data.data.count
    }
  } catch (error) {
    console.error('Failed to fetch pending count:', error)
  }
}

const fetchPendingHosts = async () => {
  try {
    const response = await fetch('/api/v1/pending-hosts')
    const data = await response.json()
    if (data.success) {
      pendingHosts.value = data.data || []
    }
  } catch (error) {
    console.error('Failed to fetch pending hosts:', error)
    message.error('获取待准入节点失败')
  }
}

const showPendingHosts = async () => {
  await fetchPendingHosts()
  pendingModalVisible.value = true
}

const approveHost = async (hostId) => {
  try {
    const response = await fetch(`/api/v1/pending-hosts/${hostId}/approve`, {
      method: 'POST'
    })
    const data = await response.json()
    
    if (data.success) {
      message.success('主机准入成功')
      await fetchPendingHosts()
      await fetchPendingCount()
      await refreshHosts()
    } else {
      message.error('准入失败: ' + data.error_message)
    }
  } catch (error) {
    message.error('准入失败: ' + error.message)
  }
}

const rejectHost = async (hostId) => {
  try {
    const response = await fetch(`/api/v1/pending-hosts/${hostId}/reject`, {
      method: 'POST'
    })
    const data = await response.json()
    
    if (data.success) {
      message.success('主机已拒绝')
      await fetchPendingHosts()
      await fetchPendingCount()
    } else {
      message.error('拒绝失败: ' + data.error_message)
    }
  } catch (error) {
    message.error('拒绝失败: ' + error.message)
  }
}

// 生命周期
onMounted(() => {
  refreshHosts()
  fetchPendingCount()
  // 每30秒自动刷新
  refreshTimer = setInterval(() => {
    refreshHosts()
    fetchPendingCount()
  }, 30000)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})
</script>

<style scoped>
.ant-card {
  border-radius: 8px;
}

.ant-statistic-title {
  font-size: 14px;
  color: #666;
}

.ant-statistic-content {
  font-size: 24px;
  font-weight: 600;
}
</style>