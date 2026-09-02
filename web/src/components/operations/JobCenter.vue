<script setup lang="ts">
import { computed } from 'vue'
import type { Job } from '@/api/types'

const props = defineProps<{ jobs: ReadonlyArray<Job>; labels: Record<string, string>; streamState: string }>()
const emit = defineEmits<{ cancel: [id: number]; retry: [id: number] }>()
const activeJobs = computed(() => props.jobs.filter(job => job.state === 'queued' || job.state === 'running'))
const historyJobs = computed(() => props.jobs.filter(job => job.state !== 'queued' && job.state !== 'running'))

function percent(job: Job): number {
  if (!job.progressTotal) return job.state === 'succeeded' ? 100 : 0
  return Math.min(100, Math.round((job.progressCurrent / job.progressTotal) * 100))
}

function jobLabel(kind = ''): string {
  return props.labels[`jobKind_${kind}`] || kind.replaceAll('_', ' ')
}
</script>

<template>
  <section class="ops-card job-center">
    <header class="section-header">
      <div>
        <p class="eyebrow">{{ labels.jobQueue }}</p>
        <h2>{{ labels.jobCenter }}</h2>
      </div>
      <span class="stream-badge" :class="`stream-badge--${streamState}`">
        <span class="status-dot" :class="streamState === 'connected' ? 'status-dot--success' : 'status-dot--warning'"></span>
        {{ labels[`stream_${streamState}`] }}
      </span>
    </header>

    <div v-if="activeJobs.length" class="job-list">
      <article v-for="job in activeJobs" :key="job.id" class="job-row job-row--active">
        <div class="job-row__top">
          <div>
            <strong>{{ jobLabel(job.kind) }}</strong>
            <span class="job-id font-code">#{{ job.id }}</span>
          </div>
          <button class="btn btn-ghost btn-sm" type="button" @click="emit('cancel', job.id)">{{ labels.cancelJob }}</button>
        </div>
        <p>{{ job.message || labels.jobWaiting }}</p>
        <div class="progress-track"><span :style="{ width: `${percent(job)}%` }"></span></div>
        <div class="job-meta font-code"><span>{{ labels[`jobState_${job.state}`] || job.state }}</span><span>{{ job.progressCurrent }} / {{ job.progressTotal || '—' }}</span></div>
      </article>
    </div>
    <p v-else class="empty-state">{{ labels.noActiveJobs }}</p>

    <h3 class="history-title">{{ labels.jobHistory }}</h3>
    <div class="history-list">
      <article v-for="job in historyJobs" :key="job.id" class="history-row">
        <span class="state-mark" :class="`state-mark--${job.state}`"></span>
        <div class="history-main">
          <strong>{{ jobLabel(job.kind) }} <span class="font-code">#{{ job.id }}</span></strong>
          <p>{{ job.errorMessage || job.message }}</p>
        </div>
        <span class="history-state font-code">{{ labels[`jobState_${job.state}`] || job.state }}</span>
        <button v-if="['failed', 'cancelled', 'interrupted'].includes(job.state)" class="btn btn-ghost btn-sm" type="button" @click="emit('retry', job.id)">{{ labels.retryJob }}</button>
      </article>
      <p v-if="!historyJobs.length" class="empty-state">{{ labels.noJobHistory }}</p>
    </div>
  </section>
</template>

<style scoped>
.ops-card { border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); background: var(--surface-container); overflow: hidden; }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: 1rem 1.1rem; border-bottom: 1px solid var(--border-subtle); }
.eyebrow { margin: 0 0 .2rem; color: var(--primary); font: 700 .65rem/1 var(--font-data); letter-spacing: .08em; text-transform: uppercase; }
h2, h3, p { margin: 0; } h2 { font-size: 1rem; color: var(--on-surface); }
.stream-badge { display: inline-flex; align-items: center; gap: .4rem; border-radius: 999px; padding: .3rem .55rem; background: var(--surface-container-high); color: var(--on-surface-variant); font: 600 .67rem/1 var(--font-data); }
.job-list, .history-list { display: grid; }
.job-row { padding: .9rem 1.1rem; border-bottom: 1px solid var(--border-subtle); }
.job-row--active { border-left: 2px solid var(--primary); }
.job-row__top { display: flex; align-items: center; justify-content: space-between; gap: .75rem; }
.job-row strong, .history-row strong { color: var(--on-surface); font-size: .8rem; text-transform: capitalize; }
.job-id { margin-left: .4rem; color: var(--outline); font-size: .68rem; }
.job-row p, .history-row p { margin-top: .3rem; color: var(--on-surface-variant); font-size: .72rem; }
.progress-track { height: 4px; margin-top: .7rem; overflow: hidden; border-radius: 999px; background: var(--surface-container-highest); }
.progress-track span { display: block; height: 100%; background: var(--primary); transition: width .2s ease; }
.job-meta { display: flex; justify-content: space-between; margin-top: .35rem; color: var(--outline); font-size: .65rem; }
.history-title { padding: .85rem 1.1rem .55rem; color: var(--on-surface-variant); font-size: .72rem; letter-spacing: .05em; text-transform: uppercase; }
.history-row { display: grid; grid-template-columns: 8px minmax(0, 1fr) auto auto; align-items: center; gap: .75rem; padding: .7rem 1.1rem; border-top: 1px solid var(--border-subtle); }
.history-main { min-width: 0; } .history-main p { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.history-state { color: var(--outline); font-size: .65rem; }
.state-mark { width: 7px; height: 7px; border-radius: 50%; background: var(--outline); }
.state-mark--succeeded { background: var(--secondary); } .state-mark--failed { background: var(--error); } .state-mark--cancelled, .state-mark--interrupted { background: var(--tertiary); }
.empty-state { padding: 1rem 1.1rem; color: var(--outline); font-size: .75rem; }
@media (max-width: 700px) { .history-row { grid-template-columns: 8px minmax(0, 1fr) auto; } .history-state { display: none; } }
</style>
