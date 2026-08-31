<script setup lang="ts">
import type { Locale } from '@/composables/useLocale'

defineProps<{ locale: Locale; labels: Record<string, string> }>()
const emit = defineEmits<{ changeLocale: [locale: Locale] }>()
</script>

<template>
  <section class="settings-card">
    <div class="card-header">
      <p class="eyebrow">{{ labels.interfaceEyebrow }}</p>
      <h2 class="section-title">{{ labels.interfaceSettings }}</h2>
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

.locale-group {
  display: grid;
  gap: 0.4rem;
  max-width: 20rem;
}

.field-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.select-input {
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

.select-input:focus {
  outline: none;
  border-color: var(--border-focus);
  box-shadow: 0 0 0 2px rgba(128, 131, 255, 0.2);
}
</style>
