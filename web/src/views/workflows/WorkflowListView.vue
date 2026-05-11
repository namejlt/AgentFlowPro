<template>
  <div class="page-container">
    <div class="page-header">
      <h2>工作流管理</h2>
      <div>
        <el-button @click="showImportDialog = true">
          <el-icon><Upload /></el-icon>导入
        </el-button>
        <el-button type="primary" @click="handleCreate">
          <el-icon><Plus /></el-icon>创建工作流
        </el-button>
      </div>
    </div>

    <el-card style="margin-bottom: 16px">
      <el-row :gutter="16" align="middle">
        <el-col :span="8">
          <el-input v-model="searchKeyword" placeholder="搜索工作流名称..." prefix-icon="Search" clearable @input="handleSearch" />
        </el-col>
        <el-col :span="4">
          <el-select v-model="filterVisibility" placeholder="可见性" clearable @change="fetchList">
            <el-option label="私有" value="private" />
            <el-option label="公开" value="public" />
            <el-option label="共享" value="shared" />
          </el-select>
        </el-col>
        <el-col :span="4">
          <el-select v-model="viewMode" style="width: 100px">
            <el-option label="卡片" value="card" />
            <el-option label="列表" value="list" />
          </el-select>
        </el-col>
      </el-row>
    </el-card>

    <div v-if="viewMode === 'card'" class="card-grid">
      <el-card v-for="wf in workflows" :key="wf.id" shadow="hover" class="workflow-card" @click="handleEdit(wf)">
        <template #header>
          <div class="card-title-row">
            <span class="card-title">{{ wf.name }}</span>
            <el-dropdown trigger="click" @command="(cmd: string) => handleAction(cmd, wf)" @click.stop>
              <el-icon><MoreFilled /></el-icon>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">编辑</el-dropdown-item>
                  <el-dropdown-item command="clone">复制</el-dropdown-item>
                  <el-dropdown-item command="export">导出</el-dropdown-item>
                  <el-dropdown-item command="share">分享</el-dropdown-item>
                  <el-dropdown-item command="delete" divided style="color: #f56c6c">删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </template>
        <p class="card-desc">{{ wf.description || '暂无描述' }}</p>
        <div class="card-tags">
          <el-tag v-for="tag in (wf.tags || []).slice(0, 3)" :key="tag" size="small" style="margin-right: 4px">{{ tag }}</el-tag>
        </div>
        <div class="card-meta">
          <span><el-icon><Clock /></el-icon> {{ timeAgo(wf.updated_at) }}</span>
          <el-tag :type="wf.visibility === 'public' ? 'success' : wf.visibility === 'shared' ? 'warning' : 'info'" size="small">
            {{ wf.visibility === 'public' ? '公开' : wf.visibility === 'shared' ? '共享' : '私有' }}
          </el-tag>
        </div>
      </el-card>
      <el-card v-if="workflows.length === 0" class="empty-card">
        <el-empty description="暂无工作流" />
      </el-card>
    </div>

    <el-card v-else>
      <el-table :data="workflows" stripe>
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column prop="visibility" label="可见性" width="90">
          <template #default="{ row }">
            <el-tag :type="row.visibility === 'public' ? 'success' : row.visibility === 'shared' ? 'warning' : 'info'" size="small">
              {{ row.visibility === 'public' ? '公开' : row.visibility === 'shared' ? '共享' : '私有' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="version" label="版本" width="70" />
        <el-table-column prop="run_count" label="运行次数" width="90" />
        <el-table-column label="更新时间" width="170">
          <template #default="{ row }">{{ formatDateTime(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="primary" @click="handleClone(row)">复制</el-button>
            <el-button link type="primary" @click="handleExport(row)">导出</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-pagination
      v-if="total > pageSize"
      style="margin-top: 16px; justify-content: center"
      :current-page="page"
      :page-size="pageSize"
      :total="total"
      layout="prev, pager, next"
      @current-change="(p: number) => { page = p; fetchList() }"
    />

    <el-dialog v-model="showImportDialog" title="导入工作流" width="500px">
      <el-upload drag :auto-upload="false" :limit="1" accept=".json" :on-change="handleImportFile">
        <el-icon :size="48"><UploadFilled /></el-icon>
        <div>拖拽或点击上传 .agentflow.json 文件</div>
      </el-upload>
      <template #footer>
        <el-button @click="showImportDialog = false">取消</el-button>
        <el-button type="primary" :loading="importing" @click="doImport">确认导入</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showShareDialog" title="分享工作流" width="450px">
      <el-form label-position="top">
        <el-form-item label="分享链接">
          <el-input :model-value="shareInfo.share_url" readonly>
            <template #append>
              <el-button @click="copyShareLink">复制</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="分享码">
          <el-input :model-value="shareInfo.share_code" readonly />
        </el-form-item>
        <el-form-item label="有效期">
          <el-date-picker v-model="shareExpiresAt" type="datetime" placeholder="可选，不填则永久有效" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showShareDialog = false">关闭</el-button>
        <el-button type="primary" @click="doShare">生成分享</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { WorkflowItem, ShareInfo, ImportMatchReport } from '@/types'
import { getWorkflows, createWorkflow, cloneWorkflow, deleteWorkflow, exportWorkflow, importWorkflow, shareWorkflow } from '@/api/workflows'
import { formatDateTime, timeAgo } from '@/utils/datetime'
import { downloadBlob } from '@/utils/download'

const router = useRouter()
const workflows = ref<WorkflowItem[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const searchKeyword = ref('')
const filterVisibility = ref('')
const viewMode = ref('card')
const showImportDialog = ref(false)
const showShareDialog = ref(false)
const importing = ref(false)
const importFile = ref<File | null>(null)
const importReport = ref<ImportMatchReport | null>(null)
const shareWfId = ref('')
const shareInfo = ref<ShareInfo>({ share_code: '', share_url: '' })
const shareExpiresAt = ref('')

async function fetchList() {
  try {
    const res = await getWorkflows({ page: page.value, page_size: pageSize.value, keyword: searchKeyword.value, visibility: filterVisibility.value || undefined })
    workflows.value = res.data.data || []
    total.value = res.data.meta?.total || 0
  } catch {}
}

function handleSearch() {
  page.value = 1
  fetchList()
}

function handleCreate() {
  router.push('/workflows/new/edit')
}

function handleEdit(wf: WorkflowItem) {
  router.push(`/workflows/${wf.id}/edit`)
}

async function handleClone(wf: WorkflowItem) {
  try {
    await cloneWorkflow(wf.id)
    ElMessage.success('复制成功')
    fetchList()
  } catch {}
}

async function handleDelete(wf: WorkflowItem) {
  await ElMessageBox.confirm(`确定删除工作流「${wf.name}」？`, '提示', { type: 'warning' })
  try {
    await deleteWorkflow(wf.id)
    ElMessage.success('删除成功')
    fetchList()
  } catch {}
}

async function handleExport(wf: WorkflowItem) {
  try {
    const res = await exportWorkflow(wf.id)
    const blob = new Blob([JSON.stringify(res.data.data, null, 2)], { type: 'application/json' })
    downloadBlob(blob, `${wf.name}.agentflow.json`)
    ElMessage.success('导出成功')
  } catch {}
}

function handleAction(cmd: string, wf: WorkflowItem) {
  if (cmd === 'edit') handleEdit(wf)
  else if (cmd === 'clone') handleClone(wf)
  else if (cmd === 'export') handleExport(wf)
  else if (cmd === 'share') { shareWfId.value = wf.id; showShareDialog.value = true }
  else if (cmd === 'delete') handleDelete(wf)
}

function handleImportFile(file: any) {
  importFile.value = file.raw
}

async function doImport() {
  if (!importFile.value) { ElMessage.warning('请选择文件'); return }
  importing.value = true
  try {
    const res = await importWorkflow(importFile.value)
    importReport.value = res.data.data
    ElMessage.success('导入成功')
    showImportDialog.value = false
    fetchList()
  } catch {} finally {
    importing.value = false
  }
}

async function doShare() {
  try {
    const res = await shareWorkflow(shareWfId.value, { expires_at: shareExpiresAt.value || undefined })
    shareInfo.value = res.data.data
    ElMessage.success('分享链接已生成')
  } catch {}
}

function copyShareLink() {
  navigator.clipboard.writeText(shareInfo.value.share_url)
  ElMessage.success('已复制到剪贴板')
}

onMounted(fetchList)
</script>

<style scoped>
.workflow-card {
  cursor: pointer;
  transition: transform 0.2s;
}
.workflow-card:hover {
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
  align-items: center;
  font-size: 12px;
  color: #c0c4cc;
}
.card-meta span {
  display: flex;
  align-items: center;
  gap: 4px;
}
.empty-card {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
}
</style>
