<template>
  <div>
    <a-card>
      <a-empty description="系统设置功能正在开发中...">
        <template #image>
          <setting-outlined style="font-size: 64px; color: #ccc" />
        </template>
        <a-button type="primary">敬请期待</a-button>
      </a-empty>
    </a-card>
  </div>
</template>

<script setup>
import { SettingOutlined } from '@ant-design/icons-vue'
</script>