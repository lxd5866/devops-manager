import { createRouter, createWebHistory } from 'vue-router'
import Layout from '@/layouts/Layout.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: Layout,
      redirect: '/hosts',
      children: [
        {
          path: '/hosts',
          name: 'Hosts',
          component: () => import('@/views/Hosts.vue'),
          meta: { title: '主机管理' }
        },
        {
          path: '/monitoring',
          name: 'Monitoring',
          component: () => import('@/views/Monitoring.vue'),
          meta: { title: '监控面板' }
        },
        {
          path: '/logs',
          name: 'Logs',
          component: () => import('@/views/Logs.vue'),
          meta: { title: '日志管理' }
        },
        {
          path: '/jobs',
          name: 'Jobs',
          redirect: '/jobs/history',
          meta: { title: '作业管理' },
          children: [
            {
              path: 'script-execution',
              name: 'ScriptExecution',
              component: () => import('@/views/jobs/ScriptExecution.vue'),
              meta: { title: '脚本执行' }
            },
            {
              path: 'history',
              name: 'JobHistory',
              component: () => import('@/views/jobs/JobHistory.vue'),
              meta: { title: '执行历史' }
            },
            {
              path: 'detail/:taskId',
              name: 'JobDetail',
              component: () => import('@/views/jobs/JobDetail.vue'),
              meta: { title: '任务详情' }
            }
          ]
        },
        {
          path: '/settings',
          name: 'Settings',
          component: () => import('@/views/Settings.vue'),
          meta: { title: '系统设置' }
        }
      ]
    }
  ]
})

export default router