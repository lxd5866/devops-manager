# DevOps 运维管理系统 - 前端

基于 Vue3 + Ant Design Vue 的现代化运维管理系统前端。

## 技术栈

- **Vue 3** - 渐进式 JavaScript 框架
- **Ant Design Vue** - 企业级 UI 组件库
- **Vue Router** - 官方路由管理器
- **Pinia** - 状态管理
- **Vite** - 现代化构建工具
- **Axios** - HTTP 客户端

## 开发环境设置

### 安装依赖

```bash
cd server/web/frontend
npm install
```

### 启动开发服务器

```bash
npm run dev
```

开发服务器将在 http://localhost:3000 启动，并自动代理 API 请求到后端服务器。

### 构建生产版本

```bash
npm run build
```

构建文件将输出到 `../dist` 目录。

## 项目结构

```
src/
├── api/           # API 接口
├── components/    # 公共组件
├── layouts/       # 布局组件
├── router/        # 路由配置
├── stores/        # 状态管理
├── views/         # 页面组件
├── App.vue        # 根组件
└── main.js        # 入口文件
```

## 功能特性

- 📊 实时主机监控
- 🖥️ 主机管理
- 📈 监控面板
- 📝 日志管理
- ⚙️ 系统设置
- 📱 响应式设计
- 🔄 自动刷新

## API 接口

前端通过 RESTful API 与后端通信：

- `GET /api/v1/hosts` - 获取主机列表
- `GET /api/v1/hosts/:id` - 获取主机详情
- `DELETE /api/v1/hosts/:id` - 删除主机
- `PUT /api/v1/hosts/:id` - 更新主机信息

## 开发说明

1. 确保后端服务器在 http://localhost:8080 运行
2. 前端开发服务器会自动代理 API 请求
3. 修改代码后会自动热重载
4. 构建前请确保所有功能正常工作