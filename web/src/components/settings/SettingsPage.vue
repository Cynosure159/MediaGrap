<script setup lang="ts">
import { computed, onMounted, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useLibrary } from '@/composables/useLibrary'
import { useSettings } from '@/composables/useSettings'
import type { Locale } from '@/composables/useLocale'
import type { Theme } from '@/composables/useTheme'
import SettingsConsoleSidebar, { type SettingsCategory } from './SettingsConsoleSidebar.vue'
import SettingsStatusSidebar from './SettingsStatusSidebar.vue'
import SettingsProviderForm from './SettingsProviderForm.vue'
import SettingsSources from './SettingsSources.vue'
import SettingsNetworkForm from './SettingsNetworkForm.vue'
import SettingsInterfaceForm from './SettingsInterfaceForm.vue'
import SettingsIntegrations from './SettingsIntegrations.vue'
import AutomationApprovals from './AutomationApprovals.vue'
import SettingsSystemInfo from './SettingsSystemInfo.vue'

const props = defineProps<{
  csrfToken: string
  labels: Record<string, string>
  locale: Locale
  theme: Theme
}>()

const emit = defineEmits<{
  changeLocale: [locale: Locale]
  changeTheme: [theme: Theme]
}>()

const route = useRoute()
const router = useRouter()

const validCategories: SettingsCategory[] = [
  'sources',
  'providers',
  'network',
  'interface',
  'integrations',
  'automation',
  'system',
]

const initialCategory = computed<SettingsCategory>(() => {
  const queryCat = route.query.section as SettingsCategory
  if (queryCat && validCategories.includes(queryCat)) return queryCat
  if (route.query.approval) return 'automation'
  return 'sources'
})

const activeCategory = shallowRef<SettingsCategory>(initialCategory.value)

watch(
  () => route.query.section,
  (newSec) => {
    if (newSec && validCategories.includes(newSec as SettingsCategory)) {
      activeCategory.value = newSec as SettingsCategory
    }
  },
)

function setCategory(cat: SettingsCategory) {
  activeCategory.value = cat
  void router.replace({ query: { ...route.query, section: cat } })
}

const sourceName = shallowRef('')
const sourcePath = shallowRef('')
const sourceFeedback = shallowRef<{ kind: 'success' | 'error'; message: string } | null>(null)
const isSaving = shallowRef(false)

const { sourceItems, refresh: refreshLibrary, createSource, removeSource, saveSourcePolicy, scan } = useLibrary(() => props.csrfToken)
const { settings, error, isLoading, providerForm, connectionTests, testingTarget, load: loadSettings, saveProvider, runConnectionTest } = useSettings(() => props.csrfToken)

const zh = computed(() => props.locale === 'zh-CN')

async function initialize() {
  await Promise.all([loadSettings(), refreshLibrary()])
}

async function save() {
  isSaving.value = true
  try {
    await saveProvider()
  } finally {
    isSaving.value = false
  }
}

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
    sourceName.value = ''
    sourcePath.value = ''
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

const categoryTitles: Record<SettingsCategory, { zh: string; en: string }> = {
  sources: { zh: '媒体源与挂载', en: 'Library Sources' },
  providers: { zh: '元数据刮削源', en: 'Metadata Providers' },
  network: { zh: '网络与代理', en: 'Network & Proxy' },
  interface: { zh: '界面与偏好', en: 'Interface & Theme' },
  integrations: { zh: '外部集成与 API', en: 'Integrations & API' },
  automation: { zh: '自动化安全审批', en: 'File Approvals' },
  system: { zh: '系统与运行环境', en: 'System & Docker' },
}

const currentCategoryTitle = computed(() => {
  const item = categoryTitles[activeCategory.value]
  return zh.value ? item.zh : item.en
})

onMounted(initialize)
</script>

<template>
  <div class="settings-console-layout">
    <!-- 1. Secondary 260px Navigation Sidebar -->
    <SettingsConsoleSidebar
      :active-category="activeCategory"
      :source-count="sourceItems.length"
      :locale="locale"
      :labels="labels"
      @select-category="setCategory"
    />

    <!-- 2. Main Content Canvas -->
    <main class="settings-main-canvas">
      <!-- Top Toolbar Bar -->
      <header class="console-top-toolbar">
        <div class="toolbar-breadcrumb">
          <span class="crumb-root">{{ zh ? '控制台' : 'Settings Console' }}</span>
          <span class="crumb-sep">/</span>
          <span class="crumb-current">{{ currentCategoryTitle }}</span>
        </div>

        <div class="toolbar-actions">
          <button
            type="button"
            class="btn btn-ghost btn-sm"
            :disabled="isLoading"
            :title="zh ? '刷新设置与状态' : 'Refresh settings & status'"
            @click="initialize"
          >
            <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14" :class="{ 'spin-slow': isLoading }">
              <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
            </svg>
            <span>{{ zh ? '刷新' : 'Refresh' }}</span>
          </button>
        </div>
      </header>

      <!-- Scrollable Canvas Area -->
      <div class="console-body-scroll">
        <div v-if="error" class="settings-error" role="alert">
          <span class="dot dot-err"></span>
          <span>{{ error }}</span>
        </div>

        <div v-if="isLoading && !settings" class="settings-loading">
          <span>{{ labels.loading }}</span>
        </div>

        <div v-else class="console-workspace-grid">
          <!-- Left Main Config Section -->
          <div class="workspace-main-column">
            <!-- 1. Sources & Storage -->
            <SettingsSources
              v-if="activeCategory === 'sources'"
              v-model:source-name="sourceName"
              v-model:source-path="sourcePath"
              :sources="sourceItems"
              :media-roots="settings?.mediaRoots ?? []"
              :labels="labels"
              :feedback="sourceFeedback"
              @add="addSource"
              @scan="scan"
              @delete="deleteSource"
              @save-policy="saveSourcePolicy"
            />

            <!-- 2. Providers Form -->
            <SettingsProviderForm
              v-else-if="activeCategory === 'providers'"
              v-model:model="providerForm"
              :settings="settings"
              :labels="labels"
              :saving="isSaving"
              :connection-tests="connectionTests"
              :testing-target="testingTarget"
              @save="save"
              @test="runConnectionTest"
            />

            <!-- 3. Network & Proxy Form -->
            <SettingsNetworkForm
              v-else-if="activeCategory === 'network'"
              v-model:model="providerForm"
              :settings="settings"
              :labels="labels"
              :saving="isSaving"
              :connection-tests="connectionTests"
              :testing-target="testingTarget"
              @save="save"
              @test="runConnectionTest"
            />

            <!-- 4. Interface Form -->
            <SettingsInterfaceForm
              v-else-if="activeCategory === 'interface'"
              :locale="locale"
              :labels="labels"
              v-model:model="providerForm"
              :saving="isSaving"
              @change-locale="emit('changeLocale', $event)"
              @change-theme="emit('changeTheme', $event)"
              @save="save"
            />

            <!-- 5. Integrations & API -->
            <SettingsIntegrations
              v-else-if="activeCategory === 'integrations'"
              :csrf-token="csrfToken"
              :sources="sourceItems"
              :locale="locale"
            />

            <!-- 6. Automation Approvals -->
            <AutomationApprovals
              v-else-if="activeCategory === 'automation'"
              :csrf-token="csrfToken"
              :locale="locale"
            />

            <!-- 7. System & Environment -->
            <SettingsSystemInfo
              v-else-if="activeCategory === 'system'"
              :locale="locale"
              :labels="labels"
            />
          </div>

          <!-- Right Status & Diagnostic Panel -->
          <div class="workspace-side-column">
            <SettingsStatusSidebar
              :settings="settings"
              :sources="sourceItems"
              :connection-tests="connectionTests"
              :testing-target="testingTarget"
              :locale="locale"
              :labels="labels"
              @test="runConnectionTest"
            />
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
.settings-console-layout {
  display: flex;
  width: 100%;
  height: 100%;
  background: var(--surface-base, #0c1324);
  overflow: hidden;
  box-sizing: border-box;
}

.settings-main-canvas {
  flex: 1;
  min-width: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--surface-base, #0c1324);
  overflow: hidden;
}

.console-top-toolbar {
  height: var(--toolbar-height, 40px);
  min-height: var(--toolbar-height, 40px);
  padding: 0 20px;
  background: var(--surface-container, #191f31);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  z-index: 10;
}

.toolbar-breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.crumb-root {
  color: var(--outline, #908fa0);
}

.crumb-sep {
  color: var(--outline-variant, #2e3447);
}

.crumb-current {
  color: var(--on-surface, #dce1fb);
  font-weight: 600;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.console-body-scroll {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  -webkit-overflow-scrolling: touch;
  padding: 24px;
}

.settings-error {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding: 10px 14px;
  border-radius: var(--radius-sm, 0.25rem);
  background: rgba(244, 63, 94, 0.1);
  border: 1px solid rgba(244, 63, 94, 0.3);
  color: var(--error, #ffb4ab);
  font-size: 13px;
}

.settings-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 0;
  color: var(--outline, #908fa0);
  font-size: 13px;
}

.console-workspace-grid {
  display: flex;
  gap: 24px;
  align-items: flex-start;
  max-width: 1440px;
  margin: 0 auto;
}

.workspace-main-column {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.workspace-side-column {
  width: 320px;
  flex-shrink: 0;
}

@media (max-width: 1100px) {
  .console-workspace-grid {
    flex-direction: column;
  }

  .workspace-side-column {
    width: 100%;
  }
}

@media (max-width: 860px) {
  .settings-console-layout {
    flex-direction: column;
  }

  .console-body-scroll {
    padding: 16px;
  }
}
</style>
