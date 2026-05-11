<template>
  <div class="exec-dialog">
    <el-alert v-if="!store.workflowId" title="请先保存工作流后再执行" type="warning" :closable="false" style="margin-bottom: 16px" />
    <el-form label-position="top">
      <el-form-item v-for="param in store.globalParams" :key="param.key" :label="param.label || param.key">
        <el-select v-if="param.type === 'select'" v-model="inputParams[param.key]" :placeholder="param.description || '请选择'">
          <el-option v-for="opt in param.options" :key="opt" :label="opt" :value="opt" />
        </el-select>
        <el-select v-else-if="param.type === 'multiselect'" v-model="inputParams[param.key]" multiple :placeholder="param.description || '请选择'">
          <el-option v-for="opt in param.options" :key="opt" :label="opt" :value="opt" />
        </el-select>
        <el-date-picker v-else-if="param.type === 'date'" v-model="inputParams[param.key]" type="date" />
        <el-input v-else-if="param.type === 'textarea'" v-model="inputParams[param.key]" type="textarea" :rows="3" :placeholder="param.description" />
        <el-input-number v-else-if="param.type === 'number'" v-model="inputParams[param.key]" />
        <el-input v-else v-model="inputParams[param.key]" :placeholder="param.description || '请输入'" />
      </el-form-item>
      <el-form-item label="执行模式">
        <el-radio-group v-model="execMode">
          <el-radio value="normal">正常</el-radio>
          <el-radio value="debug">调试</el-radio>
        </el-radio-group>
      </el-form-item>
    </el-form>
    <div class="exec-footer">
      <el-button @click="store.execDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" :disabled="!store.workflowId" @click="doExec">执行</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useWorkflowEditorStore } from '@/stores/workflowEditor'
import { createTask } from '@/api/tasks'
import { ElMessage } from 'element-plus'

const props = defineProps<{ workflowId: string }>()
const store = useWorkflowEditorStore()
const router = useRouter()
const submitting = ref(false)
const execMode = ref('normal')
const inputParams = reactive<Record<string, any>>({})

async function doExec() {
  submitting.value = true
  try {
    const res = await createTask({
      workflow_id: props.workflowId,
      input_params: { ...inputParams },
      mode: execMode.value,
    })
    ElMessage.success('任务已启动')
    store.execDialogVisible = false
    router.push(`/tasks/${res.data.data.id}`)
  } catch {} finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.exec-dialog {
  padding: 0 8px;
}
.exec-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>
