<script setup lang="ts">
import { computed, onMounted, shallowRef } from 'vue'
import * as api from '@/api/library'
import AuthPanel from '@/components/auth/AuthPanel.vue'
import AppSidebar, { type NavigationItem } from '@/components/app/AppSidebar.vue'
import LibraryWorkspace from '@/components/library/LibraryWorkspace.vue'
import SettingsPage from '@/components/settings/SettingsPage.vue'
import { useLocale } from '@/composables/useLocale'

const mode = shallowRef<'loading' | 'setup' | 'login' | 'library'>('loading')
const session = shallowRef<api.Session | null>(null)
const error = shallowRef<string | null>(null)
const activeSection = shallowRef<'movies' | 'shows' | 'sources' | 'jobs' | 'settings'>('movies')
const { locale, t, setLocale, toggleLocale } = useLocale()

const navigationItems = computed<NavigationItem[]>(() => [
  { id: 'movies', label: t.value.movies, icon: 'movie' },
  { id: 'shows', label: t.value.tvShows, icon: 'tv' },
  { id: 'sources', label: t.value.sources, icon: 'sources' },
  { id: 'jobs', label: t.value.jobs, icon: 'jobs' },
  { id: 'settings', label: t.value.settings, icon: 'settings' }
])

async function initialize() {
  try {
    const status = await api.setupStatus()
    if (status.needsSetup) { mode.value = 'setup'; return }
    session.value = await api.session()
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
    mode.value = 'library'
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : t.value.errorSignIn
  }
}

onMounted(initialize)
</script>

<template>
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
      @select-section="activeSection = $event as any"
      @toggle-locale="toggleLocale"
    />

    <!-- 2. Main Content Area Offset by 64px -->
    <div class="app-main-content">
      <!-- Library Workspace (Movies / TV Shows) -->
      <LibraryWorkspace
        v-if="activeSection === 'movies' || activeSection === 'shows'"
        :media-kind="activeSection"
        :csrf-token="session.csrfToken"
        :username="session.user.username"
        :labels="t"
        @toggle-locale="toggleLocale"
      />

      <!-- Settings / Sources Page -->
      <SettingsPage
        v-else
        :csrf-token="session.csrfToken"
        :locale="locale"
        :labels="t"
        @change-locale="setLocale"
      />
    </div>
  </div>
</template>

<style scoped>
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
@media (max-width: 700px) {
  .app-shell {
    display: block;
    height: auto;
    overflow-y: auto;
  }

  .app-main-content {
    margin-left: 0;
    height: auto;
    min-height: 100vh;
  }
}
</style>
