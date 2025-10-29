import request from './request'

// 命令相关 API
export const commandsApi = {
  // 创建命令
  createCommand(data) {
    return request.post('/api/v1/commands', data)
  },

  // 获取命令列表
  getCommands(params) {
    return request.get('/api/v1/commands', { params })
  },

  // 获取单个命令
  getCommand(commandId) {
    return request.get(`/api/v1/commands/${commandId}`)
  },

  // 更新命令
  updateCommand(commandId, data) {
    return request.put(`/api/v1/commands/${commandId}`, data)
  },

  // 删除命令
  deleteCommand(commandId) {
    return request.delete(`/api/v1/commands/${commandId}`)
  },

  // 执行命令
  executeCommand(commandId) {
    return request.post(`/api/v1/commands/${commandId}/execute`)
  },

  // 获取命令结果
  getCommandResult(commandId) {
    return request.get(`/api/v1/commands/${commandId}/result`)
  },

  // 获取任务的所有命令
  getTaskCommands(taskId) {
    return request.get(`/api/v1/tasks/${taskId}/commands`)
  },

  // 添加任务命令
  addTaskCommand(taskId, data) {
    return request.post(`/api/v1/tasks/${taskId}/commands`, data)
  },

  // 移除任务命令
  removeTaskCommand(taskId, commandId) {
    return request.delete(`/api/v1/tasks/${taskId}/commands/${commandId}`)
  }
}

export default commandsApi