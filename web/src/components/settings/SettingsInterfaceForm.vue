<script setup lang="ts">
import type { Locale } from '@/composables/useLocale'
import type { SettingsUpdate } from '@/api/library'
import type { Theme } from '@/composables/useTheme'

defineProps<{ locale: Locale; labels: Record<string, string>; saving: boolean }>()
const model = defineModel<SettingsUpdate>('model', { required: true })
const emit = defineEmits<{ changeLocale: [locale: Locale]; changeTheme: [theme: Theme]; save: [] }>()

function changeLocale(value: Locale) {
	model.value.locale = value
	emit('changeLocale', value)
}
</script>

<template>
  <section class="settings-card">
    <div class="card-header">
      <div class="card-header__info">
        <div class="header-tag">
          <svg class="header-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12.87 15.07l-2.54-2.51.03-.03c1.74-1.94 2.98-4.17 3.71-6.53H17V4h-7V2H8v2H1v1.99h11.17C11.5 7.92 10.44 9.75 9 11.35 8.07 10.32 7.3 9.19 6.69 8h-2c.73 1.63 1.73 3.17 2.98 4.56l-5.09 5.02L4 19l5-5 3.11 3.11.76-2.04zM18.5 10h-2L12 22h2l1.12-3h4.75L21 22h2l-4.5-12zm-2.62 7l1.62-4.33L19.12 17h-3.24z"/>
          </svg>
          <span class="eyebrow">{{ labels.interfaceEyebrow }}</span>
        </div>
        <h2 class="section-title">{{ labels.interfaceSettings }}</h2>
        <p class="section-hint">{{ labels.interfaceHint || (locale === 'zh-CN' ? '自定义语言偏好与深浅主题显示效果。' : 'Customize interface language and theme preferences.') }}</p>
      </div>
    </div>

    <div class="card-body">
      <div class="form-grid">
        <div class="field-group">
          <label for="interface-language" class="field-label">{{ labels.interfaceLanguage }}</label>
          <div class="select-wrapper">
            <select
              id="interface-language"
              class="form-select"
              :value="model.locale || locale"
              @change="changeLocale(($event.target as HTMLSelectElement).value as Locale)"
            >
              <option value="zh-CN">简体中文 (zh-CN)</option>
              <option value="en">English (en)</option>
            </select>
            <svg class="select-chevron" viewBox="0 0 24 24" fill="currentColor">
              <path d="M7 10l5 5 5-5z"/>
            </svg>
          </div>
        </div>

        <div class="field-group">
          <label for="interface-theme" class="field-label">{{ labels.interfaceTheme }}</label>
          <div class="select-wrapper">
            <select id="interface-theme" v-model="model.theme" class="form-select" @change="emit('changeTheme', model.theme)">
              <option value="dark">{{ labels.themeDark }}</option>
              <option value="light">{{ labels.themeLight }}</option>
              <option value="system">{{ labels.themeSystem }}</option>
            </select>
            <svg class="select-chevron" viewBox="0 0 24 24" fill="currentColor">
              <path d="M7 10l5 5 5-5z"/>
            </svg>
          </div>
        </div>
      </div>

      <div class="form-actions">
        <button type="button" class="btn btn-primary preference-save" :disabled="saving" @click="emit('save')">
          <svg v-if="saving" viewBox="0 0 24 24" fill="currentColor" width="14" height="14" class="spin-slow">
            <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
          </svg>
          <span>{{ saving ? labels.saving : labels.savePreferences }}</span>
        </button>
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

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
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

.form-actions {
  display: flex;
  align-items: center;
}

.preference-save {
  height: 36px;
  padding: 0 18px;
}

@media (max-width: 640px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
