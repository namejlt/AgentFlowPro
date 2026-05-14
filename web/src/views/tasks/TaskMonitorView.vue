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
              <div v-for="(item, idx) in riskReviews" :key="idx" class="risk-card">
                <div class="risk-card-header">
                  <el-tag :type="item.level === 'critical' ? 'danger' : item.level === 'high' ? 'warning' : item.level === 'medium' ? '' : 'success'" size="small">
                    {{ item.level === 'critical' ? '严重' : item.level === 'high' ? '高风险' : item.level === 'medium' ? '中风险' : '低风险' }}
                  </el-tag>
                  <span class="risk-dimension">{{ item.dimension || '风险评估' }}</span>
                </div>
                <div class="risk-card-body markdown-body" v-html="renderMarkdown(item.summary)" />
              </div>
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
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
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
const rawSteps = ref<TaskStepItem[]>([])
const activeTab = ref('output')

// Reactive steps: merge API-loaded steps with real-time SSE node statuses
const steps = computed<TaskStepItem[]>(() => {
  return rawSteps.value.map(s => {
    const liveStatus = streamStore.stepStatuses[s.node_id]
    if (liveStatus) {
      return { ...s, status: liveStatus as any }
    }
    return s
  })
})

const debateLogs = computed<DebateLog[]>(() => {
  const result: DebateLog[] = []

  // 1. Collect from SSE debate_round events (real-time during execution)
  streamStore.logs.filter(l => l.event === 'debate_round').forEach(l => {
    const outputs = l.data?.agent_outputs
    if (outputs) {
      if (Array.isArray(outputs)) {
        result.push(...outputs.map((a: any) => ({
          round: a.round || l.data.round || 0,
          agent_id: a.agent_id || '',
          agent_name: a.agent_name || a.agent_id || '辩手',
          output: a.output || '',
          timestamp: l.timestamp || new Date().toISOString(),
        })))
      } else if (typeof outputs === 'object') {
        for (const [agentId, output] of Object.entries(outputs)) {
          result.push({
            round: l.data.round || 0,
            agent_id: agentId,
            agent_name: String(agentId),
            output: output as string,
            timestamp: l.timestamp || new Date().toISOString(),
          })
        }
      }
    }
  })

  // 2. Extract from completed debate step outputs (for page refresh / post-completion)
  for (const step of rawSteps.value) {
    if (step.node_type === 'debate' && step.output && step.status === 'completed') {
      const outText = (step.output as any).text || ''
      if (outText) {
        // Check if we already have debate entries for this round/agent from SSE
        const existingKey = (a: DebateLog) => `${a.round}:${a.agent_id}`
        const existingKeys = new Set(result.map(existingKey))
        // Parse debate output — each agent section is delimited by 【agent_id】
        const agentBlocks = outText.split(/(?=【)/).filter(Boolean)
        for (const block of agentBlocks) {
          const match = block.match(/【([^】]+)】\n?([\s\S]*)/)
          if (match) {
            const agentId = match[1]
            const output = match[2].trim()
            const entry: DebateLog = {
              round: step.debate_round || 0,
              agent_id: agentId,
              agent_name: agentId,
              output,
              timestamp: step.finished_at || new Date().toISOString(),
            }
            if (!existingKeys.has(existingKey(entry))) {
              result.push(entry)
            }
          }
        }
        // Fallback: if no structured blocks found, show entire output as one entry
        if (agentBlocks.length <= 1 && result.length === 0) {
          result.push({
            round: step.debate_round || 1,
            agent_id: step.agent_id || 'debate',
            agent_name: step.agent_name || '辩论',
            output: outText,
            timestamp: step.finished_at || new Date().toISOString(),
          })
        }
      }
    }
  }

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
    rawSteps.value = res.data.data || []
    // Pre-populate nodeStates from historical step data so past outputs are visible
    // even before (or without) SSE connection
    for (const step of rawSteps.value) {
      const stepId = step.node_id || step.id
      const existing = streamStore.nodeStates[stepId]
      if (existing) continue // don't overwrite live data
      let output = ''
      if (step.output && typeof step.output === 'object') {
        output = (step.output as any).text || JSON.stringify(step.output)
      }
      streamStore.nodeStates[stepId] = {
        nodeId: stepId,
        status: step.status,
        output,
        agentName: step.agent_name || step.node_type || '',
      }
    }
  } catch {}
}

async function handleStop() {
  await ElMessageBox.confirm('确定停止该任务？', '提示', { type: 'warning' })
  try {
    await stopTask(task.value.id)
    ElMessage.success('已停止')
    fetchTask()
    fetchSteps()
    if (stepsTimer) { clearInterval(stepsTimer); stepsTimer = null }
  } catch {}
}

async function handleRerun() {
  try {
    const res = await rerunTask(task.value.id)
    ElMessage.success('已重新执行')
    rawSteps.value = []
    streamStore.init()
    streamStore.connect(res.data.data.id)
    await fetchTask()
    await fetchSteps()
    // Restart steps polling
    if (stepsTimer) clearInterval(stepsTimer)
    stepsTimer = setInterval(fetchSteps, 3000)
  } catch {}
}

// Auto-refresh steps while task is running
let stepsTimer: ReturnType<typeof setInterval> | null = null

watch(
  () => streamStore.stepStatuses,
  () => {
    // When SSE node_status arrives, re-fetch steps to get latest agent_name etc.
    if (Object.keys(streamStore.stepStatuses).length > 0) {
      fetchSteps()
    }
  },
  { deep: true }
)

onMounted(async () => {
  await fetchTask()
  await fetchSteps()
  if (task.value.status === 'running' || task.value.status === 'pending' || task.value.status === 'queued') {
    streamStore.connect(task.value.id)
    // Poll steps every 3s while running in case SSE misses events
    stepsTimer = setInterval(fetchSteps, 3000)
  }
})

onUnmounted(() => {
  streamStore.disconnect()
  if (stepsTimer) {
    clearInterval(stepsTimer)
    stepsTimer = null
  }
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
.risk-area {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.risk-card {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 12px;
}
.risk-card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.risk-dimension {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}
.risk-card-body {
  font-size: 13px;
  line-height: 1.6;
}
</style>
