<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import type { Webhook, APIToken } from '@/api/integrations'

defineProps<{
  webhooks: Webhook[]
  tokens: APIToken[]
  busy: boolean
  zh: boolean
}>()

const now = ref(Date.now())
let clock: ReturnType<typeof setInterval> | undefined
onMounted(() => { clock = setInterval(() => { now.value = Date.now() }, 1000) })
onUnmounted(() => { clearInterval(clock) })

const emit = defineEmits<{
  action: [
    kind: 'edit' | 'toggle' | 'test' | 'rotate' | 'delete' | 'deliveries' | 'revoke',
    id: string,
  ]
}>()
</script>

<template>
  <div class="integration-list">
    <article v-for="item in webhooks" :key="item.id" class="integration-row">
      <div class="integration-row__header">
        <strong class="item-name">{{ item.name }}</strong>
        <span class="spec-pill" :class="item.enabled ? 'spec-pill--ok' : 'spec-pill--warn'">
          <span class="dot" :class="item.enabled ? 'dot-ok' : 'dot-warn'"></span>
          {{ item.enabled ? (zh ? '已启用' : 'Enabled') : (zh ? '已暂停' : 'Paused') }}
        </span>
      </div>
      <div class="integration-detail font-code">
        <span class="event-types">{{ item.eventTypes.join(' · ') }}</span>
        <span class="key-id">{{ item.keyId }}</span>
      </div>
      <div class="integration-actions">
        <button type="button" class="btn btn-outline btn-sm" :disabled="busy" @click="emit('action', 'edit', item.id)">
          {{ zh ? '编辑' : 'Edit' }}
        </button>
        <button type="button" class="btn btn-outline btn-sm" :disabled="busy" @click="emit('action', 'toggle', item.id)">
          {{ item.enabled ? (zh ? '暂停' : 'Pause') : (zh ? '启用' : 'Enable') }}
        </button>
        <button type="button" class="btn btn-outline btn-sm" :disabled="busy || !item.enabled" @click="emit('action', 'test', item.id)">
          {{ zh ? '测试发送' : 'Send test' }}
        </button>
        <button type="button" class="btn btn-outline btn-sm" :disabled="busy" @click="emit('action', 'deliveries', item.id)">
          {{ zh ? '投递记录' : 'Deliveries' }}
        </button>
        <button type="button" class="btn btn-outline btn-sm" :disabled="busy" @click="emit('action', 'rotate', item.id)">
          {{ zh ? '轮换密钥' : 'Rotate secret' }}
        </button>
        <button type="button" class="btn btn-outline btn-sm btn-danger-subtle" :disabled="busy" @click="emit('action', 'delete', item.id)">
          {{ zh ? '删除' : 'Delete' }}
        </button>
      </div>
    </article>

    <article v-for="item in tokens" :key="item.id" class="integration-row">
      <div class="integration-row__header">
        <strong class="item-name">{{ item.name }}</strong>
        <span v-if="item.revokedAt" class="spec-pill spec-pill--err">
          <span class="dot dot-err"></span>
          {{ zh ? '已撤销' : 'Revoked' }}
        </span>
        <span v-else-if="item.expiresAt <= now" class="spec-pill spec-pill--warn">
          <span class="dot dot-warn"></span>
          {{ zh ? '已过期' : 'Expired' }}
        </span>
        <span v-else class="spec-pill spec-pill--ok">
          <span class="dot dot-ok"></span>
          {{ zh ? '有效' : 'Active' }}
        </span>
      </div>
      <div class="integration-detail font-code">
        <span>Prefix: {{ item.prefix }}… · {{ zh ? '到期：' : 'Expires: ' }}{{ new Date(item.expiresAt).toLocaleDateString() }}</span>
        <span class="scopes">{{ item.scopes.join(' · ') }}</span>
      </div>
      <div class="integration-actions">
        <button
          v-if="!item.revokedAt"
          type="button"
          class="btn btn-outline btn-sm btn-danger-subtle"
          :disabled="busy"
          @click="emit('action', 'revoke', item.id)"
        >
          {{ zh ? '撤销访问' : 'Revoke access' }}
        </button>
      </div>
    </article>

    <p v-if="!webhooks.length && !tokens.length" class="empty-list-hint">{{ zh ? '尚未配置。' : 'Nothing configured yet.' }}</p>
  </div>
</template>

<style scoped>
.integration-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 6px;
}

.integration-row {
  padding: 12px 14px;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.integration-row__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.item-name {
  font-size: 13px;
  color: var(--on-surface, #dce1fb);
}

.spec-pill--ok {
  color: var(--secondary, #4edea3);
}

.spec-pill--warn {
  color: var(--tertiary, #ffb95f);
}

.spec-pill--err {
  color: var(--error, #ffb4ab);
}

.integration-detail {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-family: var(--font-data, monospace);
  font-size: 11px;
  color: var(--on-surface-variant, #c7c4d7);
  overflow-wrap: anywhere;
}

.font-code {
  font-family: var(--font-data, monospace);
}

.integration-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding-top: 4px;
}

.empty-list-hint {
  margin: 0;
  font-size: 12px;
  color: var(--outline, #908fa0);
}
</style>
