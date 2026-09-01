<script setup lang="ts">
import { computed, onMounted, shallowRef } from 'vue'
import { useLibrary } from '@/composables/useLibrary'
import MediaCatalog from './MediaCatalog.vue'
import MovieInspector from './MovieInspector.vue'
import TVShowCatalog from './TVShowCatalog.vue'
import TVShowInspector from './TVShowInspector.vue'
import type { TVSelection } from '@/api/library'

const props = defineProps<{
  csrfToken: string
  username: string
  labels: Record<string, string>
  mediaKind?: 'movies' | 'shows'
}>()

const emit = defineEmits<{
  toggleLocale: []
}>()

const query = shallowRef('')
const selectedMovieId = shallowRef<number | null>(null)
const selectedTVSelection = shallowRef<TVSelection | null>(null)

const { sourceItems, mediaItems, tvShowItems, jobItems, error, hasSources, refresh, scan: queueScan } = useLibrary(() => props.csrfToken)
const activeJobs = computed(() => jobItems.value.filter(job => job.state === 'queued' || job.state === 'running'))

async function search(value = '') {
  query.value = value
  await refresh(query.value)
}

async function scanActiveSource() {
  const source = sourceItems.value[0]
  if (source) await queueScan(source.id, query.value)
}

onMounted(async () => {
  await refresh()
})
</script>

<template>
  <div class="workspace-shell">
    <!-- If no sources exist -->
    <div v-if="!hasSources" class="source-onboarding">
      <div class="onboarding-card">
        <svg class="onboard-icon" viewBox="0 0 24 24" fill="currentColor">
          <path d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"/>
        </svg>
        <h2>{{ labels.firstSource || 'Add your first media folder' }}</h2>
        <p>{{ labels.openSettingsForSource || 'Go to Settings to configure mounted media directories.' }}</p>
      </div>
    </div>

    <!-- Main Workspace Split Pane -->
    <div v-else class="split-pane-layout" :class="{ 'mobile-show-detail': (props.mediaKind === 'shows' ? selectedTVSelection : selectedMovieId) !== null }">
      <!-- Movies Mode -->
      <template v-if="props.mediaKind !== 'shows'">
        <MediaCatalog
          :items="mediaItems"
          :selected-id="selectedMovieId"
          :active-job="activeJobs[0]"
          :labels="labels"
          @search="search"
          @select="selectedMovieId = $event"
          @scan="scanActiveSource"
        />
        <MovieInspector
          :item-id="selectedMovieId"
          :csrf-token="csrfToken"
          :labels="labels"
          @close="selectedMovieId = null"
          @metadata-saved="refresh(query)"
        />
      </template>

      <!-- TV Shows Mode -->
      <template v-else>
        <TVShowCatalog
          :items="tvShowItems"
          :selected="selectedTVSelection"
          :active-job="activeJobs[0]"
          :labels="labels"
          @search="search"
          @select="selectedTVSelection = $event"
          @scan="scanActiveSource"
        />
        <TVShowInspector
          :selection="selectedTVSelection"
          :csrf-token="csrfToken"
          :labels="labels"
          @close="selectedTVSelection = null"
        />
      </template>
    </div>
  </div>
</template>

<style scoped>
.workspace-shell {
  width: 100%;
  height: 100vh;
  overflow: hidden;
  background: var(--surface-base, #0c1324);
  display: flex;
}

.split-pane-layout {
  display: flex;
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.source-onboarding {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.onboarding-card {
  max-width: 480px;
  text-align: center;
  padding: 3rem 2rem;
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-xl, 0.75rem);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.onboard-icon {
  width: 48px;
  height: 48px;
  color: var(--tertiary, #ffb95f);
  opacity: 0.8;
}

.onboarding-card h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.onboarding-card p {
  margin: 0;
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 13px;
  line-height: 1.6;
}

/* ── Mobile Responsive Logic ──────────────────────────────── */
@media (max-width: 760px) {
  .split-pane-layout {
    display: block;
    height: auto;
    min-height: 100vh;
    padding-bottom: 60px;
  }

  .split-pane-layout.mobile-show-detail :deep(.catalog-panel) {
    display: none;
  }

  .split-pane-layout:not(.mobile-show-detail) :deep(.inspector-workspace) {
    display: none;
  }
}
</style>
