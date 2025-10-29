import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { hostsApi } from '@/api/hosts'
import { message } from 'ant-design-vue'

export const useHostsStore = defineStore('hosts', () => {
  const hosts = ref([])
  const loading = ref(false)

  // 计算属性
  const totalHosts = computed(() => hosts.value.length)
  
  const onlineHosts = computed(() => {
    const now = Math.floor(Date.now() / 1000)
    return hosts.value.filter(host => (now - host.last_seen) < 60).length
  })
  
  const offlineHosts = computed(() => totalHosts.value - onlineHosts.value)

  const hostsWithStatus = computed(() => {
    const now = Math.floor(Date.now() / 1000)
    return hosts.value.map(host => ({
      ...host,
      status: (now - host.last_seen) < 60 ? 'online' : 'offline',
      lastSeenText: formatDateTime(new Date(host.last_seen * 1000))
    }))
  })

  // 方法
  const fetchHosts = async () => {
    try {
      loading.value = true
      const response = await hostsApi.getHosts()
      hosts.value = response.data || []
    } catch (error) {
      message.error('获取主机列表失败: ' + error.message)
      console.error('Failed to fetch hosts:', error)
    } finally {
      loading.value = false
    }
  }

  const getHost = async (id) => {
    try {
      const response = await hostsApi.getHost(id)
      return response.data
    } catch (error) {
      message.error('获取主机详情失败: ' + error.message)
      throw error
    }
  }

  const deleteHost = async (id) => {
    try {
      await hostsApi.deleteHost(id)
      hosts.value = hosts.value.filter(host => host.id !== id)
      message.success('删除主机成功')
    } catch (error) {
      message.error('删除主机失败: ' + error.message)
      throw error
    }
  }

  const updateHost = async (id, hostInfo) => {
    try {
      const response = await hostsApi.updateHost(id, hostInfo)
      const index = hosts.value.findIndex(host => host.id === id)
      if (index !== -1) {
        hosts.value[index] = response.data
      }
      message.success('更新主机成功')
    } catch (error) {
      message.error('更新主机失败: ' + error.message)
      throw error
    }
  }

  // 工具函数
  const formatDateTime = (date) => {
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  }

  return {
    hosts,
    loading,
    totalHosts,
    onlineHosts,
    offlineHosts,
    hostsWithStatus,
    fetchHosts,
    getHost,
    deleteHost,
    updateHost
  }
})