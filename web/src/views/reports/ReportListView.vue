<template>
  <div class="page-container">
    <div class="page-header">
      <h2>历史报告</h2>
      <el-button type="danger" :disabled="selectedIds.length === 0" @click="handleBatchDelete">
        <el-icon><Delete /></el-icon>批量删除 ({{ selectedIds.length }})
      </el-button>
    </div>

    <el-card style="margin-bottom: 16px">
      <el-row :gutter="16" align="middle">
        <el-col :span="6">
          <el-input v-model="keyword" placeholder="搜索报告..." prefix-icon="Search" clearable @input="fetchList" />
        </el-col>
        <el-col :span="4">
          <el-select v-model="filterStatus" placeholder="状态" clearable @change="fetchList">
            <el-option label="已完成" value="completed" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-col>
        <el-col :span="4">
          <el-select v-model="filterArchived" placeholder="归档" clearable @change="fetchList">
            <el-option label="已归档" value="true" />
            <el-option label="未归档" value="false" />
          </el-select>
        </el-col>
      </el-row>
    </el-card>

    <el-card v-loading="loading">
      <el-table :data="reports" stripe @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="50" />
        <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
        <el-table-column prop="workflow_name" label="工作流" min-width="120" />
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 'completed' ? 'success' : 'danger'" size="small">{{ row.status === 'completed' ? '完成' : '失败' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="100">
          <template #default="{ row }">{{ formatDuration(row.duration_ms) }}</template>
        </el-table-column>
        <el-table-column label="Tokens" width="90">
          <template #default="{ row }">{{ row.total_tokens || '-' }}</template>
        </el-table-column>
        <el-table-column label="归档" width="70">
          <template #default="{ row }">
            <el-tag v-if="row.archived" size="small" type="info">已归档</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="$router.push(`/reports/${row.id}`)">查看</el-button>
            <ExportMenu style="display: inline" @export="(fmt: string) => handleExport(fmt, row)" />
            <el-button link type="warning" @click="handleArchive(row)">{{ row.archived ? '取消归档' : '归档' }}</el-button>
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
import type { ReportItem } from '@/types'
import ExportMenu from '@/components/report/ExportMenu.vue'
import { getReports, deleteReport, archiveReport, batchDeleteReports, exportReportMd, exportReportPdf, exportReportDocx } from '@/api/reports'
import { formatDateTime, formatDuration } from '@/utils/datetime'
import { downloadBlob } from '@/utils/download'
import { ElMessage, ElMessageBox } from 'element-plus'

const reports = ref<ReportItem[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const filterStatus = ref('')
const filterArchived = ref('')
const selectedIds = ref<string[]>([])

const loading = ref(false)

async function fetchList() {
  loading.value = true
  try {
    const res = await getReports({ page: page.value, page_size: pageSize.value, keyword: keyword.value, status: filterStatus.value || undefined, archived: filterArchived.value === 'true' ? true : filterArchived.value === 'false' ? false : undefined })
    reports.value = res.data.data || []
    total.value = res.data.meta?.total || 0
  } catch (e: any) {
    ElMessage.error(e.message || '获取报告列表失败')
  } finally {
    loading.value = false
  }
}

function handleSelectionChange(rows: ReportItem[]) {
  selectedIds.value = rows.map(r => r.id)
}

async function handleExport(format: string, row: ReportItem) {
  try {
    let res: any
    let filename = `${row.title}.${format}`
    if (format === 'md') res = await exportReportMd(row.id)
    else if (format === 'pdf') { res = await exportReportPdf(row.id); filename = `${row.title}.pdf` }
    else { res = await exportReportDocx(row.id); filename = `${row.title}.docx` }
    downloadBlob(res.data, filename)
    ElMessage.success('导出成功')
  } catch (e: any) {
    ElMessage.error(e.message || '导出失败')
  }
}

async function handleArchive(row: ReportItem) {
  try {
    await archiveReport(row.id, !row.archived)
    ElMessage.success(row.archived ? '已取消归档' : '已归档')
    fetchList()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function handleDelete(row: ReportItem) {
  await ElMessageBox.confirm(`确定删除报告「${row.title}」？`, '提示', { type: 'warning' })
  try {
    await deleteReport(row.id)
    ElMessage.success('删除成功')
    fetchList()
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  }
}

async function handleBatchDelete() {
  await ElMessageBox.confirm(`确定批量删除 ${selectedIds.value.length} 条报告？`, '提示', { type: 'warning' })
  try {
    await batchDeleteReports(selectedIds.value)
    ElMessage.success('批量删除成功')
    fetchList()
  } catch (e: any) {
    ElMessage.error(e.message || '批量删除失败')
  }
}

onMounted(fetchList)
</script>
