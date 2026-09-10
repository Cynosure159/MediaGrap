<script setup lang="ts">
import { computed } from 'vue'
import * as api from '@/api/library'
import type { CastMember, MediaInspection, MediaItem } from '@/api/types'
import TechSpecGrid from './TechSpecGrid.vue'
import MetadataForm from './MetadataForm.vue'

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
  cast: CastMember[]
}

const props = defineProps<{
  draft: MovieDraft
  item: MediaItem
  isEditing: boolean
  labels: Record<string, string>
  inspection: MediaInspection | null
  inspectionLoading: boolean
}>()

const resolvedPoster = computed(() => {
  if (props.draft.posterUrl) return props.draft.posterUrl
  const localPoster = props.item.sidecars.find(s => {
    const filename = s.relativePath.split('/').pop()?.toLowerCase() || ''
    return ['poster', 'folder', 'cover'].some(k => filename === k || filename.startsWith(k + '.'))
  })
  if (localPoster) {
    return api.mediaArtworkUrl(props.item.id, localPoster.relativePath)
  }
  return ''
})

function formatRuntime(minutes: number | null): string {
  if (!minutes || minutes <= 0) return '—'
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

function formatFileSize(bytes: number): string {
  if (!bytes) return '—'
  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return `${gb.toFixed(1)} GB`
  return `${(bytes / (1024 * 1024)).toFixed(0)} MB`
}

const techSpecs = computed(() => {
  const specs: string[] = []
  const video = props.inspection?.video[0]
  const audio = props.inspection?.audio[0]
  if (video?.width && video.height) specs.push(`${video.width}×${video.height}`)
  if (video?.codec) specs.push(`${video.codec.toUpperCase()}${video.bitDepth ? ` ${video.bitDepth}-bit` : ''}`)
  if (video?.hdr) specs.push(video.hdr)
  if (audio?.codec) specs.push(`${audio.codec.toUpperCase()} ${audio.channelLayout || `${audio.channels} ch`}`)
  if (props.inspection?.subtitles.length) specs.push(`${props.inspection.subtitles.length} ${props.labels.subtitleTracks || 'subtitle tracks'}`)
  return specs
})

const fileHealthy = computed(() => props.inspection?.files.every(file => file.valid && !file.symlink) ?? false)

const streamSummary = computed(() => {
  const video = props.inspection?.video[0]
  const audio = props.inspection?.audio[0]
  const values: string[] = []
  if (video) values.push(`Video: ${video.width}×${video.height} ${video.codec.toUpperCase()}`)
  if (audio) values.push(`Audio: ${audio.codec.toUpperCase()} ${audio.channelLayout || `${audio.channels} ch`}`)
  return values.join(' • ')
})
</script>

<template>
  <section class="overview-view">
    <!-- ── 1. Hero Title & Metadata Banner ──────────────────────────── -->
    <div class="hero-banner">
      <div class="hero-main-group">
        <div class="title-cluster">
          <template v-if="!isEditing">
            <h1 class="main-title">{{ draft.title || item.titleHint }}</h1>
            <h2 v-if="draft.originalTitle && draft.originalTitle !== draft.title" class="sub-title">
              {{ draft.originalTitle }}
            </h2>
          </template>
          <template v-else>
            <input v-model="draft.title" type="text" class="edit-title-input" :placeholder="labels.titlePlaceholder || '电影名称'" />
            <input v-model="draft.originalTitle" type="text" class="edit-subtitle-input" :placeholder="labels.origTitlePlaceholder || '原始片名 / 英文名'" />
          </template>
        </div>

        <!-- Metadata Strip -->
        <div class="meta-strip">
          <div class="meta-left-group">
            <span class="meta-item font-code">{{ draft.year ?? item.yearHint ?? '—' }}</span>
            <span class="meta-dot"></span>
            <span class="meta-item">{{ formatRuntime(draft.runtimeMinutes) }}</span>
            
            <template v-if="draft.contentRating">
              <span class="meta-dot"></span>
              <span class="spec-pill font-code">{{ draft.contentRating }}</span>
            </template>
          </div>

          <!-- Rating Box on the right -->
          <div v-if="draft.rating !== null" class="rating-badge">
            <span class="star-score">★ {{ typeof draft.rating === 'number' ? draft.rating.toFixed(1) : draft.rating }}</span>
            <span class="score-denom">/10</span>
            <span v-if="draft.votes !== null" class="vote-count">({{ draft.votes }})</span>
            <span class="provider-tag">TMDb</span>
          </div>
        </div>

        <!-- Spec Pills Row (High-Density Tech Tags) -->
        <TechSpecGrid v-if="techSpecs.length" :specs="techSpecs" />
        <p v-else class="tech-unavailable font-code">
          {{ inspectionLoading ? labels.inspectingMedia : (inspection?.probeError || labels.mediaInfoUnavailable) }}
        </p>
      </div>

      <!-- Mobile Poster on the right -->
      <div class="poster-container mobile-hero-poster">
        <div class="compact-poster-card group">
          <img
            v-if="resolvedPoster"
            :src="resolvedPoster"
            :alt="labels.posterAlt || 'Poster'"
            class="poster-img"
          />
          <div v-else class="poster-empty">
            <svg viewBox="0 0 24 24" fill="currentColor" width="24" height="24" opacity="0.3">
              <path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
            </svg>
            <span>{{ labels.noPosterLoaded || '未加载海报' }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ── 2. Content Layout (Compact Poster + High Priority Info) ──── -->
    <div class="overview-body-layout">
      <!-- ── Left: Compact Poster on Desktop (Hidden on Mobile) ──────────────── -->
      <div class="poster-container desktop-body-poster">
        <div class="compact-poster-card group">
          <img
            v-if="resolvedPoster"
            :src="resolvedPoster"
            :alt="labels.posterAlt || 'Poster'"
            class="poster-img"
          />
          <div v-else class="poster-empty">
            <svg viewBox="0 0 24 24" fill="currentColor" width="32" height="32" opacity="0.3">
              <path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
            </svg>
            <span>{{ labels.noPosterLoaded || '未加载海报' }}</span>
          </div>
        </div>
      </div>

      <!-- ── Right: High-Priority Info + Plot + File Info ── -->
      <div class="details-container">
        <!-- ── Top Priority: Core Metadata Grid ── -->
        <MetadataForm :draft="draft" :is-editing="isEditing" :labels="labels" />

        <!-- ── Plot Summary Card ── -->
        <div class="plot-card">
          <div class="plot-box">
            <div class="plot-header">
              <span class="card-label-caps">{{ labels.plotSummary || '剧情简介' }}</span>
            </div>
            
            <!-- View Mode: Clean readable text -->
            <p v-if="!isEditing" class="plot-paragraph">
              {{ draft.overview || labels.noOverview || '暂无剧情简介。' }}
            </p>

            <!-- Edit Mode: Textarea editor -->
            <textarea
              v-else
              v-model="draft.overview"
              class="plot-textarea"
              rows="4"
              :placeholder="labels.enterPlot || '输入电影剧情简介...'"
            ></textarea>
          </div>
        </div>

        <!-- ── File Info Audit Card ── -->
        <div class="file-info-card">
          <div class="file-icon-box">
            <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
              <path d="M4 6H2v14c0 1.1.9 2 2 2h14v-2H4V6zm16-4H8c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H8V4h12v12z"/>
            </svg>
          </div>
          <div class="file-details">
            <div class="file-path-row">
              <p class="file-path font-code" :title="item.relativePath">{{ item.relativePath }}</p>
            </div>
            <div class="file-specs">
              <span class="dot" :class="inspection ? (inspection.video.length ? 'dot-ok' : 'dot-warning') : 'dot-ok'"></span>
              <span class="status-txt">{{ inspection ? (inspection.video.length ? (labels.fileHealthy || 'HEALTHY') : (labels.fileUnprobed || 'UNPROBED')) : (labels.fileHealthy || 'HEALTHY') }}</span>
              <span class="meta-sep">|</span>
              <span class="stream-summary font-code">{{ streamSummary || labels.mediaInfoUnavailable }}</span>
            </div>
          </div>

          <div class="file-size-divider"></div>

          <div class="file-size-box">
            <span class="size-val font-code">{{ formatFileSize(item.fileSize) }}</span>
            <span class="size-lbl font-code">{{ labels.fileSize || 'FILE SIZE' }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.overview-view {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
  overflow-x: hidden;
}

.hero-banner,
.overview-body-layout {
  position: relative;
  z-index: 1;
}

.tech-unavailable {
  margin: 0;
  color: var(--outline, #908fa0);
  font-size: 11px;
}

.dot-warning {
  background: var(--tertiary, #ffb95f);
}

/* ── 1. Hero Title Banner ─────────────────────────────────────────── */
.hero-banner {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
}

.hero-main-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.poster-container.mobile-hero-poster {
  display: none;
}

.poster-container.desktop-body-poster {
  display: block;
}

.title-cluster {
  display: flex;
  align-items: baseline;
  gap: 12px;
  flex-wrap: wrap;
}

.main-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  margin: 0;
  line-height: 1.2;
  letter-spacing: -0.02em;
}

.sub-title {
  font-size: 15px;
  font-weight: 400;
  color: var(--on-surface-variant, #c7c4d7);
  margin: 0;
}

.edit-title-input {
  font-size: 18px;
  font-weight: 700;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  padding: 4px 10px;
  width: 280px;
}

.edit-subtitle-input {
  font-size: 13px;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface-variant, #c7c4d7);
  padding: 4px 10px;
  width: 220px;
}

.meta-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  flex-wrap: wrap;
}

.meta-left-group {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.meta-dot {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--outline-variant, #464554);
}

.rating-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  background: var(--surface-container-low, #151b2d);
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
  font-size: 10px;
}

.vote-count {
  color: var(--outline, #908fa0);
  font-size: 10px;
}

.provider-tag {
  background: #01b4e4;
  color: #ffffff;
  font-size: 9px;
  font-weight: 700;
  padding: 1px 4px;
  border-radius: 2px;
  margin-left: 2px;
  font-family: var(--font-data);
}

.spec-pills-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 2px;
}

/* ── 2. Body Grid (Balanced 2-Column Desktop, Stacked Mobile) ─────── */
.overview-body-layout {
  display: flex;
  gap: 20px;
  align-items: flex-start;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
}

/* Poster Container */
.poster-container {
  width: 180px;
  flex-shrink: 0;
}

.compact-poster-card {
  width: 180px;
  height: 270px;
  aspect-ratio: 2 / 3;
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  overflow: hidden;
  position: relative;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
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
  gap: 6px;
  color: var(--outline, #908fa0);
  font-size: 11px;
}

.poster-res-badge {
  position: absolute;
  top: 6px;
  right: 6px;
  background: rgba(12, 19, 36, 0.85);
  backdrop-filter: blur(4px);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 1px 5px;
  font-size: 9px;
  color: var(--on-surface, #dce1fb);
}

/* Details Column */
.details-container {
  flex: 1;
  min-width: 0;
  max-width: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
  box-sizing: border-box;
  overflow: hidden;
}

/* ── Plot Summary Card ── */
.plot-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 10px 12px;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
}

.plot-box {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  max-width: 100%;
}

.plot-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.card-label-caps {
  font-size: 9px;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--outline, #908fa0);
  letter-spacing: 0.06em;
}

.plot-paragraph {
  font-size: 12px;
  line-height: 1.65;
  color: var(--on-surface, #dce1fb);
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.plot-textarea {
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  font-family: inherit;
  font-size: 12px;
  line-height: 1.6;
  padding: 8px 10px;
  resize: vertical;
  min-height: 80px;
}

.plot-textarea:focus {
  outline: none;
  border-color: var(--primary, #c0c1ff);
}

/* ── File Info Card ───────────────────────────────────────────────── */
.file-info-card {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 8px 12px;
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: auto;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
  overflow: hidden;
}

.file-icon-box {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-highest, #2e3447);
  color: var(--tertiary, #ffb95f);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.file-details {
  flex: 1;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
}

.file-path {
  font-size: 11px;
  color: var(--on-surface, #dce1fb);
  margin: 0 0 2px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.file-specs {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.status-txt {
  font-size: 9px;
  font-weight: 700;
  color: var(--secondary, #4edea3);
  letter-spacing: 0.04em;
}

.meta-sep {
  color: var(--outline-variant, #2e3447);
}

.stream-summary {
  font-size: 10px;
}

.file-size-divider {
  width: 1px;
  height: 20px;
  background: var(--outline-variant, #2e3447);
}

.file-size-box {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  flex-shrink: 0;
  min-width: 50px;
}

.size-val {
  font-size: 11px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.size-lbl {
  font-size: 8px;
  color: var(--outline, #908fa0);
  letter-spacing: 0.05em;
}

@media (max-width: 768px) {
  .overview-view {
    padding-bottom: calc(72px + env(safe-area-inset-bottom, 0px));
  }

  .hero-banner {
    display: flex;
    flex-direction: row;
    align-items: flex-start;
    gap: 14px;
  }

  .hero-main-group {
    flex: 1;
    min-width: 0;
  }

  .main-title {
    font-size: 19px;
  }

  .poster-container.mobile-hero-poster {
    display: block;
    width: 105px;
    flex-shrink: 0;
  }

  .poster-container.mobile-hero-poster .compact-poster-card {
    width: 105px;
    height: 157px;
    box-shadow: 0 4px 14px rgba(0, 0, 0, 0.4);
  }

  .poster-container.desktop-body-poster {
    display: none;
  }

  .overview-body-layout {
    display: flex;
    flex-direction: column;
    gap: 14px;
    width: 100%;
  }

  .details-container {
    width: 100%;
  }
}
</style>
