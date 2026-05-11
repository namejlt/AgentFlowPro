<template>
  <div class="markdown-viewer markdown-body" v-html="rendered" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'

const props = defineProps<{ content: string }>()

const rendered = computed(() => {
  return props.content ? marked(props.content) : ''
})
</script>

<style scoped>
.markdown-viewer {
  padding: 16px;
  line-height: 1.7;
}
</style>
