<template>
  <div class="page-container">
    <div class="page-header">
      <h2>任务监控</h2>
    </div>

    <el-card style="margin-bottom: 16px">
      <el-row :gutter="16" align="middle">
        <el-col :span="6">
          <el-input v-model="keyword" placeholder="搜索任务..." prefix-icon="Search" clearable @input="fetchList" />
        </el-col>
        <el-col :span="4">
          <el-select v-model="filterStatus" placeholder="状态筛选" clearable @change="fetchList">
            <el-option label="运行中" value="running" />
            <el-option label="已完成" value="completed" />
            <el-option label="失败" value="failed" />
            <el-option label="已停止" value="stopped" />
            <el-option label="等待中" value="pending" />
            <el-option label="排队中" value="queued" />
          </el-select>
        </el-col>
      </el-row>
    </el-card>

    <el-card v-loading="loading">
      <el-table :data="tasks" stripe>
        <el-table-column prop="workflow_name" label="工作流" min-width="140" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="mode" label="模式" width="80">
          <template #default="{ row }">{{ row.mode === 'debug' ? '调试' : '正常' }}</template>
        </el-table-column>
        <el-table-column label="耗时" width="100">
          <template #default="{ row }">{{ formatDuration(row.duration_ms) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="错误信息" min-width="160">
          <template #default="{ row }">
            <span v-if="row.error_message" class="error-text">{{ row.error_message }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="$router.push(`/tasks/${row.id}`)">监控</el-button>
            <el-button v-if="row.status === 'running'" link type="warning" @click="handleStop(row)">停止</el-button>
            <el-button v-if="row.status === 'completed' || row.status === 'failed'" link type="success" @click="handleRerun(row)">重跑</el-button>
          </template>
        </el-table-column>
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
import type { TaskItem } from '@/types'
import { getTasks, stopTask, rerunTask } from '@/api/tasks'
import { formatDateTime, formatDuration } from '@/utils/datetime'
import { ElMessage, ElMessageBox } from 'element-plus'

const tasks = ref<TaskItem[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const filterStatus = ref('')
const loading = ref(false)

function statusType(status: string) {
  const map: Record<string, string> = { completed: 'success', failed: 'danger', running: 'warning', pending: 'info', queued: 'info', stopped: 'info', paused: 'warning' }
  return map[status] || 'info'
}

function statusLabel(status: string) {
  const map: Record<string, string> = { completed: '已完成', failed: '失败', running: '运行中', pending: '等待中', queued: '排队中', stopped: '已停止', paused: '已暂停' }
  return map[status] || status
}

async function fetchList() {
  loading.value = true
  try {
    const res = await getTasks({ page: page.value, page_size: pageSize.value, keyword: keyword.value, status: filterStatus.value || undefined })
    tasks.value = res.data.data || []
    total.value = res.data.meta?.total || 0
  } catch (e: any) {
    ElMessage.error(e.message || '获取任务列表失败')
  } finally {
    loading.value = false
  }
}

async function handleStop(task: TaskItem) {
  await ElMessageBox.confirm('确定停止该任务？', '提示', { type: 'warning' })
  try {
    await stopTask(task.id)
    ElMessage.success('已停止')
    fetchList()
  } catch (e: any) {
    ElMessage.error(e.message || '停止失败')
  }
}

async function handleRerun(task: TaskItem) {
  try {
    await rerunTask(task.id)
    ElMessage.success('已重新执行')
    fetchList()
  } catch (e: any) {
    ElMessage.error(e.message || '重跑失败')
  }
}

onMounted(fetchList)
</script>

<style scoped>
.error-text {
  color: #f56c6c;
  font-size: 12px;
}
</style>
