<script setup lang="ts">
import { computed, onMounted, shallowRef } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import * as api from '@/api/library'
import AuthPanel from '@/components/auth/AuthPanel.vue'
import AppSidebar, { type NavigationItem } from '@/components/app/AppSidebar.vue'
import { useLocale } from '@/composables/useLocale'
import { useTheme } from '@/composables/useTheme'
import type { MainSection } from '@/router'

const mode = shallowRef<'loading' | 'setup' | 'login' | 'library'>('loading')
const session = shallowRef<api.Session | null>(null)
const error = shallowRef<string | null>(null)
const { locale, t, setLocale, toggleLocale } = useLocale()
const { theme, setTheme } = useTheme()
const route = useRoute()
const router = useRouter()

const activeSection = computed<MainSection>(() => {
  const section = route.meta.section
  return section === 'shows' || section === 'jobs' || section === 'settings' ? section : 'movies'
})

async function applyServerPreferences() {
	try {
		const preferences = await api.settings()
		setLocale(preferences.locale)
		setTheme(preferences.theme)
	} catch {
		// Local preferences remain usable if the settings snapshot is temporarily unavailable.
	}
}

const navigationItems = computed<NavigationItem[]>(() => [
  { id: 'movies', label: t.value.movies, icon: 'movie' },
  { id: 'shows', label: t.value.tvShows, icon: 'tv' },
  { id: 'jobs', label: t.value.jobs, icon: 'jobs' },
  { id: 'settings', label: t.value.settings, icon: 'settings' }
])

function selectSection(id: string) {
  if (id === 'movies' || id === 'shows' || id === 'jobs' || id === 'settings') {
    void router.push({ name: id })
  }
}

async function initialize() {
  try {
    const status = await api.setupStatus()
    if (status.needsSetup) { mode.value = 'setup'; return }
	session.value = await api.session()
	await applyServerPreferences()
    mode.value = 'library'
  } catch {
    mode.value = 'login'
  }
}

async function authenticate(username: string, password: string) {
  error.value = null
  try {
    session.value = mode.value === 'setup'
      ? await api.setup(username, password)
      : await api.signIn(username, password)
	await applyServerPreferences()
    mode.value = 'library'
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : t.value.errorSignIn
  }
}

onMounted(initialize)
</script>

<template>
  <a class="source-download" href="/source" download>{{ locale === 'zh-CN' ? '源码 · AGPLv3' : 'Source · AGPLv3' }}</a>
  <!-- Loading Screen -->
  <div v-if="mode === 'loading'" class="loading-screen">
    <img src="/logo-icon.svg" alt="MediaGrap" class="loading-logo" />
    <span class="loading-txt">{{ t.opening }}</span>
  </div>

  <!-- Auth Panel -->
  <AuthPanel
    v-else-if="mode === 'setup' || mode === 'login'"
    :setup="mode === 'setup'"
    :error="error"
    :labels="t"
    @submit="authenticate"
    @toggle-locale="toggleLocale"
  />

  <!-- Main App Shell -->
  <div v-else-if="session" class="app-shell">
    <!-- 1. Left Activity Rail (64px) -->
    <AppSidebar
      :items="navigationItems"
      :active-section="activeSection"
      :username="session.user.username"
      :labels="t"
      @select-section="selectSection"
      @toggle-locale="toggleLocale"
    />

    <!-- 2. Main Content Area Offset by 64px -->
    <div class="app-main-content">
      <RouterView v-slot="{ Component }">
        <component
          :is="Component"
          :csrf-token="session.csrfToken"
          :username="session.user.username"
          :labels="t"
          :locale="locale"
          :theme="theme"
          @toggle-locale="toggleLocale"
          @change-locale="setLocale"
          @change-theme="setTheme"
        />
      </RouterView>
    </div>
  </div>
</template>

<style scoped>
.source-download {
  position: fixed;
  right: 8px;
  bottom: 2px;
  z-index: 100;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 4px;
  color: var(--on-surface-variant, #c7c4d7);
  background: var(--surface-container-lowest, #070d1f);
}

.loading-screen {
  width: 100vw;
  height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: var(--surface-base, #0c1324);
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 13px;
}

.loading-logo {
  width: 48px;
  height: 48px;
  object-fit: contain;
  animation: pulse-fade 1.5s ease-in-out infinite;
}

@keyframes pulse-fade {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.96); }
}

.app-shell {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: var(--surface-base, #0c1324);
  display: flex;
}

.app-main-content {
  margin-left: var(--sidebar-width, 64px);
  flex: 1;
  height: 100vh;
  min-width: 0;
  overflow: hidden;
  display: flex;
}

/* ── Mobile Layout ────────────────────────────────────────── */
@media (max-width: 768px) {
  .app-shell {
    display: flex;
    flex-direction: column;
    height: 100vh;
    width: 100vw;
    max-width: 100vw;
    overflow: hidden;
    box-sizing: border-box;
  }

  .app-main-content {
    margin-left: 0;
    flex: 1;
    height: 100%;
    min-height: 0;
    width: 100%;
    max-width: 100vw;
    overflow: hidden;
    box-sizing: border-box;
  }
}
</style>
