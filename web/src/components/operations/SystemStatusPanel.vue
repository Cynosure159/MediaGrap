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
    <header class="card-header">
      <div class="title-group">
        <p class="eyebrow">{{ labels.runtime || 'CONTAINER RUNTIME' }}</p>
        <h2>{{ labels.systemStatus || 'System Health' }}</h2>
      </div>
    </header>

    <div v-if="status" class="status-grid">
      <!-- 1. Version -->
      <article class="status-cell">
        <span class="cell-label">{{ labels.applicationVersion || 'App Version' }}</span>
        <strong class="cell-val font-code">{{ status.application.version || 'dev' }}</strong>
        <small class="cell-sub font-code">{{ status.application.commit || '—' }}</small>
      </article>

      <!-- 2. Database -->
      <article class="status-cell">
        <span class="cell-label">{{ labels.database || 'Database' }}</span>
        <strong class="cell-val" :class="status.database.ready ? 'text-ok' : 'text-err'">
          {{ status.database.ready ? (labels.ready || 'Ready') : (labels.unavailable || 'Unavailable') }}
        </strong>
        <small class="cell-sub font-code">{{ status.database.journalMode.toUpperCase() }} · {{ bytes(status.database.sizeBytes) }}</small>
      </article>

      <!-- 3. Cache -->
      <article class="status-cell">
        <span class="cell-label">{{ labels.cache || 'Cache' }}</span>
        <strong class="cell-val" :class="status.cache.writable ? 'text-ok' : 'text-warn'">
          {{ status.cache.writable ? (labels.writable || 'Writable') : (labels.readOnly || 'Read-only') }}
        </strong>
        <small class="cell-sub font-code">{{ bytes(status.cache.usedBytes) }}</small>
      </article>

      <!-- 4. Proxy -->
      <article class="status-cell">
        <span class="cell-label">{{ labels.outboundProxy || 'Outbound Proxy' }}</span>
        <strong class="cell-val">
          {{ status.network.proxyConfigured ? (labels.configured || 'Configured') : (labels.notConfigured || 'Direct') }}
        </strong>
        <small class="cell-sub">{{ status.network.noProxyConfigured ? 'NO_PROXY active' : '—' }}</small>
      </article>
    </div>

    <!-- Mounts -->
    <div v-if="status" class="status-section">
      <div class="section-title-row">
        <h3>{{ labels.mediaMounts || 'MEDIA MOUNTS' }}</h3>
        <span class="count-tag font-code">{{ status.mounts.length }}</span>
      </div>
      <div class="mount-list">
        <div v-for="mount in status.mounts" :key="mount.id" class="mount-row">
          <div class="mount-name">
            <span class="dot" :class="mount.available && mount.writable ? 'dot-ok' : 'dot-warn'"></span>
            <span>{{ mount.name }}</span>
          </div>
          <span class="spec-pill font-code">
            {{ mount.itemCount }} · {{ !mount.available ? labels.unavailable : mount.writable ? labels.writable : labels.readOnly }}
          </span>
        </div>
        <p v-if="!status.mounts.length" class="empty-state">{{ labels.noMediaMounts }}</p>
      </div>
    </div>

    <!-- Providers & Diagnostics -->
    <div v-if="status" class="status-section">
      <div class="section-title-row">
        <h3>{{ labels.providers || 'PROVIDERS' }}</h3>
      </div>
      <div class="provider-list">
        <div v-for="provider in status.providers" :key="provider.id" class="provider-row">
          <div class="provider-info">
            <span class="dot" :class="provider.configured ? 'dot-ok' : 'dot-warn'"></span>
            <span class="provider-name">{{ provider.id }}</span>
          </div>
          <button
            type="button"
            class="btn btn-ghost btn-xs"
            :disabled="testingTarget !== null || !provider.configured"
            @click="emit('test', provider.id as ConnectionTest['target'])"
          >
            {{ testingTarget === provider.id ? (labels.testingConnection || 'Testing…') : (labels.testConnection || 'Ping') }}
          </button>
        </div>

        <div v-if="status.network.proxyConfigured" class="provider-row">
          <div class="provider-info">
            <span class="dot dot-ok"></span>
            <span class="provider-name">{{ labels.outboundProxy }}</span>
          </div>
          <button
            type="button"
            class="btn btn-ghost btn-xs"
            :disabled="testingTarget !== null"
            @click="emit('test', 'proxy')"
          >
            {{ testingTarget === 'proxy' ? labels.testingConnection : labels.testProxy }}
          </button>
        </div>

        <!-- Connection Test Results -->
        <div v-for="(test, target) in status.connectionTests" :key="target" class="test-result-row font-code">
          <span class="test-target">{{ target }}:</span>
          <span class="test-outcome" :class="`test-outcome--${test.status}`">
            {{ labels[`connection_${test.status}`] || test.status }} ({{ test.durationMs }}ms)
          </span>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.status-panel {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
  display: flex;
  flex-direction: column;
}

.card-header {
  padding: 16px 20px;
  background: var(--surface-container-low, #151b2d);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
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

.status-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1px;
  background: var(--outline-variant, #2e3447);
}

.status-cell {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 12px 16px;
  background: var(--surface-container, #191f31);
}

.cell-label {
  color: var(--outline, #908fa0);
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.cell-val {
  color: var(--on-surface, #dce1fb);
  font-size: 13px;
  font-weight: 600;
}

.cell-sub {
  color: var(--outline, #908fa0);
  font-size: 10px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.text-ok {
  color: var(--secondary, #4edea3);
}

.text-warn {
  color: var(--tertiary, #ffb95f);
}

.text-err {
  color: var(--error, #ffb4ab);
}

.status-section {
  padding: 14px 20px;
  border-top: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.section-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-title-row h3 {
  margin: 0;
  color: var(--outline, #908fa0);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.count-tag {
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.mount-list,
.provider-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.mount-row,
.provider-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  font-size: 12px;
}

.mount-name,
.provider-info {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--on-surface, #dce1fb);
}

.provider-name {
  font-weight: 500;
  text-transform: capitalize;
}

.test-result-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 8px;
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-lowest, #070d1f);
  font-size: 10px;
}

.test-target {
  color: var(--outline, #908fa0);
}

.test-outcome--reachable {
  color: var(--secondary, #4edea3);
}

.test-outcome--failed {
  color: var(--error, #ffb4ab);
}

.test-outcome--not_configured {
  color: var(--tertiary, #ffb95f);
}

.empty-state {
  margin: 0;
  color: var(--outline, #908fa0);
  font-size: 11px;
}

@media (max-width: 480px) {
  .status-grid {
    grid-template-columns: 1fr;
  }
}
</style>
