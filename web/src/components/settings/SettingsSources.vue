<script setup lang="ts">
import type { Source } from '@/api/library'
import AddSourceForm from './AddSourceForm.vue'
import SourceListItem from './SourceListItem.vue'

defineProps<{
  sources: Source[]
  mediaRoots: string[]
  labels: Record<string, string>
  feedback: { kind: 'success' | 'error'; message: string } | null
}>()

const sourceName = defineModel<string>('sourceName', { required: true })
const sourcePath = defineModel<string>('sourcePath', { required: true })

const emit = defineEmits<{
  add: []
  scan: [id: number]
  delete: [id: number]
	savePolicy: [id: number, policy: Pick<Source, 'scanMode' | 'scheduleEnabled' | 'scheduleIntervalMinutes'>]
}>()
</script>

<template>
  <section class="settings-card">
    <div class="card-header">
      <div class="card-header__info">
        <div class="header-tag">
          <svg class="header-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"/>
          </svg>
          <span class="eyebrow">{{ labels.libraryEyebrow }}</span>
        </div>
        <h2 class="section-title">{{ labels.directorySettings }}</h2>
        <p class="section-hint">{{ labels.directoryHelp }}</p>
      </div>
    </div>

    <div class="card-body">
      <!-- Add Source Form -->
      <AddSourceForm
        v-model:source-name="sourceName"
        v-model:source-path="sourcePath"
        :media-roots="mediaRoots"
        :labels="labels"
        @add="emit('add')"
      />

      <!-- Feedback Alert -->
      <div
        v-if="feedback"
        class="source-feedback"
        :class="`source-feedback--${feedback.kind}`"
        role="status"
      >
        <span
          class="feedback-dot"
          :class="feedback.kind === 'success' ? 'dot-ok' : 'dot-err'"
        ></span>
        <span class="feedback-text">{{ feedback.message }}</span>
      </div>

      <!-- Empty State -->
      <div v-if="sources.length === 0" class="empty-sources">
        <div class="empty-icon-box">
          <svg class="empty-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"/>
          </svg>
        </div>
        <p class="empty-title">{{ labels.noSources }}</p>
        <p class="empty-desc">{{ labels.firstSource }}</p>
      </div>

      <!-- Configured Source List -->
      <div v-else class="source-list">
        <SourceListItem
          v-for="source in sources"
          :key="source.id"
          :source="source"
          :labels="labels"
          @scan="emit('scan', $event)"
          @delete="emit('delete', $event)"
		  @save-policy="(id, policy) => emit('savePolicy', id, policy)"
        />
      </div>
    </div>
  </section>
</template>

<style scoped>
.settings-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-xl, 0.75rem);
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 24px 28px 20px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-container-high, #23293c);
}

.card-header__info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.header-tag {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--primary, #c0c1ff);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.header-icon {
  width: 14px;
  height: 14px;
}

.section-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  margin: 0;
  letter-spacing: -0.01em;
}

.section-hint {
  font-size: 13px;
  color: var(--on-surface-variant, #c7c4d7);
  margin: 0;
}

.card-body {
  padding: 28px;
}

.source-feedback {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-radius: var(--radius-md, 0.375rem);
  margin-bottom: 24px;
  font-size: 13px;
}

.source-feedback--success {
  background: rgba(78, 222, 163, 0.1);
  border: 1px solid rgba(78, 222, 163, 0.25);
  color: var(--secondary, #4edea3);
}

.source-feedback--error {
  background: rgba(255, 77, 79, 0.1);
  border: 1px solid rgba(255, 77, 79, 0.25);
  color: #ff4d4f;
}

.feedback-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.empty-sources {
  padding: 48px 24px;
  text-align: center;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px dashed var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
}

.empty-icon-box {
  width: 48px;
  height: 48px;
  margin: 0 auto 16px;
  border-radius: 50%;
  background: var(--surface-container-high, #23293c);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--outline, #908fa0);
}

.empty-icon {
  width: 24px;
  height: 24px;
}

.empty-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  margin: 0 0 6px 0;
}

.empty-desc {
  font-size: 13px;
  color: var(--on-surface-variant, #c7c4d7);
  margin: 0;
}

.source-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
