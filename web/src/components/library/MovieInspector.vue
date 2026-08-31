<script setup lang="ts">
import { computed, reactive, shallowRef, watch } from 'vue'
import * as api from '@/api/library'
import MatchCandidates from './MatchCandidates.vue'
import NfoPreview from './NfoPreview.vue'
import InspectorToolbar, { type InspectorTab } from './inspector/InspectorToolbar.vue'
import MovieOverviewTab, { type MovieDraft } from './inspector/MovieOverviewTab.vue'
import MovieArtworkTab from './inspector/MovieArtworkTab.vue'
import MovieCastTab, { type CastMember } from './inspector/MovieCastTab.vue'
import MovieNfoTab from './inspector/MovieNfoTab.vue'
import MovieFileAuditTab from './inspector/MovieFileAuditTab.vue'

const props = defineProps<{
  itemId: number | null
  csrfToken: string
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  close: []
  metadataSaved: []
}>()

const detail = shallowRef<{
  item: api.MediaItem
  metadata: api.Metadata
  metadataOrigin: 'draft' | 'nfo' | 'empty'
  writable: boolean
} | null>(null)

const candidatesList = shallowRef<api.Candidate[]>([])
const activeTab = shallowRef<InspectorTab>('overview')
const showCandidates = shallowRef(false)
const showNfoPreview = shallowRef(false)
const writePlan = shallowRef<api.WritePlan | null>(null)
const isSaving = shallowRef(false)
const isLoading = shallowRef(false)
const isScraping = shallowRef(false)
const error = shallowRef<string | null>(null)
const isLocked = shallowRef(false)

const draft = reactive<MovieDraft>({
  title: '',
  originalTitle: '',
  year: null,
  overview: '',
  runtimeMinutes: null,
  genres: [],
  posterUrl: '',
  backdropUrl: '',
  director: '',
  writers: '',
  studio: '',
  rating: null,
  votes: null,
  contentRating: '',
})

const castList = computed<CastMember[]>(() =>
  draft.director
    ? [{ name: draft.director, role: props.labels.director || 'Director', avatar: draft.posterUrl, id: 'director' }]
    : []
)

const nfoXmlContent = computed(() => {
  const genresXml = draft.genres.map(g => `    <genre>${g}</genre>`).join('\n')
  return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<movie>
    <title>${draft.title}</title>
    <originaltitle>${draft.originalTitle}</originaltitle>
    <year>${draft.year ?? ''}</year>
    <rating>${draft.rating ?? ''}</rating>
    <votes>${draft.votes ?? ''}</votes>
    <plot>${draft.overview}</plot>
    <runtime>${draft.runtimeMinutes ?? ''}</runtime>
    <mpaa>${draft.contentRating}</mpaa>
    <director>${draft.director}</director>
    <uniqueid type="tmdb" default="true">${detail.value?.metadata.providerId || ''}</uniqueid>
${genresXml}
</movie>`
})

function applyMetadataToDraft(meta: Partial<api.Metadata>, fallbackTitle = '', fallbackYear: number | null = null) {
  draft.title = meta.title || fallbackTitle
  draft.originalTitle = meta.originalTitle || fallbackTitle
  draft.year = meta.year ?? fallbackYear
  draft.overview = meta.overview || ''
  draft.runtimeMinutes = meta.runtimeMinutes ?? null
  draft.genres = [...(meta.genres || [])]
  draft.posterUrl = meta.posterUrl || ''
  draft.backdropUrl = meta.backdropUrl || ''
}

function buildMetadataPayload(): api.Metadata {
  return {
    mediaItemId: props.itemId ?? 0,
    provider: detail.value?.metadata.provider || 'tmdb',
    providerId: detail.value?.metadata.providerId || '',
    title: draft.title,
    originalTitle: draft.originalTitle,
    year: draft.year,
    overview: draft.overview,
    runtimeMinutes: draft.runtimeMinutes,
    genres: draft.genres,
    posterUrl: draft.posterUrl,
    backdropUrl: draft.backdropUrl,
    lockedFields: [],
  }
}

async function loadDetail(id: number) {
  isLoading.value = true
  error.value = null
  try {
    const result = await api.mediaDetail(id)
    detail.value = result
    applyMetadataToDraft(result.metadata, result.item.titleHint, result.item.yearHint)
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : props.labels.errorLoadMedia
  } finally {
    isLoading.value = false
  }
}

watch(() => props.itemId, id => {
  if (id) {
    activeTab.value = 'overview'
    loadDetail(id)
  } else {
    detail.value = null
  }
}, { immediate: true })

async function triggerScrape() {
  if (!props.itemId) return
  isScraping.value = true
  error.value = null
  try {
    const res = await api.candidates(props.itemId)
    candidatesList.value = res.items
    showCandidates.value = true
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : props.labels.errorSearchCandidates
  } finally {
    isScraping.value = false
  }
}

async function handleCandidateSelect(candidate: api.Candidate) {
  if (!props.itemId) return
  try {
    const newMeta = await api.selectCandidate(props.csrfToken, props.itemId, candidate.id)
    applyMetadataToDraft(newMeta)
    showCandidates.value = false
    emit('metadataSaved')
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : props.labels.errorApplyCandidate
  }
}

async function saveMetadata() {
  if (!props.itemId || !detail.value) return
  isSaving.value = true
  error.value = null
  try {
    await api.saveMetadata(props.csrfToken, props.itemId, buildMetadataPayload())
    emit('metadataSaved')
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : props.labels.errorSaveMetadata
  } finally {
    isSaving.value = false
  }
}

async function triggerNfoPreview() {
  if (!props.itemId || !detail.value) return
  try {
    writePlan.value = await api.previewNfo(props.csrfToken, props.itemId, buildMetadataPayload())
    showNfoPreview.value = true
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : props.labels.errorPreviewNfo
  }
}

async function handleApplyNfo() {
  if (!writePlan.value) return
  try {
    await api.applyNfo(props.csrfToken, writePlan.value.id)
    showNfoPreview.value = false
    if (props.itemId) await loadDetail(props.itemId)
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : props.labels.errorWriteNfo
  }
}
</script>

<template>
  <main class="inspector-workspace" :aria-label="labels.details">
    <!-- Toolbar with Tabs & Quick Actions -->
    <InspectorToolbar
      :active-tab="activeTab"
      :has-detail="detail !== null"
      :is-writable="detail?.writable ?? false"
      :is-saving="isSaving"
      :is-scraping="isScraping"
      :is-locked="isLocked"
      :labels="labels"
      @select-tab="activeTab = $event"
      @save-nfo="triggerNfoPreview"
      @scrape="triggerScrape"
      @toggle-lock="isLocked = !isLocked"
      @close="emit('close')"
    />

    <!-- Error Alert -->
    <div v-if="error" class="inspector-error">
      {{ error }}
    </div>

    <!-- Read-Only Banner -->
    <div v-if="detail && !detail.writable" class="read-only-banner">
      {{ labels.readOnly }}
    </div>

    <!-- Empty State -->
    <div v-if="!itemId" class="no-selection-state">
      <svg class="no-sel-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <rect x="2" y="4" width="20" height="16" rx="2" />
        <path d="M8 4v16M16 4v16M2 12h20" />
      </svg>
      <p>{{ labels.noMatch }}</p>
    </div>

    <!-- Loading State -->
    <div v-else-if="isLoading" class="loading-state">
      <div class="loading-spinner"></div>
      <p>{{ labels.loading }}</p>
    </div>

    <!-- Tab Panels Container -->
    <div v-else-if="detail" class="inspector-content">
      <MovieOverviewTab
        v-if="activeTab === 'overview'"
        :draft="draft"
        :item="detail.item"
        :is-saving="isSaving"
        :labels="labels"
        @save-draft="saveMetadata"
      />

      <MovieArtworkTab
        v-else-if="activeTab === 'artwork'"
        :poster-url="draft.posterUrl"
        :backdrop-url="draft.backdropUrl"
        :labels="labels"
      />

      <MovieCastTab
        v-else-if="activeTab === 'cast'"
        :cast="castList"
        :labels="labels"
      />

      <MovieNfoTab
        v-else-if="activeTab === 'nfo'"
        :content="nfoXmlContent"
        :labels="labels"
        @save-to-nfo="triggerNfoPreview"
      />

      <MovieFileAuditTab
        v-else-if="activeTab === 'files'"
        :item="detail.item"
        :labels="labels"
      />
    </div>

    <!-- Match Candidates Modal -->
    <MatchCandidates
      v-if="showCandidates"
      :candidates="candidatesList"
      :busy="isScraping"
      :labels="labels"
      @select="handleCandidateSelect"
      @close="showCandidates = false"
    />

    <!-- NFO Preview Modal -->
    <NfoPreview
      v-if="showNfoPreview && writePlan"
      :plan="writePlan"
      :applying="isSaving"
      :labels="labels"
      @apply="handleApplyNfo"
      @close="showNfoPreview = false"
    />
  </main>
</template>

<style scoped>
.inspector-workspace {
  flex: 1;
  height: 100vh;
  background: var(--surface-base, #0c1324);
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.inspector-error {
  padding: 8px 12px;
  background: var(--error-container, #93000a);
  color: var(--on-error-container, #ffdad6);
  font-size: 12px;
}

.read-only-banner {
  padding: 6px 12px;
  background: var(--tertiary-container, #ca8100);
  color: var(--on-tertiary-container, #3e2400);
  font-size: 12px;
  font-weight: 600;
}

.no-selection-state,
.loading-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--outline, #908fa0);
  font-size: 13px;
}

.no-sel-icon {
  width: 48px;
  height: 48px;
  opacity: 0.3;
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--primary, #c0c1ff);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.inspector-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--pane-padding, 12px);
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
