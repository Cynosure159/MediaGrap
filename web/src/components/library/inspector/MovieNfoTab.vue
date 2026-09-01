<script setup lang="ts">
import { ref } from 'vue'

const props = withDefaults(
  defineProps<{
    content: string
    labels: Record<string, string>
    filename?: string
    headerLabel?: string
    actionLabel?: string
    readOnly?: boolean
  }>(),
  {
    filename: 'movie.nfo',
    readOnly: false,
  }
)

const emit = defineEmits<{
  saveToNfo: []
}>()

const copied = ref(false)

async function copyXml() {
  try {
    await navigator.clipboard.writeText(props.content)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    // fallback
  }
}
</script>

<template>
  <section class="nfo-workshop-view">
    <div class="workshop-header">
      <div class="header-left">
        <h2>{{ headerLabel || labels.nfoXmlEditor || 'Kodi NFO Raw XML' }}</h2>
        <span class="sub-label">Deterministic UTF-8 Kodi sidecar metadata specification</span>
      </div>
      <div class="workshop-actions">
        <button class="btn btn-outline" type="button" @click="copyXml">
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
          </svg>
          {{ copied ? (labels.copied || 'Copied!') : (labels.copy || 'Copy XML') }}
        </button>
        <button v-if="!readOnly" class="btn btn-success" type="button" @click="emit('saveToNfo')">
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M17 3H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V7l-4-4zm-5 16c-1.66 0-3-1.34-3-3s1.34-3 3-3 3 1.34 3 3-1.34 3-3 3zm3-10H5V5h10v4z"/>
          </svg>
          {{ actionLabel || labels.saveToNfo || 'Save to movie.nfo' }}
        </button>
      </div>
    </div>

    <!-- XML Editor / Code Canvas -->
    <div class="xml-editor-canvas">
      <div class="xml-canvas-header">
        <div class="file-tag font-code">{{ filename }}</div>
        <div class="encoding-tag font-code">UTF-8 • XML v1.0 • Kodi v20/v21</div>
      </div>
      <pre class="xml-code font-code"><code>{{ content }}</code></pre>
    </div>
  </section>
</template>

<style scoped>
.nfo-workshop-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}

.workshop-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 4px;
}

.header-left h2 {
  font-size: 18px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  margin: 0 0 2px 0;
  letter-spacing: -0.01em;
}

.sub-label {
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
}

.workshop-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.xml-editor-canvas {
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.4);
}

.xml-canvas-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 14px;
  background: var(--surface-container-low, #151b2d);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.file-tag {
  font-size: 12px;
  font-weight: 700;
  color: var(--primary, #c0c1ff);
}

.encoding-tag {
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.xml-code {
  margin: 0;
  padding: 16px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--on-surface, #dce1fb);
  overflow-x: auto;
  max-height: 520px;
}
</style>
