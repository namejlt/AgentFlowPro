<template>
  <el-aside :width="collapsed ? '64px' : '220px'" class="side-nav">
    <div class="logo-area">
      <el-icon :size="28" color="#409eff"><Promotion /></el-icon>
      <span v-if="!collapsed" class="logo-text">AgentFlow Pro</span>
    </div>
    <el-menu
      :default-active="activeMenu"
      :collapse="collapsed"
      :collapse-transition="false"
      router
      class="side-menu"
      background-color="#1d1e2c"
      text-color="#b0b3c7"
      active-text-color="#409eff"
    >
      <el-menu-item index="/dashboard">
        <el-icon><DataBoard /></el-icon>
        <template #title>工作台</template>
      </el-menu-item>
      <el-menu-item index="/workflows">
        <el-icon><Share /></el-icon>
        <template #title>工作流</template>
      </el-menu-item>
      <el-menu-item index="/tasks">
        <el-icon><Monitor /></el-icon>
        <template #title>任务监控</template>
      </el-menu-item>
      <el-menu-item index="/agents">
        <el-icon><User /></el-icon>
        <template #title>智能体</template>
      </el-menu-item>
      <el-menu-item index="/datasources">
        <el-icon><Coin /></el-icon>
        <template #title>数据源</template>
      </el-menu-item>
      <el-menu-item index="/models">
        <el-icon><Cpu /></el-icon>
        <template #title>模型配置</template>
      </el-menu-item>
      <el-menu-item index="/reports">
        <el-icon><Document /></el-icon>
        <template #title>历史报告</template>
      </el-menu-item>
      <el-sub-menu v-if="isAdmin" index="settings-group">
        <template #title>
          <el-icon><Setting /></el-icon>
          <span>系统管理</span>
        </template>
        <el-menu-item index="/settings">全局配置</el-menu-item>
        <el-menu-item index="/settings/users">用户管理</el-menu-item>
        <el-menu-item index="/settings/audit">审计日志</el-menu-item>
      </el-sub-menu>
    </el-menu>
  </el-aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'

defineProps<{ collapsed: boolean }>()
defineEmits(['toggle'])

const route = useRoute()
const userStore = useUserStore()

const activeMenu = computed(() => route.path)
const isAdmin = computed(() => userStore.isAdmin())
</script>

<style scoped>
.side-nav {
  background: #1d1e2c;
  overflow: hidden;
  transition: width 0.2s;
}
.logo-area {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 56px;
  gap: 8px;
  border-bottom: 1px solid rgba(255,255,255,0.06);
}
.logo-text {
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  white-space: nowrap;
}
.side-menu {
  border-right: none;
}
.side-menu:not(.el-menu--collapse) {
  width: 220px;
}
</style>
