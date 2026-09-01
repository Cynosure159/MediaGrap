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
        <select
          id="interface-language"
          class="select-input"
          :value="locale"
          @change="emit('changeLocale', ($event.target as HTMLSelectElement).value as Locale)"
        >
          <option value="en">English</option>
          <option value="zh-CN">简体中文</option>
        </select>
      </div>
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

.locale-group {
  display: grid;
  gap: 0.4rem;
  max-width: 20rem;
}

.field-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--on-surface-variant);
}

.select-input {
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

.select-input:focus {
  outline: none;
  border-color: var(--primary-bright);
  box-shadow: 0 0 0 2px rgba(128, 131, 255, 0.2);
  background: var(--surface-container-lowest);
}
</style>
