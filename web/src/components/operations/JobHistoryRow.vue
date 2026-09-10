<script setup lang="ts">
import type { Job } from '@/api/types'
const props = defineProps<{ job: Job; labels: Record<string, string> }>()
const emit = defineEmits<{ retry: [id: number] }>()
function jobLabel(kind = ''): string {
  return props.labels[`jobKind_${kind}`] || kind.replaceAll('_', ' ')
}
</script>

<template>
  <article class="history-row">
    <span class="state-indicator" :class="`state-indicator--${job.state}`"></span>

    <div class="history-info">
      <div class="history-head">
        <span class="history-kind">{{ jobLabel(job.kind) }}</span>
        <span class="history-id font-code">#{{ job.id }}</span>
      </div>
      <p class="history-desc" :class="{ 'history-desc--err': !!job.errorMessage }">
        {{ job.errorMessage || job.message || '—' }}
      </p>
    </div>

    <div class="history-status-group">
      <span class="history-state-pill font-code" :class="`state-pill--${job.state}`">
        {{ labels[`jobState_${job.state}`] || job.state }}
      </span>
      <button
        v-if="['failed', 'cancelled', 'interrupted'].includes(job.state)"
        class="btn btn-ghost btn-xs btn-retry"
        type="button"
        @click="emit('retry', job.id)"
      >
        <svg viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
          <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
        </svg>
        <span>{{ labels.retryJob || 'Retry' }}</span>
      </button>
    </div>
  </article>
</template>

<style scoped>
.history-row {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 12px 20px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  transition: background 0.12s ease;
}

.history-row:hover {
  background: var(--surface-container-high, #23293c);
}

.state-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--outline, #908fa0);
}

.state-indicator--succeeded {
  background: var(--secondary, #4edea3);
  box-shadow: 0 0 6px rgba(78, 222, 163, 0.4);
}

.state-indicator--failed {
  background: var(--error, #ffb4ab);
  box-shadow: 0 0 6px rgba(255, 180, 171, 0.4);
}

.state-indicator--cancelled,
.state-indicator--interrupted {
  background: var(--tertiary, #ffb95f);
}

.history-info {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.history-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.history-kind {
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
}

.history-id {
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.history-desc {
  margin: 0;
  font-size: 11px;
  color: var(--on-surface-variant, #c7c4d7);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.history-desc--err {
  color: var(--error, #ffb4ab);
}

.history-status-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.history-state-pill {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--surface-container-high, #23293c);
  color: var(--outline, #908fa0);
  text-transform: uppercase;
}

.state-pill--succeeded {
  color: var(--secondary, #4edea3);
  background: rgba(78, 222, 163, 0.1);
}

.state-pill--failed {
  color: var(--error, #ffb4ab);
  background: rgba(255, 180, 171, 0.1);
}

.btn-retry {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--primary, #c0c1ff);
  font-size: 11px;
  padding: 3px 7px;
}

@media (max-width: 600px) {
  .history-row { padding: 10px 14px; grid-template-columns: 8px minmax(0, 1fr); }
  .history-status-group { grid-column: 2; margin-top: 4px; }
}
</style>
