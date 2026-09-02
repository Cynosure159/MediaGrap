<script setup lang="ts">
import type { ConnectionTest, OperationsStatus } from '@/api/types'

defineProps<{ status: OperationsStatus | null; labels: Record<string, string>; testingTarget: string | null }>()
const emit = defineEmits<{ test: [target: ConnectionTest['target']] }>()

function bytes(value: number): string {
  if (value < 1024) return `${value} B`
  if (value < 1024 ** 2) return `${(value / 1024).toFixed(1)} KiB`
  return `${(value / 1024 ** 2).toFixed(1)} MiB`
}
</script>

<template>
  <section class="ops-card status-panel">
    <header class="section-header"><div><p class="eyebrow">{{ labels.runtime }}</p><h2>{{ labels.systemStatus }}</h2></div></header>
    <div v-if="status" class="status-grid">
      <article class="status-block">
        <span>{{ labels.applicationVersion }}</span>
        <strong class="font-code">{{ status.application.version || 'dev' }}</strong>
        <small class="font-code">{{ status.application.commit || '—' }}</small>
      </article>
      <article class="status-block">
        <span>{{ labels.database }}</span>
        <strong :class="status.database.ready ? 'ok' : 'bad'">{{ status.database.ready ? labels.ready : labels.unavailable }}</strong>
		<small class="font-code">{{ status.database.latestMigration }} · {{ status.database.journalMode.toUpperCase() }} · {{ bytes(status.database.sizeBytes) }}</small>
      </article>
      <article class="status-block">
        <span>{{ labels.cache }}</span>
        <strong :class="status.cache.writable ? 'ok' : 'warn'">{{ status.cache.writable ? labels.writable : labels.readOnly }}</strong>
        <small class="font-code">{{ bytes(status.cache.usedBytes) }}</small>
      </article>
      <article class="status-block">
        <span>{{ labels.outboundProxy }}</span>
        <strong>{{ status.network.proxyConfigured ? labels.configured : labels.notConfigured }}</strong>
		<small>{{ status.network.noProxyConfigured ? 'NO_PROXY' : '—' }}</small>
      </article>
    </div>
    <div v-if="status" class="status-section">
      <h3>{{ labels.mediaMounts }}</h3>
      <div v-for="mount in status.mounts" :key="mount.id" class="status-line">
        <span><i class="status-dot" :class="mount.available && mount.writable ? 'status-dot--success' : 'status-dot--warning'"></i>{{ mount.name }}</span>
        <code>{{ mount.itemCount }} · {{ !mount.available ? labels.unavailable : mount.writable ? labels.writable : labels.readOnly }}</code>
      </div>
      <p v-if="!status.mounts.length" class="empty-state">{{ labels.noMediaMounts }}</p>
    </div>
    <div v-if="status" class="status-section">
      <h3>{{ labels.providers }}</h3>
      <div v-for="provider in status.providers" :key="provider.id" class="status-line">
        <span><i class="status-dot" :class="provider.configured ? 'status-dot--success' : 'status-dot--warning'"></i>{{ provider.id }}</span>
        <code>{{ labels[`providerStatus_${provider.status}`] || provider.status }}</code>
		<button type="button" class="btn btn-outline btn-sm" :disabled="testingTarget !== null || !provider.configured" @click="emit('test', provider.id as ConnectionTest['target'])">{{ testingTarget === provider.id ? labels.testingConnection : labels.testConnection }}</button>
      </div>
	  <div v-if="status.network.proxyConfigured" class="status-line"><span>{{ labels.outboundProxy }}</span><button type="button" class="btn btn-outline btn-sm" :disabled="testingTarget !== null" @click="emit('test', 'proxy')">{{ testingTarget === 'proxy' ? labels.testingConnection : labels.testProxy }}</button></div>
	  <div v-for="(test, target) in status.connectionTests" :key="target" class="test-line"><code>{{ target }}</code><span :class="`test-${test.status}`">{{ labels[`connection_${test.status}`] }} · {{ test.durationMs }} ms · {{ test.testedAt || '—' }}</span></div>
    </div>
  </section>
</template>

<style scoped>
.ops-card { border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); background: var(--surface-container); overflow: hidden; }
.section-header { padding: 1rem 1.1rem; border-bottom: 1px solid var(--border-subtle); }
.eyebrow { margin: 0 0 .2rem; color: var(--primary); font: 700 .65rem/1 var(--font-data); letter-spacing: .08em; text-transform: uppercase; }
h2, h3, p { margin: 0; } h2 { color: var(--on-surface); font-size: 1rem; }
.status-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1px; background: var(--border-subtle); }
.status-block { display: grid; gap: .28rem; padding: .85rem 1rem; background: var(--surface-container); }
.status-block span, .status-block small { color: var(--outline); font-size: .66rem; }
.status-block strong { color: var(--on-surface); font-size: .78rem; } .status-block .ok { color: var(--secondary); } .status-block .warn { color: var(--tertiary); } .status-block .bad { color: var(--error); }
.status-section { padding: .85rem 1rem; border-top: 1px solid var(--border-subtle); }
.status-section h3 { margin-bottom: .5rem; color: var(--on-surface-variant); font-size: .68rem; letter-spacing: .06em; text-transform: uppercase; }
.status-line { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: .35rem 0; color: var(--on-surface); font-size: .72rem; }
.status-line span { display: inline-flex; align-items: center; gap: .45rem; } .status-line code { color: var(--outline); font-size: .64rem; }
.empty-state { color: var(--outline); font-size: .72rem; }
.test-line { display: grid; grid-template-columns: 6rem 1fr; gap: .5rem; padding: .3rem 0; color: var(--outline); font-size: .64rem; }
.test-reachable { color: var(--secondary); } .test-failed { color: var(--error); } .test-not_configured { color: var(--tertiary); }
@media (max-width: 480px) { .status-grid { grid-template-columns: 1fr; } }
</style>
