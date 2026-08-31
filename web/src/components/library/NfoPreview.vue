<script setup lang="ts">
import type { WritePlan } from '@/api/library'

defineProps<{ plan: WritePlan | null; applying: boolean; labels: Record<string, string> }>()
const emit = defineEmits<{ apply: []; close: [] }>()
</script>

<template>
  <div v-if="plan" class="backdrop" @click.self="emit('close')">
    <section class="dialog card" role="dialog" aria-modal="true" aria-labelledby="nfo-preview-title">
      <header class="dialog-header">
        <div class="header-info">
          <div class="header-badges">
            <p class="eyebrow">{{ labels.nfoSavePreview }}</p>
            <span v-if="plan.state" class="spec-badge state-badge">{{ plan.state }}</span>
            <span v-if="plan.conflict" class="spec-badge badge-conflict">
              <span class="status-dot status-dot--error"></span> {{ labels.conflict }}
            </span>
            <span v-else-if="plan.willReplace" class="spec-badge badge-replace">
              <span class="status-dot status-dot--warning"></span> {{ labels.overwrite }}
            </span>
            <span v-else class="spec-badge badge-new">
              <span class="status-dot status-dot--ok"></span> {{ labels.newFile }}
            </span>
          </div>
          <h3 id="nfo-preview-title" class="dialog-title">{{ labels.nfoPreview }}</h3>
        </div>
        <button
          type="button"
          class="btn btn-ghost btn-icon btn-sm close-btn"
          :aria-label="labels.closeNfoPreview"
          @click="emit('close')"
        >
          ✕
        </button>
      </header>

      <div class="dialog-body">
        <div class="target-path-display">
          <span class="target-label">{{ labels.target }}</span>
          <span class="target-path" :title="plan.targetPath">{{ plan.targetPath }}</span>
        </div>

        <div v-if="plan.conflict" class="banner banner-conflict" role="alert">
          <span class="status-dot status-dot--error"></span>
          <span>{{ labels.nfoConflictHelp }}</span>
        </div>

        <div v-else-if="plan.willReplace" class="banner banner-replace">
          <span class="status-dot status-dot--ok"></span>
          <span>{{ labels.nfoReplaceHelp }}</span>
        </div>

        <div class="xml-section">
          <div class="xml-toolbar">
            <span class="xml-title">{{ labels.xmlContent }}</span>
            <span class="spec-badge">UTF-8</span>
          </div>
          <pre class="xml-code"><code>{{ plan.content }}</code></pre>
        </div>
      </div>

      <footer class="dialog-footer">
        <button type="button" class="btn btn-ghost" @click="emit('close')">
          {{ labels.cancel }}
        </button>
        <button
          type="button"
          class="btn btn-success"
          :disabled="plan.conflict || applying"
          @click="emit('apply')"
        >
          {{ applying ? labels.saving : labels.apply }}
        </button>
      </footer>
    </section>
  </div>
</template>

<style scoped>
.backdrop {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: grid;
  place-items: end center;
  padding: 1rem;
  background: rgba(7, 13, 31, 0.8);
  backdrop-filter: blur(4px);
}

@media (min-width: 700px) {
  .backdrop {
    place-items: center;
  }
}

.dialog {
  width: min(100%, 52rem);
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  box-shadow: 0 1.5rem 4rem rgba(0, 0, 0, 0.6);
  overflow: hidden;
}

.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--surface-card);
  flex-shrink: 0;
}

.header-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.header-badges {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.dialog-title {
  margin: 0;
  font-family: var(--font-display);
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.01em;
}

.close-btn {
  color: var(--text-muted);
  font-size: 0.875rem;
  flex-shrink: 0;
}

.close-btn:hover {
  color: var(--text-primary);
}

.dialog-body {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  padding: 1.25rem;
  overflow-y: auto;
  min-height: 0;
  flex: 1;
}

.target-path-display {
  display: flex;
  align-items: baseline;
  gap: 0.6rem;
  padding: 0.55rem 0.8rem;
  background: var(--surface-panel);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.target-label {
  font-family: var(--font-data);
  font-size: 0.675rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  flex-shrink: 0;
}

.target-path {
  font-family: var(--font-data);
  font-size: 0.775rem;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  direction: rtl;
  text-align: left;
}

.banner {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.65rem 0.85rem;
  border-radius: var(--radius-sm);
  font-size: 0.8125rem;
  line-height: 1.4;
}

.banner-conflict {
  background: rgba(202, 129, 0, 0.18);
  border: 1px solid var(--warning);
  color: var(--warning);
}

.banner-replace {
  background: rgba(0, 165, 114, 0.12);
  border: 1px solid var(--success-container);
  color: var(--success);
}

.badge-conflict {
  border-color: var(--error-container);
  color: var(--error);
}

.badge-replace {
  border-color: var(--warning-container);
  color: var(--warning);
}

.badge-new {
  border-color: var(--success-container);
  color: var(--success);
}

.state-badge {
  text-transform: uppercase;
}

.xml-section {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--surface-dim);
}

.xml-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.4rem 0.75rem;
  background: var(--surface-panel);
  border-bottom: 1px solid var(--border-subtle);
}

.xml-title {
  font-family: var(--font-data);
  font-size: 0.675rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-muted);
}

.xml-code {
  margin: 0;
  padding: 1rem;
  background: var(--surface-dim);
  color: var(--text-secondary);
  font-family: var(--font-data);
  font-size: 0.775rem;
  line-height: 1.5;
  overflow-x: auto;
  overflow-y: auto;
  max-height: 22rem;
  white-space: pre;
  tab-size: 2;
}

.xml-code code {
  font-family: inherit;
  color: inherit;
}

.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 0.85rem 1.25rem;
  border-top: 1px solid var(--border-subtle);
  background: var(--surface-card);
  flex-shrink: 0;
}
</style>
