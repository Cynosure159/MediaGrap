<script setup lang="ts">
import { computed } from 'vue'
import type { ConnectionTest, Settings, Source } from '@/api/library'

const props = defineProps<{
  settings: Settings | null
  sources: Source[]
  connectionTests: Record<string, ConnectionTest | null>
  testingTarget: string | null
  locale: string
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  test: [target: ConnectionTest['target']]
}>()

const zh = computed(() => props.locale === 'zh-CN')

const totalIndexed = computed(() => {
  return props.sources.reduce((acc, s) => acc + (s.itemCount || 0), 0)
})

const allWritable = computed(() => {
  return props.sources.length > 0 && props.sources.every(s => s.writable)
})

function maskKey(configured: boolean, key?: string): string {
  if (!configured) return zh.value ? '未配置' : 'Not configured'
  if (key && key.length > 8) {
    return '••••••••' + key.slice(-4)
  }
  return '••••••••••••'
}
</script>

<template>
  <aside class="settings-status-sidebar">
    <!-- Provider Connection Preview -->
    <div class="status-card">
      <div class="status-card__header">
        <div class="header-left">
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="header-icon">
            <path d="M14 12l-2 2-2-2 2-2 2 2zm-2-6l4 4-4 4-4-4 4-4zm8 8l-2-2-2 2 2 2 2-2zm-6 2l-2 2-2-2 2-2 2 2zm-6-2l-2-2-2 2 2 2 2-2zm-4-4l4-4-4-4-4 4 4 4zm16 0l4-4-4-4-4 4 4 4z"/>
          </svg>
          <h3 class="status-card__title">{{ zh ? '提供方连通性' : 'Provider Connection' }}</h3>
        </div>
      </div>

      <div class="status-items-list">
        <!-- TMDb Card -->
        <div class="diagnostic-box">
          <div class="diagnostic-box__top">
            <span class="diagnostic-name">TMDb API</span>
            <span
              v-if="connectionTests.tmdb"
              class="spec-pill"
              :class="connectionTests.tmdb.status === 'reachable' ? 'spec-pill--ok' : connectionTests.tmdb.status === 'failed' ? 'spec-pill--err' : 'spec-pill--neutral'"
              role="status"
            >
              {{ labels[`connection_${connectionTests.tmdb.status}`] }} · {{ connectionTests.tmdb.durationMs }} ms
            </span>
            <span
              v-else-if="settings?.tmdbApiKeyConfigured"
              class="spec-pill spec-pill--ok"
            >
              {{ zh ? '已配置' : 'Configured' }}
            </span>
            <span v-else class="spec-pill spec-pill--neutral">
              {{ zh ? '未配置' : 'Not set' }}
            </span>
          </div>

          <div class="diagnostic-specs font-code">
            <div class="spec-row">
              <span class="spec-k">Key</span>
              <span class="spec-v">{{ maskKey(Boolean(settings?.tmdbApiKeyConfigured)) }}</span>
            </div>
            <div class="spec-row">
              <span class="spec-k">{{ zh ? '首选' : 'Pref' }}</span>
              <span class="spec-v spec-v--highlight">{{ settings?.tmdbLanguage || 'zh-CN' }}</span>
            </div>
            <div class="spec-row">
              <span class="spec-k">{{ zh ? '回退' : 'Fallback' }}</span>
              <span class="spec-v">{{ settings?.fallbackLanguage || 'en-US' }}</span>
            </div>
          </div>

          <button
            type="button"
            class="btn btn-outline btn-sm test-action-btn"
            :disabled="testingTarget !== null || !settings?.tmdbApiKeyConfigured"
            @click="emit('test', 'tmdb')"
          >
            <span class="dot" :class="settings?.tmdbApiKeyConfigured ? 'dot-ok' : 'dot-off'"></span>
            <span>{{ testingTarget === 'tmdb' ? (zh ? '测试中…' : 'Testing…') : (zh ? '测试 TMDb' : 'Ping TMDb') }}</span>
          </button>
        </div>

        <!-- Outbound Proxy Card -->
        <div class="diagnostic-box">
          <div class="diagnostic-box__top">
            <span class="diagnostic-name">{{ zh ? '出站代理' : 'Outbound Proxy' }}</span>
            <span
              v-if="connectionTests.proxy"
              class="spec-pill"
              :class="connectionTests.proxy.status === 'reachable' ? 'spec-pill--ok' : connectionTests.proxy.status === 'failed' ? 'spec-pill--err' : 'spec-pill--neutral'"
              role="status"
            >
              {{ labels[`connection_${connectionTests.proxy.status}`] }} · {{ connectionTests.proxy.durationMs }} ms
            </span>
            <span
              v-else-if="settings?.outboundProxyConfigured"
              class="spec-pill spec-pill--ok"
            >
              {{ zh ? '已配置' : 'Configured' }}
            </span>
            <span v-else class="spec-pill spec-pill--neutral">
              {{ zh ? '直连' : 'Direct' }}
            </span>
          </div>

          <div v-if="settings?.outboundProxyConfigured" class="proxy-addr-pill font-code">
            <svg viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
              <path d="M20.2 5.9l.8-.8C19.6 3.7 17.8 3 16 3s-3.6.7-5 2.1l.8.8C13 4.8 14.5 4.2 16 4.2s3 .6 4.2 1.7zm2.8-2.8l.8-.8C21.6.8 18.9 0 16 0S10.4.8 8.2 2.3l.8.8C10.8 1.9 13.3 1.2 16 1.2s5.2.7 7 1.9zM19 13h-2V9h-2v4H5c-1.1 0-2 .9-2 2v4c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2v-4c0-1.1-.9-2-2-2zM6 18H4v-2h2v2zm3 0H7v-2h2v2zm3 0h-2v-2h2v2zm7 0h-2v-2h2v2z"/>
            </svg>
            <span class="truncate">{{ zh ? '出站代理已启用' : 'Outbound Proxy Active' }}</span>
          </div>

          <button
            type="button"
            class="btn btn-outline btn-sm test-action-btn"
            :disabled="testingTarget !== null || !settings?.outboundProxyConfigured"
            @click="emit('test', 'proxy')"
          >
            <span class="dot" :class="settings?.outboundProxyConfigured ? 'dot-ok' : 'dot-off'"></span>
            <span>{{ testingTarget === 'proxy' ? (zh ? '测速中…' : 'Testing…') : (zh ? '测速代理' : 'Ping Proxy') }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Storage & Health Summary -->
    <div class="status-card">
      <div class="status-card__header">
        <div class="header-left">
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="header-icon">
            <path d="M2 20h20v-4H2v4zm2-3h2v2H4v-2zM2 4v4h20V4H2zm4 3H4V5h2v2zm-4 7h20v-4H2v4zm2-3h2v2H4v-2z"/>
          </svg>
          <h3 class="status-card__title">{{ zh ? '存储与挂载健康' : 'Storage Health' }}</h3>
        </div>
      </div>

      <div class="storage-summary-box font-code">
        <div class="spec-row">
          <span class="spec-k">{{ zh ? '挂载源数量' : 'Sources' }}</span>
          <span class="spec-v">{{ sources.length }} {{ zh ? '个目录' : 'roots' }}</span>
        </div>
        <div class="spec-row">
          <span class="spec-k">{{ zh ? '已索引媒体' : 'Indexed' }}</span>
          <span class="spec-v">{{ totalIndexed }} {{ zh ? '项' : 'items' }}</span>
        </div>
        <div class="spec-row">
          <span class="spec-k">{{ zh ? '写入权限' : 'Permissions' }}</span>
          <span class="spec-v" :class="allWritable ? 'spec-v--ok' : 'spec-v--warn'">
            {{ allWritable ? (zh ? '可写 (Writable)' : 'Writable') : (zh ? '只读/部分受限' : 'Read-only') }}
          </span>
        </div>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.settings-status-sidebar {
  width: 320px;
  min-width: 300px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  flex-shrink: 0;
}

.status-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-xl, 0.75rem);
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}

.status-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon {
  color: var(--primary, #c0c1ff);
}

.status-card__title {
  margin: 0;
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  letter-spacing: -0.01em;
}

.status-items-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.diagnostic-box {
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.diagnostic-box__top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.diagnostic-name {
  font-size: 12px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
}

.spec-pill--ok {
  background: rgba(78, 222, 163, 0.1);
  color: var(--secondary, #4edea3);
  border-color: rgba(78, 222, 163, 0.3);
}

.spec-pill--err {
  color: var(--error, #ffb4ab);
  border-color: var(--error, #ffb4ab);
}

.spec-pill--neutral {
  background: var(--surface-container-high, #23293c);
  color: var(--outline, #908fa0);
}

.diagnostic-specs {
  display: flex;
  flex-direction: column;
  gap: 4px;
  background: var(--surface-dim, #0c1324);
  padding: 6px 8px;
  border-radius: var(--radius-sm, 0.25rem);
  border: 1px solid rgba(46, 52, 71, 0.5);
}

.spec-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
}

.spec-k {
  color: var(--outline, #908fa0);
}

.spec-v {
  color: var(--on-surface-variant, #c7c4d7);
}

.spec-v--highlight {
  color: var(--primary, #c0c1ff);
  font-weight: 600;
}

.spec-v--ok {
  color: var(--secondary, #4edea3);
  font-weight: 600;
}

.spec-v--warn {
  color: var(--tertiary, #ffb95f);
}

.proxy-addr-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  background: var(--surface-dim, #0c1324);
  padding: 6px 8px;
  border-radius: var(--radius-sm, 0.25rem);
  border: 1px solid rgba(46, 52, 71, 0.5);
  color: var(--on-surface-variant, #c7c4d7);
}

.test-action-btn {
  width: 100%;
  justify-content: center;
  font-size: 11px;
  height: 26px;
}

.storage-summary-box {
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.font-code {
  font-family: var(--font-data, monospace);
}

@media (max-width: 1100px) {
  .settings-status-sidebar {
    width: 100%;
    min-width: 0;
  }
}
</style>
