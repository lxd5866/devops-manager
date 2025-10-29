// DevOps Agent Web Interface JavaScript

// 页面加载完成后执行
document.addEventListener('DOMContentLoaded', function() {
    // 初始化页面
    initializePage();
    
    // 设置定时刷新
    if (window.location.pathname === '/status') {
        setInterval(refreshStatus, 30000); // 30秒刷新一次状态
    }
    
    if (window.location.pathname === '/tasks') {
        setInterval(refreshTasks, 10000); // 10秒刷新一次任务列表
    }
});

// 初始化页面
function initializePage() {
    // 添加表单提交处理
    const forms = document.querySelectorAll('form');
    forms.forEach(form => {
        form.addEventListener('submit', handleFormSubmit);
    });
    
    // 添加删除确认
    const deleteButtons = document.querySelectorAll('.btn-danger');
    deleteButtons.forEach(button => {
        button.addEventListener('click', handleDeleteConfirm);
    });
}

// 处理表单提交
function handleFormSubmit(event) {
    const form = event.target;
    const action = form.getAttribute('action');
    
    // 如果是任务执行表单，显示加载状态
    if (action && action.includes('/task/execute')) {
        const submitButton = form.querySelector('button[type="submit"]');
        if (submitButton) {
            submitButton.textContent = 'Executing...';
            submitButton.disabled = true;
        }
    }
}

// 处理删除确认
function handleDeleteConfirm(event) {
    const confirmed = confirm('Are you sure you want to delete this item?');
    if (!confirmed) {
        event.preventDefault();
        return false;
    }
}

// 刷新状态页面
function refreshStatus() {
    fetch('/api/v1/host/status')
        .then(response => response.json())
        .then(data => {
            if (data.error === false) {
                updateStatusDisplay(data.data);
            }
        })
        .catch(error => {
            console.error('Failed to refresh status:', error);
        });
}

// 更新状态显示
function updateStatusDisplay(status) {
    // 更新连接状态
    const connectedCell = document.querySelector('td:contains("Connected to Server:")');
    if (connectedCell) {
        const nextCell = connectedCell.nextElementSibling;
        if (nextCell) {
            nextCell.textContent = status.connected ? 'Yes' : 'No';
        }
    }
    
    // 更新运行任务数
    const tasksCell = document.querySelector('td:contains("Running Tasks:")');
    if (tasksCell) {
        const nextCell = tasksCell.nextElementSibling;
        if (nextCell) {
            nextCell.textContent = status.running_tasks || 0;
        }
    }
}

// 刷新任务列表
function refreshTasks() {
    fetch('/api/v1/task/list')
        .then(response => response.json())
        .then(data => {
            if (data.error === false) {
                updateTasksDisplay(data.data);
            }
        })
        .catch(error => {
            console.error('Failed to refresh tasks:', error);
        });
}

// 更新任务显示
function updateTasksDisplay(data) {
    const tbody = document.querySelector('table tbody');
    if (!tbody || !data.tasks) return;
    
    // 清空现有行
    tbody.innerHTML = '';
    
    // 添加新行
    data.tasks.forEach(task => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${task.task_id}</td>
            <td>${task.command}</td>
            <td><span class="status-${task.status}">${task.status}</span></td>
            <td>${new Date(task.start_time * 1000).toLocaleString()}</td>
            <td>${task.end_time ? new Date(task.end_time * 1000).toLocaleString() : '-'}</td>
            <td>
                <a href="/api/v1/task/status/${task.task_id}" class="btn">View</a>
                ${task.status === 'running' ? `<a href="/api/v1/task/cancel/${task.task_id}" class="btn btn-danger">Cancel</a>` : ''}
            </td>
        `;
        tbody.appendChild(row);
    });
}

// 工具函数：查找包含特定文本的元素
function findElementByText(text) {
    const elements = document.querySelectorAll('td');
    for (let element of elements) {
        if (element.textContent.includes(text)) {
            return element;
        }
    }
    return null;
}

// 显示通知消息
function showNotification(message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    
    // 添加样式
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 15px 20px;
        background-color: ${type === 'error' ? '#e74c3c' : type === 'success' ? '#27ae60' : '#3498db'};
        color: white;
        border-radius: 4px;
        box-shadow: 0 2px 4px rgba(0,0,0,0.2);
        z-index: 1000;
        opacity: 0;
        transition: opacity 0.3s ease;
    `;
    
    document.body.appendChild(notification);
    
    // 显示动画
    setTimeout(() => {
        notification.style.opacity = '1';
    }, 100);
    
    // 自动隐藏
    setTimeout(() => {
        notification.style.opacity = '0';
        setTimeout(() => {
            document.body.removeChild(notification);
        }, 300);
    }, 3000);
}