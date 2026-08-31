<script setup lang="ts">
import type { Settings, SettingsUpdate } from '@/api/library'

defineProps<{ settings: Settings | null; labels: Record<string, string>; saving: boolean }>()
const model = defineModel<SettingsUpdate>('model', { required: true })
const emit = defineEmits<{ save: [] }>()
</script>

<template>
  <section class="settings-card">
    <div class="section-heading"><p class="eyebrow">TMDb</p><h2>{{ labels.providerSettings }}</h2></div>
    <form class="settings-form" @submit.prevent="emit('save')">
      <label>{{ labels.tmdbApiKey }}<input v-model="model.tmdbApiKey" type="password" autocomplete="off" :placeholder="settings?.tmdbApiKeyConfigured ? labels.apiKeyConfigured : labels.apiKeyPlaceholder" /></label>
      <label class="check-label"><input v-model="model.clearTmdbApiKey" type="checkbox" />{{ labels.clearApiKey }}</label>
      <label>{{ labels.tmdbLanguage }}<select v-model="model.tmdbLanguage"><option value="en-US">English (en-US)</option><option value="zh-CN">简体中文 (zh-CN)</option><option value="zh-TW">繁體中文 (zh-TW)</option><option value="ja-JP">日本語 (ja-JP)</option><option value="ko-KR">한국어 (ko-KR)</option></select></label>
      <label>{{ labels.outboundProxy }}<input v-model="model.outboundProxy" type="url" autocomplete="off" :placeholder="settings?.outboundProxyConfigured ? labels.proxyConfigured : 'http://proxy:7890'" /></label>
      <label class="check-label"><input v-model="model.clearOutboundProxy" type="checkbox" />{{ labels.clearProxy }}</label>
      <button type="submit" :disabled="saving">{{ saving ? labels.saving : labels.saveSettings }}</button>
    </form>
  </section>
</template>

<style scoped>
.settings-card{padding:clamp(1.2rem,3vw,2rem);border:1px solid var(--mist-300);border-radius:1rem;background:var(--paper)}.section-heading h2{margin:.35rem 0 1.2rem;font:500 2rem var(--font-display);letter-spacing:-.04em}.settings-form{display:grid;gap:.85rem;max-width:38rem}.settings-form label{display:grid;gap:.35rem;font-size:.82rem;font-weight:700}.settings-form input:not([type="checkbox"]),.settings-form select{min-height:2.6rem;padding:.5rem .7rem;border:1px solid var(--mist-300);border-radius:.4rem;background:white;font:inherit}.settings-form .check-label{display:flex;align-items:center;gap:.5rem;font-weight:500}.settings-form button{justify-self:start;min-height:2.6rem;padding:.5rem .9rem;border:1px solid var(--ink-700);border-radius:.45rem;background:var(--ink-900);color:white;font-weight:700;cursor:pointer}.settings-form button:disabled{opacity:.65;cursor:wait}
</style>
