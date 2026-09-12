<script setup lang="ts">
import { computed } from 'vue'
import type { MediaInspection, TVShowDetail, TVEpisode, TVSelection } from '@/api/types'
import TechSpecGrid from './TechSpecGrid.vue'

export interface TVDraft {
  title: string
  originalTitle: string
  year: number | null
  overview: string
  genres: string[]
  posterUrl: string
  backdropUrl: string
  rating: number | null
  votes: number | null
  status: string
  network: string
  cast: { name: string; role: string; profileUrl?: string }[]
  episodes: {
    seasonNumber: number
    episodeNumber: number
    title: string
    overview: string
    airDate: string
    runtimeMinutes?: number | null
    stillUrl: string
  }[]
}

const props = withDefaults(defineProps<{
  selection: TVSelection | null
  detail: TVShowDetail
  draft: TVDraft
  isEditing: boolean
  resolvedPosterUrl: string
  selectedUnitTitle: string
  selectedUnitEpisode: TVEpisode | null
  selectedUnitSeasonEpisodes: TVEpisode[]
  selectedUnitOverview: string
  selectedUnitPath: string
  selectedUnitSize: number
  seasonsMap: Map<number, TVEpisode[]>
  selectedEpisodeId: number | null
  inspection?: MediaInspection | null
  inspectionLoading?: boolean
  inspectionError?: string | null
  labels: Record<string, string>
}>(), {
  inspection: null,
  inspectionLoading: false,
  inspectionError: null,
})

const emit = defineEmits<{
  (e: 'update:selectedEpisodeId', id: number): void
  (e: 'selectEpisode', episode: TVEpisode): void
}>()

const genresInput = computed({
  get: () => props.draft.genres.join(', '),
  set: (val: string) => {
    props.draft.genres = val.split(',').map(s => s.trim()).filter(Boolean)
  },
})

function formatEpisodeCode(ep: TVEpisode): string {
  const s = String(ep.seasonNumber).padStart(2, '0')
  const e = String(ep.episodeStart).padStart(2, '0')
  return `S${s}E${e}`
}

function episodeTitle(ep: TVEpisode): string {
  const remote = props.draft.episodes.find(
    item => item.seasonNumber === ep.seasonNumber && item.episodeNumber === ep.episodeStart
  )
  return remote?.title || ep.titleHint || ep.relativePath.split('/').pop() || ''
}

function remoteEpisode(ep: TVEpisode) {
  return props.draft.episodes.find(
    item => item.seasonNumber === ep.seasonNumber && item.episodeNumber === ep.episodeStart
  )
}

function formatFileSize(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function nfoState(ep: TVEpisode): string {
  return ep.sidecars.some(sidecar => sidecar.kind === 'nfo') ? (props.labels.nfoFound || 'NFO Present') : (props.labels.nfoMissing || 'NFO Missing')
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

const streamSummary = computed(() => {
  const values: string[] = []
  const video = props.inspection?.video[0]
  const audio = props.inspection?.audio[0]
  if (video) values.push(`Video: ${video.width}×${video.height} ${video.codec.toUpperCase()}`)
  if (audio) values.push(`Audio: ${audio.codec.toUpperCase()} ${audio.channelLayout || `${audio.channels} ch`}`)
  return values.join(' • ')
})
</script>

<template>
  <section class="overview-view">
    <!-- Hero Title & Identity -->
    <div class="hero-banner">
      <div class="hero-main-group">
        <div class="title-cluster">
          <template v-if="!isEditing">
            <h1 class="main-title">{{ selectedUnitTitle }}</h1>
            <h2 v-if="draft.originalTitle && draft.originalTitle !== draft.title" class="sub-title">
              {{ draft.originalTitle }}
            </h2>
          </template>
          <template v-else>
            <input v-model="draft.title" type="text" class="edit-title-input" :placeholder="labels.titlePlaceholder || '电视剧名称'" />
            <input v-model="draft.originalTitle" type="text" class="edit-subtitle-input" :placeholder="labels.origTitlePlaceholder || '原始片名 / 英文名'" />
          </template>
        </div>

        <div class="meta-strip">
          <div class="meta-left-group">
            <span class="meta-item font-code">{{ selectedUnitEpisode ? formatEpisodeCode(selectedUnitEpisode) : (draft.year ?? detail.show.yearHint ?? labels.tvSeries) }}</span>
            <span class="meta-dot"></span>
            <span class="spec-pill font-code">{{ detail.show.seasonCount }} {{ labels.seasons }}</span>
            <span class="spec-pill font-code">{{ detail.show.episodeCount }} {{ labels.episodes }}</span>
            
            <template v-if="draft.status">
              <span class="meta-dot"></span>
              <span class="spec-pill font-code">{{ draft.status }}</span>
            </template>
          </div>

          <div v-if="draft.rating !== null" class="rating-badge">
            <span class="star-score">★ {{ typeof draft.rating === 'number' ? draft.rating.toFixed(1) : draft.rating }}</span>
            <span class="score-denom">/10</span>
            <span v-if="draft.votes !== null" class="vote-count">({{ draft.votes }})</span>
            <span class="provider-tag">TMDb</span>
          </div>
        </div>

        <TechSpecGrid v-if="techSpecs.length" :specs="techSpecs" />
        <p v-else-if="selection?.kind === 'episode'" class="tech-unavailable font-code">
          {{ inspectionLoading ? labels.inspectingMedia : (inspectionError || inspection?.probeError || labels.mediaInfoUnavailable) }}
        </p>
      </div>

      <!-- Mobile Poster on the right -->
      <div class="poster-container mobile-hero-poster">
        <div class="compact-poster-card group">
          <img
            v-if="resolvedPosterUrl"
            :src="resolvedPosterUrl"
            :alt="draft.title || 'Poster'"
            class="poster-img"
          />
          <div v-else class="poster-empty">
            <svg viewBox="0 0 24 24" fill="currentColor" width="24" height="24" opacity="0.3">
              <path d="M21 3H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h5v2h8v-2h5c1.1 0 1.99-.9 1.99-2L23 5c0-1.1-.9-2-2-2zm0 14H3V5h18v12z"/>
            </svg>
            <span>{{ labels.noPosterLoaded || '未加载海报' }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="overview-body-layout">
      <!-- Desktop Poster on Left -->
      <div class="poster-container desktop-body-poster">
        <div class="compact-poster-card group">
          <img
            v-if="resolvedPosterUrl"
            :src="resolvedPosterUrl"
            :alt="draft.title || 'Poster'"
            class="poster-img"
          />
          <div v-else class="poster-empty">
            <svg viewBox="0 0 24 24" fill="currentColor" width="32" height="32" opacity="0.3">
              <path d="M21 3H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h5v2h8v-2h5c1.1 0 1.99-.9 1.99-2L23 5c0-1.1-.9-2-2-2zm0 14H3V5h18v12z"/>
            </svg>
            <span>{{ labels.noPosterLoaded || '未加载海报' }}</span>
          </div>
        </div>
      </div>

      <div class="details-container">
        <div class="meta-blocks-grid">
          <div class="meta-card">
            <span class="card-label-caps">{{ labels.releaseDate || '首播年份' }}</span>
            <div v-if="!isEditing" class="meta-val font-code">
              {{ draft.year || '—' }}
            </div>
            <div v-else class="meta-input-wrap">
              <input v-model="draft.year" type="number" class="card-input font-code" :placeholder="labels.yearPlaceholder || 'YYYY'" />
            </div>
          </div>

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
              :placeholder="labels.genresPlaceholder || '剧情, 动作, 奇幻'"
            />
          </div>

          <div class="meta-card">
            <span class="card-label-caps">NETWORK / 平台</span>
            <div v-if="!isEditing" class="meta-val meta-val-highlight">
              {{ draft.network || '—' }}
            </div>
            <input v-else v-model="draft.network" type="text" class="card-input" placeholder="HBO, Netflix..." />
          </div>

          <div class="meta-card">
            <span class="card-label-caps">STATUS / 连载状态</span>
            <div v-if="!isEditing" class="meta-val">
              {{ draft.status || '—' }}
            </div>
            <input v-else v-model="draft.status" type="text" class="card-input" placeholder="Ended, Returning Series..." />
          </div>

          <div class="meta-card meta-card-full">
            <span class="card-label-caps">STORAGE PATH / 目录路径</span>
            <div class="meta-val font-code" :title="selectedUnitPath">
              {{ selectedUnitPath }}
            </div>
          </div>
        </div>

        <div class="plot-card">
          <div class="plot-box">
            <div class="plot-header">
              <span class="card-label-caps">{{ labels.plotSummary || '剧情简介' }}</span>
            </div>
            <p v-if="!isEditing" class="plot-paragraph">
              {{ selectedUnitOverview || labels.noOverview || '暂无剧情简介。' }}
            </p>
            <textarea
              v-else
              v-model="draft.overview"
              class="plot-textarea"
              rows="4"
              :placeholder="labels.enterPlot || '输入电视剧剧情简介...'"
            ></textarea>
          </div>
        </div>

        <div class="file-info-card">
          <div class="file-icon-box">
            <svg viewBox="0 0 24 24" fill="currentColor" width="18" height="18">
              <path d="M14 2H6c-1.1 0-1.99.9-1.99 2L4 20c0 1.1.89 2 1.99 2H18c1.1 0 2-.9 2-2V8l-6-6zm2 16H8v-2h8v2zm0-4H8v-2h8v2zm-3-5V3.5L18.5 9H13z"/>
            </svg>
          </div>
          <div class="file-details">
            <p class="file-path font-code" :title="selectedUnitPath">{{ selectedUnitPath }}</p>
            <div class="file-specs">
              <span class="dot dot-ok"></span>
              <span class="status-txt">{{ selectedUnitEpisode ? formatEpisodeCode(selectedUnitEpisode) : (selection?.kind === 'season' ? `${selectedUnitSeasonEpisodes.length} ${labels.episodes}` : `${detail.show.seasonCount} ${labels.seasons} · ${detail.show.episodeCount} ${labels.episodes}`) }}</span>
              <span class="meta-sep">|</span>
              <span class="stream-summary font-code">{{ streamSummary || (inspectionLoading ? labels.inspectingMedia : (inspectionError || inspection?.probeError || labels.mediaInfoUnavailable)) }}</span>
            </div>
          </div>
          <div class="file-size-divider"></div>
          <div class="file-size-box">
            <span class="size-val font-code">{{ formatFileSize(selectedUnitSize) }}</span>
            <span class="size-lbl font-code">{{ labels.fileSize || 'TOTAL SIZE' }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Seasons & Episodes Section -->
    <div v-if="selection?.kind !== 'episode'" class="seasons-container">
      <div class="seasons-header-title">
        <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="title-icon">
          <path d="M4 6H2v14c0 1.1.9 2 2 2h14v-2H4V6zm16-4H8c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H8V4h12v12z"/>
        </svg>
        <h2>{{ labels.episodesAndSeasons || 'Seasons & Episodes' }}</h2>
      </div>

      <section
        v-for="[seasonNum, episodes] in (selection?.kind === 'season' ? [[selection.seasonNumber, selectedUnitSeasonEpisodes]] : seasonsMap)"
        :key="seasonNum"
        class="season-card"
      >
        <header class="season-header">
          <div class="season-title-box">
            <span class="season-badge">{{ labels.season }} {{ seasonNum }}</span>
            <span class="season-count">{{ episodes.length }} {{ labels.episodesCount }}</span>
          </div>
          <span class="spec-pill font-code">{{ draft.title ? labels.metadataReady : labels.metadataEmpty }}</span>
        </header>

        <table class="episode-table">
          <thead>
            <tr>
              <th class="col-code">{{ labels.code }}</th>
              <th class="col-title">{{ labels.titleAndPath }}</th>
              <th class="col-res">{{ labels.quality }}</th>
              <th class="col-status">{{ labels.nfo }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="ep in episodes"
              :key="ep.id"
              class="ep-row"
              :class="{ 'ep-row--active': selectedEpisodeId === ep.id }"
              @click="emit('update:selectedEpisodeId', ep.id); emit('selectEpisode', ep)"
            >
              <td class="col-code font-code">{{ formatEpisodeCode(ep) }}</td>
              <td class="col-title">
                <div class="ep-title-main">{{ episodeTitle(ep) }}</div>
                <div class="ep-path font-code">{{ ep.relativePath }}</div>
              </td>
              <td class="col-res">
                <span class="spec-badge font-code">{{ remoteEpisode(ep)?.runtimeMinutes ? `${remoteEpisode(ep)?.runtimeMinutes} ${labels.runtimeMinutes || 'min'}` : (labels.qualityUnavailable || '—') }}</span>
              </td>
              <td class="col-status">
                <span class="dot" :class="ep.sidecars.some(sidecar => sidecar.kind === 'nfo') ? 'dot-ok' : 'dot-off'" :title="nfoState(ep)"></span>
              </td>
            </tr>
          </tbody>
        </table>
      </section>
    </div>
  </section>
</template>

<style scoped>
.overview-view {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
  overflow-x: hidden;
}

.hero-banner,
.overview-body-layout,
.seasons-container {
  position: relative;
  z-index: 1;
}

.hero-banner {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  max-width: 100%;
  min-width: 0;
}

.hero-main-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  flex: 1;
}

.poster-container.mobile-hero-poster {
  display: none;
}

.poster-container.desktop-body-poster {
  display: block;
  width: 180px;
  flex-shrink: 0;
}

.title-cluster {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.main-title {
  font-size: 26px;
  font-weight: 800;
  color: var(--on-surface, #dce1fb);
  line-height: 1.2;
  letter-spacing: -0.02em;
  margin: 0;
}

.sub-title {
  font-size: 15px;
  font-weight: 500;
  color: var(--on-surface-variant, #c7c4d7);
  margin: 0;
}

.edit-title-input {
  font-size: 20px;
  font-weight: 700;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  padding: 6px 10px;
  width: 100%;
}

.edit-subtitle-input {
  font-size: 14px;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface-variant, #c7c4d7);
  padding: 4px 10px;
  width: 100%;
}

.meta-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
}

.meta-left-group {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.meta-item {
  font-size: 13px;
  font-weight: 600;
  color: var(--primary, #c0c1ff);
}

.meta-dot {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--outline-variant, #2e3447);
}

.rating-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  background: var(--surface-container-high, #23293c);
  padding: 2px 8px;
  border-radius: var(--radius-sm, 0.25rem);
  font-size: 12px;
}

.star-score {
  color: #ffc107;
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
  background: var(--primary-container, #8083ff);
  color: #ffffff;
  font-size: 9px;
  font-weight: 700;
  padding: 1px 4px;
  border-radius: 2px;
  margin-left: 4px;
}

.spec-pills-row {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.overview-body-layout {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 20px;
}

.poster-container {
  display: flex;
  flex-direction: column;
}

.compact-poster-card {
  width: 100%;
  aspect-ratio: 2 / 3;
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  overflow: hidden;
  position: relative;
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
  color: var(--outline, #908fa0);
  font-size: 11px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.poster-res-badge {
  position: absolute;
  bottom: 6px;
  right: 6px;
  font-size: 9px;
  font-weight: 700;
  background: rgba(0, 0, 0, 0.7);
  color: #ffffff;
  padding: 2px 6px;
  border-radius: 3px;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.details-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
  max-width: 100%;
  box-sizing: border-box;
  overflow: hidden;
}

.meta-blocks-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 10px;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
}

.meta-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  max-width: 100%;
  box-sizing: border-box;
  overflow: hidden;
}

.meta-card-full {
  grid-column: span 2;
  min-width: 0;
  max-width: 100%;
}

.card-label-caps {
  font-size: 10px;
  font-weight: 700;
  color: var(--primary, #c0c1ff);
  letter-spacing: 0.05em;
}

.meta-val {
  font-size: 13px;
  color: var(--on-surface, #dce1fb);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.meta-val-highlight {
  font-weight: 600;
}

.card-input {
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  padding: 4px 8px;
  font-size: 12px;
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
}

.genres-pills-list {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.genre-tag-pill {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface, #dce1fb);
  font-size: 11px;
  padding: 2px 6px;
  border-radius: var(--radius-sm, 0.25rem);
}

.genre-empty-hint {
  font-size: 12px;
  color: var(--outline, #908fa0);
}

.plot-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 12px;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
}

.plot-header {
  margin-bottom: 6px;
}

.plot-paragraph {
  font-size: 13px;
  line-height: 1.6;
  color: var(--on-surface, #dce1fb);
  margin: 0;
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
  padding: 8px;
  font-size: 13px;
  line-height: 1.5;
  resize: vertical;
}

.file-info-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 10px 14px;
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
  overflow: hidden;
}

.file-icon-box {
  color: var(--primary, #c0c1ff);
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.file-details {
  flex: 1;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.file-path {
  font-size: 12px;
  color: var(--on-surface, #dce1fb);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin: 0;
}

.file-specs {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
}

.status-txt {
  color: var(--on-surface-variant, #c7c4d7);
}

.meta-sep {
  color: var(--outline-variant, #2e3447);
}

.stream-summary {
  color: var(--outline, #908fa0);
}

.file-size-divider {
  width: 1px;
  height: 28px;
  background: var(--outline-variant, #2e3447);
}

.file-size-box {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 1px;
}

.size-val {
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.size-lbl {
  font-size: 9px;
  color: var(--outline, #908fa0);
}

.seasons-container {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.seasons-header-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.seasons-header-title h2 {
  font-size: 16px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  margin: 0;
}

.season-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  overflow: hidden;
}

.season-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: var(--surface-container-high, #23293c);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.season-title-box {
  display: flex;
  align-items: center;
  gap: 8px;
}

.season-badge {
  font-size: 13px;
  font-weight: 700;
  color: var(--primary, #c0c1ff);
}

.season-count {
  font-size: 11px;
  color: var(--outline, #908fa0);
}

.episode-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.episode-table th {
  padding: 8px 12px;
  text-align: left;
  font-size: 10px;
  font-weight: 700;
  color: var(--outline, #908fa0);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-container-low, #151b2d);
}

.episode-table td {
  padding: 8px 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.03);
}

.ep-row {
  cursor: pointer;
  transition: background 0.15s ease;
}

.ep-row:hover {
  background: var(--surface-container-high, #23293c);
}

.ep-row--active {
  background: var(--surface-container-highest, #2d344b) !important;
}

.col-code {
  width: 70px;
  font-weight: 700;
  color: var(--primary, #c0c1ff);
}

.col-title {
  min-width: 0;
}

.ep-title-main {
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
}

.ep-path {
  font-size: 10px;
  color: var(--outline, #908fa0);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 380px;
}

.col-res {
  width: 90px;
}

.col-status {
  width: 40px;
  text-align: center;
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
    grid-template-columns: 1fr;
    gap: 14px;
  }

  .episode-table-container {
    overflow-x: auto;
  }
}
</style>
