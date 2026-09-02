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
  try { await action() } catch (caught) { actionError.value = caught instanceof Error ? caught.message : props.labels.operationFailed }
}

onMounted(async () => {
  await operations.refreshAll()
  operations.connect()
})
onUnmounted(operations.disconnect)
</script>

<template>
  <main class="operations-page">
    <div class="operations-container">
      <header class="page-header">
        <div><p class="eyebrow">{{ labels.operationsEyebrow }}</p><h1>{{ labels.operations }}</h1><p>{{ labels.operationsIntro }}</p></div>
        <button class="btn btn-ghost btn-sm" type="button" :disabled="operations.isLoading.value" @click="operations.refreshAll">{{ labels.refresh }}</button>
      </header>
      <div v-if="operations.error.value || actionError" class="error-banner" role="alert">{{ actionError || operations.error.value }}</div>
      <div v-if="operations.isLoading.value && !operations.status.value" class="loading-state">{{ labels.loading }}</div>
      <div v-else class="operations-grid">
        <JobCenter
          :jobs="operations.jobs.value"
          :labels="labels"
          :stream-state="operations.streamState.value"
          @cancel="runAction(() => operations.cancel($event))"
          @retry="runAction(() => operations.retry($event))"
        />
        <div class="side-stack">
          <SystemStatusPanel :status="operations.status.value" :labels="labels" />
          <AuditCenter :entries="operations.audits.value" :labels="labels" />
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.operations-page { width: 100%; height: 100%; overflow-y: auto; background: var(--surface-base); }
.operations-container { width: 100%; max-width: 1440px; margin: 0 auto; padding: clamp(1rem, 2vw, 2.5rem); box-sizing: border-box; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; padding-bottom: 1.25rem; border-bottom: 1px solid var(--border-subtle); }
.page-header .eyebrow { margin: 0 0 .25rem; color: var(--primary); font: 700 .68rem/1 var(--font-data); letter-spacing: .08em; text-transform: uppercase; }
.page-header h1 { margin: 0; color: var(--on-surface); font-size: 1.4rem; } .page-header p:not(.eyebrow) { margin: .3rem 0 0; color: var(--on-surface-variant); font-size: .8rem; }
.operations-grid { display: grid; grid-template-columns: minmax(0, 1.45fr) minmax(320px, .8fr); gap: 1.25rem; margin-top: 1.25rem; align-items: start; }
.side-stack { display: grid; gap: 1.25rem; }
.error-banner, .loading-state { margin-top: 1rem; border: 1px solid var(--error-container); border-radius: var(--radius-sm); padding: .8rem 1rem; background: var(--surface-container-low); color: var(--error); font-size: .8rem; }
.loading-state { border-color: var(--border-subtle); color: var(--outline); }
@media (max-width: 980px) { .operations-grid { grid-template-columns: 1fr; } }
@media (max-width: 768px) { .operations-container { padding-bottom: calc(84px + env(safe-area-inset-bottom)); } }
</style>
