<script setup lang="ts">
import type { Webhook, APIToken } from '@/api/integrations'

defineProps<{
  webhooks: Webhook[]
  tokens: APIToken[]
  busy: boolean
  zh: boolean
}>()

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
      <strong>{{ item.name }}</strong>
      <span class="spec-pill">
        {{ item.enabled ? (zh ? '已启用' : 'Enabled') : (zh ? '已暂停' : 'Paused') }}
      </span>
      <p class="integration-detail">
        {{ item.eventTypes.join(' · ') }}<br />{{ item.keyId }}
      </p>
      <div class="integration-actions">
        <button class="btn-ghost" :disabled="busy" @click="emit('action', 'edit', item.id)">
          {{ zh ? '编辑' : 'Edit' }}
        </button>
        <button class="btn-ghost" :disabled="busy" @click="emit('action', 'toggle', item.id)">
          {{ item.enabled ? (zh ? '暂停' : 'Pause') : (zh ? '启用' : 'Enable') }}
        </button>
        <button class="btn-ghost" :disabled="busy || !item.enabled" @click="emit('action', 'test', item.id)">
          {{ zh ? '测试发送' : 'Send test' }}
        </button>
        <button class="btn-ghost" :disabled="busy" @click="emit('action', 'deliveries', item.id)">
          {{ zh ? '投递记录' : 'Deliveries' }}
        </button>
        <button class="btn-ghost" :disabled="busy" @click="emit('action', 'rotate', item.id)">
          {{ zh ? '轮换密钥' : 'Rotate secret' }}
        </button>
        <button class="btn-ghost" :disabled="busy" @click="emit('action', 'delete', item.id)">
          {{ zh ? '删除' : 'Delete' }}
        </button>
      </div>
    </article>

    <article v-for="item in tokens" :key="item.id" class="integration-row">
      <strong>{{ item.name }}</strong>
      <p class="integration-detail">
        {{ item.prefix }}… · {{ new Date(item.expiresAt).toLocaleDateString() }}<br />
        {{ item.scopes.join(' · ') }}
      </p>
      <span v-if="item.revokedAt" class="spec-pill">{{ zh ? '已撤销' : 'Revoked' }}</span>
      <button
        v-else
        class="btn-ghost"
        :disabled="busy"
        @click="emit('action', 'revoke', item.id)"
      >
        {{ zh ? '撤销访问' : 'Revoke access' }}
      </button>
    </article>

    <p v-if="!webhooks.length && !tokens.length">{{ zh ? '尚未配置。' : 'Nothing configured yet.' }}</p>
  </div>
</template>

<style scoped>
.integration-list {
  display: grid;
  gap: 0.65rem;
  margin-top: 1rem;
}
.integration-row {
  padding: 0.8rem;
  background: var(--surface-container-high);
  border-radius: 0.5rem;
  overflow-wrap: anywhere;
}
.integration-detail {
  font-family: var(--font-data);
  font-size: 0.7rem;
  color: var(--on-surface-variant);
}
.integration-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}
</style>
