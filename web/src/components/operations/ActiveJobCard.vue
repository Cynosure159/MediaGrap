<script setup lang="ts">
import type { Job } from '@/api/types'
const props = defineProps<{ job: Job; labels: Record<string, string> }>()
const emit = defineEmits<{ cancel: [id: number] }>()
function percent(job: Job): number {
  if (!job.progressTotal) return job.state === 'succeeded' ? 100 : 0
  return Math.min(100, Math.round((job.progressCurrent / job.progressTotal) * 100))
}

function jobLabel(kind = ''): string {
  return props.labels[`jobKind_${kind}`] || kind.replaceAll('_', ' ')
}
</script>

<template>
  <article
    class="active-job-card"
    :class="`active-job-card--${job.state}`"
  >
    <div class="job-card-top">
      <div class="job-identity">
        <div class="job-kind-badge font-code">{{ jobLabel(job.kind) }}</div>
        <span class="job-id-tag font-code">#{{ job.id }}</span>
      </div>
      <div class="job-top-actions">
        <span class="state-chip" :class="`state-chip--${job.state}`">
          <span class="spinner-sm" v-if="job.state === 'running'"></span>
          {{ labels[`jobState_${job.state}`] || job.state }}
        </span>
        <button
          class="btn btn-ghost btn-xs btn-cancel"
          type="button"
          :title="labels.cancelJob || 'Cancel'"
          @click="emit('cancel', job.id)"
        >
          {{ labels.cancelJob || 'Cancel' }}
        </button>
      </div>
    </div>

    <p class="job-message">{{ job.message || labels.jobWaiting }}</p>

    <!-- Progress Bar -->
    <div class="progress-wrap">
      <div class="progress-track">
        <span :style="{ width: `${percent(job)}%` }"></span>
      </div>
      <div class="progress-meta font-code">
        <span class="percent-val">{{ percent(job) }}%</span>
        <span class="count-val">{{ job.progressCurrent }} / {{ job.progressTotal || '—' }}</span>
      </div>
    </div>
  </article>
</template>

<style scoped>
.active-job-card {
  padding: 14px 16px;
  border-radius: var(--radius-md, 0.375rem);
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  border-left: 3px solid var(--primary, #c0c1ff);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.active-job-card--running {
  border-left-color: var(--tertiary, #ffb95f);
  background: linear-gradient(90deg, rgba(255, 185, 95, 0.04) 0%, var(--surface-container-high, #23293c) 100%);
}

.job-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.job-identity {
  display: flex;
  align-items: center;
  gap: 8px;
}

.job-kind-badge {
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.job-id-tag {
  font-size: 11px;
  color: var(--outline, #908fa0);
}

.job-top-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.state-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 7px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
}

.state-chip--running {
  background: rgba(255, 185, 95, 0.15);
  color: var(--tertiary, #ffb95f);
}

.state-chip--queued {
  background: rgba(192, 193, 255, 0.15);
  color: var(--primary, #c0c1ff);
}

.spinner-sm {
  width: 9px;
  height: 9px;
  border: 1.5px solid var(--tertiary, #ffb95f);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.btn-cancel {
  color: var(--error, #ffb4ab);
  font-size: 11px;
}

.btn-cancel:hover {
  background: rgba(244, 63, 94, 0.15);
}

.job-message {
  margin: 0;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  line-height: 1.4;
}

.progress-wrap {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.progress-track {
  height: 6px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--surface-container-lowest, #070d1f);
}

.progress-track span {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, var(--primary, #8083ff) 0%, var(--tertiary, #ffb95f) 100%);
  border-radius: 999px;
  transition: width 0.3s ease;
}

.progress-meta {
  display: flex;
  justify-content: space-between;
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.percent-val {
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

</style>
