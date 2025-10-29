import request from './request'

export const hostsApi = {
  // 获取所有主机
  getHosts() {
    return request.get('/api/v1/hosts')
  },

  // 获取单个主机
  getHost(id) {
    return request.get(`/api/v1/hosts/${id}`)
  },

  // 注册主机
  registerHost(hostInfo) {
    return request.post('/api/v1/hosts/register', hostInfo)
  },

  // 更新主机
  updateHost(id, hostInfo) {
    return request.put(`/api/v1/hosts/${id}`, hostInfo)
  },

  // 删除主机
  deleteHost(id) {
    return request.delete(`/api/v1/hosts/${id}`)
  },

  // 获取待准入主机
  getPendingHosts() {
    return request.get('/api/v1/pending-hosts')
  },

  // 准入主机
  approveHost(id) {
    return request.post(`/api/v1/pending-hosts/${id}/approve`)
  },

  // 拒绝主机
  rejectHost(id) {
    return request.post(`/api/v1/pending-hosts/${id}/reject`)
  },

  // 获取主机状态
  getHostStatus(id) {
    return request.get(`/api/v1/hosts/${id}/status`)
  }
}

export default hostsApi