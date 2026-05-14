<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ isEdit ? '编辑智能体' : '创建智能体' }}</h2>
      <div>
        <el-button @click="$router.push('/agents')">返回</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="基本信息" name="basic">
        <el-form ref="basicFormRef" :model="form" :rules="basicRules" label-width="120px" style="max-width: 700px">
          <el-form-item label="名称" prop="name">
            <el-input v-model="form.name" placeholder="智能体名称" />
          </el-form-item>
          <el-form-item label="角色描述" prop="role_desc">
            <el-input v-model="form.role_desc" type="textarea" :rows="3" placeholder="描述智能体的角色定位" />
          </el-form-item>
          <el-form-item label="标签">
            <el-select v-model="form.tags" multiple filterable allow-create placeholder="添加标签" style="width: 100%">
              <el-option v-for="tag in form.tags" :key="tag" :label="tag" :value="tag" />
            </el-select>
          </el-form-item>
          <el-form-item label="图标">
            <el-input v-model="form.icon" placeholder="图标名称" />
          </el-form-item>
          <el-form-item label="输出格式">
            <el-select v-model="form.output_format">
              <el-option label="Markdown" value="markdown" />
              <el-option label="纯文本" value="plaintext" />
              <el-option label="JSON" value="json" />
            </el-select>
          </el-form-item>
          <el-form-item label="输出语言">
            <el-select v-model="form.output_lang">
              <el-option label="中文" value="zh-CN" />
              <el-option label="英文" value="en-US" />
            </el-select>
          </el-form-item>
          <el-form-item label="最大输出字数">
            <el-input-number v-model="form.max_output_chars" :min="100" :max="100000" :step="1000" />
          </el-form-item>
          <el-form-item label="启用状态">
            <el-switch v-model="form.enabled" />
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="Prompt 配置" name="prompt">
        <el-form label-width="120px" style="max-width: 800px">
          <el-form-item label="System Prompt">
            <div style="width: 100%">
              <div style="margin-bottom: 8px">
                <el-button size="small" @click="insertVariable('global')">插入全局变量</el-button>
                <el-button size="small" @click="insertVariable('datasource')">插入数据源结果</el-button>
                <el-button size="small" @click="insertVariable('agent')">插入智能体输出</el-button>
                <el-button size="small" @click="insertVariable('history')">插入历史轮次</el-button>
              </div>
              <el-input v-model="form.system_prompt" type="textarea" :rows="12" placeholder="支持占位符: {{全局变量}}, {{datasource.result}}, {{agent.xxx.output}}, {{history.round_N.output}}" ref="promptInputRef" />
            </div>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="模型与数据源" name="binding">
        <el-form label-width="120px" style="max-width: 700px">
          <el-form-item label="绑定模型">
            <el-select v-model="form.llm_model_id" placeholder="选择模型" clearable style="width: 100%">
              <el-option v-for="m in models" :key="m.id" :label="m.name" :value="m.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="绑定数据源">
            <el-select v-model="form.datasource_id" placeholder="选择数据源" clearable style="width: 100%">
              <el-option v-for="ds in datasources" :key="ds.id" :label="ds.name" :value="ds.id" />
            </el-select>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="参数映射" name="params">
        <div style="max-width: 800px">
          <el-table :data="form.param_mappings" border>
            <el-table-column label="参数名" min-width="150">
              <template #default="{ row }">
                <el-input v-model="row.param_key" placeholder="参数名" />
              </template>
            </el-table-column>
            <el-table-column label="来源类型" width="150">
              <template #default="{ row }">
                <el-select v-model="row.source_type">
                  <el-option label="全局变量" value="global_var" />
                  <el-option label="固定值" value="fixed_value" />
                  <el-option label="表达式" value="expression" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="来源值" min-width="200">
              <template #default="{ row }">
                <el-input v-model="row.source_value" placeholder="来源值" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="80">
              <template #default="{ $index }">
                <el-button link type="danger" @click="(form.param_mappings as any[]).splice($index, 1)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-button type="primary" plain style="margin-top: 12px; width: 100%" @click="addParamMapping">
            <el-icon><Plus /></el-icon>添加参数映射
          </el-button>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { AgentItem, LlmModelItem, DataSourceItem, ParamMapping } from '@/types'
import { getAgent, createAgent, updateAgent } from '@/api/agents'
import { getModels } from '@/api/models'
import { getDataSources } from '@/api/datasources'
import { ElMessage } from 'element-plus'
import type { FormInstance } from 'element-plus'

const route = useRoute()
const router = useRouter()
const basicFormRef = ref<FormInstance>()
const promptInputRef = ref<any>(null)
const activeTab = ref('basic')
const saving = ref(false)
const models = ref<LlmModelItem[]>([])
const datasources = ref<DataSourceItem[]>([])

const isEdit = computed(() => route.params.id && route.params.id !== 'new')

const form = reactive<Partial<AgentItem>>({
  name: '',
  role_desc: '',
  tags: [],
  icon: '',
  system_prompt: '',
  llm_model_id: '',
  datasource_id: '',
  param_mappings: [],
  output_format: 'markdown',
  output_lang: 'zh-CN',
  max_output_chars: 8000,
  enabled: true,
})

const basicRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  system_prompt: [{ required: true, message: '请输入 System Prompt', trigger: 'blur' }],
}

function addParamMapping() {
  form.param_mappings!.push({ param_key: '', source_type: 'global_var', source_value: '' })
}

function insertVariable(type: string) {
  let placeholder = ''
  if (type === 'global') placeholder = '{{变量名}}'
  else if (type === 'datasource') placeholder = '{{datasource.result}}'
  else if (type === 'agent') placeholder = '{{agent.xxx.output}}'
  else if (type === 'history') placeholder = '{{history.round_1.output}}'
  form.system_prompt = (form.system_prompt || '') + placeholder
}

async function handleSave() {
  const valid = await basicFormRef.value?.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    if (isEdit.value) {
      await updateAgent(route.params.id as string, form)
      ElMessage.success('更新成功')
    } else {
      await createAgent(form)
      ElMessage.success('创建成功')
    }
    router.push('/agents')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    const [mRes, dsRes] = await Promise.all([getModels(), getDataSources()])
    models.value = mRes.data.data || []
    datasources.value = dsRes.data.data || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载数据失败')
  }

  if (isEdit.value) {
    try {
      const res = await getAgent(route.params.id as string)
      Object.assign(form, res.data.data)
    } catch (e: any) {
      ElMessage.error(e.message || '加载智能体失败')
    }
  }
})
</script>
