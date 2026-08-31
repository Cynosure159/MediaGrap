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

    <button v-if="error" class="retry-button" type="button" :disabled="isLoading" @click="emit('refresh')">
      Try again
    </button>

    <dl class="status-grid">
      <div class="status-cell">
        <dt>Sources</dt>
        <dd>{{ summary?.library.sources ?? '—' }}</dd>
      </div>
      <div class="status-cell">
        <dt>Indexed items</dt>
        <dd>{{ summary?.library.items ?? '—' }}</dd>
      </div>
      <div class="status-cell">
        <dt>Active jobs</dt>
        <dd>{{ summary?.jobs.running ?? '—' }}</dd>
      </div>
    </dl>
  </section>
</template>

<style scoped>
.system-pulse { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 1.8rem; padding: clamp(1.4rem, 3vw, 2.5rem); border: 1px solid var(--water-800); border-radius: 1rem; background: linear-gradient(130deg, var(--ink-900), #12344a); color: var(--mist-100); overflow: hidden; }
.pulse-copy { max-width: 38rem; }
.pulse-title { margin: 0.3rem 0 0.75rem; font-family: var(--font-display); font-size: clamp(2.25rem, 6vw, 4.5rem); font-weight: 500; letter-spacing: -0.07em; line-height: 0.95; }
.pulse-description { max-width: 34rem; margin: 0; color: var(--mist-300); line-height: 1.6; }
.pulse-description--error { color: #ffd0b8; }
.pulse-signal { display: flex; align-items: end; gap: 0.3rem; height: 4rem; padding-top: 0.5rem; }
.pulse-signal span { width: 0.45rem; background: var(--water-400); border-radius: 99px; animation: pulse 1.6s ease-in-out infinite; }
.pulse-signal span:nth-child(1) { height: 38%; animation-delay: -0.4s; }.pulse-signal span:nth-child(2) { height: 100%; animation-delay: -0.2s; }.pulse-signal span:nth-child(3) { height: 64%; }
.pulse-signal--error span { background: var(--amber-400); animation: none; }
.retry-button { align-self: start; padding: 0.65rem 0.8rem; border: 1px solid var(--mist-400); border-radius: 0.45rem; background: transparent; color: var(--mist-100); cursor: pointer; }
.status-grid { grid-column: 1 / -1; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 1px; margin: 0; background: var(--water-800); border: 1px solid var(--water-800); border-radius: 0.6rem; overflow: hidden; }
.status-cell { padding: 0.9rem 1rem; background: rgb(8 29 44 / 55%); }.status-cell dt { color: var(--mist-400); font: 0.7rem var(--font-data); letter-spacing: 0.08em; text-transform: uppercase; }.status-cell dd { margin: 0.3rem 0 0; font-family: var(--font-display); font-size: 1.7rem; }
@keyframes pulse { 50% { transform: scaleY(0.55); opacity: 0.5; } }
@media (prefers-reduced-motion: reduce) { .pulse-signal span { animation: none; } }
@media (max-width: 580px) { .system-pulse { grid-template-columns: 1fr; gap: 1rem; }.pulse-signal { display: none; }.status-grid { grid-template-columns: 1fr; }.status-cell { display: flex; justify-content: space-between; align-items: baseline; }.status-cell dd { margin: 0; } }
</style>

