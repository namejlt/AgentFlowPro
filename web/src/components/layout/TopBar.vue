<template>
  <el-header class="top-bar" height="56px">
    <div class="top-bar-left">
      <el-icon class="toggle-btn" @click="$emit('toggle-nav')"><Fold /></el-icon>
      <el-breadcrumb separator="/">
        <el-breadcrumb-item v-for="item in breadcrumbs" :key="item.path" :to="item.path">
          {{ item.title }}
        </el-breadcrumb-item>
      </el-breadcrumb>
    </div>
    <div class="top-bar-right">
      <el-input
        v-model="searchKeyword"
        placeholder="全局搜索..."
        prefix-icon="Search"
        size="default"
        style="width: 220px"
        clearable
        @keyup.enter="handleSearch"
      />
      <el-badge :value="taskRunningCount" :max="99" :hidden="taskRunningCount === 0" class="notify-badge" @click="handleNotifications">
        <el-icon :size="20"><Bell /></el-icon>
      </el-badge>
      <el-dropdown trigger="click" @command="handleUserCommand">
        <span class="user-menu">
          <el-avatar :size="32" :src="userStore.user?.avatar_url">
            {{ userStore.user?.username?.charAt(0) || 'U' }}
          </el-avatar>
          <span class="username">{{ userStore.user?.username || '用户' }}</span>
          <el-icon><ArrowDown /></el-icon>
        </span>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="profile">个人资料</el-dropdown-item>
            <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </el-header>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'

defineEmits(['toggle-nav'])

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const searchKeyword = ref('')
const taskRunningCount = ref(0)

const BREADCRUMB_MAP: Record<string, string> = {
  dashboard: '工作台',
  workflows: '工作流',
  edit: '编辑',
  tasks: '任务监控',
  agents: '智能体',
  datasources: '数据源',
  models: '模型配置',
  reports: '历史报告',
  settings: '系统配置',
  users: '用户管理',
  audit: '审计日志',
}

const breadcrumbs = computed(() => {
  const items = route.path.split('/').filter(Boolean)
  return items.map((seg, i) => ({
    path: '/' + items.slice(0, i + 1).join('/'),
    title: BREADCRUMB_MAP[seg] || seg,
  }))
})

function handleSearch() {
  if (searchKeyword.value.trim()) {
    router.push({ path: '/workflows', query: { keyword: searchKeyword.value } })
  }
}

async function handleUserCommand(cmd: string) {
  if (cmd === 'profile') {
    router.push('/profile')
  } else if (cmd === 'logout') {
    await userStore.logout()
  }
}

function handleNotifications() {
  router.push('/tasks')
}
</script>

<style scoped>
.top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  padding: 0 20px;
}
.top-bar-left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.toggle-btn {
  cursor: pointer;
  font-size: 20px;
  color: #606266;
}
.top-bar-right {
  display: flex;
  align-items: center;
  gap: 16px;
}
.notify-badge {
  cursor: pointer;
}
.user-menu {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}
.username {
  font-size: 14px;
  color: #303133;
}
</style>
