<script setup lang="ts">
import type { ConnectionTest, Settings, SettingsUpdate } from '@/api/library'

defineProps<{ settings: Settings | null; labels: Record<string, string>; saving: boolean; connectionTests: Record<string, ConnectionTest | null>; testingTarget: string | null }>()
const model = defineModel<SettingsUpdate>('model', { required: true })
const emit = defineEmits<{ save: []; test: [target: ConnectionTest['target']] }>()
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
            class="input-control font-code"
            :placeholder="settings?.tmdbApiKeyConfigured ? labels.apiKeyConfigured : labels.apiKeyPlaceholder"
          />
          <p v-if="!settings?.tmdbApiKeyConfigured" class="field-hint">{{ labels.apiKeyPlaceholder }}</p>
          <label v-if="settings?.tmdbApiKeyConfigured" class="custom-checkbox-row">
            <input v-model="model.clearTmdbApiKey" type="checkbox" class="sr-only" />
            <span class="custom-checkbox-box" :class="{ 'custom-checkbox-box--checked': model.clearTmdbApiKey }">
              <svg v-if="model.clearTmdbApiKey" viewBox="0 0 24 24" fill="currentColor" width="11" height="11">
                <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
              </svg>
            </span>
            <span class="custom-checkbox-label" :class="{ 'custom-checkbox-label--checked': model.clearTmdbApiKey }">
              {{ labels.clearApiKey }}
            </span>
          </label>
		  <button type="button" class="btn btn-outline btn-sm test-btn" :disabled="testingTarget !== null || !settings?.tmdbApiKeyConfigured" @click="emit('test', 'tmdb')">{{ testingTarget === 'tmdb' ? labels.testingConnection : labels.testConnection }}</button>
		  <p v-if="connectionTests.tmdb" class="test-result" :class="`test-result--${connectionTests.tmdb.status}`">{{ labels[`connection_${connectionTests.tmdb.status}`] }} · {{ connectionTests.tmdb.durationMs }} ms</p>
        </div>

        <!-- TMDb Language -->
        <div class="field-group">
          <label for="tmdb-language" class="field-label">{{ labels.tmdbLanguage }}</label>
          <div class="select-wrapper">
            <select id="tmdb-language" v-model="model.tmdbLanguage" class="form-select">
              <option value="zh-CN">简体中文 (zh-CN)</option>
              <option value="zh-TW">繁體中文 (zh-TW)</option>
              <option value="en-US">English (en-US)</option>
              <option value="ja-JP">日本語 (ja-JP)</option>
              <option value="ko-KR">한국어 (ko-KR)</option>
            </select>
            <svg class="select-chevron" viewBox="0 0 24 24" fill="currentColor">
              <path d="M7 10l5 5 5-5z"/>
            </svg>
          </div>
        </div>

		<div class="field-group">
		  <label for="fallback-language" class="field-label">{{ labels.fallbackLanguage }}</label>
		  <div class="select-wrapper"><select id="fallback-language" v-model="model.fallbackLanguage" class="form-select"><option value="en-US">English (en-US)</option><option value="zh-CN">简体中文 (zh-CN)</option><option value="zh-TW">繁體中文 (zh-TW)</option><option value="ja-JP">日本語 (ja-JP)</option><option value="ko-KR">한국어 (ko-KR)</option></select></div>
		  <p class="field-hint">{{ labels.fallbackLanguageHelp }}</p>
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
          <input
            id="fanart-tv-api-key"
            v-model="model.fanartTvApiKey"
            type="password"
            autocomplete="off"
            class="input-control font-code"
            :placeholder="settings?.fanartTvApiKeyConfigured ? labels.apiKeyConfigured : labels.fanartTvApiKeyPlaceholder"
          />
          <p class="field-hint">{{ labels.fanartTvApiKeyHelp }}</p>
          <label v-if="settings?.fanartTvApiKeyConfigured" class="custom-checkbox-row">
            <input v-model="model.clearFanartTvApiKey" type="checkbox" class="sr-only" />
            <span class="custom-checkbox-box" :class="{ 'custom-checkbox-box--checked': model.clearFanartTvApiKey }">
              <svg v-if="model.clearFanartTvApiKey" viewBox="0 0 24 24" fill="currentColor" width="11" height="11">
                <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
              </svg>
            </span>
            <span class="custom-checkbox-label" :class="{ 'custom-checkbox-label--checked': model.clearFanartTvApiKey }">
              {{ labels.clearFanartTvApiKey }}
            </span>
          </label>
          <button type="button" class="btn btn-outline btn-sm test-btn" :disabled="testingTarget !== null || !settings?.fanartTvApiKeyConfigured" @click="emit('test', 'fanart_tv')">{{ testingTarget === 'fanart_tv' ? labels.testingConnection : labels.testConnection }}</button>
          <p v-if="connectionTests.fanart_tv" class="test-result" :class="`test-result--${connectionTests.fanart_tv.status}`">{{ labels[`connection_${connectionTests.fanart_tv.status}`] }} · {{ connectionTests.fanart_tv.durationMs }} ms</p>
        </div>

        <div class="field-group">
          <div class="field-header">
            <label for="fanart-tv-personal-api-key" class="field-label">{{ labels.fanartTvPersonalApiKey }}</label>
            <span class="status-pill" :class="settings?.fanartTvPersonalApiKeyConfigured ? 'status-pill--configured' : 'status-pill--unset'">
              {{ settings?.fanartTvPersonalApiKeyConfigured ? labels.configured : labels.notSet }}
            </span>
          </div>
          <input id="fanart-tv-personal-api-key" v-model="model.fanartTvPersonalApiKey" type="password" autocomplete="off" class="input-control font-code" :placeholder="settings?.fanartTvPersonalApiKeyConfigured ? labels.apiKeyConfigured : labels.fanartTvPersonalApiKeyPlaceholder" />
          <p class="field-hint">{{ labels.fanartTvPersonalApiKeyHelp }}</p>
          <label v-if="settings?.fanartTvPersonalApiKeyConfigured" class="custom-checkbox-row">
            <input v-model="model.clearFanartTvPersonalApiKey" type="checkbox" class="sr-only" />
            <span class="custom-checkbox-label">{{ labels.clearFanartTvPersonalApiKey }}</span>
          </label>
        </div>

        <!-- Submit Button -->
        <div class="form-actions">
          <button type="submit" class="btn btn-primary save-btn" :disabled="saving">
            <svg v-if="saving" viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="spin-slow">
              <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
            </svg>
            <svg v-else viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
              <path d="M17 3H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V7l-4-4zm2 16H5V5h11.17L19 7.83V19zm-7-7c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3zM6 6h9v4H6z"/>
            </svg>
            <span>{{ saving ? (labels.saving || '正在保存…') : (labels.saveSettings || '保存提供方设置') }}</span>
          </button>
        </div>
      </form>
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
  padding: 20px 24px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-container-high, #23293c);
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

.card-body {
  padding: 24px;
}

.settings-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.field-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--on-surface-variant, #c7c4d7);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.field-hint {
  margin: 0;
  font-size: 11px;
  color: var(--outline, #908fa0);
  line-height: 1.4;
}
.test-btn { width: fit-content; margin-top: 4px; }
.test-result { margin: 0; font: 11px var(--font-data); color: var(--outline); }
.test-result--reachable { color: var(--secondary); }
.test-result--failed { color: var(--error); }
.test-result--not_configured { color: var(--tertiary); }

.input-control {
  width: 100%;
  height: 38px;
  padding: 0 12px;
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-low, #151b2d);
  color: var(--on-surface, #dce1fb);
  font-size: 13px;
  outline: none;
  box-sizing: border-box;
  transition: all 0.15s ease;
}

.input-control:focus {
  border-color: var(--primary, #c0c1ff);
  box-shadow: 0 0 0 2px rgba(192, 193, 255, 0.15);
  background: var(--surface-container-lowest, #070d1f);
}

.font-code {
  font-family: var(--font-code, monospace);
}

.select-wrapper {
  position: relative;
  width: 100%;
}

.form-select {
  width: 100%;
  height: 38px;
  padding: 0 32px 0 12px;
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-low, #151b2d);
  color: var(--on-surface, #dce1fb);
  font-size: 13px;
  outline: none;
  appearance: none;
  cursor: pointer;
  box-sizing: border-box;
  transition: all 0.15s ease;
}

.form-select:focus {
  border-color: var(--primary, #c0c1ff);
  box-shadow: 0 0 0 2px rgba(192, 193, 255, 0.15);
  background: var(--surface-container-lowest, #070d1f);
}

.select-chevron {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  width: 18px;
  height: 18px;
  color: var(--outline, #908fa0);
  pointer-events: none;
}

/* ── Custom Dark Theme Checkbox ───────────────────────────────────── */
.custom-checkbox-row {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
  margin-top: 4px;
  padding: 4px 10px 4px 6px;
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  transition: all 0.15s ease;
  width: fit-content;
}

.custom-checkbox-row:hover {
  border-color: var(--outline, #908fa0);
  background: var(--surface-container-high, #23293c);
}

.custom-checkbox-box {
  width: 16px;
  height: 16px;
  border-radius: 3px;
  border: 1.5px solid var(--outline, #908fa0);
  background: var(--surface-container-low, #151b2d);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
  flex-shrink: 0;
  color: #ffffff;
}

.custom-checkbox-box--checked {
  background: #ff4d4f;
  border-color: #ff4d4f;
}

.custom-checkbox-label {
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  font-weight: 500;
  transition: color 0.15s ease;
}

.custom-checkbox-label--checked {
  color: #ff7875;
  font-weight: 600;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border-width: 0;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 1px 8px;
  border-radius: 9999px;
  font-size: 11px;
  font-weight: 600;
  font-family: var(--font-data, monospace);
}

.status-pill--configured {
  background: rgba(78, 222, 163, 0.12);
  color: var(--secondary, #4edea3);
  border: 1px solid rgba(78, 222, 163, 0.3);
}

.status-pill--unset {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  color: var(--outline, #908fa0);
}

.status-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
}

.status-dot--ok {
  background: var(--secondary, #4edea3);
}

.status-dot--neutral {
  background: var(--outline, #908fa0);
}

.form-actions {
  padding-top: 4px;
}

.save-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 38px;
  padding: 0 20px;
  font-weight: 600;
  font-size: 13px;
}

.spin-slow {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
