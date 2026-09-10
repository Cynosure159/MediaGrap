<script setup lang="ts">
import { computed, onMounted, shallowRef } from 'vue'
import { operationsStatus } from '@/api/library'
import type { OperationsStatus } from '@/api/types'

const props = defineProps<{
  locale: string
  labels: Record<string, string>
}>()

const zh = computed(() => props.locale === 'zh-CN')
const status = shallowRef<OperationsStatus | null>(null)
const isLoading = shallowRef(false)
const error = shallowRef<string | null>(null)

async function refresh() {
  isLoading.value = true
  error.value = null
  try {
    status.value = await operationsStatus()
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Unable to load system status'
  } finally {
    isLoading.value = false
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

onMounted(refresh)
</script>

<template>
  <section class="settings-card" aria-labelledby="system-title">
    <div class="card-header">
      <div class="card-header__info">
        <div class="header-tag">
          <svg class="header-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 18c1.1 0 1.99-.9 1.99-2L22 6c0-1.1-.9-2-2-2H4c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"/>
          </svg>
          <span class="eyebrow">{{ zh ? '系统与容器运行环境' : 'SYSTEM RUNTIME & DOCKER' }}</span>
        </div>
        <h2 id="system-title" class="section-title">{{ zh ? '系统与运行环境信息' : 'System & Container Runtime' }}</h2>
        <p class="section-hint">{{ zh ? '监控 MediaGrap 核心服务、SQLite WAL 存储模式与持久化缓存状态。' : 'Monitor application build metadata, SQLite WAL storage, and persistent cache.' }}</p>
      </div>
      <div class="card-header__actions">
        <button type="button" class="btn btn-outline btn-sm" :disabled="isLoading" @click="refresh">
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14" :class="{ 'spin-slow': isLoading }">
            <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
          </svg>
          <span>{{ zh ? '刷新' : 'Refresh' }}</span>
        </button>
      </div>
    </div>

    <div class="card-body">
      <div v-if="error" role="alert" class="system-banner system-banner--error">
        <span class="dot dot-err"></span>
        <span>{{ error }}</span>
      </div>

      <div class="system-grid">
        <!-- Application Meta -->
        <div class="info-block">
          <h3 class="block-title">{{ zh ? '应用程序构建信息' : 'Application Build' }}</h3>
          <div class="specs-table font-code">
            <div class="spec-row">
              <span class="spec-k">Version</span>
              <span class="spec-v spec-v--pill">{{ status?.application.version || 'v0.1.0' }}</span>
            </div>
            <div class="spec-row">
              <span class="spec-k">Commit</span>
              <span class="spec-v">{{ status?.application.commit ? status.application.commit.slice(0, 8) : 'development' }}</span>
            </div>
            <div class="spec-row">
              <span class="spec-k">Built At</span>
              <span class="spec-v">{{ status?.application.builtAt || 'Local dev' }}</span>
            </div>
            <div class="spec-row">
              <span class="spec-k">Architecture</span>
              <span class="spec-v">Go Monolith + Embedded Vue PWA</span>
            </div>
          </div>
        </div>

        <!-- Database Meta -->
        <div class="info-block">
          <h3 class="block-title">{{ zh ? '数据库与持久化' : 'Database & Persistence' }}</h3>
          <div class="specs-table font-code">
            <div class="spec-row">
              <span class="spec-k">Engine</span>
              <span class="spec-v spec-v--pill">SQLite 3 (WAL Mode)</span>
            </div>
            <div class="spec-row">
              <span class="spec-k">Status</span>
              <span class="spec-v spec-v--ok">
                <span class="dot dot-ok"></span>
                {{ status?.database.ready ? (zh ? '就绪 (Ready)' : 'Ready') : (zh ? '异常' : 'Not ready') }}
              </span>
            </div>
            <div class="spec-row">
              <span class="spec-k">Size</span>
              <span class="spec-v">{{ status?.database.sizeBytes ? formatBytes(status.database.sizeBytes) : '—' }}</span>
            </div>
            <div class="spec-row">
              <span class="spec-k">Migration</span>
              <span class="spec-v truncate">{{ status?.database.latestMigration || 'latest' }}</span>
            </div>
          </div>
        </div>

        <!-- Cache & Storage -->
        <div class="info-block">
          <h3 class="block-title">{{ zh ? '缓存与临时存储' : 'Cache & Storage' }}</h3>
          <div class="specs-table font-code">
            <div class="spec-row">
              <span class="spec-k">Cache Path</span>
              <span class="spec-v truncate">{{ status?.cache.path || '/cache' }}</span>
            </div>
            <div class="spec-row">
              <span class="spec-k">Writable</span>
              <span class="spec-v" :class="status?.cache.writable ? 'spec-v--ok' : 'spec-v--warn'">
                {{ status?.cache.writable ? (zh ? '读写正常 (Writable)' : 'Writable') : (zh ? '受限' : 'Read-only') }}
              </span>
            </div>
            <div class="spec-row">
              <span class="spec-k">Used</span>
              <span class="spec-v">{{ status?.cache.usedBytes ? formatBytes(status.cache.usedBytes) : '0 B' }}</span>
            </div>
          </div>
        </div>

        <!-- Security & Principles -->
        <div class="info-block">
          <h3 class="block-title">{{ zh ? '安全与文件原则' : 'Safety & Write Policy' }}</h3>
          <div class="safety-notes">
            <p class="safety-item">
              <span class="dot dot-ok"></span>
              <span><strong>Atomic Replace</strong>: NFO 与图片使用临时文件 + 原子重命名，杜绝脏写。</span>
            </p>
            <p class="safety-item">
              <span class="dot dot-ok"></span>
              <span><strong>Dry-Run Verification</strong>: 重命名与批量变更强制先执行预览与冲突排查。</span>
            </p>
            <p class="safety-item">
              <span class="dot dot-ok"></span>
              <span><strong>Agent Gate</strong>: MCP 外部智能体文件修改须通过网页人工审批。</span>
            </p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.settings-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-xl, 0.75rem);
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px 24px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-container-high, #23293c);
  gap: 16px;
}

.card-header__info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.header-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--primary, #c0c1ff);
}

.header-icon {
  width: 14px;
  height: 14px;
}

.eyebrow {
  margin: 0;
  font-family: var(--font-data, monospace);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--primary, #c0c1ff);
}

.section-title {
  margin: 0;
  color: var(--on-surface, #dce1fb);
  font-size: 17px;
  font-weight: 700;
  line-height: 1.3;
  letter-spacing: -0.01em;
}

.section-hint {
  margin: 0;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  line-height: 1.4;
}

.card-body {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.system-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-sm, 0.25rem);
  font-size: 12px;
}

.system-banner--error {
  background: rgba(244, 63, 94, 0.1);
  border: 1px solid rgba(244, 63, 94, 0.3);
  color: var(--error, #ffb4ab);
}

.system-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}

.info-block {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.block-title {
  margin: 0;
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.specs-table {
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: var(--surface-container-lowest, #070d1f);
  padding: 12px;
  border-radius: var(--radius-sm, 0.25rem);
  border: 1px solid var(--outline-variant, #2e3447);
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

.spec-v--pill {
  background: var(--surface-container-high, #23293c);
  color: var(--primary, #c0c1ff);
  padding: 2px 6px;
  border-radius: var(--radius-sm, 0.25rem);
}

.spec-v--ok {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--secondary, #4edea3);
}

.spec-v--warn {
  color: var(--tertiary, #ffb95f);
}

.safety-notes {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--on-surface-variant, #c7c4d7);
}

.safety-item {
  margin: 0;
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.safety-item .dot {
  margin-top: 6px;
  flex-shrink: 0;
}

.font-code {
  font-family: var(--font-data, monospace);
}

@media (max-width: 860px) {
  .system-grid {
    grid-template-columns: 1fr;
  }
}
</style>
