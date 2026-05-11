<template>
  <div class="page-container">
    <div class="page-header">
      <h2>审计日志</h2>
    </div>

    <el-card style="margin-bottom: 16px">
      <el-row :gutter="16" align="middle">
        <el-col :span="6">
          <el-input v-model="keyword" placeholder="搜索操作..." prefix-icon="Search" clearable @input="fetchList" />
        </el-col>
        <el-col :span="4">
          <el-select v-model="filterAction" placeholder="操作类型" clearable @change="fetchList">
            <el-option label="创建" value="create" />
            <el-option label="更新" value="update" />
            <el-option label="删除" value="delete" />
            <el-option label="登录" value="login" />
            <el-option label="执行" value="execute" />
          </el-select>
        </el-col>
      </el-row>
    </el-card>

    <el-card>
      <el-table :data="logs" stripe>
        <el-table-column label="时间" width="170">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="username" label="用户" width="120" />
        <el-table-column prop="action" label="操作" width="100" />
        <el-table-column prop="resource_type" label="资源类型" width="120" />
        <el-table-column prop="resource_id" label="资源ID" width="140" show-overflow-tooltip />
        <el-table-column label="详情" min-width="300">
          <template #default="{ row }">
            <span style="font-size: 12px; color: #606266">{{ row.detail ? JSON.stringify(row.detail).substring(0, 200) : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="IP" width="130" />
      </el-table>
    </el-card>

    <el-pagination
      v-if="total > pageSize"
      style="margin-top: 16px; justify-content: center"
      :current-page="page"
      :page-size="pageSize"
      :total="total"
      layout="prev, pager, next"
      @current-change="(p: number) => { page = p; fetchList() }"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { AuditLogItem } from '@/types'
import { getAuditLogs } from '@/api/system'
import { formatDateTime } from '@/utils/datetime'

const logs = ref<AuditLogItem[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const filterAction = ref('')

async function fetchList() {
  try {
    const res = await getAuditLogs({ page: page.value, page_size: pageSize.value, keyword: keyword.value, action: filterAction.value || undefined })
    logs.value = res.data.data || []
    total.value = res.data.meta?.total || 0
  } catch {}
}

onMounted(fetchList)
</script>
