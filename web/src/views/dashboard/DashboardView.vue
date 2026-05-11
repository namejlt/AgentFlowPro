<template>
  <div class="page-container">
    <div class="page-header">
      <h2>工作台</h2>
      <el-button type="primary" @click="$router.push('/workflows')">
        <el-icon><Plus /></el-icon>创建工作流
      </el-button>
    </div>

    <el-row :gutter="16" class="stat-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-value">{{ stats.workflow_count }}</div>
          <div class="stat-label">工作流总数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-value" style="color: #67c23a">{{ stats.agent_count }}</div>
          <div class="stat-label">智能体总数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-value" style="color: #e6a23c">{{ stats.task_running_count }}</div>
          <div class="stat-label">运行中任务</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-value" style="color: #f56c6c">{{ stats.success_rate_24h }}%</div>
          <div class="stat-label">24h 成功率</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top: 20px">
      <el-col :span="16">
        <el-card>
          <template #header>
            <span>最近任务</span>
          </template>
          <el-table :data="stats.recent_tasks" stripe style="width: 100%">
            <el-table-column prop="workflow_name" label="工作流" min-width="120" />
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="耗时" width="100">
              <template #default="{ row }">{{ formatDuration(row.duration_ms) }}</template>
            </el-table-column>
            <el-table-column label="创建时间" width="170">
              <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button link type="primary" @click="$router.push(`/tasks/${row.id}`)">查看</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card>
          <template #header>
            <span>快捷操作</span>
          </template>
          <div class="quick-actions">
            <el-button class="quick-btn" @click="$router.push('/workflows')">
              <el-icon :size="24"><Share /></el-icon>
              <span>工作流管理</span>
            </el-button>
            <el-button class="quick-btn" @click="$router.push('/agents')">
              <el-icon :size="24"><User /></el-icon>
              <span>智能体管理</span>
            </el-button>
            <el-button class="quick-btn" @click="$router.push('/datasources')">
              <el-icon :size="24"><Coin /></el-icon>
              <span>数据源管理</span>
            </el-button>
            <el-button class="quick-btn" @click="$router.push('/models')">
              <el-icon :size="24"><Cpu /></el-icon>
              <span>模型配置</span>
            </el-button>
            <el-button class="quick-btn" @click="$router.push('/reports')">
              <el-icon :size="24"><Document /></el-icon>
              <span>历史报告</span>
            </el-button>
            <el-button class="quick-btn" @click="$router.push('/tasks')">
              <el-icon :size="24"><Monitor /></el-icon>
              <span>任务监控</span>
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getDashboard } from '@/api/system'
import type { DashboardStats, TaskItem } from '@/types'
import { formatDateTime, formatDuration } from '@/utils/datetime'

const stats = ref<DashboardStats>({
  workflow_count: 0,
  agent_count: 0,
  task_running_count: 0,
  report_count: 0,
  success_rate_24h: 0,
  recent_tasks: [],
})

function statusType(status: string) {
  const map: Record<string, string> = { completed: 'success', failed: 'danger', running: 'warning', pending: 'info', queued: 'info', stopped: 'info', paused: 'warning' }
  return map[status] || 'info'
}

function statusLabel(status: string) {
  const map: Record<string, string> = { completed: '已完成', failed: '失败', running: '运行中', pending: '等待中', queued: '排队中', stopped: '已停止', paused: '已暂停' }
  return map[status] || status
}

onMounted(async () => {
  try {
    const res = await getDashboard()
    stats.value = res.data.data
  } catch {}
})
</script>

<style scoped>
.stat-row {
  margin-bottom: 0;
}
.stat-card {
  text-align: center;
  padding: 10px 0;
}
.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: #409eff;
}
.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 4px;
}
.quick-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.quick-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 80px;
  gap: 8px;
}
</style>
