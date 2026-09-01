<script setup lang="ts">
import type { Locale } from '@/composables/useLocale'

defineProps<{ locale: Locale; labels: Record<string, string> }>()
const emit = defineEmits<{ changeLocale: [locale: Locale] }>()
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
      </div>
    </div>

    <div class="card-body">
      <div class="field-group locale-group">
        <label for="interface-language" class="field-label">{{ labels.interfaceLanguage }}</label>
        <div class="select-wrapper">
          <select
            id="interface-language"
            class="form-select"
            :value="locale"
            @change="emit('changeLocale', ($event.target as HTMLSelectElement).value as Locale)"
          >
            <option value="zh-CN">简体中文 (zh-CN)</option>
            <option value="en">English (en)</option>
          </select>
          <svg class="select-chevron" viewBox="0 0 24 24" fill="currentColor">
            <path d="M7 10l5 5 5-5z"/>
          </svg>
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

.locale-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-width: 22rem;
}

.field-label {
  font-size: 12px;
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
</style>
