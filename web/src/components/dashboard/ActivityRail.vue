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
        <div>
          <p class="activity-label">{{ activity.label }}</p>
          <p class="activity-detail">{{ activity.detail }}</p>
        </div>
        <p class="activity-value">{{ activity.value }}</p>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.activity-rail { margin-top: 1.5rem; padding: clamp(1.2rem, 2vw, 1.8rem); border: 1px solid var(--mist-300); border-radius: 1rem; background: var(--paper); }
.activity-header { display: flex; justify-content: space-between; gap: 1rem; align-items: start; }.activity-title { margin: 0.25rem 0 0; font-family: var(--font-display); font-size: 1.7rem; letter-spacing: -0.04em; }.phase-tag { padding: 0.35rem 0.55rem; border: 1px solid var(--mist-300); border-radius: 999px; color: var(--ink-600); font: 0.7rem var(--font-data); letter-spacing: 0.08em; text-transform: uppercase; }
.activity-list { display: grid; gap: 0.2rem; margin: 1.5rem 0 0; padding: 0; list-style: none; }.activity-item { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; gap: 0.8rem; align-items: center; padding: 0.85rem 0; border-top: 1px solid var(--mist-200); }.activity-marker { width: 0.55rem; height: 0.55rem; border-radius: 50%; }.activity-marker--complete { background: var(--water-600); box-shadow: 0 0 0 0.25rem var(--water-100); }.activity-marker--waiting { border: 1px solid var(--mist-400); }.activity-label, .activity-detail, .activity-value { margin: 0; }.activity-label { font-weight: 650; }.activity-detail { margin-top: 0.18rem; color: var(--ink-500); font-size: 0.88rem; }.activity-value { color: var(--ink-600); font: 0.75rem var(--font-data); text-align: right; }
@media (max-width: 580px) { .activity-item { grid-template-columns: auto minmax(0, 1fr); }.activity-value { grid-column: 2; text-align: left; }.activity-detail { max-width: 24rem; } }
</style>

