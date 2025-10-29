<template>
  <a-layout style="min-height: 100vh">
    <!-- 侧边栏 -->
    <a-layout-sider v-model:collapsed="collapsed" :trigger="null" collapsible>
      <div class="logo">
        <h2 v-if="!collapsed" style="color: white; text-align: center; margin: 16px 0">
          DevOps 运维管理
        </h2>
        <h2 v-else style="color: white; text-align: center; margin: 16px 0">
          DevOps
        </h2>
      </div>
      
      <a-menu
        v-model:selectedKeys="selectedKeys"
        theme="dark"
        mode="inline"
        @click="handleMenuClick"
      >
        <a-menu-item key="/hosts">
          <desktop-outlined />
          <span>主机管理</span>
        </a-menu-item>
        
        <a-menu-item key="/monitoring">
          <dashboard-outlined />
          <span>监控面板</span>
        </a-menu-item>
        
        <a-sub-menu key="/jobs">
          <template #icon>
            <play-circle-outlined />
          </template>
          <template #title>作业管理</template>
          <a-menu-item key="/jobs/script-execution">脚本执行</a-menu-item>
          <a-menu-item key="/jobs/history">执行历史</a-menu-item>
        </a-sub-menu>
        
        <a-menu-item key="/logs">
          <file-text-outlined />
          <span>日志管理</span>
        </a-menu-item>
        
        <a-menu-item key="/settings">
          <setting-outlined />
          <span>系统设置</span>
        </a-menu-item>
      </a-menu>
    </a-layout-sider>

    <a-layout>
      <!-- 顶部导航栏 -->
      <a-layout-header style="background: #fff; padding: 0; box-shadow: 0 1px 4px rgba(0,21,41,.08)">
        <div style="display: flex; justify-content: space-between; align-items: center; padding: 0 24px">
          <div style="display: flex; align-items: center">
            <menu-unfold-outlined
              v-if="collapsed"
              class="trigger"
              @click="() => (collapsed = !collapsed)"
            />
            <menu-fold-outlined
              v-else
              class="trigger"
              @click="() => (collapsed = !collapsed)"
            />
            <h3 style="margin: 0 0 0 16px; color: #001529">{{ currentTitle }}</h3>
          </div>
          
          <div style="display: flex; align-items: center; gap: 16px">
            <span style="color: #666">{{ currentTime }}</span>
            <a-avatar style="background-color: #1890ff">管</a-avatar>
            <span>管理员</span>
          </div>
        </div>
      </a-layout-header>

      <!-- 主内容区域 -->
      <a-layout-content style="margin: 24px 16px; padding: 24px; background: #fff; min-height: 280px">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  MenuUnfoldOutlined,
  MenuFoldOutlined,
  DesktopOutlined,
  DashboardOutlined,
  FileTextOutlined,
  SettingOutlined,
  PlayCircleOutlined
} from '@ant-design/icons-vue'
import dayjs from 'dayjs'

const router = useRouter()
const route = useRoute()

const collapsed = ref(false)
const selectedKeys = ref([route.path])
const currentTime = ref('')

let timer = null

const currentTitle = computed(() => {
  return route.meta?.title || 'DevOps 运维管理系统'
})

const handleMenuClick = ({ key }) => {
  selectedKeys.value = [key]
  router.push(key)
}

const updateTime = () => {
  currentTime.value = dayjs().format('YYYY-MM-DD HH:mm:ss')
}

onMounted(() => {
  updateTime()
  timer = setInterval(updateTime, 1000)
})

onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
  }
})

// 监听路由变化
router.afterEach((to) => {
  selectedKeys.value = [to.path]
})
</script>

<style scoped>
.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.2);
  margin: 16px;
  border-radius: 6px;
}

.trigger {
  font-size: 18px;
  line-height: 64px;
  padding: 0 24px;
  cursor: pointer;
  transition: color 0.3s;
}

.trigger:hover {
  color: #1890ff;
}
</style>