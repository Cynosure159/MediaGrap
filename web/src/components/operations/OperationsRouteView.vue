<script setup lang="ts">
import OperationsPage from './OperationsPage.vue'

defineOptions({ inheritAttrs: false })

defineProps<{
  csrfToken: string
  labels: Record<string, string>
}>()
</script>

<template>
  <OperationsPage :csrf-token="csrfToken" :labels="labels" />
</template>
