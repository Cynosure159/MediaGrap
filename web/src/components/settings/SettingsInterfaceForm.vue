<script setup lang="ts">
import type { Locale } from '@/composables/useLocale'

defineProps<{ locale: Locale; labels: Record<string, string> }>()
const emit = defineEmits<{ changeLocale: [locale: Locale] }>()
</script>

<template>
  <section class="settings-card">
    <div class="section-heading"><p class="eyebrow">Interface</p><h2>{{ labels.interfaceSettings }}</h2></div>
    <label class="locale-label">{{ labels.interfaceLanguage }}<select :value="locale" @change="emit('changeLocale', ($event.target as HTMLSelectElement).value as Locale)"><option value="en">English</option><option value="zh-CN">简体中文</option></select></label>
  </section>
</template>

<style scoped>
.settings-card{padding:clamp(1.2rem,3vw,2rem);border:1px solid var(--mist-300);border-radius:1rem;background:var(--paper)}.section-heading h2{margin:.35rem 0 1.2rem;font:500 2rem var(--font-display);letter-spacing:-.04em}.locale-label{display:grid;gap:.35rem;max-width:20rem;font-size:.82rem;font-weight:700}.locale-label select{min-height:2.6rem;padding:.5rem .7rem;border:1px solid var(--mist-300);border-radius:.4rem;background:white;font:inherit}
</style>
