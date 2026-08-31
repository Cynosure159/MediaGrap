<script setup lang="ts">
import { computed, reactive, shallowRef, watch } from 'vue'
import * as api from '@/api/library'
import MatchCandidates from './MatchCandidates.vue'
import NfoPreview from './NfoPreview.vue'

const props = defineProps<{
  itemId: number | null
  csrfToken: string
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  close: []
  metadataSaved: []
}>()

const detail = shallowRef<{ item: api.MediaItem; metadata: api.Metadata; metadataOrigin: 'draft' | 'nfo' | 'empty'; writable: boolean } | null>(null)
const candidatesList = shallowRef<api.Candidate[]>([])
const activeTab = shallowRef<'overview' | 'artwork' | 'cast' | 'nfo' | 'files'>('overview')
const showCandidates = shallowRef(false)
const showNfoPreview = shallowRef(false)
const writePlan = shallowRef<api.WritePlan | null>(null)
const isSaving = shallowRef(false)
const isLoading = shallowRef(false)
const isScraping = shallowRef(false)
const error = shallowRef<string | null>(null)
const isLocked = shallowRef(false)

const draft = reactive({
  title: '',
  originalTitle: '',
  year: null as number | null,
  overview: '',
  runtimeMinutes: null as number | null,
  genres: [] as string[],
  posterUrl: '',
  backdropUrl: '',
  director: '',
  writers: '',
  studio: '',
  rating: null as number | null,
  votes: null as number | null,
  contentRating: ''
})

const castList = computed(() => draft.director
  ? [{ name: draft.director, role: props.labels.director, avatar: draft.posterUrl, id: 'director' }]
  : [])

const genresInput = computed({
  get: () => draft.genres.join(', '),
  set: (val: string) => {
    draft.genres = val.split(',').map(s => s.trim()).filter(Boolean)
  }
})

const nfoXmlContent = computed(() => {
  const genresXml = draft.genres.map(g => `    <genre>${g}</genre>`).join('\n')
  return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<movie>
    <title>${draft.title}</title>
    <originaltitle>${draft.originalTitle}</originaltitle>
    <year>${draft.year ?? ''}</year>
    <rating>${draft.rating}</rating>
    <votes>${draft.votes}</votes>
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
    lockedFields: []
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

function formatFileSize(bytes: number): string {
  if (!bytes) return '—'
  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return `${gb.toFixed(1)} GB`
  return `${(bytes / (1024 * 1024)).toFixed(0)} MB`
}
</script>

<template>
  <main class="inspector-workspace" :aria-label="labels.details">
    <!-- Top Action Bar with Tabs -->
    <header class="inspector-toolbar">
      <!-- Tabs Navigation -->
      <nav class="workshop-tabs">
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': activeTab === 'overview' }"
          type="button"
          @click="activeTab = 'overview'"
        >
          {{ labels.overview }}
        </button>
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': activeTab === 'artwork' }"
          type="button"
          @click="activeTab = 'artwork'"
        >
          {{ labels.artworkTab }}
        </button>
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': activeTab === 'cast' }"
          type="button"
          @click="activeTab = 'cast'"
        >
          {{ labels.castTab }}
        </button>
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': activeTab === 'nfo' }"
          type="button"
          @click="activeTab = 'nfo'"
        >
          {{ labels.nfoRaw }}
        </button>
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': activeTab === 'files' }"
          type="button"
          @click="activeTab = 'files'"
        >
          {{ labels.fileAudit }}
        </button>
      </nav>

      <!-- Action Buttons -->
      <div v-if="detail" class="toolbar-actions">
        <!-- Close Mobile Button -->
        <button class="mobile-close-btn" :title="labels.backToList" type="button" @click="emit('close')">
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
            <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
          </svg>
        </button>

        <button
          class="btn btn-success"
          :disabled="isSaving || !detail.writable"
          type="button"
          @click="triggerNfoPreview"
        >
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M17 3H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V7l-4-4zm-5 16c-1.66 0-3-1.34-3-3s1.34-3 3-3 3 1.34 3 3-1.34 3-3 3zm3-10H5V5h10v4z"/>
          </svg>
          {{ labels.saveAndWriteNfo }}
        </button>

        <button
          class="btn btn-primary"
          :disabled="isScraping"
          type="button"
          @click="triggerScrape"
        >
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
          </svg>
          {{ isScraping ? labels.searching : labels.scrape }}
        </button>

        <div class="divider-v"></div>

        <button
          class="icon-btn"
          :class="{ active: isLocked }"
          :title="isLocked ? labels.metadataLocked : labels.lockMetadata"
          type="button"
          @click="isLocked = !isLocked"
        >
          <svg viewBox="0 0 24 24" fill="currentColor" width="15" height="15">
            <path d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z"/>
          </svg>
        </button>
      </div>
    </header>

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

    <!-- Content Workspace -->
    <div v-else-if="detail" class="inspector-content">
      <!-- ── TAB 1: OVERVIEW ──────────────────────────────────────── -->
      <section v-if="activeTab === 'overview'" class="overview-view">
        <!-- Hero Title Banner -->
        <div class="hero-banner">
          <div class="title-cluster">
            <h1 class="main-title">{{ draft.title || detail.item.titleHint }}</h1>
            <h2 class="sub-title">{{ draft.originalTitle || detail.item.titleHint }}</h2>
          </div>

          <div class="meta-strip">
            <span class="meta-item font-code">{{ draft.year ?? detail.item.yearHint ?? '—' }}</span>
            <span class="meta-dot"></span>
            <span class="meta-item">{{ draft.runtimeMinutes ? `${draft.runtimeMinutes} ${labels.runtimeMinutes}` : '—' }}</span>
            <span class="meta-dot"></span>
            <span v-if="draft.contentRating" class="spec-pill font-code">{{ draft.contentRating }}</span>
            <span class="meta-dot"></span>
            <span class="meta-item">{{ draft.studio }}</span>

            <!-- Rating Box -->
            <div class="rating-badge">
              <span class="star-score">★ {{ draft.rating ?? '—' }}</span>
              <span class="score-denom">/10</span>
              <span v-if="draft.votes !== null" class="vote-count">({{ draft.votes }})</span>
              <span class="provider-tag">TMDb</span>
            </div>
          </div>

          <!-- Spec Pills Row -->
        </div>

        <!-- Bento Grid Layout: 1/3 Artwork + 2/3 Metadata Form -->
        <div class="bento-grid">
          <!-- Left Column: Artwork Hub -->
          <div class="bento-col-artwork">
            <div class="main-poster-card">
              <img
                v-if="draft.posterUrl"
                :src="draft.posterUrl"
                :alt="labels.posterAlt"
                class="poster-img"
              />
              <div v-else class="poster-empty">
                <svg viewBox="0 0 24 24" fill="currentColor" width="48" height="48" opacity="0.3">
                  <path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
                </svg>
                <span>{{ labels.noPosterLoaded }}</span>
              </div>
            </div>

            <!-- Mini Artwork Grid -->
            <div class="mini-art-grid">
              <!-- Fanart / Backdrop -->
              <div class="mini-art-card">
                <img v-if="draft.backdropUrl" :src="draft.backdropUrl" :alt="labels.fanart" class="mini-art-img" />
                <div v-else class="mini-art-missing">{{ labels.fanartUnavailable }}</div>
                <span class="mini-art-label">{{ labels.fanart }}</span>
              </div>
              <!-- Clear Logo -->
              <div class="mini-art-card logo-card">
                <div class="logo-preview-box">
                  <img src="/assets/logo-icon.png" :alt="labels.logo" class="logo-preview-img" />
                </div>
                <span class="mini-art-label">{{ labels.logo }}</span>
              </div>
            </div>
          </div>

          <!-- Right Column: Metadata Form -->
          <div class="bento-col-form">
            <!-- Plot Summary Card -->
            <div class="plot-card">
              <div class="plot-box">
                <label class="plot-label">
                  <span>{{ labels.plotSummary }}</span>
                  <button class="btn-link" :title="labels.saveDraft" type="button" @click="saveMetadata">
                    {{ isSaving ? labels.saving : labels.saveDraft }}
                  </button>
                </label>
                <textarea
                  v-model="draft.overview"
                  class="plot-textarea"
                  rows="4"
                  :placeholder="labels.enterPlot"
                ></textarea>
              </div>
            </div>

            <!-- Metadata Key-Value Blocks -->
            <div class="meta-blocks-grid">
              <div class="meta-card">
                <span class="card-caption">{{ labels.releaseDate }}</span>
                <input v-model="draft.year" type="number" class="card-input" :placeholder="labels.yearPlaceholder" />
              </div>

              <div class="meta-card">
                <span class="card-caption">{{ labels.director }}</span>
                <input v-model="draft.director" type="text" class="card-input" :placeholder="labels.directorPlaceholder" />
              </div>

              <div class="meta-card meta-card-full">
                <span class="card-caption">{{ labels.genres }}</span>
                <input v-model="genresInput" type="text" class="card-input" :placeholder="labels.genresPlaceholder" />
              </div>
            </div>

            <!-- File Info Card -->
            <div class="file-info-card">
              <div class="file-icon-box">
                <svg viewBox="0 0 24 24" fill="currentColor" width="20" height="20">
                  <path d="M14 2H6c-1.1 0-1.99.9-1.99 2L4 20c0 1.1.89 2 1.99 2H18c1.1 0 2-.9 2-2V8l-6-6zm2 16H8v-2h8v2zm0-4H8v-2h8v2zm-3-5V3.5L18.5 9H13z"/>
                </svg>
              </div>
              <div class="file-details">
                <p class="file-path" :title="detail.item.relativePath">{{ detail.item.relativePath }}</p>
                <div class="file-specs">
                  <span class="dot dot-ok"></span>
                  <span class="status-txt">{{ labels.fileHealthy }}</span>
                  <span class="meta-sep">|</span>
                  <span class="stream-summary">{{ labels.videoSummaryPlaceholder }}</span>
                </div>
              </div>
              <div class="file-size-box">
                <span class="size-val">{{ formatFileSize(detail.item.fileSize) }}</span>
                <span class="size-lbl">{{ labels.fileSize }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- ── TAB 2: ARTWORK WORKSHOP ──────────────────────────────── -->
      <section v-else-if="activeTab === 'artwork'" class="workshop-view">
        <div class="workshop-header">
          <h2>{{ labels.artworkGallery }}</h2>
          <div class="workshop-actions">
            <button class="btn btn-outline" type="button">{{ labels.uploadCustomImage }}</button>
            <button class="btn btn-primary" type="button">{{ labels.scrapeFanart }}</button>
          </div>
        </div>

        <div class="artwork-grid">
          <div class="art-slot-card">
            <div class="slot-header">
              <span class="slot-title">{{ labels.poster }}</span>
            </div>
            <div class="slot-preview poster-slot">
              <img v-if="draft.posterUrl" :src="draft.posterUrl" :alt="labels.poster" />
            </div>
          </div>

          <div class="art-slot-card">
            <div class="slot-header">
              <span class="slot-title">{{ labels.fanart }}</span>
            </div>
            <div class="slot-preview fanart-slot">
              <img v-if="draft.backdropUrl" :src="draft.backdropUrl" :alt="labels.fanart" />
            </div>
          </div>

          <div class="art-slot-card">
            <div class="slot-header">
              <span class="slot-title">{{ labels.clearLogo }}</span>
            </div>
            <div class="slot-preview logo-slot">
              <img src="/assets/logo-icon.png" :alt="labels.logo" class="logo-sample" />
            </div>
          </div>

          <div class="art-slot-card">
            <div class="slot-header">
              <span class="slot-title">{{ labels.banner }}</span>
              <span class="spec-badge">1000x185</span>
            </div>
            <div class="slot-preview banner-slot">
              <div class="banner-sample">{{ labels.bannerPreview }}</div>
            </div>
          </div>
        </div>
      </section>

      <!-- ── TAB 3: CAST & CREW WORKSHOP ───────────────────────────── -->
      <section v-else-if="activeTab === 'cast'" class="workshop-view">
        <div class="workshop-header">
          <h2>{{ labels.castAndCrew }}</h2>
          <button class="btn btn-outline" type="button">{{ labels.scrapeHeadshots }}</button>
        </div>

        <div class="cast-grid">
          <div v-for="person in castList" :key="person.id" class="cast-card">
            <div class="cast-avatar">
              <img v-if="person.avatar" :src="person.avatar" :alt="labels.avatarAlt" class="avatar-img" />
              <span v-else class="avatar-placeholder">{{ person.name.charAt(0) }}</span>
            </div>
            <div class="cast-info">
              <strong class="cast-name">{{ person.name }}</strong>
              <span class="cast-role">{{ person.role }}</span>
            </div>
          </div>
        </div>
      </section>

      <!-- ── TAB 4: NFO RAW XML ────────────────────────────────────── -->
      <section v-else-if="activeTab === 'nfo'" class="workshop-view">
        <div class="workshop-header">
          <h2>{{ labels.nfoXmlEditor }}</h2>
          <button class="btn btn-success" type="button" @click="triggerNfoPreview">{{ labels.saveToNfo }}</button>
        </div>

        <div class="xml-editor-box">
          <pre class="xml-code"><code>{{ nfoXmlContent }}</code></pre>
        </div>
      </section>

      <!-- ── TAB 5: FILE AUDIT & RENAME ────────────────────────────── -->
      <section v-else-if="activeTab === 'files'" class="workshop-view">
        <div class="workshop-header">
          <h2>{{ labels.filesRenamePlanner }}</h2>
          <button class="btn btn-primary" type="button">{{ labels.runDryRun }}</button>
        </div>

        <div class="file-audit-list">
          <div class="audit-item">
            <span class="spec-badge">{{ labels.video }}</span>
            <span class="audit-path font-code">{{ detail.item.relativePath }}</span>
            <span class="audit-status text-ok">{{ labels.matched }}</span>
          </div>
          <div class="audit-item">
            <span class="spec-badge">NFO</span>
            <span class="audit-path font-code">movie.nfo</span>
            <span class="audit-status text-ok">{{ labels.validKodi }}</span>
          </div>
          <div class="audit-item">
            <span class="spec-badge">POSTER</span>
            <span class="audit-path font-code">poster.jpg</span>
            <span class="audit-status text-ok">{{ labels.fileResolutionUnavailable }}</span>
          </div>
        </div>
      </section>
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

/* ── Toolbar ──────────────────────────────────────────────── */
.inspector-toolbar {
  height: var(--toolbar-height, 40px);
  padding: 0 var(--pane-padding, 12px);
  background: var(--surface-dim, #0c1324);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.workshop-tabs {
  display: flex;
  align-items: center;
  gap: 16px;
  height: 100%;
}

.tab-btn {
  height: 100%;
  padding: 0 4px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.tab-btn:hover {
  color: var(--primary, #c0c1ff);
}

.tab-btn--active {
  color: var(--primary, #c0c1ff);
  border-bottom-color: var(--primary, #c0c1ff);
  font-weight: 700;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mobile-close-btn {
  display: none;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  background: transparent;
  border: 1px solid var(--outline-variant);
  color: var(--on-surface-variant);
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.divider-v {
  width: 1px;
  height: 16px;
  background: var(--outline-variant, #2e3447);
  margin: 0 2px;
}

.icon-btn {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm, 0.25rem);
  background: transparent;
  border: 1px solid transparent;
  color: var(--on-surface-variant, #c7c4d7);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.15s ease;
}

.icon-btn:hover {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface, #dce1fb);
}

.icon-btn.active {
  color: var(--tertiary, #ffb95f);
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

/* ── Main Content Scroll ──────────────────────────────────── */
.inspector-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--pane-padding, 12px);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ── Hero Banner ──────────────────────────────────────────── */
.hero-banner {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.title-cluster {
  display: flex;
  align-items: baseline;
  gap: 12px;
  flex-wrap: wrap;
}

.main-title {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  letter-spacing: -0.02em;
}

.sub-title {
  margin: 0;
  font-size: 16px;
  font-weight: 400;
  color: var(--on-surface-variant, #c7c4d7);
}

.meta-strip {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  flex-wrap: wrap;
}

.font-code {
  font-family: var(--font-data);
}

.meta-dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--outline-variant, #2e3447);
}

.rating-badge {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm);
  font-size: 11px;
}

.star-score {
  font-weight: 700;
  color: var(--tertiary, #ffb95f);
}

.score-denom, .vote-count {
  font-size: 9px;
  color: var(--outline, #908fa0);
}

.provider-tag {
  font-family: var(--font-data);
  font-size: 9px;
  background: var(--primary-container, #6366f1);
  color: #fff;
  padding: 1px 4px;
  border-radius: 2px;
  margin-left: 4px;
}

.spec-pills-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 4px;
}

/* ── Bento Grid ───────────────────────────────────────────── */
.bento-grid {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}

.bento-col-artwork {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.main-poster-card {
  position: relative;
  width: 100%;
  aspect-ratio: 2/3;
  border-radius: var(--radius-md, 0.375rem);
  overflow: hidden;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  align-items: center;
  justify-content: center;
}

.poster-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.poster-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  color: var(--outline, #908fa0);
  font-size: 11px;
}

.res-tag {
  position: absolute;
  top: 6px;
  right: 6px;
  background: rgba(12, 19, 36, 0.85);
  backdrop-filter: blur(4px);
  color: var(--on-surface, #dce1fb);
  font-family: var(--font-data);
  font-size: 9px;
  padding: 2px 4px;
  border-radius: 2px;
  border: 1px solid var(--outline-variant, #2e3447);
}

.mini-art-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.mini-art-card {
  position: relative;
  aspect-ratio: 16/9;
  border-radius: var(--radius-sm);
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.mini-art-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.mini-art-missing {
  color: var(--error-bright, #f43f5e);
  font-size: 8px;
  font-weight: 700;
  text-align: center;
  padding: 2px;
}

.mini-art-label {
  position: absolute;
  bottom: 0;
  right: 0;
  background: rgba(12, 19, 36, 0.9);
  color: var(--on-surface, #dce1fb);
  font-size: 8px;
  font-weight: 700;
  padding: 1px 4px;
  border-top-left-radius: 2px;
}

.logo-card {
  padding: 6px;
}

.logo-preview-box {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo-preview-img {
  max-width: 80%;
  max-height: 80%;
  object-fit: contain;
}

/* ── Bento Form ───────────────────────────────────────────── */
.bento-col-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.plot-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.tagline-txt {
  margin: 0;
  font-size: 13px;
  font-style: italic;
  color: var(--primary, #c0c1ff);
}

.plot-box {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.plot-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 10px;
  font-weight: 700;
  color: var(--on-surface-variant, #c7c4d7);
  letter-spacing: 0.05em;
}

.btn-link {
  background: transparent;
  border: none;
  color: var(--primary, #c0c1ff);
  font-size: 11px;
  cursor: pointer;
  font-weight: 600;
}

.btn-link:hover {
  text-decoration: underline;
}

.plot-textarea {
  width: 100%;
  min-height: 90px;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm);
  color: var(--on-surface, #dce1fb);
  font-size: 12px;
  line-height: 1.6;
  padding: 8px;
  resize: vertical;
}

.plot-textarea:focus {
  border-color: var(--primary, #c0c1ff);
  outline: none;
}

.meta-blocks-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.meta-card {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm);
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.meta-card-full {
  grid-column: span 2;
}

.card-caption {
  font-size: 9px;
  font-weight: 700;
  color: var(--on-surface-variant, #c7c4d7);
  letter-spacing: 0.05em;
}

.card-input {
  background: transparent;
  border: none;
  border-bottom: 1px solid transparent;
  color: var(--on-surface, #dce1fb);
  font-size: 12px;
  font-weight: 500;
  padding: 2px 0;
  width: 100%;
}

.card-input:focus {
  border-bottom-color: var(--primary, #c0c1ff);
  outline: none;
}

.file-info-card {
  margin-top: 4px;
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md);
  padding: 10px 12px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.file-icon-box {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm);
  background: var(--surface-container-high, #23293c);
  color: var(--tertiary, #ffb95f);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.file-details {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.file-path {
  margin: 0;
  font-family: var(--font-data);
  font-size: 11px;
  color: var(--on-surface, #dce1fb);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-specs {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 10px;
  color: var(--on-surface-variant, #c7c4d7);
}

.status-txt {
  font-size: 9px;
  font-weight: 700;
  color: var(--secondary, #4edea3);
}

.meta-sep {
  color: var(--outline-variant, #2e3447);
}

.file-size-box {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  border-left: 1px solid var(--outline-variant, #2e3447);
  padding-left: 12px;
}

.size-val {
  font-size: 12px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.size-lbl {
  font-size: 8px;
  color: var(--outline, #908fa0);
}

/* ── Workshops Generic ────────────────────────────────────── */
.workshop-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.workshop-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.workshop-header h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.artwork-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.art-slot-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md);
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.slot-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.slot-title {
  font-size: 10px;
  font-weight: 700;
  color: var(--on-surface-variant, #c7c4d7);
}

.slot-preview {
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm);
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.poster-slot { aspect-ratio: 2/3; }
.fanart-slot { aspect-ratio: 16/9; }
.logo-slot { aspect-ratio: 16/9; padding: 12px; }
.banner-slot { aspect-ratio: 5/1; }

.slot-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.logo-slot img {
  object-fit: contain;
}

.banner-sample {
  font-size: 11px;
  color: var(--outline, #908fa0);
}

.cast-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 12px;
}

.cast-card {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm);
  padding: 8px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.cast-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  overflow: hidden;
  background: var(--surface-container-high, #23293c);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  font-weight: 700;
}

.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.cast-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.cast-name {
  font-size: 12px;
  color: var(--on-surface, #dce1fb);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cast-role {
  font-size: 10px;
  color: var(--outline, #908fa0);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.xml-editor-box {
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md);
  padding: 12px;
  max-height: 480px;
  overflow: auto;
}

.xml-code {
  margin: 0;
  font-family: var(--font-data);
  font-size: 11px;
  line-height: 1.6;
  color: var(--on-surface-variant, #c7c4d7);
}

.file-audit-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.audit-item {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm);
  padding: 8px 12px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.audit-path {
  flex: 1;
  font-size: 11px;
  color: var(--on-surface, #dce1fb);
}

.audit-status {
  font-size: 10px;
  font-weight: 700;
}

.text-ok {
  color: var(--secondary, #4edea3);
}

/* ── Mobile Layout ────────────────────────────────────────── */
@media (max-width: 760px) {
  .mobile-close-btn {
    display: inline-flex;
  }
  .bento-grid {
    grid-template-columns: 1fr;
  }
  .workshop-tabs {
    overflow-x: auto;
  }
}
</style>
