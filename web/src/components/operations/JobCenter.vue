<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import ActiveJobCard from './ActiveJobCard.vue'
import JobHistoryRow from './JobHistoryRow.vue'
import JobFilterBar from './JobFilterBar.vue'
import type { JobFilter } from './jobFilters'
import type { Job } from '@/api/types'

const props = defineProps<{ jobs: ReadonlyArray<Job>; labels: Record<string, string>; streamState: string }>()
const emit = defineEmits<{ cancel: [id: number]; retry: [id: number] }>()

const activeFilter = shallowRef<JobFilter>('all')

const runningJobs = computed(() => props.jobs.filter(job => job.state === 'running'))
const queuedJobs = computed(() => props.jobs.filter(job => job.state === 'queued'))
const succeededJobs = computed(() => props.jobs.filter(job => job.state === 'succeeded'))
const failedJobs = computed(() => props.jobs.filter(job => ['failed', 'cancelled', 'interrupted'].includes(job.state)))

const activeJobs = computed(() => props.jobs.filter(job => job.state === 'queued' || job.state === 'running'))
const historyJobs = computed(() => props.jobs.filter(job => job.state !== 'queued' && job.state !== 'running'))

const filterCounts = computed(() => ({
  all: props.jobs.length,
  running: runningJobs.value.length,
  queued: queuedJobs.value.length,
  succeeded: succeededJobs.value.length,
  failed: failedJobs.value.length,
}))

const filteredHistoryJobs = computed(() => {
  if (activeFilter.value === 'all') return historyJobs.value
  if (activeFilter.value === 'succeeded') return succeededJobs.value
  if (activeFilter.value === 'failed') return failedJobs.value
  return []
})

const showActiveSection = computed(() => {
  if (activeFilter.value === 'all' || activeFilter.value === 'running' || activeFilter.value === 'queued') {
    return activeJobs.value.length > 0
  }
  return false
})

const displayActiveJobs = computed(() => {
  if (activeFilter.value === 'running') return runningJobs.value
  if (activeFilter.value === 'queued') return queuedJobs.value
  return activeJobs.value
})
</script>

<template>
  <section class="ops-card job-center">
    <!-- Header with Eyebrow, Title & Filter Tabs -->
    <header class="card-header">
      <div class="header-main">
        <div class="title-group">
          <p class="eyebrow">{{ labels.jobQueue || 'DURABLE WORK' }}</p>
          <h2>{{ labels.jobCenter || 'Job Center' }}</h2>
        </div>
        <span class="total-badge font-code">{{ jobs.length }}</span>
      </div>

      <!-- State Filter Tabs -->
      <JobFilterBar v-model="activeFilter" :labels="labels" :counts="filterCounts" />
    </header>

    <!-- Active Jobs Section -->
    <div v-if="showActiveSection" class="active-jobs-section">
      <div class="section-subhead">
        <span class="subhead-title">{{ labels.jobQueue || 'ACTIVE QUEUE' }}</span>
        <span class="live-tag">
          <span class="pulse-ring"></span>
          <span>LIVE</span>
        </span>
      </div>

      <div class="active-job-grid">
        <ActiveJobCard v-for="job in displayActiveJobs" :key="job.id" :job="job" :labels="labels" @cancel="emit('cancel', $event)" />
      </div>
    </div>

    <!-- Job History Section -->
    <div v-if="activeFilter !== 'running' && activeFilter !== 'queued'" class="history-section">
      <div class="section-subhead">
        <span class="subhead-title">{{ labels.jobHistory || 'JOB HISTORY' }}</span>
      </div>

      <div v-if="filteredHistoryJobs.length" class="history-list">
        <JobHistoryRow v-for="job in filteredHistoryJobs" :key="job.id" :job="job" :labels="labels" @retry="emit('retry', $event)" />
      </div>

      <div v-else class="empty-state">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="28" height="28" class="empty-icon">
          <circle cx="12" cy="12" r="9" />
          <path d="M12 8v4M12 16h.01" />
        </svg>
        <p>{{ labels.noJobHistory || 'No completed jobs recorded yet.' }}</p>
      </div>
    </div>
  </section>
</template>

<style scoped>
.job-center {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
}

.card-header {
  padding: 16px 20px;
  background: var(--surface-container-low, #151b2d);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.header-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.title-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.eyebrow {
  margin: 0;
  color: var(--primary, #c0c1ff);
  font: 700 0.65rem/1 var(--font-data);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

h2 {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.total-badge {
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--surface-container-highest, #2e3447);
  color: var(--on-surface, #dce1fb);
  font-size: 11px;
}

.section-subhead {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px 8px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  background: rgba(12, 19, 36, 0.2);
}

.subhead-title {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--outline, #908fa0);
}

.live-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 9px;
  font-weight: 700;
  color: var(--secondary, #4edea3);
  letter-spacing: 0.05em;
}

.pulse-ring {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--secondary, #4edea3);
  box-shadow: 0 0 0 0 rgba(78, 222, 163, 0.6);
  animation: pulse-green 1.8s infinite;
}

.active-jobs-section {
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.active-job-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 20px;
}

.history-list {
  display: flex;
  flex-direction: column;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px 20px;
  color: var(--outline, #908fa0);
  font-size: 12px;
}

.empty-icon {
  opacity: 0.4;
}

@media (max-width: 600px) {
  .card-header {
    padding: 12px 14px;
  }
  .active-job-grid {
    padding: 12px 14px;
  }
}
</style>
