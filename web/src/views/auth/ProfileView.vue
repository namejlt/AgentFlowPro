<template>
  <div class="page-container">
    <div class="page-header">
      <h2>个人资料</h2>
    </div>
    <el-card style="max-width: 600px">
      <el-descriptions :column="1" border>
        <el-descriptions-item label="用户名">{{ user?.username || '-' }}</el-descriptions-item>
        <el-descriptions-item label="邮箱">{{ user?.email || '-' }}</el-descriptions-item>
        <el-descriptions-item label="角色">
          <el-tag :type="roleTagType" size="small">{{ roleLabel }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="上次登录">{{ user?.last_login_at ? formatDateTime(user.last_login_at) : '-' }}</el-descriptions-item>
        <el-descriptions-item label="注册时间">{{ user?.created_at ? formatDateTime(user.created_at) : '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { formatDateTime } from '@/utils/datetime'

const userStore = useUserStore()
const user = computed(() => userStore.user)

const roleLabel = computed(() => {
  const role = user.value?.role
  if (role === 'admin') return '管理员'
  if (role === 'creator') return '创作者'
  return '普通用户'
})

const roleTagType = computed(() => {
  const role = user.value?.role
  if (role === 'admin') return 'danger'
  if (role === 'creator') return 'warning'
  return 'info'
})
</script>
