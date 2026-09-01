<script setup lang="ts">
import { computed } from 'vue'
import * as api from '@/api/library'
import type { CastMember, MediaItem } from '@/api/types'

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

const genresInput = computed({
  get: () => props.draft.genres.join(', '),
  set: (val: string) => {
    props.draft.genres = val.split(',').map(s => s.trim()).filter(Boolean)
  },
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

// Extract media tech specs hints from filename
const techSpecs = computed(() => {
  const path = props.item.relativePath.toUpperCase()
  const specs: string[] = []
  
  if (path.includes('2160P') || path.includes('4K') || path.includes('UHD')) specs.push('4K UHD')
  else if (path.includes('1080P') || path.includes('FHD')) specs.push('1080p FHD')
  else if (path.includes('720P')) specs.push('720p HD')

  if (path.includes('HEVC') || path.includes('X265') || path.includes('H.265') || path.includes('H265')) specs.push('HEVC 10-bit')
  else if (path.includes('AVC') || path.includes('X264') || path.includes('H.264') || path.includes('H264')) specs.push('AVC 8-bit')

  if (path.includes('ATMOS')) specs.push('Dolby Atmos 7.1')
  else if (path.includes('DDP5.1') || path.includes('DD+5.1') || path.includes('EAC3')) specs.push('E-AC3 5.1')
  else if (path.includes('AAC')) specs.push('AAC 2.0')

  if (path.includes('HDR10+') || path.includes('HDR10PLUS')) specs.push('HDR10+')
  else if (path.includes('HDR')) specs.push('HDR10')
  else if (path.includes('DV') || path.includes('DOVI')) specs.push('Dolby Vision')

  if (path.includes('DUAL') || path.includes('CHS') || path.includes('CHT')) specs.push('Chs/Eng Sub')

  return specs.length > 0 ? specs : ['1080p FHD', 'AVC 8-bit', 'AAC 2.0', 'Chs/Eng Sub']
})
</script>

<template>
  <section class="overview-view">
    <!-- ── 1. Hero Title & Metadata Banner ──────────────────────────── -->
    <div class="hero-banner">
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
        <span class="meta-item font-code">{{ draft.year ?? item.yearHint ?? '—' }}</span>
        <span class="meta-dot"></span>
        <span class="meta-item">{{ formatRuntime(draft.runtimeMinutes) }}</span>
        
        <template v-if="draft.contentRating">
          <span class="meta-dot"></span>
          <span class="spec-pill font-code">{{ draft.contentRating }}</span>
        </template>

        <template v-if="draft.studio">
          <span class="meta-dot"></span>
          <span class="meta-studio" :title="draft.studio">{{ draft.studio }}</span>
        </template>

        <!-- Rating Box on the right -->
        <div v-if="draft.rating !== null" class="rating-badge">
          <span class="star-score">★ {{ typeof draft.rating === 'number' ? draft.rating.toFixed(1) : draft.rating }}</span>
          <span class="score-denom">/10</span>
          <span v-if="draft.votes !== null" class="vote-count">({{ draft.votes }})</span>
          <span class="provider-tag">TMDb</span>
        </div>
      </div>

      <!-- Spec Pills Row (High-Density Tech Tags) -->
      <div class="spec-pills-row">
        <span v-for="spec in techSpecs" :key="spec" class="spec-pill">
          {{ spec }}
        </span>
      </div>
    </div>

    <!-- ── 2. Content Layout (Compact Poster + High Priority Info) ──── -->
    <div class="overview-body-layout">
      <!-- ── Left: Compact Poster (Fixed Proportion) ──────────────── -->
      <div class="poster-container">
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

          <!-- Top-Right Resolution Badge -->
          <div v-if="resolvedPoster" class="poster-res-badge font-code">
            1000×1500
          </div>
        </div>
      </div>

      <!-- ── Right: High-Priority Info + Plot + File Info ── -->
      <div class="details-container">
        <!-- ── Top Priority: Core Metadata Grid (2 Columns) ── -->
        <div class="meta-blocks-grid">
          <!-- Row 1, Col 1: Release Date -->
          <div class="meta-card">
            <span class="card-label-caps">{{ labels.releaseDate || '上映日期' }}</span>
            <div v-if="!isEditing" class="meta-val font-code">
              {{ draft.year || '—' }}
            </div>
            <div v-else class="meta-input-wrap">
              <input v-model="draft.year" type="number" class="card-input font-code" :placeholder="labels.yearPlaceholder || 'YYYY'" />
              <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14" class="meta-field-icon">
                <path d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM7 10h5v5H7z"/>
              </svg>
            </div>
          </div>

          <!-- Row 1, Col 2: Genres (Pill Badges) -->
          <div class="meta-card">
            <span class="card-label-caps">{{ labels.genres || '类型' }}</span>
            <div v-if="!isEditing" class="genres-pills-list">
              <span v-for="g in draft.genres" :key="g" class="genre-tag-pill">
                {{ g }}
              </span>
              <span v-if="draft.genres.length === 0" class="genre-empty-hint">
                {{ labels.noGenres || '暂无分类' }}
              </span>
            </div>
            <input
              v-else
              v-model="genresInput"
              type="text"
              class="card-input"
              :placeholder="labels.genresPlaceholder || '剧情, 动作, 历史'"
            />
          </div>

          <!-- Row 2, Col 1: Director -->
          <div class="meta-card">
            <span class="card-label-caps">{{ labels.director || '导演' }}</span>
            <div v-if="!isEditing" class="meta-val meta-val-highlight">
              {{ draft.director || '—' }}
            </div>
            <input v-else v-model="draft.director" type="text" class="card-input" :placeholder="labels.directorPlaceholder || '导演'" />
          </div>

          <!-- Row 2, Col 2: Writers -->
          <div class="meta-card">
            <span class="card-label-caps">{{ labels.writers || '编剧' }}</span>
            <div v-if="!isEditing" class="meta-val" :title="draft.writers">
              {{ draft.writers || '—' }}
            </div>
            <input v-else v-model="draft.writers" type="text" class="card-input" :placeholder="labels.writerPlaceholder || '编剧、剧本'" />
          </div>

          <!-- Row 3: Studio / Production (Full Width across 2 columns) -->
          <div class="meta-card meta-card-full">
            <span class="card-label-caps">{{ labels.studio || '制作公司' }}</span>
            <div v-if="!isEditing" class="meta-val" :title="draft.studio">
              {{ draft.studio || '—' }}
            </div>
            <input v-else v-model="draft.studio" type="text" class="card-input" :placeholder="labels.studioPlaceholder || '制作公司'" />
          </div>
        </div>

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
            <svg viewBox="0 0 24 24" fill="currentColor" width="18" height="18">
              <path d="M14 2H6c-1.1 0-1.99.9-1.99 2L4 20c0 1.1.89 2 1.99 2H18c1.1 0 2-.9 2-2V8l-6-6zm2 16H8v-2h8v2zm0-4H8v-2h8v2zm-3-5V3.5L18.5 9H13z"/>
            </svg>
          </div>

          <div class="file-details">
            <p class="file-path font-code" :title="item.relativePath">{{ item.relativePath }}</p>
            <div class="file-specs">
              <span class="dot dot-ok"></span>
              <span class="status-txt">{{ labels.fileHealthy || 'HEALTHY' }}</span>
              <span class="meta-sep">|</span>
              <span class="stream-summary font-code">Video: 1080p AVC • Audio: AAC 2.0</span>
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
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}

/* ── 1. Hero Title Banner ─────────────────────────────────────────── */
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
  gap: 8px;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  flex-wrap: wrap;
}

.meta-dot {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--outline-variant, #464554);
}

.meta-studio {
  max-width: 380px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
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
  margin-left: auto;
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
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* ── Top Priority: Core Metadata Grid ── */
.meta-blocks-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}

.meta-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 6px 10px;
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-height: 48px;
  justify-content: center;
}

.meta-card-full {
  grid-column: span 2;
}

.meta-val {
  font-size: 12px;
  font-weight: 500;
  color: var(--on-surface, #dce1fb);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.meta-val-highlight {
  color: var(--primary, #c0c1ff);
}

.meta-input-wrap {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
}

.meta-field-icon {
  color: var(--outline, #908fa0);
  flex-shrink: 0;
}

.card-input {
  width: 100%;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  font-family: inherit;
  font-size: 12px;
  font-weight: 500;
  padding: 3px 6px;
}

.card-input:focus {
  outline: none;
  border-color: var(--primary, #c0c1ff);
}

.genres-pills-list {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
  padding: 1px 0;
}

.genre-tag-pill {
  background: var(--surface-container-highest, #2e3447);
  border: 1px solid var(--outline-variant, #464554);
  color: var(--on-surface, #dce1fb);
  font-size: 11px;
  padding: 1px 6px;
  border-radius: var(--radius-sm, 0.25rem);
}

.genre-empty-hint {
  font-size: 11px;
  color: var(--outline, #908fa0);
}

/* ── Plot Summary Card ── */
.plot-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 10px 12px;
}

.plot-box {
  display: flex;
  flex-direction: column;
  gap: 6px;
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
}

.plot-textarea {
  width: 100%;
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
}

.file-path {
  font-size: 11px;
  color: var(--on-surface, #dce1fb);
  margin: 0 0 2px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
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

/* ── Mobile / Narrow Responsive Layout ────────────────────────────── */
@media (max-width: 640px) {
  .overview-body-layout {
    flex-direction: column;
    gap: 12px;
  }

  .poster-container {
    width: 100%;
    display: flex;
    justify-content: center;
  }

  .compact-poster-card {
    width: 130px;
    height: 195px;
  }
}
</style>
