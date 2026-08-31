<script setup lang="ts">
import { onMounted, shallowRef } from 'vue'
import AppSidebar, { type NavigationItem } from '@/components/app/AppSidebar.vue'
import ActivityRail from '@/components/dashboard/ActivityRail.vue'
import SystemPulse from '@/components/dashboard/SystemPulse.vue'
import { useSystemSummary } from '@/composables/useSystemSummary'

const navigation: NavigationItem[] = [
  { id: 'overview', label: 'Overview', detail: 'Foundation status' },
  { id: 'library', label: 'Library', detail: 'Coming in phase 1', disabled: true },
  { id: 'jobs', label: 'Jobs', detail: 'Coming in phase 1', disabled: true },
  { id: 'settings', label: 'Settings', detail: 'Coming in phase 1', disabled: true }
]

const activeSection = shallowRef('overview')
const { summary, isLoading, error, connectionLabel, refresh } = useSystemSummary()

function selectSection(id: string) {
  activeSection.value = id
}

onMounted(refresh)
</script>

<template>
  <main class="application-shell">
    <AppSidebar
      :active-section="activeSection"
      :items="navigation"
      @select-section="selectSection"
    />

    <section class="workspace" aria-labelledby="page-title">
      <header class="workspace-header">
        <div>
          <p class="eyebrow">Media library control room</p>
          <h1 id="page-title" class="page-title">Foundation</h1>
          <p class="page-description">The service is ready for its first media source.</p>
        </div>
        <button class="refresh-button" type="button" :disabled="isLoading" @click="refresh">
          {{ isLoading ? 'Checking…' : 'Refresh status' }}
        </button>
      </header>

      <SystemPulse
        :summary="summary"
        :is-loading="isLoading"
        :error="error"
        :connection-label="connectionLabel"
        @refresh="refresh"
      />

      <ActivityRail :summary="summary" />
    </section>
  </main>
</template>

