<template>
  <div class="page-container">
    <div class="page-header">
      <h2>数据源管理</h2>
      <el-button type="primary" @click="handleCreate">
        <el-icon><Plus /></el-icon>创建数据源
      </el-button>
    </div>

    <el-card style="margin-bottom: 16px">
      <el-row :gutter="16" align="middle">
        <el-col :span="6">
          <el-input v-model="keyword" placeholder="搜索数据源..." prefix-icon="Search" clearable @input="fetchList" />
        </el-col>
        <el-col :span="4">
          <el-select v-model="filterType" placeholder="类型" clearable @change="fetchList">
            <el-option label="HTTP GET" value="HTTP_GET" />
            <el-option label="HTTP POST" value="HTTP_POST" />
            <el-option label="文件上传" value="FILE_UPLOAD" />
            <el-option label="手动输入" value="MANUAL_INPUT" />
            <el-option label="WebSocket" value="WEBSOCKET_STREAM" />
          </el-select>
        </el-col>
        <el-col :span="4">
          <el-select v-model="filterCategory" placeholder="分类" clearable @change="fetchList">
            <el-option label="金融" value="finance" />
            <el-option label="教育" value="education" />
            <el-option label="通用" value="general" />
          </el-select>
        </el-col>
      </el-row>
    </el-card>

    <el-card v-loading="loading">
      <el-table :data="datasources" stripe>
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="ds_type" label="类型" width="130">
          <template #default="{ row }">
            <el-tag size="small">{{ dsTypeLabel(row.ds_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="category" label="分类" width="80" />
        <el-table-column prop="auth_type" label="鉴权" width="110">
          <template #default="{ row }">{{ authTypeLabel(row.auth_type) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="测试状态" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.last_test_status === 'success'" type="success" size="small">通过</el-tag>
            <el-tag v-else-if="row.last_test_status === 'failed'" type="danger" size="small">失败</el-tag>
            <span v-else style="color: #c0c4cc">未测试</span>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="170">
          <template #default="{ row }">{{ formatDateTime(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="primary" @click="handleTest(row)">测试</el-button>
            <el-button link type="primary" @click="handleClone(row)">复制</el-button>
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import type { DataSourceItem } from '@/types'
import { getDataSources, cloneDataSource, deleteDataSource, testDataSource, patchDataSourceStatus } from '@/api/datasources'
import { formatDateTime } from '@/utils/datetime'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()
const datasources = ref<DataSourceItem[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const filterType = ref('')
const filterCategory = ref('')

function dsTypeLabel(type: string) {
  const map: Record<string, string> = { HTTP_GET: 'HTTP GET', HTTP_POST: 'HTTP POST', FILE_UPLOAD: '文件上传', MANUAL_INPUT: '手动输入', WEBSOCKET_STREAM: 'WebSocket' }
  return map[type] || type
}

function authTypeLabel(type: string) {
  const map: Record<string, string> = { none: '无', bearer: 'Bearer', api_key_header: 'API Key', custom_header: '自定义' }
  return map[type] || type
}

const loading = ref(false)

async function fetchList() {
  loading.value = true
  try {
    const res = await getDataSources({ page: page.value, page_size: pageSize.value, keyword: keyword.value, type: filterType.value || undefined, category: filterCategory.value || undefined })
    datasources.value = res.data.data || []
    total.value = res.data.meta?.total || 0
  } catch (e: any) {
    ElMessage.error(e.message || '获取数据源列表失败')
  } finally {
    loading.value = false
  }
}

function handleCreate() {
  router.push('/datasources/new')
}

function handleEdit(ds: DataSourceItem) {
  router.push(`/datasources/${ds.id}`)
}

async function handleTest(ds: DataSourceItem) {
  try {
    const res = await testDataSource(ds.id)
    if (res.data.data.ok) {
      ElMessage.success('测试通过')
    } else {
      ElMessage.error(`测试失败: ${res.data.data.error || '未知错误'}`)
    }
    fetchList()
  } catch (e: any) {
    ElMessage.error(e.message || '测试失败')
  }
}

async function handleClone(ds: DataSourceItem) {
  try {
    await cloneDataSource(ds.id)
    ElMessage.success('复制成功')
    fetchList()
  } catch (e: any) {
    ElMessage.error(e.message || '复制失败')
  }
}

async function handleDelete(ds: DataSourceItem) {
  await ElMessageBox.confirm(`确定删除数据源「${ds.name}」？`, '提示', { type: 'warning' })
  try {
    await deleteDataSource(ds.id)
    ElMessage.success('删除成功')
    fetchList()
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  }
}

onMounted(fetchList)
</script>
