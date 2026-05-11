<template>
  <el-container class="app-shell">
    <SideNav :collapsed="navCollapsed" @toggle="navCollapsed = !navCollapsed" />
    <el-container direction="vertical">
      <TopBar @toggle-nav="navCollapsed = !navCollapsed" />
      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import SideNav from './SideNav.vue'
import TopBar from './TopBar.vue'

const navCollapsed = ref(false)
</script>

<style scoped>
.app-shell {
  height: 100vh;
}
.app-main {
  background: #f5f7fa;
  overflow-y: auto;
  padding: 0;
}
</style>
