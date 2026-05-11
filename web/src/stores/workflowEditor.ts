import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { WorkflowNode, WorkflowEdge, GlobalParam, NodeType, ExecConfig } from '@/types'
import { v4 as uuidv4 } from 'uuid'

let idCounter = 0
function genId() {
  return `node_${++idCounter}_${Date.now()}`
}

export const useWorkflowEditorStore = defineStore('workflowEditor', () => {
  const workflowId = ref<string>('')
  const workflowName = ref('未命名工作流')
  const workflowDesc = ref('')
  const workflowTags = ref<string[]>([])
  const visibility = ref<'private' | 'public' | 'shared'>('private')
  const defaultModelId = ref<string>('')
  const globalParams = ref<GlobalParam[]>([])
  const nodes = ref<WorkflowNode[]>([])
  const edges = ref<WorkflowEdge[]>([])
  const execConfig = ref<ExecConfig>({})
  const selectedNodeId = ref<string | null>(null)
  const globalParamsDrawerVisible = ref(false)
  const execDialogVisible = ref(false)

  const selectedNode = computed(() => {
    return nodes.value.find((n) => n.id === selectedNodeId.value) || null
  })

  function initFromWorkflow(wf: any) {
    workflowId.value = wf.id
    workflowName.value = wf.name
    workflowDesc.value = wf.description || ''
    workflowTags.value = wf.tags || []
    visibility.value = wf.visibility || 'private'
    defaultModelId.value = wf.default_model_id || ''
    globalParams.value = wf.global_params || []
    nodes.value = wf.nodes || []
    edges.value = wf.edges || []
    execConfig.value = wf.exec_config || {}
    selectedNodeId.value = null
  }

  function addNode(type: NodeType, position: { x: number; y: number }) {
    const id = genId()
    const node: WorkflowNode = {
      id,
      type,
      label: type,
      position,
      data: {},
    }
    nodes.value.push(node)
    selectedNodeId.value = id
    return id
  }

  function removeNode(id: string) {
    nodes.value = nodes.value.filter((n) => n.id !== id)
    edges.value = edges.value.filter((e) => e.source !== id && e.target !== id)
    if (selectedNodeId.value === id) {
      selectedNodeId.value = null
    }
  }

  function updateNodeData(id: string, data: Record<string, any>) {
    const node = nodes.value.find((n) => n.id === id)
    if (node) {
      node.data = { ...node.data, ...data }
    }
  }

  function updateNodeLabel(id: string, label: string) {
    const node = nodes.value.find((n) => n.id === id)
    if (node) {
      node.label = label
    }
  }

  function addEdge(edge: WorkflowEdge) {
    if (!edges.value.find((e) => e.source === edge.source && e.target === edge.target)) {
      edges.value.push(edge)
    }
  }

  function removeEdge(id: string) {
    edges.value = edges.value.filter((e) => e.id !== id)
  }

  function reset() {
    workflowId.value = ''
    workflowName.value = '未命名工作流'
    workflowDesc.value = ''
    workflowTags.value = []
    visibility.value = 'private'
    defaultModelId.value = ''
    globalParams.value = []
    nodes.value = []
    edges.value = []
    execConfig.value = {}
    selectedNodeId.value = null
  }

  return {
    workflowId, workflowName, workflowDesc, workflowTags, visibility,
    defaultModelId, globalParams, nodes, edges, execConfig,
    selectedNodeId, selectedNode, globalParamsDrawerVisible, execDialogVisible,
    initFromWorkflow, addNode, removeNode, updateNodeData, updateNodeLabel,
    addEdge, removeEdge, reset,
  }
})
