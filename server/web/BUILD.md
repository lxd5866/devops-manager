# DevOps 运维管理系统 - Web 前端构建说明

## 项目结构

```
server/web/
├── frontend/          # Vue3 + Ant Design 前端源码
│   ├── src/
│   ├── package.json
│   ├── vite.config.js
│   └── README.md
├── dist/              # 构建输出目录
│   └── index.html     # 当前使用的简化版本
├── static/            # 旧版静态文件（向后兼容）
└── templates/         # 旧版模板文件（向后兼容）
```

## 当前状态

目前系统使用的是 `dist/index.html` 中的简化版前端，它是一个单文件的 Vue3 应用，通过 CDN 加载依赖。

### 功能特性

✅ **已实现功能**：
- 实时主机列表显示
- 主机状态监控（在线/离线）
- 统计卡片（总数、在线数、离线数）
- 自动刷新（30秒间隔）
- 响应式设计
- 现代化 UI

🚧 **开发中功能**：
- 完整的 Vue3 + Ant Design 应用
- 路由管理
- 状态管理（Pinia）
- 组件化架构

## 开发环境设置

### 1. 安装 Node.js 依赖

```bash
cd server/web/frontend
npm install
```

### 2. 启动开发服务器

```bash
npm run dev
```

开发服务器将在 http://localhost:3000 启动，API 请求会自动代理到 http://localhost:8080

### 3. 构建生产版本

```bash
npm run build
```

构建文件将输出到 `../dist` 目录，替换当前的简化版本。

## 技术栈

### 当前版本（简化版）
- **Vue 3** (CDN)
- **原生 JavaScript**
- **CSS Grid/Flexbox**
- **Fetch API**

### 完整版本（开发中）
- **Vue 3** + **Composition API**
- **Ant Design Vue** - 企业级 UI 组件库
- **Vue Router** - 路由管理
- **Pinia** - 状态管理
- **Vite** - 构建工具
- **Axios** - HTTP 客户端

## API 接口

前端通过以下 RESTful API 与后端通信：

```
GET    /api/v1/hosts           # 获取主机列表
GET    /api/v1/hosts/:id       # 获取主机详情
POST   /api/v1/hosts/register  # 注册主机
PUT    /api/v1/hosts/:id       # 更新主机
DELETE /api/v1/hosts/:id       # 删除主机
```

## 部署说明

### 开发环境
1. 启动后端服务器：`go run server/cmd/main.go`
2. 访问 http://localhost:8080 查看 Web 界面

### 生产环境
1. 构建前端：`cd server/web/frontend && npm run build`
2. 启动后端服务器
3. 前端文件会自动从 `dist` 目录提供服务

## 下一步开发计划

1. **完善 Vue3 应用**：
   - 完成所有页面组件
   - 实现路由切换
   - 添加状态管理

2. **功能扩展**：
   - 主机详情页面
   - 监控图表
   - 日志查看
   - 系统设置

3. **用户体验优化**：
   - 加载状态
   - 错误处理
   - 通知提醒
   - 主题切换

4. **性能优化**：
   - 代码分割
   - 懒加载
   - 缓存策略
   - PWA 支持