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
  <section class="integrations" aria-labelledby="integrations-title">
    <header class="integrations-header">
      <h2 id="integrations-title">{{ zh ? '外部集成' : 'Integrations' }}</h2>
      <button class="btn-ghost" :disabled="busy" @click="perform(refresh)">
        {{ zh ? '刷新' : 'Refresh' }}
      </button>
    </header>

    <p v-if="error" role="alert" class="integration-error">{{ error }}</p>
    <p v-if="notice" role="status">{{ notice }}</p>

    <div v-if="secret" class="secret-panel" role="status">
      <p>{{ zh ? '请保存此密钥，关闭后无法再次查看。' : 'Save this secret. It cannot be displayed again after dismissal.' }}</p>
      <code class="secret-value">{{ secret }}</code>
      <button class="btn-secondary" @click="secret = ''">
        {{ zh ? '已保存，关闭' : 'Saved, dismiss' }}
      </button>
    </div>

    <div class="integrations-grid">
      <section class="card integration-card">
        <h3>Webhook</h3>
        <p v-if="status">
          {{ zh ? '待投递 / 失败' : 'Pending / Dead' }}: {{ status.pendingDeliveries }} / {{ status.deadDeliveries }}
          <span v-if="status.paused" class="spec-pill">{{ zh ? '发送已暂停' : 'Sending paused' }}</span>
        </p>
        <p v-if="status && !status.signingReady">
          {{ zh ? '需配置 Webhook 主密钥文件后重启服务。' : 'Configure the Webhook master key file and restart the server.' }}
        </p>
        <button v-if="editing" class="btn-ghost" @click="editingId = ''">
          {{ zh ? '取消编辑' : 'Cancel editing' }}
        </button>
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
      </section>

      <section class="card integration-card">
        <h3>MCP</h3>
        <p>{{ zh ? '访问指定媒体源，默认只读。文件应用必须在网页审批。连接路径 /mcp。' : 'Access selected sources at /mcp. Read only by default; file changes require Web approval.' }}</p>
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
      </section>
    </div>

    <section v-if="selectedEndpoint" class="card integration-card">
      <h3>{{ zh ? '最近 100 条投递记录' : 'Latest 100 deliveries' }}</h3>
      <button class="btn-ghost" :disabled="busy" @click="action('deliveries', selectedEndpoint)">
        {{ zh ? '刷新记录' : 'Refresh deliveries' }}
      </button>
      <p v-if="!deliveryItems.length">{{ zh ? '暂无投递。' : 'No deliveries yet.' }}</p>
      <div v-for="item in deliveryItems" :key="item.id" class="delivery-row">
        <code>{{ item.eventId }}</code>
        <span class="spec-pill">{{ item.state }}</span>
        <span>{{ item.status || '—' }} · {{ item.attempts }} · {{ item.errorCode }}</span>
        <button
          v-if="item.state === 'dead' || item.state === 'cancelled'"
          class="btn-ghost"
          :disabled="busy"
          @click="retry(item.id)"
        >
          {{ zh ? '重试' : 'Retry' }}
        </button>
      </div>
    </section>
  </section>
</template>

<style scoped>
.integrations {
  margin-top: 1.5rem;
  font-size: 0.8125rem;
}
.integrations-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.integrations-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1.5rem;
}
.integration-card {
  padding: 1.25rem;
  margin-bottom: 1rem;
  min-width: 0;
}
.integration-error {
  color: var(--error);
}
.secret-panel {
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--surface-container-high);
  border: 1px solid var(--primary);
  border-radius: 0.5rem;
  margin-bottom: 1rem;
}
.secret-value {
  overflow-wrap: anywhere;
  font-family: var(--font-data);
}
.delivery-row {
  display: flex;
  gap: 0.6rem;
  flex-wrap: wrap;
  align-items: center;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--outline-variant);
  overflow-wrap: anywhere;
}
@media (max-width: 959px) {
  .integrations-grid {
    grid-template-columns: 1fr;
  }
}
</style>
