<template>
  <div class="page-container">
    <div class="page-header">
      <h2>LLM 模型配置</h2>
      <el-button type="primary" @click="showCreateDialog">
        <el-icon><Plus /></el-icon>添加模型
      </el-button>
    </div>

    <el-card>
      <el-table :data="models" stripe>
        <el-table-column prop="name" label="展示名" min-width="120" />
        <el-table-column prop="vendor" label="厂商" width="100" />
        <el-table-column prop="model_id" label="模型ID" width="160" />
        <el-table-column prop="endpoint" label="Endpoint" min-width="200" show-overflow-tooltip />
        <el-table-column label="API Key" width="140">
          <template #default="{ row }">
            <span class="masked-key">{{ row.api_key_masked || '****' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="temperature" label="Temperature" width="110" />
        <el-table-column prop="max_tokens" label="Max Tokens" width="110" />
        <el-table-column label="流式" width="70">
          <template #default="{ row }">
            <el-tag :type="row.stream_enabled ? 'success' : 'info'" size="small">{{ row.stream_enabled ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="默认" width="70">
          <template #default="{ row }">
            <el-tag v-if="row.is_default" type="warning" size="small">默认</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="primary" @click="handleTest(row)">测试</el-button>
            <el-button v-if="!row.is_default" link type="warning" @click="handleSetDefault(row)">设默认</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑模型' : '添加模型'" width="600px">
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="120px">
        <el-form-item label="展示名" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="厂商" prop="vendor">
          <el-input v-model="form.vendor" placeholder="如: OpenAI, Anthropic" />
        </el-form-item>
        <el-form-item label="Endpoint" prop="endpoint">
          <el-input v-model="form.endpoint" placeholder="https://api.openai.com/v1/chat/completions" />
        </el-form-item>
        <el-form-item label="模型ID" prop="model_id">
          <el-input v-model="form.model_id" placeholder="如: gpt-4o" />
        </el-form-item>
        <el-form-item label="API Key" :prop="editingId ? '' : 'api_key'">
          <el-input v-model="form.api_key" :placeholder="editingId ? '留空则保留原密钥' : '请输入 API Key'" show-password />
        </el-form-item>
        <el-form-item label="Temperature">
          <el-slider v-model="form.temperature" :min="0" :max="2" :step="0.1" show-input />
        </el-form-item>
        <el-form-item label="Max Tokens">
          <el-input-number v-model="form.max_tokens" :min="1" :max="128000" :step="1024" />
        </el-form-item>
        <el-form-item label="超时(ms)">
          <el-input-number v-model="form.timeout_ms" :min="5000" :step="10000" />
        </el-form-item>
        <el-form-item label="重试次数">
          <el-input-number v-model="form.retry_count" :min="0" :max="10" />
        </el-form-item>
        <el-form-item label="流式输出">
          <el-switch v-model="form.stream_enabled" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="testResultVisible" title="连通性测试" width="400px">
      <el-result v-if="testResultData.success" icon="success" title="连接成功" :sub-title="`延迟: ${testResultData.latency_ms}ms`" />
      <el-result v-else icon="error" title="连接失败" :sub-title="testResultData.error" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { LlmModelItem } from '@/types'
import { getModels, createModel, updateModel, deleteModel, testModel, setDefaultModel } from '@/api/models'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance } from 'element-plus'

const models = ref<LlmModelItem[]>([])
const dialogVisible = ref(false)
const editingId = ref('')
const saving = ref(false)
const formRef = ref<FormInstance>()
const testResultVisible = ref(false)
const testResultData = reactive({ success: false, latency_ms: 0, error: '' })

const form = reactive({
  name: '',
  vendor: '',
  endpoint: '',
  model_id: '',
  api_key: '',
  temperature: 0.7,
  max_tokens: 4096,
  timeout_ms: 60000,
  retry_count: 3,
  stream_enabled: false,
  enabled: true,
})

const formRules = {
  name: [{ required: true, message: '请输入展示名', trigger: 'blur' }],
  vendor: [{ required: true, message: '请输入厂商', trigger: 'blur' }],
  endpoint: [{ required: true, message: '请输入 Endpoint', trigger: 'blur' }],
  model_id: [{ required: true, message: '请输入模型ID', trigger: 'blur' }],
  api_key: [{ required: true, message: '请输入 API Key', trigger: 'blur' }],
}

async function fetchList() {
  try {
    const res = await getModels()
    models.value = res.data.data || []
  } catch {}
}

function showCreateDialog() {
  editingId.value = ''
  Object.assign(form, { name: '', vendor: '', endpoint: '', model_id: '', api_key: '', temperature: 0.7, max_tokens: 4096, timeout_ms: 60000, retry_count: 3, stream_enabled: false, enabled: true })
  dialogVisible.value = true
}

function handleEdit(row: LlmModelItem) {
  editingId.value = row.id
  Object.assign(form, { ...row, api_key: '' })
  dialogVisible.value = true
}

async function handleSave() {
  await formRef.value?.validate()
  saving.value = true
  try {
    if (editingId.value) {
      await updateModel(editingId.value, form)
      ElMessage.success('更新成功')
    } else {
      await createModel(form)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchList()
  } catch {} finally {
    saving.value = false
  }
}

async function handleTest(row: LlmModelItem) {
  try {
    const res = await testModel(row.id)
    testResultData.success = res.data.data.success
    testResultData.latency_ms = res.data.data.latency_ms
    testResultData.error = res.data.data.error || ''
    testResultVisible.value = true
  } catch {}
}

async function handleSetDefault(row: LlmModelItem) {
  try {
    await setDefaultModel(row.id)
    ElMessage.success('已设为默认模型')
    fetchList()
  } catch {}
}

async function handleDelete(row: LlmModelItem) {
  await ElMessageBox.confirm(`确定删除模型「${row.name}」？`, '提示', { type: 'warning' })
  try {
    await deleteModel(row.id)
    ElMessage.success('删除成功')
    fetchList()
  } catch {}
}

onMounted(fetchList)
</script>

<style scoped>
.masked-key {
  font-family: monospace;
  color: #909399;
  font-size: 12px;
}
</style>
