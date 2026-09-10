<script setup lang="ts">
import { computed, onMounted, shallowRef } from 'vue'
import type { Source } from '@/api/types'
import * as api from '@/api/integrations'
import IntegrationCreateForm from './IntegrationCreateForm.vue'
import IntegrationList from './IntegrationList.vue'

const props = defineProps<{
  csrfToken: string
  sources: Source[]
  locale: string
}>()

const zh = computed(() => props.locale === 'zh-CN')
const webhooks = shallowRef<api.Webhook[]>([])
const tokens = shallowRef<api.APIToken[]>([])
const status = shallowRef<api.IntegrationStatus | null>(null)
const busy = shallowRef(false)
const error = shallowRef('')
const notice = shallowRef('')
const secret = shallowRef('')
const deliveryItems = shallowRef<api.Delivery[]>([])
const selectedEndpoint = shallowRef('')
const editingId = shallowRef('')

const editing = computed(() => webhooks.value.find(item => item.id === editingId.value))

async function refresh() {
  const [hooks, keys, state] = await Promise.all([
    api.listWebhooks(),
    api.listTokens(),
    api.integrationStatus(),
  ])
  webhooks.value = hooks.items
  tokens.value = keys.items
  status.value = state
}

async function perform(work: () => Promise<void>) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    await work()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}

async function create(kind: 'webhook' | 'token', input: api.IntegrationInput) {
  await perform(async () => {
    secret.value = ''
    if (kind === 'webhook' && editing.value) {
      await api.mutate(props.csrfToken, `webhooks/${editing.value.id}`, 'PATCH', {
        ...input,
        enabled: editing.value.enabled,
        version: editing.value.version,
      })
      editingId.value = ''
      await refresh()
      return
    }
    const result = await api.mutate<{ secret: string }>(
      props.csrfToken,
      kind === 'webhook' ? 'webhooks' : 'api-tokens',
      'POST',
      kind === 'webhook'
        ? { ...input, enabled: true, eventTypes: input.eventTypes }
        : { ...input, scopes: input.scopes },
    )
    secret.value = result.secret
    await refresh()
  })
}

async function action(
  kind: 'edit' | 'toggle' | 'test' | 'rotate' | 'delete' | 'deliveries' | 'revoke',
  id: string,
) {
  await perform(async () => {
    if (kind === 'edit') {
      editingId.value = id
      return
    }
    if (kind === 'deliveries') {
      selectedEndpoint.value = id
      deliveryItems.value = (await api.deliveries(id)).items
      return
    }
    if (kind === 'toggle') {
      const endpoint = webhooks.value.find(item => item.id === id)!
      await api.mutate(props.csrfToken, `webhooks/${id}`, 'PATCH', {
        ...endpoint,
        url: '',
        enabled: !endpoint.enabled,
      })
    }
    if (kind === 'test') {
      const result = await api.mutate<{ deliveryId: string }>(props.csrfToken, `webhooks/${id}/test`, 'POST')
      notice.value = `${zh.value ? '测试已排队' : 'Test queued'}: ${result.deliveryId}`
    }
    if (kind === 'rotate') {
      secret.value = ''
      const result = await api.mutate<{ secret: string }>(props.csrfToken, `webhooks/${id}/rotate-secret`, 'POST')
      secret.value = result.secret
    }
    if (kind === 'delete' || kind === 'revoke') {
      await api.mutate(props.csrfToken, `${kind === 'delete' ? 'webhooks' : 'api-tokens'}/${id}`, 'DELETE')
    }
    await refresh()
  })
}

async function retry(id: string) {
  await perform(async () => {
    await api.mutate(props.csrfToken, `webhooks/${selectedEndpoint.value}/deliveries/${id}/retry`, 'POST')
    deliveryItems.value = (await api.deliveries(selectedEndpoint.value)).items
    await refresh()
  })
}

onMounted(() => perform(refresh))
</script>

<template>
  <section class="settings-card" aria-labelledby="integrations-title">
    <div class="card-header">
      <div class="card-header__info">
        <div class="header-tag">
          <svg class="header-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
          </svg>
          <span class="eyebrow">{{ zh ? '外部集成 & API' : 'EXTERNAL INTEGRATIONS' }}</span>
        </div>
        <h2 id="integrations-title" class="section-title">{{ zh ? '外部集成与 Webhook' : 'External Integrations & Webhooks' }}</h2>
        <p class="section-hint">{{ zh ? '配置 Webhook 事件推送与 MCP 模型上下文协议访问令牌。' : 'Configure outbound Webhook notifications and Model Context Protocol (MCP) access tokens.' }}</p>
      </div>
      <div class="card-header__actions">
        <button type="button" class="btn btn-outline btn-sm" :disabled="busy" @click="perform(refresh)">
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14" :class="{ 'spin-slow': busy }">
            <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
          </svg>
          <span>{{ zh ? '刷新' : 'Refresh' }}</span>
        </button>
      </div>
    </div>

    <div class="card-body">
      <div v-if="error" role="alert" class="integration-banner integration-banner--error">
        <span class="dot dot-err"></span>
        <span>{{ error }}</span>
      </div>
      <div v-if="notice" role="status" class="integration-banner integration-banner--info">
        <span class="dot dot-ok"></span>
        <span>{{ notice }}</span>
      </div>

      <div v-if="secret" class="secret-panel" role="status">
        <div class="secret-panel__header">
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="text-primary">
            <path d="M12.65 10C11.83 7.67 9.61 6 7 6c-3.31 0-6 2.69-6 6s2.69 6 6 6c2.61 0 4.83-1.67 5.65-4H17v4h4v-4h2v-4H12.65zM7 14c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2z"/>
          </svg>
          <span>{{ zh ? '请妥善保存此令牌密钥，关闭后将无法再次查看：' : 'Save this secret. It cannot be displayed again after dismissal:' }}</span>
        </div>
        <div class="secret-value-box">
          <code class="secret-value">{{ secret }}</code>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" @click="secret = ''">
          {{ zh ? '已保存，关闭' : 'Saved, dismiss' }}
        </button>
      </div>

      <div class="integrations-grid">
        <!-- Webhook Section -->
        <div class="sub-panel">
          <div class="sub-panel__header">
            <div class="sub-panel__title-row">
              <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="sub-panel__icon">
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
              </svg>
              <h3 class="sub-panel__title">Webhook</h3>
            </div>
            <div v-if="status" class="sub-panel__status">
              <span class="spec-pill">
                {{ zh ? '待投递' : 'Pending' }}: {{ status.pendingDeliveries }} · {{ zh ? '失败' : 'Dead' }}: {{ status.deadDeliveries }}
              </span>
              <span v-if="status.paused" class="spec-pill spec-pill--warn">{{ zh ? '发送已暂停' : 'Sending paused' }}</span>
            </div>
          </div>

          <p v-if="status && !status.signingReady" class="warning-text">
            {{ zh ? '需配置 Webhook 主密钥文件后重启服务。' : 'Configure the Webhook master key file and restart the server.' }}
          </p>

          <div v-if="editing" class="edit-banner">
            <span>{{ zh ? '正在编辑 Webhook: ' + editing.name : 'Editing Webhook: ' + editing.name }}</span>
            <button type="button" class="btn btn-ghost btn-sm" @click="editingId = ''">
              {{ zh ? '取消编辑' : 'Cancel editing' }}
            </button>
          </div>

          <IntegrationCreateForm
            :key="editingId"
            :endpoint="editing"
            kind="webhook"
            :sources="sources"
            :busy="busy || !status?.signingReady"
            :zh="zh"
            @create="create('webhook', $event)"
          />

          <IntegrationList
            :webhooks="webhooks"
            :tokens="[]"
            :busy="busy"
            :zh="zh"
            @action="action"
          />
        </div>

        <!-- MCP Tokens Section -->
        <div class="sub-panel">
          <div class="sub-panel__header">
            <div class="sub-panel__title-row">
              <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="sub-panel__icon">
                <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14c-1.66 0-3-1.34-3-3s1.34-3 3-3 3 1.34 3 3-1.34 3-3 3zm5-8H7V7h10v2z"/>
              </svg>
              <h3 class="sub-panel__title">MCP (Model Context Protocol)</h3>
            </div>
          </div>

          <p class="sub-panel__desc">
            {{ zh ? '访问指定媒体源，默认只读。文件应用必须在网页审批。连接路径 /mcp。' : 'Access selected sources at /mcp. Read only by default; file changes require Web approval.' }}
          </p>

          <IntegrationCreateForm
            kind="token"
            :sources="sources"
            :busy="busy"
            :zh="zh"
            @create="create('token', $event)"
          />

          <IntegrationList
            :webhooks="[]"
            :tokens="tokens"
            :busy="busy"
            :zh="zh"
            @action="action"
          />
        </div>
      </div>

      <!-- Deliveries History -->
      <div v-if="selectedEndpoint" class="deliveries-panel">
        <div class="deliveries-panel__header">
          <div class="deliveries-panel__title-row">
            <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="text-primary">
              <path d="M13 3c-4.97 0-9 4.03-9 9H1l3.89 3.89.07.14L9 12H6c0-3.87 3.13-7 7-7s7 3.13 7 7-3.13 7-7 7c-1.93 0-3.68-.79-4.94-2.06l-1.42 1.42C8.27 19.99 10.51 21 13 21c4.97 0 9-4.03 9-9s-4.03-9-9-9zm-1 5v5l4.28 2.54.72-1.21-3.5-2.08V8H12z"/>
            </svg>
            <h3 class="sub-panel__title">{{ zh ? '最近 100 条投递记录' : 'Latest 100 deliveries' }}</h3>
          </div>
          <button type="button" class="btn btn-outline btn-sm" :disabled="busy" @click="action('deliveries', selectedEndpoint)">
            {{ zh ? '刷新记录' : 'Refresh deliveries' }}
          </button>
        </div>

        <p v-if="!deliveryItems.length" class="empty-hint">{{ zh ? '暂无投递记录。' : 'No deliveries yet.' }}</p>

        <div v-else class="deliveries-list">
          <div v-for="item in deliveryItems" :key="item.id" class="delivery-row">
            <div class="delivery-row__info">
              <code class="delivery-event-id">{{ item.eventId }}</code>
              <span class="spec-pill" :class="item.state === 'delivered' ? 'status-pill--ok' : (item.state === 'dead' ? 'status-pill--warn' : '')">
                {{ item.state }}
              </span>
              <span class="delivery-meta">{{ item.status || '—' }} · {{ item.attempts }} attempts · {{ item.errorCode || 'OK' }}</span>
            </div>
            <button
              v-if="item.state === 'dead' || item.state === 'cancelled'"
              type="button"
              class="btn btn-outline btn-sm"
              :disabled="busy"
              @click="retry(item.id)"
            >
              {{ zh ? '重试' : 'Retry' }}
            </button>
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

.card-header__actions {
  flex-shrink: 0;
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

.integration-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-sm, 0.25rem);
  font-size: 12px;
}

.integration-banner--error {
  background: rgba(244, 63, 94, 0.1);
  border: 1px solid rgba(244, 63, 94, 0.3);
  color: var(--error, #ffb4ab);
}

.integration-banner--info {
  background: rgba(78, 222, 163, 0.1);
  border: 1px solid rgba(78, 222, 163, 0.3);
  color: var(--secondary, #4edea3);
}

.secret-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px;
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--primary-bright, #8083ff);
  border-radius: var(--radius-md, 0.375rem);
}

.secret-panel__header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
}

.secret-value-box {
  background: var(--surface-container-lowest, #070d1f);
  padding: 10px 12px;
  border-radius: var(--radius-sm, 0.25rem);
  border: 1px solid var(--outline-variant, #2e3447);
  overflow-x: auto;
}

.secret-value {
  font-family: var(--font-data, monospace);
  font-size: 12px;
  color: var(--secondary, #4edea3);
  word-break: break-all;
}

.integrations-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}

.sub-panel {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.sub-panel__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.sub-panel__title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sub-panel__icon {
  color: var(--primary, #c0c1ff);
}

.sub-panel__title {
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.sub-panel__status {
  display: flex;
  align-items: center;
  gap: 6px;
}

.sub-panel__desc {
  margin: 0;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  line-height: 1.4;
}

.warning-text {
  margin: 0;
  font-size: 11px;
  color: var(--tertiary, #ffb95f);
  background: rgba(255, 185, 95, 0.1);
  padding: 6px 10px;
  border-radius: var(--radius-sm, 0.25rem);
  border: 1px solid rgba(255, 185, 95, 0.25);
}

.edit-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--surface-container-high, #23293c);
  padding: 6px 12px;
  border-radius: var(--radius-sm, 0.25rem);
  border-left: 3px solid var(--primary, #c0c1ff);
  font-size: 12px;
  color: var(--primary, #c0c1ff);
}

.deliveries-panel {
  margin-top: 4px;
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.deliveries-panel__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.deliveries-panel__title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.empty-hint {
  margin: 0;
  font-size: 12px;
  color: var(--outline, #908fa0);
}

.deliveries-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.delivery-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
}

.delivery-row__info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.delivery-event-id {
  font-family: var(--font-data, monospace);
  font-size: 11px;
  color: var(--on-surface, #dce1fb);
}

.delivery-meta {
  font-family: var(--font-data, monospace);
  font-size: 11px;
  color: var(--on-surface-variant, #c7c4d7);
}

@media (max-width: 959px) {
  .integrations-grid {
    grid-template-columns: 1fr;
  }
}
</style>
