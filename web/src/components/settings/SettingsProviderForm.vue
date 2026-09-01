<script setup lang="ts">
import type { Settings, SettingsUpdate } from '@/api/library'

defineProps<{ settings: Settings | null; labels: Record<string, string>; saving: boolean }>()
const model = defineModel<SettingsUpdate>('model', { required: true })
const emit = defineEmits<{ save: [] }>()
</script>

<template>
  <section class="settings-card">
    <div class="card-header">
      <div class="card-header__info">
        <div class="header-tag">
          <svg class="header-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
          </svg>
          <span class="eyebrow">{{ labels.providerEyebrow }}</span>
        </div>
        <h2 class="section-title">{{ labels.providerSettings }}</h2>
      </div>
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

        <!-- Fanart.tv API Key -->
        <div class="field-group">
          <div class="field-header">
            <label for="fanart-tv-api-key" class="field-label">{{ labels.fanartTvApiKey }}</label>
            <span class="status-pill" :class="settings?.fanartTvApiKeyConfigured ? 'status-pill--configured' : 'status-pill--unset'">
              <span class="status-dot" :class="settings?.fanartTvApiKeyConfigured ? 'status-dot--ok' : 'status-dot--neutral'"></span>
              {{ settings?.fanartTvApiKeyConfigured ? labels.configured : labels.notSet }}
            </span>
          </div>
          <input id="fanart-tv-api-key" v-model="model.fanartTvApiKey" type="password" autocomplete="off" :placeholder="settings?.fanartTvApiKeyConfigured ? labels.apiKeyConfigured : labels.fanartTvApiKeyPlaceholder" />
          <p class="field-hint">{{ labels.fanartTvApiKeyHelp }}</p>
          <label class="check-label"><input v-model="model.clearFanartTvApiKey" type="checkbox" class="form-checkbox" /><span>{{ labels.clearFanartTvApiKey }}</span></label>
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
  background: var(--surface-container);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
}

.card-header {
  padding: 1.125rem 1.5rem;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--surface-container);
}

.card-header__info {
  display: grid;
  gap: 0.3rem;
}

.header-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--primary);
}

.header-icon {
  width: 14px;
  height: 14px;
}

.eyebrow {
  margin: 0;
  font-family: var(--font-data);
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--primary);
}

.section-title {
  margin: 0;
  color: var(--on-surface);
  font-family: var(--font-display);
  font-size: 1.0625rem;
  font-weight: 700;
  line-height: 1.3;
  letter-spacing: -0.01em;
}

.card-body {
  padding: 1.5rem;
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
  color: var(--on-surface-variant);
}

.field-hint {
  margin: 0;
  font-size: 0.75rem;
  color: var(--outline);
  line-height: 1.4;
}

.settings-form input:not([type="checkbox"]),
.form-select {
  width: 100%;
  height: 34px;
  padding: 0.35rem 0.75rem;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-container-low);
  color: var(--on-surface);
  font-size: 0.8125rem;
  transition: all 0.15s ease;
}

.settings-form input:not([type="checkbox"]):focus,
.form-select:focus {
  outline: none;
  border-color: var(--primary-bright);
  box-shadow: 0 0 0 2px rgba(128, 131, 255, 0.2);
  background: var(--surface-container-lowest);
}

.check-label {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--on-surface-variant);
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
  background: rgba(0, 165, 114, 0.18);
  color: var(--secondary);
  border: 1px solid rgba(78, 222, 163, 0.3);
}

.status-pill--unset {
  background: var(--surface-container-low);
  border: 1px solid var(--border-default);
  color: var(--outline);
}

.form-actions {
  padding-top: 0.25rem;
}
</style>
