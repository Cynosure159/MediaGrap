<script setup lang="ts">
import { computed } from 'vue'
import type { MediaItem } from '@/api/types'

export interface MovieDraft {
  title: string
  originalTitle: string
  year: number | null
  overview: string
  runtimeMinutes: number | null
  genres: string[]
  posterUrl: string
  backdropUrl: string
  director: string
  writers: string
  studio: string
  rating: number | null
  votes: number | null
  contentRating: string
}

const props = defineProps<{
  draft: MovieDraft
  item: MediaItem
  isSaving: boolean
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  saveDraft: []
}>()

const genresInput = computed({
  get: () => props.draft.genres.join(', '),
  set: (val: string) => {
    props.draft.genres = val.split(',').map(s => s.trim()).filter(Boolean)
  },
})

function formatFileSize(bytes: number): string {
  if (!bytes) return '—'
  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return `${gb.toFixed(1)} GB`
  return `${(bytes / (1024 * 1024)).toFixed(0)} MB`
}
</script>

<template>
  <section class="overview-view">
    <!-- Hero Title Banner -->
    <div class="hero-banner">
      <div class="title-cluster">
        <h1 class="main-title">{{ draft.title || item.titleHint }}</h1>
        <h2 class="sub-title">{{ draft.originalTitle || item.titleHint }}</h2>
      </div>

      <div class="meta-strip">
        <span class="meta-item font-code">{{ draft.year ?? item.yearHint ?? '—' }}</span>
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
              <button class="btn-link" :title="labels.saveDraft" type="button" @click="emit('saveDraft')">
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
            <p class="file-path" :title="item.relativePath">{{ item.relativePath }}</p>
            <div class="file-specs">
              <span class="dot dot-ok"></span>
              <span class="status-txt">{{ labels.fileHealthy }}</span>
              <span class="meta-sep">|</span>
              <span class="stream-summary">{{ labels.videoSummaryPlaceholder }}</span>
            </div>
          </div>
          <div class="file-size-box">
            <span class="size-val">{{ formatFileSize(item.fileSize) }}</span>
            <span class="size-lbl">{{ labels.fileSize }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.overview-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

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
  font-size: 20px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  margin: 0;
  line-height: 1.2;
}

.sub-title {
  font-size: 14px;
  font-weight: 400;
  color: var(--on-surface-variant, #c7c4d7);
  margin: 0;
}

.meta-strip {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  flex-wrap: wrap;
}

.meta-dot {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--outline, #908fa0);
}

.rating-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 6px;
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  font-size: 11px;
}

.star-score {
  color: #f59e0b;
  font-weight: 700;
}

.score-denom {
  color: var(--outline, #908fa0);
}

.vote-count {
  color: var(--outline, #908fa0);
  font-size: 10px;
}

.provider-tag {
  background: #01b4e4;
  color: #fff;
  font-size: 9px;
  font-weight: 700;
  padding: 1px 3px;
  border-radius: 2px;
  margin-left: 2px;
}

/* Bento Grid */
.bento-grid {
  display: grid;
  grid-template-columns: 240px 1fr;
  gap: 16px;
}

.bento-col-artwork {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.main-poster-card {
  width: 100%;
  aspect-ratio: 2 / 3;
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.5rem);
  overflow: hidden;
}

.poster-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.poster-empty {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--outline, #908fa0);
  font-size: 12px;
}

.mini-art-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.mini-art-card {
  height: 70px;
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  overflow: hidden;
  position: relative;
}

.mini-art-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.mini-art-missing {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  color: var(--outline, #908fa0);
  text-align: center;
  padding: 4px;
}

.mini-art-label {
  position: absolute;
  bottom: 2px;
  left: 4px;
  font-size: 9px;
  color: rgba(255, 255, 255, 0.8);
  background: rgba(0, 0, 0, 0.6);
  padding: 1px 4px;
  border-radius: 2px;
}

.logo-card {
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo-preview-box {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo-preview-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.bento-col-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.plot-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.5rem);
  padding: 12px;
}

.plot-box {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.plot-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--outline, #908fa0);
  letter-spacing: 0.04em;
}

.btn-link {
  background: none;
  border: none;
  color: var(--primary, #c0c1ff);
  font-size: 11px;
  cursor: pointer;
  padding: 0;
  text-transform: none;
}

.btn-link:hover {
  text-decoration: underline;
}

.plot-textarea {
  width: 100%;
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  font-family: inherit;
  font-size: 13px;
  line-height: 1.5;
  padding: 8px 10px;
  resize: vertical;
  min-height: 80px;
}

.plot-textarea:focus {
  outline: none;
  border-color: var(--primary, #c0c1ff);
}

.meta-blocks-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.meta-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.5rem);
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.meta-card-full {
  grid-column: span 2;
}

.card-caption {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--outline, #908fa0);
  letter-spacing: 0.04em;
}

.card-input {
  background: transparent;
  border: none;
  color: var(--on-surface, #dce1fb);
  font-family: inherit;
  font-size: 13px;
  padding: 2px 0;
}

.card-input:focus {
  outline: none;
  color: var(--primary, #c0c1ff);
}

.file-info-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.5rem);
  padding: 10px 14px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.file-icon-box {
  color: var(--outline, #908fa0);
  display: flex;
  align-items: center;
}

.file-details {
  flex: 1;
  min-width: 0;
}

.file-path {
  font-size: 12px;
  font-family: var(--font-code, monospace);
  color: var(--on-surface, #dce1fb);
  margin: 0 0 2px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-specs {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--outline, #908fa0);
}

.meta-sep {
  color: var(--outline-variant, #2e3447);
}

.file-size-box {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.size-val {
  font-size: 12px;
  font-weight: 600;
  font-family: var(--font-code, monospace);
  color: var(--on-surface, #dce1fb);
}

.size-lbl {
  font-size: 9px;
  color: var(--outline, #908fa0);
  text-transform: uppercase;
}

@media (max-width: 900px) {
  .bento-grid {
    grid-template-columns: 1fr;
  }
}
</style>
