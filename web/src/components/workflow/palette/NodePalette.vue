<template>
  <div class="node-palette">
    <div class="palette-title">节点面板</div>
    <div class="palette-list">
      <div
        v-for="(cfg, type) in NODE_TYPE_CONFIG"
        :key="type"
        class="palette-item"
        :style="{ borderLeftColor: cfg.color }"
        @click="$emit('add-node', type)"
      >
        <el-icon :size="18" :style="{ color: cfg.color }"><component :is="cfg.icon" /></el-icon>
        <span class="palette-label">{{ cfg.label }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NODE_TYPE_CONFIG } from '@/types/api'

defineEmits(['add-node'])
</script>

<style scoped>
.node-palette {
  width: 180px;
  background: #fff;
  border-right: 1px solid #e8e8e8;
  padding: 12px;
  overflow-y: auto;
}
.palette-title {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 12px;
  color: #303133;
}
.palette-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.palette-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 6px;
  border-left: 3px solid;
  cursor: pointer;
  transition: background 0.2s;
  font-size: 13px;
}
.palette-item:hover {
  background: #f0f5ff;
}
.palette-label {
  color: #303133;
}
</style>
