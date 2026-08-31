<script setup lang="ts">
import { onMounted, shallowRef } from 'vue'
import { useLibrary } from '@/composables/useLibrary'
import { useSettings } from '@/composables/useSettings'
import type { Locale } from '@/composables/useLocale'
import SettingsInterfaceForm from './SettingsInterfaceForm.vue'
import SettingsProviderForm from './SettingsProviderForm.vue'
import SettingsSources from './SettingsSources.vue'

const props = defineProps<{ csrfToken: string; labels: Record<string, string>; locale: Locale }>()
const emit = defineEmits<{ changeLocale: [locale: Locale] }>()
const sourceName = shallowRef('Movies')
const sourcePath = shallowRef('/media/movies')
const isSaving = shallowRef(false)
const { sourceItems, refresh, createSource, scan } = useLibrary(() => props.csrfToken)
const { settings, error, isLoading, providerForm, load, saveProvider } = useSettings(() => props.csrfToken)

async function initialize() { await Promise.all([load(), refresh()]) }
async function save() { isSaving.value = true; try { await saveProvider() } finally { isSaving.value = false } }
async function addSource() { await createSource(sourceName.value, sourcePath.value) }
onMounted(initialize)
</script>

<template>
  <main class="settings-page"><header class="settings-header"><p class="eyebrow">MediaGrap · control room</p><h1>{{ labels.settings }}</h1><p>{{ labels.settingsIntro }}</p></header><p v-if="error" class="settings-error">{{ error }}</p><div v-if="isLoading" class="settings-loading">{{ labels.loading }}</div><div v-else class="settings-stack"><SettingsProviderForm v-model:model="providerForm" :settings="settings" :labels="labels" :saving="isSaving" @save="save" /><SettingsSources v-model:source-name="sourceName" v-model:source-path="sourcePath" :sources="sourceItems" :media-roots="settings?.mediaRoots ?? []" :labels="labels" @add="addSource" @scan="scan" /><SettingsInterfaceForm :locale="locale" :labels="labels" @change-locale="emit('changeLocale', $event)" /></div></main>
</template>

<style scoped>
.settings-page{max-width:68rem;margin:auto;padding:clamp(1.25rem,3vw,3rem)}.settings-header{padding-bottom:1.6rem;border-bottom:1px solid var(--mist-300)}.settings-header h1{margin:.35rem 0;font:500 clamp(2.5rem,6vw,5rem)/.9 var(--font-display);letter-spacing:-.07em}.settings-header p{color:var(--ink-600)}.settings-stack{display:grid;gap:1rem;margin-top:1.3rem}.settings-error{padding:.7rem;border-radius:.4rem;background:var(--water-100);color:var(--ink-800)}.settings-loading{padding:2rem;color:var(--ink-600)}
</style>
