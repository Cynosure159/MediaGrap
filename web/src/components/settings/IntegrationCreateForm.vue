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
    <div class="field-group">
      <label class="field-label">
        {{ zh ? '名称' : 'Name' }}
      </label>
      <input
        v-model="form.name"
        required
        maxlength="100"
        :disabled="busy"
        class="input-control"
        :placeholder="zh ? '例如：Plex Webhook 或 AI Assistant' : 'e.g., Plex Webhook or AI Assistant'"
      />
    </div>

    <div v-if="kind === 'webhook'" class="field-group">
      <label class="field-label">
        Webhook URL
      </label>
      <input
        v-model="form.url"
        type="url"
        :required="!endpoint"
        maxlength="2048"
        :placeholder="endpoint ? (zh ? '留空保留当前 URL' : 'Leave blank to keep current URL') : 'https://…'"
        autocomplete="off"
        :disabled="busy"
        class="input-control font-code"
      />
    </div>

    <div v-else class="form-grid">
      <div class="field-group">
        <label class="field-label">
          {{ zh ? '有效期' : 'Expires after' }}
        </label>
        <div class="select-wrapper">
          <select v-model="form.days" :disabled="busy" class="form-select">
            <option :value="7">7 {{ zh ? '天' : 'days' }}</option>
            <option :value="30">30 {{ zh ? '天' : 'days' }}</option>
            <option :value="90">90 {{ zh ? '天' : 'days' }}</option>
          </select>
          <svg class="select-chevron" viewBox="0 0 24 24" fill="currentColor">
            <path d="M7 10l5 5 5-5z"/>
          </svg>
        </div>
      </div>

      <div class="field-group">
        <label class="field-label">
          {{ zh ? '访问权限' : 'Access' }}
        </label>
        <div class="select-wrapper">
          <select v-model="form.access" :disabled="busy" class="form-select">
            <option value="read">{{ zh ? '只读 (Read only)' : 'Read only' }}</option>
            <option value="tasks">{{ zh ? '查询和后台任务 (Tasks)' : 'Queries and background tasks' }}</option>
            <option value="plans">{{ zh ? '文件计划 (需网页审批)' : 'File plans (Web approval required)' }}</option>
          </select>
          <svg class="select-chevron" viewBox="0 0 24 24" fill="currentColor">
            <path d="M7 10l5 5 5-5z"/>
          </svg>
        </div>
      </div>
    </div>

    <fieldset class="options-fieldset" :disabled="busy">
      <legend class="options-legend">{{ zh ? '授权媒体源（必选）' : 'Authorized sources (required)' }}</legend>
      <div class="options-list">
        <label v-for="source in sources" :key="source.id" class="source-option">
          <input v-model="form.sourceIds" type="checkbox" :value="source.id" class="custom-check" />
          <span class="source-option__name">{{ source.name }}</span>
        </label>
      </div>
      <p v-if="!sources.length" class="empty-field-msg">{{ zh ? '请先在上方添加媒体源。' : 'Add a media source first.' }}</p>
    </fieldset>

    <fieldset v-if="kind === 'webhook'" class="options-fieldset" :disabled="busy">
      <legend class="options-legend">{{ zh ? '订阅事件' : 'Events' }}</legend>
      <div class="options-list">
        <label
          v-for="event in ['job.succeeded', 'job.failed', 'write_plan.applied', 'artwork_plan.applied', 'rename_plan.applied']"
          :key="event"
          class="source-option"
        >
          <input v-model="form.eventTypes" type="checkbox" :value="event" class="custom-check" />
          <span class="source-option__name font-code">{{ event }}</span>
        </label>
      </div>
    </fieldset>

    <div class="form-actions">
      <button class="btn btn-primary form-submit" type="submit" :disabled="busy || !valid">
        <span>{{ endpoint ? (zh ? '保存修改' : 'Save changes') : (zh ? '创建集成' : 'Create') }}</span>
      </button>
    </div>
  </form>
</template>

<style scoped>
.integration-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--on-surface-variant, #c7c4d7);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.input-control {
  width: 100%;
  height: 36px;
  padding: 0 12px;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  font-size: 13px;
  outline: none;
  transition: all 0.15s ease;
}

.input-control:focus {
  border-color: var(--primary, #c0c1ff);
  box-shadow: 0 0 0 2px rgba(192, 193, 255, 0.15);
}

.font-code {
  font-family: var(--font-data, monospace);
  font-size: 11px;
}

.select-wrapper {
  position: relative;
  width: 100%;
}

.form-select {
  width: 100%;
  height: 36px;
  padding: 0 30px 0 10px;
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-lowest, #070d1f);
  color: var(--on-surface, #dce1fb);
  font-size: 12px;
  outline: none;
  appearance: none;
  cursor: pointer;
  transition: all 0.15s ease;
}

.form-select:focus {
  border-color: var(--primary, #c0c1ff);
}

.select-chevron {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  color: var(--outline, #908fa0);
  pointer-events: none;
}

.options-fieldset {
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 10px 12px;
  background: var(--surface-container-lowest, #070d1f);
}

.options-legend {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--primary, #c0c1ff);
  padding: 0 4px;
}

.options-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  margin-top: 4px;
}

.source-option {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  font-size: 12px;
  color: var(--on-surface, #dce1fb);
}

.custom-check {
  cursor: pointer;
  accent-color: var(--primary-container, #6366f1);
}

.source-option__name {
  user-select: none;
}

.empty-field-msg {
  margin: 0;
  font-size: 11px;
  color: var(--outline, #908fa0);
}

.form-actions {
  display: flex;
  justify-content: flex-start;
  margin-top: 4px;
}

.form-submit {
  height: 32px;
  padding: 0 16px;
}

@media (max-width: 640px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
