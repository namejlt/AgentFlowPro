<template>
  <div class="global-params-editor">
    <div v-for="(param, idx) in store.globalParams" :key="idx" class="param-item">
      <el-card shadow="never">
        <el-form label-position="top" size="small">
          <el-row :gutter="8">
            <el-col :span="8">
              <el-form-item label="变量名">
                <el-input v-model="param.key" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="显示标签">
                <el-input v-model="param.label" />
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="类型">
                <el-select v-model="param.type">
                  <el-option label="字符串" value="string" />
                  <el-option label="数字" value="number" />
                  <el-option label="日期" value="date" />
                  <el-option label="下拉选择" value="select" />
                  <el-option label="多选" value="multiselect" />
                  <el-option label="文本域" value="textarea" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="2">
              <el-button link type="danger" @click="removeParam(idx)"><el-icon><Delete /></el-icon></el-button>
            </el-col>
          </el-row>
          <el-row :gutter="8">
            <el-col :span="8">
              <el-form-item label="必填">
                <el-switch v-model="param.required" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="默认值">
                <el-input v-model="param.default_value" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="描述">
                <el-input v-model="param.description" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row v-if="param.type === 'select' || param.type === 'multiselect'" :gutter="8">
            <el-col :span="24">
              <el-form-item label="选项列表(逗号分隔)">
                <el-input :model-value="(param.options || []).join(',')" @update:model-value="(v: string) => { param.options = v ? v.split(',').map(s => s.trim()) : [] }" />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
      </el-card>
    </div>
    <el-button type="primary" plain style="width: 100%; margin-top: 12px" @click="addParam">
      <el-icon><Plus /></el-icon>添加全局入参
    </el-button>
  </div>
</template>

<script setup lang="ts">
import { useWorkflowEditorStore } from '@/stores/workflowEditor'
import type { GlobalParam } from '@/types'

const store = useWorkflowEditorStore()

function addParam() {
  store.globalParams.push({
    key: '',
    label: '',
    type: 'string',
    required: false,
    default_value: '',
    options: [],
    description: '',
    sort_order: store.globalParams.length,
  } as GlobalParam)
}

function removeParam(idx: number) {
  store.globalParams.splice(idx, 1)
}
</script>

<style scoped>
.param-item {
  margin-bottom: 12px;
}
</style>
