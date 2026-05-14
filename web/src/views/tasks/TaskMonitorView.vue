<template>
  <div class="task-monitor">
    <div class="monitor-toolbar">
      <div class="toolbar-left">
        <el-button @click="$router.push('/tasks')"><el-icon><Back /></el-icon>返回</el-button>
        <h3>{{ task.workflow_name || '任务监控' }}</h3>
        <el-tag :type="statusType(task.status)" size="large">{{ statusLabel(task.status) }}</el-tag>
      </div>
      <div class="toolbar-right">
        <el-button v-if="task.status === 'running'" type="warning" @click="handleStop">
          <el-icon><VideoPause /></el-icon>停止
        </el-button>
        <el-button v-if="task.status === 'completed' || task.status === 'failed'" type="success" @click="handleRerun">
          <el-icon><RefreshRight /></el-icon>重跑
        </el-button>
        <el-button v-if="task.report_id" type="primary" @click="$router.push(`/reports/${task.report_id}`)">
          <el-icon><Document /></el-icon>查看报告
        </el-button>
      </div>
    </div>

    <div class="monitor-body">
      <div class="monitor-left">
        <el-card>
          <template #header><span>执行流程</span></template>
          <div class="mini-dag">
            <div
              v-for="step in steps"
              :key="step.id"
              class="step-item"
              :class="stepStatusClass(step.status)"
            >
              <div class="step-icon">
                <el-icon v-if="step.status === 'completed'" color="#67c23a"><CircleCheck /></el-icon>
                <el-icon v-else-if="step.status === 'running'" color="#e6a23c" class="is-loading"><Loading /></el-icon>
                <el-icon v-else-if="step.status === 'failed'" color="#f56c6c"><CircleClose /></el-icon>
                <el-icon v-else color="#c0c4cc"><Clock /></el-icon>
              </div>
              <div class="step-info">
                <div class="step-name">{{ step.node_type }} {{ step.agent_name ? `- ${step.agent_name}` : '' }}</div>
                <div class="step-meta">
                  <span v-if="step.debate_round">第{{ step.debate_round }}轮</span>
                  <span v-if="step.tokens_used">{{ step.tokens_used }} tokens</span>
                </div>
              </div>
            </div>
          </div>
        </el-card>
      </div>

      <div class="monitor-right">
        <el-tabs v-model="activeTab">
          <el-tab-pane label="实时输出" name="output">
            <div class="output-area">
              <div v-for="(state, key) in streamStore.nodeStates" :key="key" class="output-block">
                <div class="output-header">
                  <el-tag size="small">{{ state.agentName || key }}</el-tag>
                  <el-tag :type="state.status === 'completed' ? 'success' : state.status === 'running' ? 'warning' : 'info'" size="small">
                    {{ state.status === 'completed' ? '完成' : state.status === 'running' ? '输出中...' : state.status }}
                  </el-tag>
                </div>
                <div class="output-content markdown-body" v-html="renderMarkdown(state.output)" />
              </div>
              <el-empty v-if="Object.keys(streamStore.nodeStates).length === 0" description="等待输出..." />
            </div>
          </el-tab-pane>

          <el-tab-pane label="辩论过程" name="debate">
            <div class="debate-area">
              <div v-for="log in debateLogs" :key="log.round + log.agent_id" class="debate-bubble" :class="log.round % 2 === 0 ? 'left' : 'right'">
                <div class="agent-name">{{ log.agent_name }} <span class="round-tag">第{{ log.round }}轮</span></div>
                <div class="debate-content markdown-body" v-html="renderMarkdown(log.output)" />
              </div>
              <el-empty v-if="debateLogs.length === 0" description="暂无辩论记录" />
            </div>
          </el-tab-pane>

          <el-tab-pane label="风险评审" name="risk">
            <div class="risk-area">
              <el-table :data="riskReviews" stripe>
                <el-table-column prop="dimension" label="维度" width="120" />
                <el-table-column prop="level" label="等级" width="100">
                  <template #default="{ row }">
                    <el-tag :type="row.level === 'critical' ? 'danger' : row.level === 'high' ? 'warning' : row.level === 'medium' ? '' : 'success'" size="small">
                      {{ row.level }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="summary" label="摘要" min-width="300" />
              </el-table>
              <el-empty v-if="riskReviews.length === 0" description="暂无风险评审" />
            </div>
          </el-tab-pane>

          <el-tab-pane label="执行日志" name="logs">
            <div class="log-area">
              <div v-for="(log, idx) in streamStore.logs" :key="idx" class="log-item">
                <span class="log-time">{{ formatDateTime(log.timestamp) }}</span>
                <el-tag size="small" :type="log.event === 'task_failed' ? 'danger' : log.event === 'task_complete' ? 'success' : 'info'">{{ log.event }}</el-tag>
                <span class="log-data">{{ JSON.stringify(log.data) }}</span>
              </div>
              <el-empty v-if="streamStore.logs.length === 0" description="暂无日志" />
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import type { TaskItem, TaskStepItem, DebateLog, RiskReview } from '@/types'
import { getTask, getTaskSteps, stopTask, rerunTask } from '@/api/tasks'
import { useTaskStreamStore } from '@/stores/taskStream'
import { formatDateTime } from '@/utils/datetime'
import { marked } from 'marked'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const streamStore = useTaskStreamStore()

const task = ref<TaskItem>({} as TaskItem)
const steps = ref<TaskStepItem[]>([])
const activeTab = ref('output')

const debateLogs = computed<DebateLog[]>(() => {
  const result: DebateLog[] = []
  streamStore.logs.filter(l => l.event === 'debate_round').forEach(l => {
    const outputs = l.data?.agent_outputs
    if (outputs) {
      if (Array.isArray(outputs)) {
        result.push(...outputs.map((a: any) => ({
          round: a.round || l.data.round,
          agent_id: a.agent_id || '',
          agent_name: a.agent_name || a.agent_id || '',
          output: a.output || '',
          timestamp: l.timestamp,
        })))
      } else if (typeof outputs === 'object') {
        for (const [agentId, output] of Object.entries(outputs)) {
          result.push({
            round: l.data.round,
            agent_id: agentId,
            agent_name: agentId,
            output: output as string,
            timestamp: l.timestamp,
          })
        }
      }
    }
  })
  return result
})

const riskReviews = computed<RiskReview[]>(() => streamStore.riskItems)

function statusType(status: string) {
  const map: Record<string, string> = { completed: 'success', failed: 'danger', running: 'warning', pending: 'info', queued: 'info', stopped: 'info', paused: 'warning' }
  return map[status] || 'info'
}

function statusLabel(status: string) {
  const map: Record<string, string> = { completed: '已完成', failed: '失败', running: '运行中', pending: '等待中', queued: '排队中', stopped: '已停止', paused: '已暂停' }
  return map[status] || status
}

function stepStatusClass(status: string) {
  return `step-${status}`
}

function renderMarkdown(content: string) {
  if (!content) return ''
  return marked(content)
}

async function fetchTask() {
  const id = route.params.id as string
  try {
    const res = await getTask(id)
    task.value = res.data.data
  } catch {}
}

async function fetchSteps() {
  const id = route.params.id as string
  try {
    const res = await getTaskSteps(id)
    steps.value = res.data.data || []
  } catch {}
}

async function handleStop() {
  await ElMessageBox.confirm('确定停止该任务？', '提示', { type: 'warning' })
  try {
    await stopTask(task.value.id)
    ElMessage.success('已停止')
    fetchTask()
  } catch {}
}

async function handleRerun() {
  try {
    const res = await rerunTask(task.value.id)
    ElMessage.success('已重新执行')
    streamStore.connect(res.data.data.id)
    fetchTask()
  } catch {}
}

onMounted(async () => {
  await fetchTask()
  await fetchSteps()
  if (task.value.status === 'running' || task.value.status === 'pending' || task.value.status === 'queued') {
    streamStore.connect(task.value.id)
  }
})

onUnmounted(() => {
  streamStore.disconnect()
})
</script>

<style scoped>
.task-monitor {
  height: calc(100vh - 56px);
  display: flex;
  flex-direction: column;
}
.monitor-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  height: 48px;
}
.toolbar-left, .toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.toolbar-left h3 {
  font-size: 16px;
  margin: 0;
}
.monitor-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}
.monitor-left {
  width: 35%;
  padding: 12px;
  overflow-y: auto;
  border-right: 1px solid #e8e8e8;
}
.monitor-right {
  flex: 1;
  padding: 12px;
  overflow-y: auto;
}
.step-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}
.step-icon {
  margin-top: 2px;
}
.step-name {
  font-size: 13px;
  font-weight: 500;
}
.step-meta {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}
.output-area {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.output-block {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 12px;
}
.output-header {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
.output-content {
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
}
.debate-area {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.debate-content {
  font-size: 13px;
  line-height: 1.6;
}
.log-area {
  font-family: monospace;
  font-size: 12px;
}
.log-item {
  display: flex;
  gap: 8px;
  padding: 4px 0;
  border-bottom: 1px solid #f5f5f5;
}
.log-time {
  color: #909399;
  white-space: nowrap;
}
.log-data {
  color: #606266;
  word-break: break-all;
}
</style>
