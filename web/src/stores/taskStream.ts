import { defineStore } from 'pinia'
import { ref } from 'vue'

interface StreamState {
  nodeId: string
  status: string
  output: string
  agentName: string
  debateRound?: number
}

export const useTaskStreamStore = defineStore('taskStream', () => {
  const nodeStates = ref<Record<string, StreamState>>({})
  const stepStatuses = ref<Record<string, string>>({}) // node_id -> status, updated by node_status SSE
  const logs = ref<{ timestamp: string; event: string; data: any }[]>([])
  const completed = ref(false)
  const failed = ref(false)
  const errorMessage = ref('')
  const reportId = ref('')
  const eventSource = ref<EventSource | null>(null)
  const riskItems = ref<{ dimension: string; level: 'low' | 'medium' | 'high' | 'critical'; summary: string; timestamp: string }[]>([])

  function init() {
    nodeStates.value = {}
    stepStatuses.value = {}
    logs.value = []
    completed.value = false
    failed.value = false
    errorMessage.value = ''
    reportId.value = ''
    riskItems.value = []
  }

  // Shared helper: append a summarized log entry (not raw chunks)
  function appendLog(event: string, data: any) {
    if (logs.value.length < 1000) {
      logs.value.push({ timestamp: new Date().toISOString(), event, data })
    }
  }

  // Shared helper: update streaming node state with accumulated output
  function upsertStreamState(stepId: string, agentName: string, output: string, status: string) {
    const existing = nodeStates.value[stepId] || { nodeId: stepId, status: '', output: '', agentName: '' }
    nodeStates.value[stepId] = { ...existing, agentName: agentName || existing.agentName, output, status }
  }

  function connect(taskId: string) {
    disconnect()
    init()
    const token = localStorage.getItem('token')
    const url = `/api/v1/tasks/${taskId}/stream${token ? `?token=${token}` : ''}`
    const es = new EventSource(url)
    eventSource.value = es

    // --- Node lifecycle ---
    es.addEventListener('node_status', (e) => {
      const data = JSON.parse(e.data)
      const existing = nodeStates.value[data.node_id] || { nodeId: data.node_id, status: '', output: '', agentName: '' }
      nodeStates.value[data.node_id] = { ...existing, status: data.status }
      // Update step statuses for the left-panel execution flow
      stepStatuses.value[data.node_id] = data.status
      appendLog('node_status', data)
    })

    // --- Agent stream ---
    es.addEventListener('agent_stream_start', (e) => {
      const data = JSON.parse(e.data)
      upsertStreamState(data.step_id, data.agent_name, '', 'running')
      appendLog('agent_stream_start', data)
    })
    es.addEventListener('agent_stream_chunk', (e) => {
      const data = JSON.parse(e.data)
      const existing = nodeStates.value[data.step_id] || { nodeId: data.step_id, status: 'running', output: '', agentName: '' }
      nodeStates.value[data.step_id] = { ...existing, output: data.accumulated || (existing.output + (data.chunk || '')) }
    })
    es.addEventListener('agent_stream_end', (e) => {
      const data = JSON.parse(e.data)
      upsertStreamState(data.step_id, '', data.full_output || '', 'completed')
      appendLog('agent_stream_end', data)
    })

    // --- Summarize stream ---
    es.addEventListener('summarize_stream_start', (e) => {
      const data = JSON.parse(e.data)
      upsertStreamState(data.step_id, '报告汇总', '', 'running')
      appendLog('summarize_stream_start', data)
    })
    es.addEventListener('summarize_stream_chunk', (e) => {
      const data = JSON.parse(e.data)
      const existing = nodeStates.value[data.step_id] || { nodeId: data.step_id, status: 'running', output: '', agentName: '报告汇总' }
      nodeStates.value[data.step_id] = { ...existing, output: data.accumulated || (existing.output + (data.chunk || '')) }
    })
    es.addEventListener('summarize_stream_end', (e) => {
      const data = JSON.parse(e.data)
      upsertStreamState(data.step_id, '报告汇总', data.full_output || '', 'completed')
      appendLog('summarize_stream_end', data)
    })
    // Backward-compat full result event
    es.addEventListener('summarize', (e) => {
      const data = JSON.parse(e.data)
      upsertStreamState(data.node_id, '报告汇总', data.result || '', 'completed')
      appendLog('summarize', data)
    })

    // --- Cross validate stream ---
    es.addEventListener('cross_validate_stream_start', (e) => {
      const data = JSON.parse(e.data)
      upsertStreamState(data.step_id, '交叉验证', '', 'running')
      appendLog('cross_validate_stream_start', data)
    })
    es.addEventListener('cross_validate_stream_chunk', (e) => {
      const data = JSON.parse(e.data)
      const existing = nodeStates.value[data.step_id] || { nodeId: data.step_id, status: 'running', output: '', agentName: '交叉验证' }
      nodeStates.value[data.step_id] = { ...existing, output: data.accumulated || (existing.output + (data.chunk || '')) }
    })
    es.addEventListener('cross_validate_stream_end', (e) => {
      const data = JSON.parse(e.data)
      upsertStreamState(data.step_id, '交叉验证', data.full_output || '', 'completed')
      appendLog('cross_validate_stream_end', data)
    })
    // Backward-compat full result event
    es.addEventListener('cross_validate', (e) => {
      const data = JSON.parse(e.data)
      upsertStreamState(data.node_id, '交叉验证', data.result || '', 'completed')
      appendLog('cross_validate', data)
    })

    // --- Risk review stream ---
    es.addEventListener('risk_review_stream_start', (e) => {
      const data = JSON.parse(e.data)
      upsertStreamState(data.step_id, '风险评审', '', 'running')
      appendLog('risk_review_stream_start', data)
    })
    es.addEventListener('risk_review_stream_chunk', (e) => {
      const data = JSON.parse(e.data)
      const existing = nodeStates.value[data.step_id] || { nodeId: data.step_id, status: 'running', output: '', agentName: '风险评审' }
      nodeStates.value[data.step_id] = { ...existing, output: data.accumulated || (existing.output + (data.chunk || '')) }
    })
    es.addEventListener('risk_review_stream_end', (e) => {
      const data = JSON.parse(e.data)
      upsertStreamState(data.step_id, '风险评审', data.full_output || '', 'completed')
      appendLog('risk_review_stream_end', data)
    })
    // Full result event — also update risk items
    es.addEventListener('risk_review', (e) => {
      const data = JSON.parse(e.data)
      const text: string = data.result || ''
      upsertStreamState(data.node_id, '风险评审', text, 'completed')
      let level: 'low' | 'medium' | 'high' | 'critical' = 'medium'
      if (text.includes('严重') || text.includes('critical')) level = 'critical'
      else if (text.includes('高') || text.includes('high')) level = 'high'
      else if (text.includes('低') || text.includes('low')) level = 'low'
      riskItems.value.push({
        dimension: data.node_id || '风险评估',
        level,
        summary: text,
        timestamp: new Date().toISOString(),
      })
      appendLog('risk_review', data)
    })

    // --- Debate stream ---
    es.addEventListener('debate_stream_start', (e) => {
      const data = JSON.parse(e.data)
      upsertStreamState(data.step_id, data.agent_name || '辩手', '', 'running')
      appendLog('debate_stream_start', data)
    })
    es.addEventListener('debate_stream_chunk', (e) => {
      const data = JSON.parse(e.data)
      const existing = nodeStates.value[data.step_id] || { nodeId: data.step_id, status: 'running', output: '', agentName: '辩手' }
      nodeStates.value[data.step_id] = { ...existing, output: data.accumulated || (existing.output + (data.chunk || '')) }
    })
    es.addEventListener('debate_stream_end', (e) => {
      const data = JSON.parse(e.data)
      upsertStreamState(data.step_id, '', data.full_output || '', 'completed')
      appendLog('debate_stream_end', data)
    })
    // Full round result
    es.addEventListener('debate_round', (e) => {
      const data = JSON.parse(e.data)
      appendLog('debate_round', data)
    })

    // --- Task lifecycle ---
    es.addEventListener('task_complete', (e) => {
      const data = JSON.parse(e.data)
      completed.value = true
      reportId.value = data.report_id
      appendLog('task_complete', data)
    })
    es.addEventListener('task_failed', (e) => {
      const data = JSON.parse(e.data)
      failed.value = true
      errorMessage.value = data.error
      appendLog('task_failed', data)
    })

    es.onerror = () => {
      es.close()
      eventSource.value = null
    }
  }

  function disconnect() {
    if (eventSource.value) {
      eventSource.value.close()
      eventSource.value = null
    }
  }

  return { nodeStates, stepStatuses, logs, completed, failed, errorMessage, reportId, riskItems, connect, disconnect, init }
})