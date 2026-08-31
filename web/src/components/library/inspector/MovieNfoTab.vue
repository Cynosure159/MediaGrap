<script setup lang="ts">
defineProps<{
  content: string
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  saveToNfo: []
}>()
</script>

<template>
  <section class="workshop-view">
    <div class="workshop-header">
      <h2>{{ labels.nfoXmlEditor }}</h2>
      <button class="btn btn-success" type="button" @click="emit('saveToNfo')">
        {{ labels.saveToNfo }}
      </button>
    </div>

    <div class="xml-editor-box">
      <pre class="xml-code"><code>{{ content }}</code></pre>
    </div>
  </section>
</template>

<style scoped>
.workshop-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.workshop-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.workshop-header h2 {
  font-size: 16px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  margin: 0;
}

.xml-editor-box {
  background: var(--surface-container-lowest, #070d1e);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.5rem);
  padding: 14px;
  overflow: auto;
  max-height: 480px;
}

.xml-code {
  font-family: var(--font-code, monospace);
  font-size: 12px;
  line-height: 1.6;
  color: var(--primary, #c0c1ff);
  margin: 0;
  white-space: pre;
}
</style>
