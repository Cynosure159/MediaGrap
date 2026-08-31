<script setup lang="ts">
import { computed } from 'vue'
import type { SystemSummary } from '@/api/system'

const props = defineProps<{
  summary: SystemSummary | null
}>()

const activities = computed(() => [
  { label: 'Database migration', value: 'Applied', detail: 'Schema is ready for phase 1.', tone: 'complete' },
  { label: 'Media sources', value: `${props.summary?.library.sources ?? 0} configured`, detail: 'Source configuration comes next.', tone: 'waiting' },
  { label: 'Background jobs', value: `${props.summary?.jobs.queued ?? 0} queued`, detail: 'Durable queues arrive in phase 1.', tone: 'waiting' }
])
</script>

<template>
  <section class="activity-rail" aria-labelledby="activity-title">
    <div class="activity-header">
      <div>
        <p class="eyebrow">Build log</p>
        <h2 id="activity-title" class="activity-title">What is ready</h2>
      </div>
      <span class="phase-tag">{{ summary?.phase ?? 'foundation' }}</span>
    </div>

    <ol class="activity-list">
      <li v-for="activity in activities" :key="activity.label" class="activity-item">
        <span class="activity-marker" :class="`activity-marker--${activity.tone}`" aria-hidden="true"></span>
        <div class="activity-content">
          <p class="activity-label">{{ activity.label }}</p>
          <p class="activity-detail">{{ activity.detail }}</p>
        </div>
        <p class="activity-value">{{ activity.value }}</p>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.activity-rail {
  margin-top: 1.5rem;
  padding: clamp(1.2rem, 2vw, 1.8rem);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  background: var(--surface-card);
}

.activity-header {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
}

.eyebrow {
  margin: 0;
  color: var(--text-muted);
  font: 500 0.65rem/1.4 var(--font-data);
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.activity-title {
  margin: 0.25rem 0 0;
  font-family: var(--font-display);
  font-size: clamp(1.25rem, 3vw, 1.5rem);
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: -0.02em;
}

.phase-tag {
  padding: 0.25rem 0.6rem;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  background: var(--surface-elevated);
  color: var(--text-muted);
  font: 500 0.6875rem/1.2 var(--font-data);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.activity-list {
  display: grid;
  gap: 0;
  margin: 1.25rem 0 0;
  padding: 0;
  list-style: none;
}

.activity-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 0.85rem;
  align-items: center;
  padding: 0.75rem 0;
  border-top: 1px solid var(--border-subtle);
}

.activity-marker {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.activity-marker--complete {
  background: var(--success-bright);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.2);
}

.activity-marker--waiting {
  border: 1px solid var(--border-strong);
  background: transparent;
}

.activity-content {
  display: grid;
  gap: 0.15rem;
}

.activity-label {
  margin: 0;
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--text-primary);
}

.activity-detail {
  margin: 0;
  color: var(--text-muted);
  font-size: 0.8125rem;
  line-height: 1.4;
}

.activity-value {
  margin: 0;
  color: var(--text-secondary);
  font-family: var(--font-data);
  font-size: 0.75rem;
  text-align: right;
  white-space: nowrap;
}

@media (max-width: 580px) {
  .activity-item {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .activity-value {
    grid-column: 2;
    text-align: left;
  }
}
</style>
