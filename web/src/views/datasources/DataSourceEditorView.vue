<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ isEdit ? '编辑数据源' : '创建数据源' }}</h2>
      <div>
        <el-button @click="$router.push('/datasources')">返回</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </div>
    </div>

    <el-steps :active="currentStep" finish-status="success" align-center style="margin-bottom: 24px">
      <el-step title="基本信息" />
      <el-step title="请求配置" />
      <el-step title="鉴权与缓存" />
      <el-step title="参数模板" />
    </el-steps>

    <el-card v-show="currentStep === 0">
      <el-form ref="step0FormRef" :model="form" :rules="step0Rules" label-width="120px" style="max-width: 700px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="数据源名称" />
        </el-form-item>
        <el-form-item label="类型" prop="ds_type">
          <el-select v-model="form.ds_type" placeholder="选择类型" @change="onTypeChange">
            <el-option label="HTTP GET" value="HTTP_GET" />
            <el-option label="HTTP POST" value="HTTP_POST" />
            <el-option label="文件上传" value="FILE_UPLOAD" />
            <el-option label="手动输入" value="MANUAL_INPUT" />
            <el-option label="WebSocket 流" value="WEBSOCKET_STREAM" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="分类">
          <el-input v-model="form.category" placeholder="如: finance, education" />
        </el-form-item>
        <el-form-item label="标签">
          <el-select v-model="form.tags" multiple filterable allow-create placeholder="添加标签" style="width: 100%">
            <el-option v-for="tag in form.tags" :key="tag" :label="tag" :value="tag" />
          </el-select>
        </el-form-item>
      </el-form>
      <div style="text-align: right">
        <el-button type="primary" @click="nextStep(0)">下一步</el-button>
      </div>
    </el-card>

    <el-card v-show="currentStep === 1">
      <el-form label-width="120px" style="max-width: 700px">
        <el-form-item v-if="isHttp" label="URL 模板">
          <el-input v-model="form.url_template" placeholder="支持 {{变量名}} 占位符" />
        </el-form-item>
        <el-form-item v-if="form.ds_type === 'HTTP_POST'" label="Content-Type">
          <el-input v-model="form.content_type" placeholder="application/json" />
        </el-form-item>
        <el-form-item v-if="isHttp" label="超时(ms)">
          <el-input-number v-model="form.timeout_ms" :min="1000" :step="5000" />
        </el-form-item>
        <el-form-item v-if="isHttp" label="重试次数">
          <el-input-number v-model="form.retry_count" :min="0" :max="10" />
        </el-form-item>
        <el-form-item v-if="isHttp" label="Headers">
          <div style="width: 100%">
            <div v-for="(_, idx) in headerEntries" :key="idx" style="display: flex; gap: 8px; margin-bottom: 8px">
              <el-input v-model="headerEntries[idx].key" placeholder="Key" style="flex: 1" />
              <el-input v-model="headerEntries[idx].value" placeholder="Value" style="flex: 1" />
              <el-button link type="danger" @click="headerEntries.splice(idx, 1)"><el-icon><Delete /></el-icon></el-button>
            </div>
            <el-button size="small" @click="headerEntries.push({ key: '', value: '' })">添加 Header</el-button>
          </div>
        </el-form-item>
        <el-form-item v-if="form.ds_type === 'HTTP_POST'" label="Body 模板">
          <el-input v-model="bodyTemplateStr" type="textarea" :rows="6" placeholder="JSON Body 模板，支持占位符" />
        </el-form-item>
        <el-form-item v-if="isHttp" label="响应提取 JSONPath">
          <el-input v-model="form.response_jsonpath" placeholder="如: $.data.result" />
        </el-form-item>
        <el-form-item v-if="form.ds_type === 'FILE_UPLOAD'" label="上传文件">
          <el-upload :auto-upload="false" :limit="1" :on-change="handleFileChange">
            <el-button>选择文件</el-button>
          </el-upload>
        </el-form-item>
        <el-form-item v-if="form.ds_type === 'MANUAL_INPUT'" label="手动输入内容">
          <el-input v-model="manualContent" type="textarea" :rows="6" placeholder="输入静态数据" />
        </el-form-item>
        <template v-if="form.ds_type === 'WEBSOCKET_STREAM'">
          <el-form-item label="WebSocket URL">
            <el-input v-model="form.url_template" placeholder="wss://example.com/stream" />
          </el-form-item>
          <el-form-item label="连接超时(ms)">
            <el-input-number v-model="form.timeout_ms" :min="1000" :step="5000" />
          </el-form-item>
          <el-form-item label="Ping 间隔(秒)">
            <el-input-number v-model="wsPingInterval" :min="5" :max="300" />
          </el-form-item>
          <el-form-item label="自动重连">
            <el-switch v-model="wsAutoReconnect" />
          </el-form-item>
        </template>
      </el-form>
      <div style="text-align: right; display: flex; gap: 8px; justify-content: flex-end">
        <el-button @click="currentStep--">上一步</el-button>
        <el-button type="primary" @click="nextStep(1)">下一步</el-button>
      </div>
    </el-card>

    <el-card v-show="currentStep === 2">
      <el-form label-width="120px" style="max-width: 700px">
        <el-form-item label="鉴权类型">
          <el-select v-model="form.auth_type">
            <el-option label="无" value="none" />
            <el-option label="Bearer Token" value="bearer" />
            <el-option label="API Key Header" value="api_key_header" />
            <el-option label="自定义 Header" value="custom_header" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.auth_type === 'bearer'" label="Bearer Token">
          <el-input v-model="authConfig.token" placeholder="Bearer Token（加密存储）" show-password />
        </el-form-item>
        <el-form-item v-if="form.auth_type === 'api_key_header'" label="Header Name">
          <el-input v-model="authConfig.header_name" placeholder="如: X-API-Key" />
        </el-form-item>
        <el-form-item v-if="form.auth_type === 'api_key_header'" label="API Key">
          <el-input v-model="authConfig.api_key" placeholder="API Key（加密存储）" show-password />
        </el-form-item>
        <el-form-item v-if="form.auth_type === 'custom_header'" label="自定义 Headers">
          <div style="width: 100%">
            <div v-for="(_, idx) in authHeaderEntries" :key="idx" style="display: flex; gap: 8px; margin-bottom: 8px">
              <el-input v-model="authHeaderEntries[idx].key" placeholder="Key" style="flex: 1" />
              <el-input v-model="authHeaderEntries[idx].value" placeholder="Value（加密存储）" show-password style="flex: 1" />
              <el-button link type="danger" @click="authHeaderEntries.splice(idx, 1)"><el-icon><Delete /></el-icon></el-button>
            </div>
            <el-button size="small" @click="authHeaderEntries.push({ key: '', value: '' })">添加</el-button>
          </div>
        </el-form-item>
        <el-form-item label="缓存策略">
          <el-select v-model="form.cache_policy">
            <el-option label="不缓存" value="none" />
            <el-option label="TTL" value="ttl" />
            <el-option label="固定时长" value="fixed" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.cache_policy !== 'none'" label="缓存时长(秒)">
          <el-input-number v-model="form.cache_ttl_seconds!" :min="1" />
        </el-form-item>
      </el-form>
      <div style="text-align: right; display: flex; gap: 8px; justify-content: flex-end">
        <el-button @click="currentStep--">上一步</el-button>
        <el-button type="primary" @click="nextStep(2)">下一步</el-button>
      </div>
    </el-card>

    <el-card v-show="currentStep === 3">
      <div style="max-width: 800px">
        <el-table :data="form.params_schema" border>
          <el-table-column label="参数名" min-width="120">
            <template #default="{ row }"><el-input v-model="row.name" /></template>
          </el-table-column>
          <el-table-column label="类型" width="120">
            <template #default="{ row }">
              <el-select v-model="row.type">
                <el-option label="字符串" value="string" />
                <el-option label="数字" value="number" />
                <el-option label="日期" value="date" />
                <el-option label="数组" value="array" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="必填" width="70">
            <template #default="{ row }"><el-switch v-model="row.required" /></template>
          </el-table-column>
          <el-table-column label="默认值" min-width="120">
            <template #default="{ row }"><el-input v-model="row.default_value" /></template>
          </el-table-column>
          <el-table-column label="来源" width="130">
            <template #default="{ row }">
              <el-select v-model="row.source">
                <el-option label="全局变量" value="global_var" />
                <el-option label="固定值" value="fixed_value" />
                <el-option label="运行时输入" value="runtime_input" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="描述" min-width="120">
            <template #default="{ row }"><el-input v-model="row.description" /></template>
          </el-table-column>
          <el-table-column label="操作" width="80">
            <template #default="{ $index }">
              <el-button link type="danger" @click="(form.params_schema as any[]).splice($index, 1)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-button type="primary" plain style="margin-top: 12px; width: 100%" @click="addParam">
          <el-icon><Plus /></el-icon>添加参数
        </el-button>
        <div style="margin-top: 24px; text-align: right; display: flex; gap: 8px; justify-content: flex-end">
          <el-button @click="currentStep--">上一步</el-button>
          <el-button type="warning" @click="handleTest">测试调用</el-button>
          <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
        </div>
      </div>
    </el-card>

    <el-dialog v-model="testResultVisible" title="测试结果" width="500px">
      <el-alert v-if="testResult.ok" title="测试通过" type="success" :closable="false" />
      <el-alert v-else :title="`测试失败: ${testResult.error}`" type="error" :closable="false" />
      <div v-if="testResult.data" style="margin-top: 12px">
        <pre style="background: #f5f7fa; padding: 12px; border-radius: 4px; max-height: 300px; overflow: auto; font-size: 12px">{{ JSON.stringify(testResult.data, null, 2) }}</pre>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { DataSourceItem, ParamSchema } from '@/types'
import { getDataSource, createDataSource, updateDataSource, testDataSource, uploadFile } from '@/api/datasources'
import { ElMessage } from 'element-plus'
import type { FormInstance } from 'element-plus'

const route = useRoute()
const router = useRouter()
const step0FormRef = ref<FormInstance>()
const currentStep = ref(0)
const saving = ref(false)
const isEdit = computed(() => route.params.id && route.params.id !== 'new')
const isHttp = computed(() => form.ds_type === 'HTTP_GET' || form.ds_type === 'HTTP_POST')

const headerEntries = ref<{ key: string; value: string }[]>([])
const authHeaderEntries = ref<{ key: string; value: string }[]>([])
const authConfig = reactive<Record<string, string>>({ token: '', header_name: '', api_key: '' })
const bodyTemplateStr = ref('')
const manualContent = ref('')
const uploadFileId = ref('')
const testResultVisible = ref(false)
const testResult = reactive({ ok: false, error: '', data: null as any })
const wsPingInterval = ref(30)
const wsAutoReconnect = ref(true)

const form = reactive<Partial<DataSourceItem>>({
  name: '',
  description: '',
  category: '',
  tags: [],
  ds_type: 'HTTP_GET',
  url_template: '',
  http_method: 'GET',
  content_type: 'application/json',
  timeout_ms: 30000,
  retry_count: 2,
  auth_type: 'none',
  cache_policy: 'none',
  cache_ttl_seconds: 300,
  response_jsonpath: '',
  params_schema: [],
  enabled: true,
})

const step0Rules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  ds_type: [{ required: true, message: '请选择类型', trigger: 'change' }],
}

function onTypeChange() {
  if (form.ds_type === 'HTTP_GET') form.http_method = 'GET'
  else if (form.ds_type === 'HTTP_POST') form.http_method = 'POST'
}

function addParam() {
  form.params_schema!.push({ name: '', type: 'string', required: false, default_value: '', description: '', source: 'runtime_input' })
}

async function nextStep(step: number) {
  if (step === 0) {
    const valid = await step0FormRef.value?.validate().catch(() => false)
    if (!valid) return
  }
  currentStep.value++
}

async function handleFileChange(file: any) {
  try {
    const res = await uploadFile(file.raw)
    uploadFileId.value = res.data.data.file_id
    ElMessage.success('上传成功')
  } catch (e: any) {
    ElMessage.error(e.message || '上传失败')
  }
}

async function handleTest() {
  if (!isEdit.value) {
    ElMessage.warning('请先保存数据源再测试')
    return
  }
  try {
    const res = await testDataSource(route.params.id as string)
    testResult.ok = res.data.data.ok
    testResult.error = res.data.data.error || ''
    testResult.data = res.data.data.extracted || res.data.data.raw
    testResultVisible.value = true
  } catch (e: any) {
    ElMessage.error(e.message || '测试失败')
  }
}

async function handleSave() {
  saving.value = true
  try {
    const headers: Record<string, string> = {}
    headerEntries.value.forEach(h => { if (h.key) headers[h.key] = h.value })
    form.headers = headers

    if (bodyTemplateStr.value) {
      try { form.body_template = JSON.parse(bodyTemplateStr.value) } catch { form.body_template = { raw: bodyTemplateStr.value } }
    }

    if (form.auth_type === 'bearer') form.auth_config = { token: authConfig.token }
    else if (form.auth_type === 'api_key_header') form.auth_config = { header_name: authConfig.header_name, api_key: authConfig.api_key }
    else if (form.auth_type === 'custom_header') {
      const cfg: Record<string, string> = {}
      authHeaderEntries.value.forEach(h => { if (h.key) cfg[h.key] = h.value })
      form.auth_config = cfg
    }

    if (uploadFileId.value) form.uploaded_file_id = uploadFileId.value

    if (form.ds_type === 'WEBSOCKET_STREAM') {
      form.extra_config = { ping_interval_sec: wsPingInterval.value, auto_reconnect: wsAutoReconnect.value }
    }

    if (isEdit.value) {
      await updateDataSource(route.params.id as string, form)
      ElMessage.success('更新成功')
    } else {
      await createDataSource(form)
      ElMessage.success('创建成功')
    }
    router.push('/datasources')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  if (isEdit.value) {
    try {
      const res = await getDataSource(route.params.id as string)
      Object.assign(form, res.data.data)
      if (form.headers) {
        headerEntries.value = Object.entries(form.headers).map(([key, value]) => ({ key, value: String(value) }))
      }
      if (form.body_template) bodyTemplateStr.value = JSON.stringify(form.body_template, null, 2)
      if (form.auth_config) Object.assign(authConfig, form.auth_config)
    } catch (e: any) {
      ElMessage.error(e.message || '加载数据源失败')
    }
  }
})
</script>
