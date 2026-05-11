<template>
  <div class="page-container">
    <div class="page-header">
      <h2>全局配置</h2>
    </div>

    <el-card>
      <el-form label-width="200px" style="max-width: 700px">
        <el-form-item v-for="cfg in configs" :key="cfg.id" :label="cfg.cfg_key">
          <div style="display: flex; gap: 8px; width: 100%">
            <el-input v-if="typeof cfg.cfg_value === 'string'" v-model="cfg.cfg_value" style="flex: 1" />
            <el-input-number v-else-if="typeof cfg.cfg_value === 'number'" v-model="cfg.cfg_value" style="flex: 1" />
            <el-switch v-else-if="typeof cfg.cfg_value === 'boolean'" v-model="cfg.cfg_value" />
            <el-input v-else :model-value="JSON.stringify(cfg.cfg_value)" style="flex: 1" @update:model-value="(v: string) => { try { cfg.cfg_value = JSON.parse(v) } catch {} }" />
          </div>
          <div v-if="cfg.description" style="font-size: 12px; color: #909399; margin-top: 4px">{{ cfg.description }}</div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="handleSave">保存配置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { SystemConfigItem } from '@/types'
import { getSystemConfig, updateSystemConfig } from '@/api/system'
import { ElMessage } from 'element-plus'

const configs = ref<SystemConfigItem[]>([])
const saving = ref(false)

async function fetchConfig() {
  try {
    const res = await getSystemConfig()
    configs.value = res.data.data || []
  } catch {}
}

async function handleSave() {
  saving.value = true
  try {
    const data: Record<string, any> = {}
    configs.value.forEach(cfg => { data[cfg.cfg_key] = cfg.cfg_value })
    await updateSystemConfig(data)
    ElMessage.success('配置已保存')
  } catch {} finally {
    saving.value = false
  }
}

onMounted(fetchConfig)
</script>
