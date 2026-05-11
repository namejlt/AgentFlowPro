<template>
  <div class="custom-node summarize-node">
    <Handle type="target" :position="Position.Left" />
    <Handle type="source" :position="Position.Right" />
    <div class="node-icon"><el-icon :size="20" color="#607d8b"><Document /></el-icon></div>
    <div class="node-label">{{ data.label || '汇总' }}</div>
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
defineProps<{ data: { label: string } }>()
</script>

<style scoped>
.custom-node { padding: 10px 16px; border-radius: 8px; background: #fff; border: 2px solid; min-width: 120px; text-align: center; box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
.summarize-node { border-color: #607d8b; }
.node-icon { margin-bottom: 4px; }
.node-label { font-size: 13px; font-weight: 600; color: #303133; }
</style>
