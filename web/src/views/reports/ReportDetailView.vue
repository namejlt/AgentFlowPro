<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ report.title || '报告详情' }}</h2>
      <div>
        <ExportMenu style="margin-right: 8px" @export="handleExport" />
        <el-button @click="$router.push('/reports')">返回</el-button>
      </div>
    </div>

    <el-descriptions :column="4" border style="margin-bottom: 16px">
      <el-descriptions-item label="状态">
        <el-tag :type="report.status === 'completed' ? 'success' : 'danger'">{{ report.status === 'completed' ? '完成' : '失败' }}</el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="耗时">{{ formatDuration(report.duration_ms) }}</el-descriptions-item>
      <el-descriptions-item label="Tokens">{{ report.total_tokens || '-' }}</el-descriptions-item>
      <el-descriptions-item label="创建时间">{{ formatDateTime(report.created_at) }}</el-descriptions-item>
    </el-descriptions>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="报告正文" name="content">
        <el-card>
          <MarkdownViewer :content="report.content_md" />
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="智能体输出" name="agents">
        <el-card>
          <el-collapse>
            <el-collapse-item v-for="(output, key) in report.agent_outputs" :key="key" :title="String(key)">
              <MarkdownViewer :content="typeof output === 'string' ? output : JSON.stringify(output, null, 2)" />
            </el-collapse-item>
          </el-collapse>
          <el-empty v-if="!report.agent_outputs || Object.keys(report.agent_outputs).length === 0" description="暂无智能体输出" />
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="辩论过程" name="debate">
        <el-card>
          <div v-for="log in report.debate_logs" :key="log.round + log.agent_id" class="debate-bubble" :class="log.round % 2 === 0 ? 'left' : 'right'">
            <div class="agent-name">{{ log.agent_name }} <span class="round-tag">第{{ log.round }}轮</span></div>
            <div class="debate-content"><MarkdownViewer :content="log.output" /></div>
          </div>
          <el-empty v-if="!report.debate_logs || report.debate_logs.length === 0" description="暂无辩论记录" />
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="执行日志" name="logs">
        <el-card>
          <el-table :data="report.exec_logs" stripe>
            <el-table-column label="时间" width="170">
              <template #default="{ row }">{{ formatDateTime(row.timestamp) }}</template>
            </el-table-column>
            <el-table-column prop="node_id" label="节点" width="120" />
            <el-table-column prop="node_type" label="类型" width="120" />
            <el-table-column prop="action" label="动作" width="120" />
            <el-table-column prop="detail" label="详情" min-width="300" show-overflow-tooltip />
          </el-table>
          <el-empty v-if="!report.exec_logs || report.exec_logs.length === 0" description="暂无执行日志" />
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import type { ReportItem } from '@/types'
import ExportMenu from '@/components/report/ExportMenu.vue'
import MarkdownViewer from '@/components/report/MarkdownViewer.vue'
import { getReport, exportReportMd, exportReportPdf, exportReportDocx } from '@/api/reports'
import { formatDateTime, formatDuration } from '@/utils/datetime'
import { downloadBlob } from '@/utils/download'
import { ElMessage } from 'element-plus'

const route = useRoute()
const report = ref<ReportItem>({} as ReportItem)
const activeTab = ref('content')

async function handleExport(format: string) {
  try {
    let res: any
    let filename = `${report.value.title}.${format}`
    if (format === 'md') res = await exportReportMd(report.value.id)
    else if (format === 'pdf') { res = await exportReportPdf(report.value.id); filename = `${report.value.title}.pdf` }
    else { res = await exportReportDocx(report.value.id); filename = `${report.value.title}.docx` }
    downloadBlob(res.data, filename)
    ElMessage.success('导出成功')
  } catch {}
}

onMounted(async () => {
  try {
    const res = await getReport(route.params.id as string)
    report.value = res.data.data
  } catch {}
})
</script>

<style scoped>
.debate-content {
  font-size: 13px;
  line-height: 1.6;
}
</style>
