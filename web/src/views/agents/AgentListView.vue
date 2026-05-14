<template>
  <div class="page-container">
    <div class="page-header">
      <h2>智能体管理</h2>
      <el-button type="primary" @click="handleCreate">
        <el-icon><Plus /></el-icon>创建智能体
      </el-button>
    </div>

    <el-card style="margin-bottom: 16px">
      <el-row :gutter="16" align="middle">
        <el-col :span="8">
          <el-input v-model="keyword" placeholder="搜索智能体..." prefix-icon="Search" clearable @input="fetchList" />
        </el-col>
        <el-col :span="4">
          <el-select v-model="filterEnabled" placeholder="状态" clearable @change="fetchList">
            <el-option label="启用" value="true" />
            <el-option label="禁用" value="false" />
          </el-select>
        </el-col>
      </el-row>
    </el-card>

    <el-card v-loading="loading">
    <div class="card-grid">
      <el-card v-for="agent in agents" :key="agent.id" shadow="hover" class="agent-card" @click="handleEdit(agent)">
        <template #header>
          <div class="card-title-row">
            <span class="card-title">{{ agent.name }}</span>
            <el-tag :type="agent.enabled ? 'success' : 'info'" size="small">{{ agent.enabled ? '启用' : '禁用' }}</el-tag>
          </div>
        </template>
        <p class="card-desc">{{ agent.role_desc || '暂无角色描述' }}</p>
        <div class="card-tags">
          <el-tag v-for="tag in (agent.tags || []).slice(0, 3)" :key="tag" size="small" style="margin-right: 4px">{{ tag }}</el-tag>
        </div>
        <div class="card-meta">
          <span>输出格式: {{ agent.output_format }}</span>
          <span>语言: {{ agent.output_lang }}</span>
        </div>
        <div class="card-actions">
          <el-button size="small" @click.stop="handlePreview(agent)">预览测试</el-button>
          <el-button size="small" @click.stop="handleClone(agent)">复制</el-button>
          <el-button size="small" type="danger" @click.stop="handleDelete(agent)">删除</el-button>
        </div>
      </el-card>
      <el-card v-if="agents.length === 0" class="empty-card">
        <el-empty description="暂无智能体" />
      </el-card>
    </div>

    <el-dialog v-model="previewVisible" title="智能体预览测试" width="600px" @closed="previewOutput = ''; previewError = ''">
      <div v-if="previewAgent">
        <p style="margin-bottom: 12px; color: #909399">测试智能体: {{ previewAgent.name }}</p>
        <el-input v-model="previewInput" type="textarea" :rows="3" placeholder="输入测试内容（可选）" />
        <div v-if="previewLoading" style="margin-top: 12px; text-align: center">
          <el-icon class="is-loading" :size="24"><Loading /></el-icon>
          <span style="margin-left: 8px">生成中...</span>
        </div>
        <el-alert v-if="previewError" :title="previewError" type="error" :closable="false" style="margin-top: 12px" />
        <div v-if="previewOutput" class="preview-output markdown-body" style="margin-top: 12px" v-html="renderMarkdown(previewOutput)" />
      </div>
      <template #footer>
        <el-button @click="previewVisible = false">关闭</el-button>
        <el-button type="primary" :loading="previewLoading" @click="doPreview">执行测试</el-button>
      </template>
    </el-dialog>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import type { AgentItem } from '@/types'
import { getAgents, cloneAgent, deleteAgent, previewAgent as apiPreviewAgent } from '@/api/agents'
import { ElMessage, ElMessageBox } from 'element-plus'
import { marked } from 'marked'

const router = useRouter()
const agents = ref<AgentItem[]>([])
const keyword = ref('')
const filterEnabled = ref('')
const previewVisible = ref(false)
const previewAgent = ref<AgentItem | null>(null)
const previewInput = ref('')
const previewLoading = ref(false)
const previewOutput = ref('')
const previewError = ref('')
const loading = ref(false)

async function fetchList() {
  loading.value = true
  try {
    const res = await getAgents({ keyword: keyword.value, enabled: filterEnabled.value === 'true' ? true : filterEnabled.value === 'false' ? false : undefined })
    agents.value = res.data.data || []
  } catch (e: any) {
    ElMessage.error(e.message || '获取智能体列表失败')
  } finally {
    loading.value = false
  }
}

function handleCreate() {
  router.push('/agents/new')
}

function handleEdit(agent: AgentItem) {
  router.push(`/agents/${agent.id}`)
}

async function handleClone(agent: AgentItem) {
  try {
    await cloneAgent(agent.id)
    ElMessage.success('复制成功')
    fetchList()
  } catch (e: any) {
    ElMessage.error(e.message || '复制失败')
  }
}

async function handleDelete(agent: AgentItem) {
  await ElMessageBox.confirm(`确定删除智能体「${agent.name}」？`, '提示', { type: 'warning' })
  try {
    await deleteAgent(agent.id)
    ElMessage.success('删除成功')
    fetchList()
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  }
}

function handlePreview(agent: AgentItem) {
  previewAgent.value = agent
  previewInput.value = ''
  previewOutput.value = ''
  previewVisible.value = true
}

async function doPreview() {
  if (!previewAgent.value) return
  previewLoading.value = true
  previewOutput.value = ''
  previewError.value = ''
  try {
    const res = await apiPreviewAgent(previewAgent.value.id, { content: previewInput.value })
    const data = res.data.data
    if (data.error) {
      previewError.value = data.error
    } else if (data.output) {
      previewOutput.value = data.output
    } else {
      previewError.value = '模型返回了空内容，请检查模型配置和系统提示词'
    }
  } catch (e: any) {
    previewError.value = e.response?.data?.message || e.message || '网络请求失败'
  } finally {
    previewLoading.value = false
  }
}

function renderMarkdown(content: string) {
  return content ? marked(content) : ''
}

onMounted(fetchList)
</script>

<style scoped>
.agent-card {
  cursor: pointer;
  transition: transform 0.2s;
}
.agent-card:hover {
  transform: translateY(-2px);
}
.card-title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.card-title {
  font-weight: 600;
  font-size: 15px;
}
.card-desc {
  color: #909399;
  font-size: 13px;
  margin: 8px 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.card-tags {
  margin-bottom: 8px;
}
.card-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #c0c4cc;
  margin-bottom: 8px;
}
.card-actions {
  display: flex;
  gap: 8px;
}
.empty-card {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
}
.preview-output {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 12px;
  max-height: 300px;
  overflow-y: auto;
}
</style>
