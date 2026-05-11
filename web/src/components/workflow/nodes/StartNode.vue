<template>
  <div class="custom-node start-node">
    <Handle type="source" :position="Position.Right" />
    <div class="node-icon"><el-icon :size="20" color="#67c23a"><VideoPlay /></el-icon></div>
    <div class="node-label">{{ data.label || '开始' }}</div>
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
defineProps<{ data: { label: string } }>()
</script>

<style scoped>
.custom-node { padding: 10px 16px; border-radius: 8px; background: #fff; border: 2px solid; min-width: 120px; text-align: center; box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
.start-node { border-color: #67c23a; }
.node-icon { margin-bottom: 4px; }
.node-label { font-size: 13px; font-weight: 600; color: #303133; }
</style>
