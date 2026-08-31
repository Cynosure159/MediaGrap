<script setup lang="ts">
import type { Settings, SettingsUpdate } from '@/api/library'

defineProps<{ settings: Settings | null; labels: Record<string, string>; saving: boolean }>()
const model = defineModel<SettingsUpdate>('model', { required: true })
const emit = defineEmits<{ save: [] }>()
</script>

<template>
  <section class="settings-card">
    <div class="card-header">
      <p class="eyebrow">{{ labels.providerEyebrow }}</p>
      <h2 class="section-title">{{ labels.providerSettings }}</h2>
    </div>

    <div class="card-body">
      <form class="settings-form" @submit.prevent="emit('save')">
        <!-- TMDb API Key -->
        <div class="field-group">
          <div class="field-header">
            <label for="tmdb-api-key" class="field-label">{{ labels.tmdbApiKey }}</label>
            <span
              class="status-pill"
              :class="settings?.tmdbApiKeyConfigured ? 'status-pill--configured' : 'status-pill--unset'"
            >
              <span
                class="status-dot"
                :class="settings?.tmdbApiKeyConfigured ? 'status-dot--ok' : 'status-dot--neutral'"
              ></span>
              {{ settings?.tmdbApiKeyConfigured ? labels.configured : labels.notSet }}
            </span>
          </div>
          <input
            id="tmdb-api-key"
            v-model="model.tmdbApiKey"
            type="password"
            autocomplete="off"
            :placeholder="settings?.tmdbApiKeyConfigured ? labels.apiKeyConfigured : labels.apiKeyPlaceholder"
          />
          <p class="field-hint">{{ settings?.tmdbApiKeyConfigured ? labels.apiKeyConfigured : labels.apiKeyPlaceholder }}</p>
          <label class="check-label">
            <input v-model="model.clearTmdbApiKey" type="checkbox" class="form-checkbox" />
            <span>{{ labels.clearApiKey }}</span>
          </label>
        </div>

        <!-- TMDb Language -->
        <div class="field-group">
          <label for="tmdb-language" class="field-label">{{ labels.tmdbLanguage }}</label>
          <select id="tmdb-language" v-model="model.tmdbLanguage" class="form-select">
            <option value="en-US">English (en-US)</option>
            <option value="zh-CN">简体中文 (zh-CN)</option>
            <option value="zh-TW">繁體中文 (zh-TW)</option>
            <option value="ja-JP">日本語 (ja-JP)</option>
            <option value="ko-KR">한국어 (ko-KR)</option>
          </select>
        </div>

        <!-- Outbound Proxy -->
        <div class="field-group">
          <div class="field-header">
            <label for="outbound-proxy" class="field-label">{{ labels.outboundProxy }}</label>
            <span
              class="status-pill"
              :class="settings?.outboundProxyConfigured ? 'status-pill--configured' : 'status-pill--unset'"
            >
              <span
                class="status-dot"
                :class="settings?.outboundProxyConfigured ? 'status-dot--ok' : 'status-dot--neutral'"
              ></span>
              {{ settings?.outboundProxyConfigured ? labels.configured : labels.notSet }}
            </span>
          </div>
          <input
            id="outbound-proxy"
            v-model="model.outboundProxy"
            type="url"
            autocomplete="off"
            :placeholder="settings?.outboundProxyConfigured ? labels.proxyConfigured : labels.proxyPlaceholder"
          />
          <p class="field-hint">{{ settings?.outboundProxyConfigured ? labels.proxyConfigured : labels.proxyPlaceholder }}</p>
          <label class="check-label">
            <input v-model="model.clearOutboundProxy" type="checkbox" class="form-checkbox" />
            <span>{{ labels.clearProxy }}</span>
          </label>
        </div>

        <!-- Submit Button -->
        <div class="form-actions">
          <button type="submit" class="btn btn-primary" :disabled="saving">
            {{ saving ? labels.saving : labels.saveSettings }}
          </button>
        </div>
      </form>
    </div>
  </section>
</template>

<style scoped>
.settings-card {
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  overflow: hidden;
}

.card-header {
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--surface-card);
}

.section-title {
  margin: 0.25rem 0 0;
  color: var(--text-primary);
  font-family: var(--font-display);
  font-size: 0.9375rem;
  font-weight: 600;
  line-height: 1.3;
  letter-spacing: -0.01em;
}

.card-body {
  padding: 1.25rem;
}

.settings-form {
  display: grid;
  gap: 1.25rem;
  max-width: 38rem;
}

.field-group {
  display: grid;
  gap: 0.4rem;
}

.field-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.field-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.field-hint {
  margin: 0;
  font-size: 0.75rem;
  color: var(--text-muted);
  line-height: 1.4;
}

.settings-form input:not([type="checkbox"]),
.form-select {
  width: 100%;
  min-height: 2.375rem;
  padding: 0.45rem 0.75rem;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-elevated);
  color: var(--text-primary);
  font-size: 0.8125rem;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.settings-form input:not([type="checkbox"]):focus,
.form-select:focus {
  outline: none;
  border-color: var(--border-focus);
  box-shadow: 0 0 0 2px rgba(128, 131, 255, 0.2);
}

.check-label {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--text-secondary);
  cursor: pointer;
  user-select: none;
  margin-top: 0.15rem;
}

.form-checkbox {
  width: 1rem;
  height: 1rem;
  accent-color: var(--primary-bright);
  cursor: pointer;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.15rem 0.5rem;
  border-radius: var(--radius-full);
  font: 500 0.6875rem/1.2 var(--font-data);
  letter-spacing: 0.02em;
}

.status-pill--configured {
  background: var(--success-container);
  color: var(--success);
}

.status-pill--unset {
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  color: var(--text-muted);
}

.form-actions {
  padding-top: 0.25rem;
}
</style>
