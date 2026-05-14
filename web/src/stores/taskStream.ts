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
  const logs = ref<{ timestamp: string; event: string; data: any }[]>([])
  const completed = ref(false)
  const failed = ref(false)
  const errorMessage = ref('')
  const reportId = ref('')
  const eventSource = ref<EventSource | null>(null)
  const riskItems = ref<{ dimension: string; level: 'low' | 'medium' | 'high' | 'critical'; summary: string; timestamp: string }[]>([])

  function init() {
    nodeStates.value = {}
    logs.value = []
    completed.value = false
    failed.value = false
    errorMessage.value = ''
    reportId.value = ''
    riskItems.value = []
  }

  function connect(taskId: string) {
    disconnect()
    init()
    const token = localStorage.getItem('token')
    const url = `/api/v1/tasks/${taskId}/stream${token ? `?token=${token}` : ''}`
    const es = new EventSource(url)
    eventSource.value = es

    es.addEventListener('node_status', (e) => {
      const data = JSON.parse(e.data)
      const existing = nodeStates.value[data.node_id] || { nodeId: data.node_id, status: '', output: '', agentName: '' }
      nodeStates.value[data.node_id] = { ...existing, status: data.status }
      logs.value.push({ timestamp: data.timestamp, event: 'node_status', data })
    })

    es.addEventListener('agent_stream_start', (e) => {
      const data = JSON.parse(e.data)
      nodeStates.value[data.step_id] = {
        nodeId: data.step_id,
        status: 'running',
        output: '',
        agentName: data.agent_name,
      }
      logs.value.push({ timestamp: new Date().toISOString(), event: 'agent_stream_start', data })
    })

    es.addEventListener('agent_stream_chunk', (e) => {
      const data = JSON.parse(e.data)
      const existing = nodeStates.value[data.step_id] || { nodeId: data.step_id, status: 'running', output: '', agentName: '' }
      const newOutput = data.accumulated || (existing.output + (data.chunk || ''))
      nodeStates.value[data.step_id] = { ...existing, output: newOutput }
      // Limit logs to prevent memory issues
      if (logs.value.length < 1000) {
        logs.value.push({ timestamp: new Date().toISOString(), event: 'agent_stream_chunk', data: { step_id: data.step_id, chunk_length: (data.chunk || '').length } })
      }
    })

    es.addEventListener('agent_stream_end', (e) => {
      const data = JSON.parse(e.data)
      const existing = nodeStates.value[data.step_id] || { nodeId: data.step_id, status: '', output: '', agentName: '' }
      nodeStates.value[data.step_id] = { ...existing, status: 'completed', output: data.full_output }
      logs.value.push({ timestamp: new Date().toISOString(), event: 'agent_stream_end', data })
    })

    es.addEventListener('debate_round', (e) => {
      const data = JSON.parse(e.data)
      logs.value.push({ timestamp: new Date().toISOString(), event: 'debate_round', data })
    })

    es.addEventListener('risk_review', (e) => {
      const data = JSON.parse(e.data)
      const text: string = data.result || ''
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
      logs.value.push({ timestamp: new Date().toISOString(), event: 'risk_review', data })
    })

    es.addEventListener('task_complete', (e) => {
      const data = JSON.parse(e.data)
      completed.value = true
      reportId.value = data.report_id
      logs.value.push({ timestamp: new Date().toISOString(), event: 'task_complete', data })
    })

    es.addEventListener('task_failed', (e) => {
      const data = JSON.parse(e.data)
      failed.value = true
      errorMessage.value = data.error
      logs.value.push({ timestamp: new Date().toISOString(), event: 'task_failed', data })
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

  return { nodeStates, logs, completed, failed, errorMessage, reportId, riskItems, connect, disconnect, init }
})
