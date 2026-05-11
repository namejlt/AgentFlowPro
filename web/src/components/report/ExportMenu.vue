<template>
  <el-dropdown trigger="click" @command="(cmd: string) => $emit('export', cmd)">
    <el-button><el-icon><Download /></el-icon>导出</el-button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item command="md">Markdown</el-dropdown-item>
        <el-dropdown-item command="pdf">PDF</el-dropdown-item>
        <el-dropdown-item command="docx">DOCX</el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
defineEmits(['export'])
</script>
