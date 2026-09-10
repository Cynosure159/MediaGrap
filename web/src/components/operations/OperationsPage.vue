<script setup lang="ts">
import { onMounted, onUnmounted, shallowRef } from 'vue'
import AuditCenter from './AuditCenter.vue'
import JobCenter from './JobCenter.vue'
import SystemStatusPanel from './SystemStatusPanel.vue'
import { useOperations } from '@/composables/useOperations'

const props = defineProps<{ csrfToken: string; labels: Record<string, string> }>()
const actionError = shallowRef<string | null>(null)
const operations = useOperations(() => props.csrfToken)

async function runAction(action: () => Promise<void>) {
  actionError.value = null
  try {
    await action()
  } catch (caught) {
    actionError.value = caught instanceof Error ? caught.message : props.labels.operationFailed
  }
}

onMounted(async () => {
  await operations.refreshAll()
  operations.connect()
})
onUnmounted(operations.disconnect)
</script>

<template>
  <main class="operations-page">
    <!-- Top 40px Breadcrumb Toolbar matching Settings Console -->
    <header class="console-top-toolbar">
      <div class="toolbar-breadcrumb">
        <span class="crumb-root">{{ labels.operations || 'Operations' }}</span>
        <span class="crumb-sep">/</span>
        <span class="crumb-current">{{ labels.jobCenter || 'Job Center' }} & {{ labels.auditCenter || 'Audits' }}</span>
      </div>

      <div class="toolbar-actions">
        <!-- Live SSE Status Pill -->
        <div class="stream-pill" :class="`stream-pill--${operations.streamState.value}`">
          <span class="stream-dot" :class="{ 'stream-dot--pulse': operations.streamState.value === 'connected' }"></span>
          <span class="stream-text">{{ labels[`stream_${operations.streamState.value}`] || operations.streamState.value }}</span>
        </div>

        <button
          type="button"
          class="btn btn-ghost btn-sm"
          :disabled="operations.isLoading.value"
          :title="labels.refresh || 'Refresh'"
          @click="operations.refreshAll"
        >
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14" :class="{ 'spin-slow': operations.isLoading.value }">
            <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
          </svg>
          <span>{{ labels.refresh || 'Refresh' }}</span>
        </button>
      </div>
    </header>

    <!-- Scrollable Canvas Area -->
    <div class="operations-body-scroll">
      <div class="operations-container">
        <div v-if="operations.error.value || actionError" class="error-banner" role="alert">
          <span class="dot dot-err"></span>
          <span>{{ actionError || operations.error.value }}</span>
        </div>

        <div v-if="operations.isLoading.value && !operations.status.value" class="loading-state">
          <div class="loading-spinner"></div>
          <span>{{ labels.loading }}</span>
        </div>

        <div v-else class="operations-grid">
          <!-- Left: Job Center Pro with State Filter Tabs & Rich Cards -->
          <div class="grid-main-col">
            <JobCenter
              :jobs="operations.jobs.value"
              :labels="labels"
              :stream-state="operations.streamState.value"
              @cancel="runAction(() => operations.cancel($event))"
              @retry="runAction(() => operations.retry($event))"
            />
          </div>

          <!-- Right: Audit Center & System Health Monitor -->
          <div class="grid-side-col">
            <SystemStatusPanel
              :status="operations.status.value"
              :labels="labels"
              :testing-target="operations.testingTarget.value"
              @test="runAction(() => operations.testConnection($event))"
            />
            <AuditCenter :entries="operations.audits.value" :labels="labels" />
          </div>
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.operations-page {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--surface-base, #0c1324);
  overflow: hidden;
  box-sizing: border-box;
}

.console-top-toolbar {
  height: var(--toolbar-height, 40px);
  min-height: var(--toolbar-height, 40px);
  padding: 0 20px;
  background: var(--surface-container, #191f31);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  z-index: 10;
}

.toolbar-breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.crumb-root {
  color: var(--outline, #908fa0);
}

.crumb-sep {
  color: var(--outline-variant, #2e3447);
}

.crumb-current {
  color: var(--on-surface, #dce1fb);
  font-weight: 600;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.stream-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  font-size: 11px;
  font-weight: 500;
  color: var(--on-surface-variant, #c7c4d7);
}

.stream-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--outline, #908fa0);
}

.stream-pill--connected .stream-dot {
  background: var(--secondary, #4edea3);
}

.stream-pill--connecting .stream-dot,
.stream-pill--reconnecting .stream-dot {
  background: var(--tertiary, #ffb95f);
}

.stream-dot--pulse {
  box-shadow: 0 0 0 0 rgba(78, 222, 163, 0.7);
  animation: pulse-green 2s infinite;
}

@keyframes pulse-green {
  0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(78, 222, 163, 0.7); }
  70% { transform: scale(1); box-shadow: 0 0 0 5px rgba(78, 222, 163, 0); }
  100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(78, 222, 163, 0); }
}

.spin-slow {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.operations-body-scroll {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  -webkit-overflow-scrolling: touch;
  padding: 20px;
}

.operations-container {
  width: 100%;
  max-width: 1440px;
  margin: 0 auto;
}

.operations-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(340px, 0.9fr);
  gap: 20px;
  align-items: start;
}

.grid-main-col {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.grid-side-col {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.error-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding: 10px 14px;
  border-radius: var(--radius-sm, 0.25rem);
  background: rgba(244, 63, 94, 0.1);
  border: 1px solid rgba(244, 63, 94, 0.3);
  color: var(--error, #ffb4ab);
  font-size: 13px;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 60px 0;
  color: var(--outline, #908fa0);
  font-size: 13px;
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--primary, #c0c1ff);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@media (max-width: 1024px) {
  .operations-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .operations-body-scroll {
    padding: 14px 14px calc(80px + env(safe-area-inset-bottom, 0px));
  }
}
</style>
