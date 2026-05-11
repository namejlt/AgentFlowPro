<template>
  <div class="node-inspector">
    <div class="inspector-header">
      <span>节点配置 - {{ NODE_TYPE_CONFIG[node.type as NodeType]?.label || node.label }}</span>
      <el-button link @click="$emit('close')"><el-icon><Close /></el-icon></el-button>
    </div>
    <div class="inspector-body">
      <el-form label-position="top" size="small">
        <el-form-item label="节点名称">
          <el-input :model-value="node.label" @update:model-value="(v: string) => store.updateNodeLabel(node.id, v)" />
        </el-form-item>

        <template v-if="node.type === 'agent_run'">
          <el-form-item label="智能体">
            <el-select :model-value="node.data.agent_id" placeholder="选择智能体" @update:model-value="(v: string) => updateField('agent_id', v)">
              <el-option v-for="a in agents" :key="a.id" :label="a.name" :value="a.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="执行超时(ms)">
            <el-input-number :model-value="node.data.timeout_ms || 60000" :min="5000" :step="10000" @update:model-value="(v: number) => updateField('timeout_ms', v)" />
          </el-form-item>
          <el-form-item label="覆盖模型">
            <el-select :model-value="node.data.override_model_id" clearable placeholder="可选" @update:model-value="(v: string) => updateField('override_model_id', v)">
              <el-option v-for="m in models" :key="m.id" :label="m.name" :value="m.id" />
            </el-select>
          </el-form-item>
        </template>

        <template v-if="node.type === 'parallel'">
          <el-form-item label="等待策略">
            <el-select :model-value="node.data.wait_strategy || 'all'" @update:model-value="(v: string) => updateField('wait_strategy', v)">
              <el-option label="全部完成" value="all" />
              <el-option label="任一完成" value="any" />
            </el-select>
          </el-form-item>
        </template>

        <template v-if="node.type === 'debate'">
          <el-form-item label="最大轮次">
            <el-input-number :model-value="node.data.max_rounds || 3" :min="1" :max="5" @update:model-value="(v: number) => updateField('max_rounds', v)" />
          </el-form-item>
          <el-form-item label="终止条件">
            <el-select :model-value="node.data.stop_condition || 'max_rounds'" @update:model-value="(v: string) => updateField('stop_condition', v)">
              <el-option label="达到最大轮次" value="max_rounds" />
              <el-option label="结论一致" value="consensus" />
              <el-option label="置信度收敛" value="confidence" />
            </el-select>
          </el-form-item>
          <el-form-item label="参与智能体">
            <el-select :model-value="node.data.agent_ids || []" multiple placeholder="选择智能体" @update:model-value="(v: string[]) => updateField('agent_ids', v)">
              <el-option v-for="a in agents" :key="a.id" :label="a.name" :value="a.id" />
            </el-select>
          </el-form-item>
        </template>

        <template v-if="node.type === 'cross_validate'">
          <el-form-item label="待验证智能体">
            <el-select :model-value="node.data.agent_ids || []" multiple placeholder="选择智能体" @update:model-value="(v: string[]) => updateField('agent_ids', v)">
              <el-option v-for="a in agents" :key="a.id" :label="a.name" :value="a.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="验证维度">
            <el-input :model-value="node.data.validate_dimensions" placeholder="如: 准确性,完整性" @update:model-value="(v: string) => updateField('validate_dimensions', v)" />
          </el-form-item>
        </template>

        <template v-if="node.type === 'risk_review'">
          <el-form-item label="风险维度">
            <el-input :model-value="node.data.risk_dimensions" placeholder="如: 合规性,安全性" @update:model-value="(v: string) => updateField('risk_dimensions', v)" />
          </el-form-item>
          <el-form-item label="等级阈值">
            <el-select :model-value="node.data.risk_threshold || 'medium'" @update:model-value="(v: string) => updateField('risk_threshold', v)">
              <el-option label="低" value="low" />
              <el-option label="中" value="medium" />
              <el-option label="高" value="high" />
              <el-option label="严重" value="critical" />
            </el-select>
          </el-form-item>
        </template>

        <template v-if="node.type === 'condition'">
          <el-form-item label="条件表达式">
            <el-input :model-value="node.data.condition_expr" type="textarea" :rows="3" placeholder="如: {{agent.xxx.output}} 包含 '是'" @update:model-value="(v: string) => updateField('condition_expr', v)" />
          </el-form-item>
        </template>

        <template v-if="node.type === 'summarize'">
          <el-form-item label="汇总模板 Prompt">
            <el-input :model-value="node.data.summary_prompt" type="textarea" :rows="4" placeholder="请输入汇总 Prompt" @update:model-value="(v: string) => updateField('summary_prompt', v)" />
          </el-form-item>
          <el-form-item label="输出格式">
            <el-select :model-value="node.data.output_format || 'markdown'" @update:model-value="(v: string) => updateField('output_format', v)">
              <el-option label="Markdown" value="markdown" />
              <el-option label="纯文本" value="plaintext" />
              <el-option label="JSON" value="json" />
            </el-select>
          </el-form-item>
        </template>

        <template v-if="node.type === 'transform'">
          <el-form-item label="转换表达式">
            <el-input :model-value="node.data.transform_expr" type="textarea" :rows="3" placeholder="JSONPath 或字段映射" @update:model-value="(v: string) => updateField('transform_expr', v)" />
          </el-form-item>
        </template>

        <template v-if="node.type === 'end'">
          <el-form-item label="报告标题模板">
            <el-input :model-value="node.data.report_title_template" placeholder="如: {{global.topic}} 分析报告" @update:model-value="(v: string) => updateField('report_title_template', v)" />
          </el-form-item>
        </template>

        <el-divider />
        <el-button type="danger" size="small" @click="handleDelete">删除节点</el-button>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { WorkflowNode, NodeType, AgentItem, LlmModelItem } from '@/types'
import { NODE_TYPE_CONFIG } from '@/types/api'
import { useWorkflowEditorStore } from '@/stores/workflowEditor'
import { getAgents } from '@/api/agents'
import { getModels } from '@/api/models'

const props = defineProps<{ node: WorkflowNode }>()
const emit = defineEmits(['close', 'update'])
const store = useWorkflowEditorStore()

const agents = ref<AgentItem[]>([])
const models = ref<LlmModelItem[]>([])

function updateField(key: string, value: any) {
  emit('update', { ...props.node.data, [key]: value })
}

function handleDelete() {
  store.removeNode(props.node.id)
  emit('close')
}

onMounted(async () => {
  try {
    const [aRes, mRes] = await Promise.all([getAgents(), getModels()])
    agents.value = aRes.data.data || []
    models.value = mRes.data.data || []
  } catch {}
})
</script>

<style scoped>
.node-inspector {
  width: 320px;
  background: #fff;
  border-left: 1px solid #e8e8e8;
  display: flex;
  flex-direction: column;
}
.inspector-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #e8e8e8;
  font-weight: 600;
  font-size: 14px;
}
.inspector-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}
</style>
