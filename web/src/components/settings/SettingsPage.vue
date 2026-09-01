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
const sourceName = shallowRef('Media')
const sourcePath = shallowRef('/media')
const sourceFeedback = shallowRef<{ kind: 'success' | 'error'; message: string } | null>(null)
const isSaving = shallowRef(false)
const { sourceItems, refresh, createSource, removeSource, scan } = useLibrary(() => props.csrfToken)
const { settings, error, isLoading, providerForm, load, saveProvider } = useSettings(() => props.csrfToken)

async function initialize() { await Promise.all([load(), refresh()]) }
async function save() { isSaving.value = true; try { await saveProvider() } finally { isSaving.value = false } }
function sourceErrorMessage(error: unknown): string {
  const message = error instanceof Error ? error.message : ''
  if (message.includes('outside configured media roots')) return props.labels.errorSourceOutsideRoots
  if (message.includes('must be an accessible directory')) return props.labels.errorSourceMissing
  if (message.includes('UNIQUE constraint failed')) return props.labels.errorSourceExists
  return message || props.labels.errorAddSource
}
async function addSource() {
  sourceFeedback.value = null
  try {
    await createSource(sourceName.value, sourcePath.value)
    sourceFeedback.value = { kind: 'success', message: props.labels.sourceAdded }
  } catch (caught) {
    sourceFeedback.value = { kind: 'error', message: sourceErrorMessage(caught) }
  }
}
async function deleteSource(id: number) {
  if (!window.confirm(props.labels.confirmDeleteSource)) return
  sourceFeedback.value = null
  try {
    await removeSource(id)
    sourceFeedback.value = { kind: 'success', message: props.labels.sourceDeleted }
  } catch (caught) {
    sourceFeedback.value = { kind: 'error', message: sourceErrorMessage(caught) }
  }
}
onMounted(initialize)
</script>

<template>
  <main class="settings-page">
    <div class="settings-container">
      <header class="settings-header">
        <div class="settings-header__info">
          <p class="eyebrow">{{ labels.settingsEyebrow }}</p>
          <h1 class="settings-title">{{ labels.settings }}</h1>
          <p class="settings-intro">{{ labels.settingsIntro }}</p>
        </div>
        <button
          type="button"
          class="btn btn-ghost btn-sm locale-toggle"
          @click="emit('changeLocale', locale === 'en' ? 'zh-CN' : 'en')"
        >
          {{ labels.language }}
        </button>
      </header>

      <div v-if="error" class="settings-error" role="alert">
        <span class="status-dot status-dot--error"></span>
        <span>{{ error }}</span>
      </div>

      <div v-if="isLoading" class="settings-loading">
        <span>{{ labels.loading }}</span>
      </div>

      <div v-else class="settings-grid">
        <div class="settings-col">
          <SettingsProviderForm
            v-model:model="providerForm"
            :settings="settings"
            :labels="labels"
            :saving="isSaving"
            @save="save"
          />
        </div>
        <div class="settings-col">
          <SettingsSources
            v-model:source-name="sourceName"
            v-model:source-path="sourcePath"
            :sources="sourceItems"
            :media-roots="settings?.mediaRoots ?? []"
            :labels="labels"
            :feedback="sourceFeedback"
            @add="addSource"
            @scan="scan"
            @delete="deleteSource"
          />
          <SettingsInterfaceForm
            :locale="locale"
            :labels="labels"
            @change-locale="emit('changeLocale', $event)"
          />
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.settings-page {
  flex: 1;
  width: 100%;
  height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  -webkit-overflow-scrolling: touch;
  background: var(--surface-base, #0c1324);
  box-sizing: border-box;
}

.settings-container {
  width: 100%;
  max-width: 1360px;
  margin: 0 auto;
  padding: clamp(1rem, 2vw, 2.5rem);
  box-sizing: border-box;
}

.settings-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  padding-bottom: 1.25rem;
  border-bottom: 1px solid var(--border-subtle);
}

.settings-header__info {
  display: grid;
  gap: 0.25rem;
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

.settings-title {
  margin: 0;
  font-family: var(--font-display);
  font-size: 1.375rem;
  font-weight: 700;
  color: var(--on-surface);
  letter-spacing: -0.02em;
  line-height: 1.25;
}

.settings-intro {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--on-surface-variant);
  line-height: 1.5;
}

.locale-toggle {
  flex-shrink: 0;
}

.settings-error {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  margin-top: 1.25rem;
  padding: 0.75rem 1rem;
  border-radius: var(--radius-sm);
  background: var(--surface-container-low);
  border: 1px solid var(--error-container);
  color: var(--error);
  font-size: 0.8125rem;
}

.settings-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 3rem 1rem;
  color: var(--outline);
  font-size: 0.875rem;
}

.settings-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1.5rem;
  margin-top: 1.5rem;
  align-items: start;
}

.settings-col {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  min-width: 0;
}

@media (min-width: 960px) {
  .settings-grid {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  }
}

@media (max-width: 768px) {
  .settings-container {
    padding-bottom: calc(84px + env(safe-area-inset-bottom, 0px));
  }
}
</style>
