<template>
  <div class="workflow-editor">
    <div class="editor-toolbar">
      <div class="toolbar-left">
        <el-button @click="$router.push('/workflows')"><el-icon><Back /></el-icon>返回</el-button>
        <el-input v-model="store.workflowName" style="width: 240px" placeholder="工作流名称" />
      </div>
      <div class="toolbar-center">
        <el-tag>版本: {{ currentVersion }}</el-tag>
        <el-select v-model="store.visibility" style="width: 100px" placeholder="可见性">
          <el-option label="私有" value="private" />
          <el-option label="公开" value="public" />
          <el-option label="共享" value="shared" />
        </el-select>
      </div>
      <div class="toolbar-right">
        <el-button @click="store.globalParamsDrawerVisible = true">
          <el-icon><Setting /></el-icon>全局入参
        </el-button>
        <el-button type="warning" @click="handleExec">
          <el-icon><VideoPlay /></el-icon>执行
        </el-button>
        <el-button type="primary" @click="handleSave">
          <el-icon><Check /></el-icon>保存
        </el-button>
      </div>
    </div>

    <div class="editor-body">
      <NodePalette @add-node="handleAddNode" />

      <div class="canvas-area">
        <VueFlow
          v-model:nodes="flowNodes"
          v-model:edges="flowEdges"
          :node-types="nodeTypes as any"
          fit-view-on-init
          :default-viewport="{ zoom: 1 }"
          :min-zoom="0.3"
          :max-zoom="2"
          @node-click="onNodeClick"
          @connect="onConnect"
          @pane-click="onPaneClick"
        >
          <Background :gap="16" />
          <Controls />
          <MiniMap />
        </VueFlow>
      </div>

      <NodeInspector
        v-if="store.selectedNode"
        :node="store.selectedNode"
        @close="store.selectedNodeId = null"
        @update="handleUpdateNode"
      />
    </div>

    <el-drawer v-model="store.globalParamsDrawerVisible" title="全局入参配置" size="480px" direction="rtl">
      <GlobalParamsEditor />
    </el-drawer>

    <el-dialog v-model="store.execDialogVisible" title="执行工作流" width="600px">
      <ExecDialog :workflow-id="store.workflowId" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, markRaw } from 'vue'
import { useRoute } from 'vue-router'
import { VueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import type { Connection } from '@vue-flow/core'
import { useWorkflowEditorStore } from '@/stores/workflowEditor'
import { getWorkflow, createWorkflow, updateWorkflow } from '@/api/workflows'
import { ElMessage } from 'element-plus'
import NodePalette from '@/components/workflow/palette/NodePalette.vue'
import NodeInspector from '@/components/workflow/panel/NodeInspector.vue'
import GlobalParamsEditor from '@/components/workflow/panel/GlobalParamsEditor.vue'
import ExecDialog from '@/components/workflow/panel/ExecDialog.vue'
import StartNode from '@/components/workflow/nodes/StartNode.vue'
import EndNode from '@/components/workflow/nodes/EndNode.vue'
import AgentRunNode from '@/components/workflow/nodes/AgentRunNode.vue'
import ParallelNode from '@/components/workflow/nodes/ParallelNode.vue'
import DebateNode from '@/components/workflow/nodes/DebateNode.vue'
import CrossValidateNode from '@/components/workflow/nodes/CrossValidateNode.vue'
import RiskReviewNode from '@/components/workflow/nodes/RiskReviewNode.vue'
import ConditionNode from '@/components/workflow/nodes/ConditionNode.vue'
import SummarizeNode from '@/components/workflow/nodes/SummarizeNode.vue'
import TransformNode from '@/components/workflow/nodes/TransformNode.vue'

const nodeTypes = {
  start: markRaw(StartNode),
  end: markRaw(EndNode),
  agent_run: markRaw(AgentRunNode),
  parallel: markRaw(ParallelNode),
  debate: markRaw(DebateNode),
  cross_validate: markRaw(CrossValidateNode),
  risk_review: markRaw(RiskReviewNode),
  condition: markRaw(ConditionNode),
  summarize: markRaw(SummarizeNode),
  transform: markRaw(TransformNode),
}

const route = useRoute()
const store = useWorkflowEditorStore()
const currentVersion = ref(1)

const flowNodes = computed({
  get: () => store.nodes.map((n) => ({
    id: n.id,
    type: n.type,
    position: n.position,
    data: { label: n.label, ...n.data },
  })),
  set: (val) => {
    store.nodes = val.map((n) => ({
      id: n.id,
      type: (n.type as any) || 'start',
      label: n.data?.label || n.type || '',
      position: n.position,
      data: n.data || {},
    }))
  },
})

const flowEdges = computed({
  get: () => store.edges.map((e) => ({
    id: e.id,
    source: e.source,
    target: e.target,
    sourceHandle: e.sourceHandle,
    targetHandle: e.targetHandle,
    label: e.label,
    animated: true,
    style: { stroke: '#409eff' },
  })),
  set: (val) => {
    store.edges = val.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
      sourceHandle: e.sourceHandle,
      targetHandle: e.targetHandle,
      label: e.label,
    }))
  },
})

function handleAddNode(type: string) {
  const position = { x: 200 + Math.random() * 300, y: 200 + Math.random() * 200 }
  store.addNode(type as any, position)
}

function onNodeClick({ node }: any) {
  store.selectedNodeId = node.id
}

function onPaneClick() {
  store.selectedNodeId = null
}

function onConnect(params: Connection) {
  store.addEdge({
    id: `e-${params.source}-${params.target}`,
    source: params.source,
    target: params.target,
    sourceHandle: params.sourceHandle || undefined,
    targetHandle: params.targetHandle || undefined,
  })
}

function handleUpdateNode(data: Record<string, any>) {
  if (store.selectedNodeId) {
    store.updateNodeData(store.selectedNodeId, data)
  }
}

function handleExec() {
  store.execDialogVisible = true
}

async function handleSave() {
  try {
    const payload = {
      name: store.workflowName,
      description: store.workflowDesc,
      tags: store.workflowTags,
      visibility: store.visibility,
      default_model_id: store.defaultModelId || undefined,
      global_params: store.globalParams,
      nodes: store.nodes,
      edges: store.edges,
      exec_config: store.execConfig,
    }
    if (store.workflowId) {
      await updateWorkflow(store.workflowId, payload)
      ElMessage.success('保存成功')
    } else {
      const res = await createWorkflow(payload)
      store.workflowId = res.data.data.id
      ElMessage.success('创建成功')
    }
  } catch {}
}

onMounted(async () => {
  const id = route.params.id as string
  if (id && id !== 'new') {
    try {
      const res = await getWorkflow(id)
      store.initFromWorkflow(res.data.data)
      currentVersion.value = res.data.data.version || 1
    } catch {}
  } else {
    store.reset()
    store.addNode('start', { x: 100, y: 200 })
    store.addNode('end', { x: 600, y: 200 })
  }
})
</script>

<style scoped>
.workflow-editor {
  height: calc(100vh - 56px);
  display: flex;
  flex-direction: column;
}
.editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  height: 48px;
}
.toolbar-left, .toolbar-center, .toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.editor-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}
.canvas-area {
  flex: 1;
  height: 100%;
}
</style>

<style>
@import '@vue-flow/core/dist/style.css';
@import '@vue-flow/core/dist/theme-default.css';
@import '@vue-flow/controls/dist/style.css';
@import '@vue-flow/minimap/dist/style.css';
</style>
