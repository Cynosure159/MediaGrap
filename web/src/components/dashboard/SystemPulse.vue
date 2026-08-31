<script setup lang="ts">
import type { SystemSummary } from '@/api/system'

defineProps<{
  summary: SystemSummary | null
  isLoading: boolean
  error: string | null
  connectionLabel: string
}>()

const emit = defineEmits<{
  refresh: []
}>()
</script>

<template>
  <section class="system-pulse" aria-labelledby="pulse-title">
    <div class="pulse-copy">
      <p class="eyebrow">Service pulse</p>
      <h2 id="pulse-title" class="pulse-title">{{ connectionLabel }}</h2>
      <p v-if="error" class="pulse-description pulse-description--error">{{ error }}</p>
      <p v-else class="pulse-description">
        SQLite is initialized in WAL mode. The API and embedded application shell are responding.
      </p>
    </div>

    <div class="pulse-signal" :class="{ 'pulse-signal--error': error }" aria-hidden="true">
      <span></span><span></span><span></span>
    </div>

    <button
      v-if="error"
      class="btn btn-outline btn-sm retry-button"
      type="button"
      :disabled="isLoading"
      @click="emit('refresh')"
    >
      Try again
    </button>

    <dl class="status-grid">
      <div class="status-cell">
        <dt class="status-label">Sources</dt>
        <dd class="status-value">{{ summary?.library.sources ?? '—' }}</dd>
      </div>
      <div class="status-cell">
        <dt class="status-label">Indexed items</dt>
        <dd class="status-value">{{ summary?.library.items ?? '—' }}</dd>
      </div>
      <div class="status-cell">
        <dt class="status-label">Active jobs</dt>
        <dd class="status-value">{{ summary?.jobs.running ?? '—' }}</dd>
      </div>
    </dl>
  </section>
</template>

<style scoped>
.system-pulse {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 1.5rem;
  padding: clamp(1.25rem, 3vw, 2rem);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  background: linear-gradient(135deg, var(--surface-card) 0%, var(--surface-panel) 100%);
  color: var(--text-primary);
  overflow: hidden;
}

.pulse-copy {
  max-width: 38rem;
}

.eyebrow {
  margin: 0;
  color: var(--text-muted);
  font: 500 0.65rem/1.4 var(--font-data);
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.pulse-title {
  margin: 0.35rem 0 0.6rem;
  font-family: var(--font-display);
  font-size: clamp(2rem, 5vw, 3.25rem);
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: -0.04em;
  line-height: 1.05;
}

.pulse-description {
  max-width: 36rem;
  margin: 0;
  color: var(--text-secondary);
  font-size: 0.875rem;
  line-height: 1.55;
}

.pulse-description--error {
  color: var(--error);
}

.pulse-signal {
  display: flex;
  align-items: flex-end;
  gap: 0.35rem;
  height: 3.5rem;
  padding-top: 0.5rem;
}

.pulse-signal span {
  width: 0.4rem;
  background: var(--success);
  border-radius: var(--radius-full);
  animation: pulse 1.6s ease-in-out infinite;
}

.pulse-signal span:nth-child(1) {
  height: 40%;
  animation-delay: -0.4s;
}

.pulse-signal span:nth-child(2) {
  height: 100%;
  animation-delay: -0.2s;
}

.pulse-signal span:nth-child(3) {
  height: 65%;
}

.pulse-signal--error span {
  background: var(--warning);
  animation: none;
}

.retry-button {
  grid-column: 1 / -1;
  justify-self: start;
}

.status-grid {
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
  margin: 0;
  background: var(--border-subtle);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.status-cell {
  padding: 0.85rem 1rem;
  background: var(--surface-elevated);
}

.status-label {
  color: var(--text-muted);
  font: 500 0.6875rem/1.2 var(--font-data);
  letter-spacing: 0.08em;
  text-transform: uppercase;
  margin: 0;
}

.status-value {
  margin: 0.35rem 0 0;
  font-family: var(--font-display);
  font-size: 1.625rem;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1.15;
}

@keyframes pulse {
  50% {
    transform: scaleY(0.5);
    opacity: 0.6;
  }
}

@media (prefers-reduced-motion: reduce) {
  .pulse-signal span {
    animation: none;
  }
}

@media (max-width: 580px) {
  .system-pulse {
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  .pulse-signal {
    display: none;
  }

  .status-grid {
    grid-template-columns: 1fr;
  }

  .status-cell {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
  }

  .status-value {
    margin: 0;
    font-size: 1.25rem;
  }
}
</style>
