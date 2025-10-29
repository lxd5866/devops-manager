// DevOps 运维管理系统 - 主要 JavaScript 文件

class DevOpsManager {
    constructor() {
        this.init();
    }

    init() {
        this.setupSidebar();
        this.setupHostsTable();
        this.setupRealTimeUpdates();
    }

    setupSidebar() {
        // 侧边栏菜单切换
        const sidebarToggle = document.getElementById('sidebar-toggle');
        const sidebar = document.getElementById('sidebar');
        
        if (sidebarToggle && sidebar) {
            sidebarToggle.addEventListener('click', () => {
                sidebar.classList.toggle('-translate-x-full');
            });
        }

        // 菜单项激活状态
        const menuItems = document.querySelectorAll('.sidebar-menu-item');
        menuItems.forEach(item => {
            item.addEventListener('click', (e) => {
                menuItems.forEach(mi => mi.classList.remove('active'));
                e.currentTarget.classList.add('active');
            });
        });
    }

    setupHostsTable() {
        this.loadHosts();
        
        // 刷新按钮
        const refreshBtn = document.getElementById('refresh-hosts');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.loadHosts();
            });
        }
    }

    async loadHosts() {
        try {
            const response = await fetch('/api/v1/hosts');
            const result = await response.json();
            
            if (result.success) {
                this.renderHostsTable(result.data || []);
            } else {
                console.error('Failed to load hosts:', result.error_message);
            }
        } catch (error) {
            console.error('Error loading hosts:', error);
        }
    }

    renderHostsTable(hosts) {
        const tbody = document.getElementById('hosts-table-body');
        if (!tbody) return;

        tbody.innerHTML = '';

        hosts.forEach(host => {
            const row = document.createElement('tr');
            row.className = 'hover:bg-gray-50';
            
            const lastSeen = new Date(host.last_seen * 1000);
            const isOnline = (Date.now() - lastSeen.getTime()) < 60000; // 1分钟内为在线
            
            // 获取系统状态信息
            this.loadHostStatus(host.id).then(status => {
                const systemStatusHtml = this.formatSystemStatus(status);
                const statusCell = row.querySelector('.system-status-cell');
                if (statusCell) {
                    statusCell.innerHTML = systemStatusHtml;
                }
            });
            
            row.innerHTML = `
                <td class="px-6 py-4 whitespace-nowrap">
                    <div class="flex items-center">
                        <div class="flex-shrink-0 h-10 w-10">
                            <div class="h-10 w-10 rounded-full bg-gray-300 flex items-center justify-center">
                                <svg class="h-6 w-6 text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                                </svg>
                            </div>
                        </div>
                        <div class="ml-4">
                            <div class="text-sm font-medium text-gray-900">${host.hostname}</div>
                            <div class="text-sm text-gray-500">${host.id}</div>
                        </div>
                    </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <span class="status-badge ${isOnline ? 'status-online' : 'status-offline'}">
                        ${isOnline ? '在线' : '离线'}
                    </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 system-status-cell">
                    <div class="animate-pulse">加载中...</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${host.ip}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${host.os}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    ${this.formatTags(host.tags)}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    ${this.formatDateTime(lastSeen)}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button class="text-blue-600 hover:text-blue-900 mr-3" onclick="devopsManager.viewHost('${host.id}')">查看</button>
                    <button class="text-green-600 hover:text-green-900 mr-3" onclick="devopsManager.viewHostStatus('${host.id}')">状态</button>
                    <button class="text-red-600 hover:text-red-900" onclick="devopsManager.deleteHost('${host.id}')">删除</button>
                </td>
            `;
            
            tbody.appendChild(row);
        });

        // 更新统计信息
        this.updateStats(hosts);
    }

    formatTags(tags) {
        if (!tags) return '-';
        
        const tagArray = Object.entries(tags).map(([key, value]) => `${key}: ${value}`);
        return tagArray.slice(0, 2).join(', ') + (tagArray.length > 2 ? '...' : '');
    }

    formatDateTime(date) {
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
    }

    updateStats(hosts) {
        const totalHosts = hosts.length;
        const onlineHosts = hosts.filter(host => {
            const lastSeen = new Date(host.last_seen * 1000);
            return (Date.now() - lastSeen.getTime()) < 60000;
        }).length;
        const offlineHosts = totalHosts - onlineHosts;

        document.getElementById('total-hosts').textContent = totalHosts;
        document.getElementById('online-hosts').textContent = onlineHosts;
        document.getElementById('offline-hosts').textContent = offlineHosts;
    }

    setupRealTimeUpdates() {
        // 每30秒自动刷新数据
        setInterval(() => {
            this.loadHosts();
        }, 30000);
    }

    async viewHost(hostId) {
        try {
            const response = await fetch(`/api/v1/hosts/${hostId}`);
            const result = await response.json();
            
            if (result.success) {
                this.showHostDetails(result.data);
            } else {
                alert('获取主机详情失败: ' + result.error_message);
            }
        } catch (error) {
            alert('获取主机详情失败: ' + error.message);
        }
    }

    showHostDetails(host) {
        const modal = document.getElementById('host-details-modal');
        const content = document.getElementById('host-details-content');
        
        if (modal && content) {
            content.innerHTML = `
                <div class="space-y-4">
                    <div>
                        <label class="block text-sm font-medium text-gray-700">主机ID</label>
                        <p class="mt-1 text-sm text-gray-900">${host.id}</p>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">主机名</label>
                        <p class="mt-1 text-sm text-gray-900">${host.hostname}</p>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">IP地址</label>
                        <p class="mt-1 text-sm text-gray-900">${host.ip}</p>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">操作系统</label>
                        <p class="mt-1 text-sm text-gray-900">${host.os}</p>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">标签</label>
                        <div class="mt-1 space-y-1">
                            ${Object.entries(host.tags || {}).map(([key, value]) => 
                                `<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">${key}: ${value}</span>`
                            ).join(' ')}
                        </div>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700">最后上报时间</label>
                        <p class="mt-1 text-sm text-gray-900">${this.formatDateTime(new Date(host.last_seen * 1000))}</p>
                    </div>
                </div>
            `;
            
            modal.classList.remove('hidden');
        }
    }

    closeModal() {
        const modal = document.getElementById('host-details-modal');
        if (modal) {
            modal.classList.add('hidden');
        }
    }

    async loadHostStatus(hostId) {
        try {
            const response = await fetch(`/api/v1/hosts/${hostId}/status`);
            const result = await response.json();
            
            if (result.success) {
                return result.data;
            }
        } catch (error) {
            console.error('Error loading host status:', error);
        }
        return null;
    }

    formatSystemStatus(status) {
        if (!status) {
            return '<span class="text-gray-400">无数据</span>';
        }

        const cpuUsage = status.cpu ? status.cpu.usage_percent.toFixed(1) : 'N/A';
        const memUsage = status.memory ? status.memory.usage_percent.toFixed(1) : 'N/A';
        const healthStatus = status.health_status || 'unknown';
        
        const healthColor = healthStatus === 'healthy' ? 'text-green-600' : 
                           healthStatus === 'warning' ? 'text-yellow-600' : 'text-red-600';

        return `
            <div class="space-y-1">
                <div class="flex items-center space-x-2">
                    <span class="text-xs text-gray-500">CPU:</span>
                    <span class="text-xs font-medium">${cpuUsage}%</span>
                </div>
                <div class="flex items-center space-x-2">
                    <span class="text-xs text-gray-500">内存:</span>
                    <span class="text-xs font-medium">${memUsage}%</span>
                </div>
                <div class="flex items-center space-x-2">
                    <span class="text-xs text-gray-500">健康:</span>
                    <span class="text-xs font-medium ${healthColor}">${healthStatus}</span>
                </div>
            </div>
        `;
    }

    async viewHostStatus(hostId) {
        try {
            const [hostResponse, statusResponse] = await Promise.all([
                fetch(`/api/v1/hosts/${hostId}`),
                fetch(`/api/v1/hosts/${hostId}/status`)
            ]);
            
            const hostResult = await hostResponse.json();
            const statusResult = await statusResponse.json();
            
            if (hostResult.success) {
                this.showHostStatusDetails(hostResult.data, statusResult.success ? statusResult.data : null);
            } else {
                alert('获取主机信息失败: ' + hostResult.error_message);
            }
        } catch (error) {
            alert('获取主机状态失败: ' + error.message);
        }
    }

    showHostStatusDetails(host, status) {
        const modal = document.getElementById('host-status-modal') || this.createStatusModal();
        const content = document.getElementById('host-status-content');
        
        if (modal && content) {
            content.innerHTML = this.renderHostStatusContent(host, status);
            modal.classList.remove('hidden');
        }
    }

    createStatusModal() {
        const modal = document.createElement('div');
        modal.id = 'host-status-modal';
        modal.className = 'fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50 hidden';
        modal.innerHTML = `
            <div class="relative top-20 mx-auto p-5 border w-11/12 md:w-3/4 lg:w-1/2 shadow-lg rounded-md bg-white">
                <div class="flex items-center justify-between mb-4">
                    <h3 class="text-lg font-medium text-gray-900">主机状态详情</h3>
                    <button onclick="devopsManager.closeStatusModal()" class="text-gray-400 hover:text-gray-600">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                        </svg>
                    </button>
                </div>
                <div id="host-status-content"></div>
            </div>
        `;
        document.body.appendChild(modal);
        return modal;
    }

    renderHostStatusContent(host, status) {
        if (!status) {
            return `
                <div class="text-center py-8">
                    <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                    </svg>
                    <h3 class="mt-2 text-sm font-medium text-gray-900">暂无状态数据</h3>
                    <p class="mt-1 text-sm text-gray-500">主机 ${host.hostname} 还没有上报状态信息。</p>
                </div>
            `;
        }

        const lastUpdate = new Date(status.timestamp * 1000);
        const uptime = this.formatUptime(status.uptime_seconds);

        return `
            <div class="space-y-6">
                <!-- 基本信息 -->
                <div class="bg-gray-50 p-4 rounded-lg">
                    <h4 class="text-sm font-medium text-gray-900 mb-3">基本信息</h4>
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <span class="text-sm text-gray-500">主机名:</span>
                            <span class="ml-2 text-sm font-medium">${host.hostname}</span>
                        </div>
                        <div>
                            <span class="text-sm text-gray-500">运行时间:</span>
                            <span class="ml-2 text-sm font-medium">${uptime}</span>
                        </div>
                        <div>
                            <span class="text-sm text-gray-500">IP地址:</span>
                            <span class="ml-2 text-sm font-medium">${status.ip || host.ip}</span>
                        </div>
                        <div>
                            <span class="text-sm text-gray-500">最后更新:</span>
                            <span class="ml-2 text-sm font-medium">${this.formatDateTime(lastUpdate)}</span>
                        </div>
                    </div>
                </div>

                <!-- CPU 信息 -->
                ${status.cpu ? `
                <div class="bg-blue-50 p-4 rounded-lg">
                    <h4 class="text-sm font-medium text-gray-900 mb-3">CPU 信息</h4>
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <span class="text-sm text-gray-500">使用率:</span>
                            <span class="ml-2 text-sm font-medium">${status.cpu.usage_percent.toFixed(1)}%</span>
                        </div>
                        <div>
                            <span class="text-sm text-gray-500">核心数:</span>
                            <span class="ml-2 text-sm font-medium">${status.cpu.core_count}</span>
                        </div>
                        ${status.cpu.load_avg_1m !== undefined ? `
                        <div>
                            <span class="text-sm text-gray-500">负载(1m):</span>
                            <span class="ml-2 text-sm font-medium">${status.cpu.load_avg_1m.toFixed(2)}</span>
                        </div>
                        <div>
                            <span class="text-sm text-gray-500">负载(5m):</span>
                            <span class="ml-2 text-sm font-medium">${status.cpu.load_avg_5m.toFixed(2)}</span>
                        </div>
                        ` : ''}
                    </div>
                </div>
                ` : ''}

                <!-- 内存信息 -->
                ${status.memory ? `
                <div class="bg-green-50 p-4 rounded-lg">
                    <h4 class="text-sm font-medium text-gray-900 mb-3">内存信息</h4>
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <span class="text-sm text-gray-500">使用率:</span>
                            <span class="ml-2 text-sm font-medium">${status.memory.usage_percent.toFixed(1)}%</span>
                        </div>
                        <div>
                            <span class="text-sm text-gray-500">总内存:</span>
                            <span class="ml-2 text-sm font-medium">${this.formatBytes(status.memory.total_bytes)}</span>
                        </div>
                        <div>
                            <span class="text-sm text-gray-500">已使用:</span>
                            <span class="ml-2 text-sm font-medium">${this.formatBytes(status.memory.used_bytes)}</span>
                        </div>
                        <div>
                            <span class="text-sm text-gray-500">可用:</span>
                            <span class="ml-2 text-sm font-medium">${this.formatBytes(status.memory.total_bytes - status.memory.used_bytes)}</span>
                        </div>
                    </div>
                </div>
                ` : ''}

                <!-- 磁盘信息 -->
                ${status.disks && status.disks.length > 0 ? `
                <div class="bg-yellow-50 p-4 rounded-lg">
                    <h4 class="text-sm font-medium text-gray-900 mb-3">磁盘信息</h4>
                    ${status.disks.map(disk => `
                        <div class="mb-3 last:mb-0">
                            <div class="text-sm font-medium text-gray-700">${disk.mount_point}</div>
                            <div class="grid grid-cols-3 gap-4 mt-1">
                                <div>
                                    <span class="text-xs text-gray-500">使用率:</span>
                                    <span class="ml-1 text-xs font-medium">${disk.usage_percent.toFixed(1)}%</span>
                                </div>
                                <div>
                                    <span class="text-xs text-gray-500">总容量:</span>
                                    <span class="ml-1 text-xs font-medium">${this.formatBytes(disk.total_bytes)}</span>
                                </div>
                                <div>
                                    <span class="text-xs text-gray-500">可用:</span>
                                    <span class="ml-1 text-xs font-medium">${this.formatBytes(disk.free_bytes)}</span>
                                </div>
                            </div>
                        </div>
                    `).join('')}
                </div>
                ` : ''}



                <!-- 自定义标签 -->
                ${status.custom_tags && Object.keys(status.custom_tags).length > 0 ? `
                <div class="bg-gray-50 p-4 rounded-lg">
                    <h4 class="text-sm font-medium text-gray-900 mb-3">自定义标签</h4>
                    <div class="flex flex-wrap gap-2">
                        ${Object.entries(status.custom_tags).map(([key, value]) => 
                            `<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">${key}: ${value}</span>`
                        ).join('')}
                    </div>
                </div>
                ` : ''}
            </div>
        `;
    }

    formatUptime(seconds) {
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        
        if (days > 0) {
            return `${days}天 ${hours}小时 ${minutes}分钟`;
        } else if (hours > 0) {
            return `${hours}小时 ${minutes}分钟`;
        } else {
            return `${minutes}分钟`;
        }
    }

    formatBytes(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    closeStatusModal() {
        const modal = document.getElementById('host-status-modal');
        if (modal) {
            modal.classList.add('hidden');
        }
    }

    async deleteHost(hostId) {
        if (!confirm('确定要删除这台主机吗？')) {
            return;
        }

        try {
            const response = await fetch(`/api/v1/hosts/${hostId}`, {
                method: 'DELETE'
            });
            const result = await response.json();
            
            if (result.success) {
                this.loadHosts(); // 重新加载列表
            } else {
                alert('删除失败: ' + result.error_message);
            }
        } catch (error) {
            alert('删除失败: ' + error.message);
        }
    }
}

// 初始化应用
let devopsManager;
document.addEventListener('DOMContentLoaded', () => {
    devopsManager = new DevOpsManager();
});