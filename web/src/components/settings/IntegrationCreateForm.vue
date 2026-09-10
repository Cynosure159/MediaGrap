<script setup lang="ts">
import { reactive, computed } from 'vue'
import type { Source } from '@/api/types'
import type { IntegrationInput, Webhook } from '@/api/integrations'

const props = defineProps<{
  kind: 'webhook' | 'token'
  sources: Source[]
  busy: boolean
  zh: boolean
  endpoint?: Webhook
}>()

const emit = defineEmits<{
  create: [input: IntegrationInput]
}>()

const form = reactive({
  name: props.endpoint?.name ?? '',
  url: '',
  sourceIds: [...(props.endpoint?.sourceIds ?? [])],
  days: 30,
  access: 'read',
  eventTypes: [...(props.endpoint?.eventTypes ?? ['job.succeeded', 'job.failed'])],
})

const valid = computed(() => {
  if (!form.name.trim() || form.sourceIds.length === 0) return false
  if (props.kind === 'token') return true
  return form.eventTypes.length > 0 && Boolean(props.endpoint || form.url.trim())
})

function submit() {
  if (!valid.value) return

  const scopes = [
    'media:read',
    'metadata:read',
    'jobs:read',
    ...(form.access !== 'read' ? ['jobs:write', 'metadata:write'] : []),
    ...(form.access === 'plans' ? ['plans:preview', 'plans:apply'] : []),
  ]

  emit('create', {
    name: form.name,
    url: form.url,
    sourceIds: [...form.sourceIds],
    expiresAt: Date.now() + form.days * 86400000,
    eventTypes: [...form.eventTypes],
    scopes,
  })
}
</script>

<template>
  <form class="integration-form" @submit.prevent="submit">
    <label>
      {{ zh ? '名称' : 'Name' }}
      <input v-model="form.name" required maxlength="100" :disabled="busy" />
    </label>

    <label v-if="kind === 'webhook'">
      Webhook URL
      <input
        v-model="form.url"
        type="url"
        :required="!endpoint"
        maxlength="2048"
        :placeholder="endpoint ? (zh ? '留空保留当前 URL' : 'Leave blank to keep current URL') : 'https://…'"
        autocomplete="off"
        :disabled="busy"
      />
    </label>
    <label v-else>
      {{ zh ? '有效期' : 'Expires after' }}
      <select v-model="form.days" :disabled="busy">
        <option :value="7">7 {{ zh ? '天' : 'days' }}</option>
        <option :value="30">30 {{ zh ? '天' : 'days' }}</option>
        <option :value="90">90 {{ zh ? '天' : 'days' }}</option>
      </select>
    </label>

    <label v-if="kind === 'token'">
      {{ zh ? '访问权限' : 'Access' }}
      <select v-model="form.access" :disabled="busy">
        <option value="read">{{ zh ? '只读' : 'Read only' }}</option>
        <option value="tasks">{{ zh ? '查询和后台任务' : 'Queries and background tasks' }}</option>
        <option value="plans">{{ zh ? '文件计划（每次需网页审批）' : 'File plans (Web approval required)' }}</option>
      </select>
    </label>

    <fieldset :disabled="busy">
      <legend>{{ zh ? '授权媒体源（必选）' : 'Authorized sources (required)' }}</legend>
      <label v-for="source in sources" :key="source.id" class="source-option">
        <input v-model="form.sourceIds" type="checkbox" :value="source.id" />
        {{ source.name }}
      </label>
      <p v-if="!sources.length">{{ zh ? '请先添加媒体源。' : 'Add a media source first.' }}</p>
    </fieldset>

    <fieldset v-if="kind === 'webhook'" :disabled="busy">
      <legend>{{ zh ? '订阅事件' : 'Events' }}</legend>
      <label
        v-for="event in ['job.succeeded', 'job.failed', 'write_plan.applied', 'artwork_plan.applied', 'rename_plan.applied']"
        :key="event"
        class="source-option"
      >
        <input v-model="form.eventTypes" type="checkbox" :value="event" />
        {{ event }}
      </label>
    </fieldset>

    <button class="btn-primary" type="submit" :disabled="busy || !valid">
      {{ endpoint ? (zh ? '保存修改' : 'Save changes') : (zh ? '创建' : 'Create') }}
    </button>
  </form>
</template>

<style scoped>
.integration-form {
  display: grid;
  gap: 0.8rem;
}
.integration-form label {
  display: grid;
  gap: 0.3rem;
  font-size: 0.8rem;
}
.integration-form input:not([type=checkbox]),
.integration-form select {
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
  padding: 0.55rem;
  color: var(--on-surface);
  background: var(--surface-container-high);
  border: 1px solid var(--outline-variant);
  border-radius: 0.35rem;
}
.integration-form fieldset {
  border: 1px solid var(--outline-variant);
  border-radius: 0.35rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  font-size: 0.8rem;
}
.integration-form .source-option {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}
</style>
