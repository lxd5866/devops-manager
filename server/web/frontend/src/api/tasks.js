import request from './request'

// 任务相关 API
export const tasksApi = {
  // 创建任务
  createTask(data) {
    return request.post('/api/v1/tasks', data)
  },

  // 获取任务列表
  getTasks(params) {
    return request.get('/api/v1/tasks', { params })
  },

  // 获取单个任务
  getTask(taskId) {
    return request.get(`/api/v1/tasks/${taskId}`)
  },

  // 更新任务
  updateTask(taskId, data) {
    return request.put(`/api/v1/tasks/${taskId}`, data)
  },

  // 删除任务
  deleteTask(taskId) {
    return request.delete(`/api/v1/tasks/${taskId}`)
  },

  // 启动任务
  startTask(taskId) {
    return request.post(`/api/v1/tasks/${taskId}/start`)
  },

  // 停止任务
  stopTask(taskId) {
    return request.post(`/api/v1/tasks/${taskId}/stop`)
  },

  // 取消任务
  cancelTask(taskId) {
    return request.post(`/api/v1/tasks/${taskId}/cancel`)
  },

  // 重新执行任务
  retryTask(taskId) {
    return request.post(`/api/v1/tasks/${taskId}/retry`)
  },

  // 获取任务状态
  getTaskStatus(taskId) {
    return request.get(`/api/v1/tasks/${taskId}/status`)
  },

  // 获取任务进度
  getTaskProgress(taskId) {
    return request.get(`/api/v1/tasks/${taskId}/progress`)
  },

  // 获取任务日志
  getTaskLogs(taskId, params) {
    return request.get(`/api/v1/tasks/${taskId}/logs`, { params })
  },

  // 添加任务主机
  addTaskHosts(taskId, hostIds) {
    return request.post(`/api/v1/tasks/${taskId}/hosts`, { host_ids: hostIds })
  },

  // 移除任务主机
  removeTaskHost(taskId, hostId) {
    return request.delete(`/api/v1/tasks/${taskId}/hosts/${hostId}`)
  },

  // 获取任务主机列表
  getTaskHosts(taskId) {
    return request.get(`/api/v1/tasks/${taskId}/hosts`)
  },

  // 导出任务结果
  exportTaskResult(taskId) {
    return request.get(`/api/v1/tasks/${taskId}/export`, {
      responseType: 'blob'
    })
  }
}

export default tasksApi