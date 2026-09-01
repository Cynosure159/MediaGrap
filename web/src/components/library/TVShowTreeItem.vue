<script setup lang="ts">
import { computed } from 'vue'
import type { TVEpisode, TVSelection, TVShow, TVShowDetail } from '@/api/library'

const props = defineProps<{
  show: TVShow
  detail?: TVShowDetail
  expanded: boolean
  selected: TVSelection | null
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  toggle: [showId: number]
  select: [selection: TVSelection]
}>()

const seasons = computed(() => {
  const grouped = new Map<number, TVEpisode[]>()
  for (const episode of props.detail?.episodes ?? []) {
    const episodes = grouped.get(episode.seasonNumber) ?? []
    episodes.push(episode)
    grouped.set(episode.seasonNumber, episodes)
  }
  return [...grouped.entries()]
})

function episodeCode(episode: TVEpisode) {
  return `S${String(episode.seasonNumber).padStart(2, '0')}E${String(episode.episodeStart).padStart(2, '0')}`
}

function episodeTitle(episode: TVEpisode) {
  const metadata = props.detail?.metadata.episodes.find(
    item => item.seasonNumber === episode.seasonNumber && item.episodeNumber === episode.episodeStart
  )
  return metadata?.title || episode.titleHint || episode.relativePath.split('/').pop() || ''
}

function isShowActive(): boolean {
  if (!props.selected) return false
  return props.selected.showId === props.show.id && props.selected.kind === 'show'
}

function isSeasonActive(seasonNumber: number): boolean {
  if (!props.selected) return false
  return props.selected.showId === props.show.id && props.selected.kind === 'season' && props.selected.seasonNumber === seasonNumber
}

function isEpisodeActive(seasonNumber: number, episodeId: number): boolean {
  if (!props.selected) return false
  return props.selected.showId === props.show.id && props.selected.kind === 'episode' && props.selected.episodeId === episodeId
}
</script>

<template>
  <div class="tv-tree-node">
    <!-- Show Level Main Card (Movie media-row style with Left Expand Chevron) -->
    <div
      class="media-row group"
      :class="{ 'media-row--active': isShowActive() }"
      @click="emit('select', { kind: 'show', showId: show.id })"
    >
      <!-- Left Expand / Collapse Button -->
      <button
        class="tree-chevron-btn"
        :class="{ 'tree-chevron-btn--expanded': expanded }"
        :title="expanded ? '收起季集' : '展开季集'"
        type="button"
        @click.stop="emit('toggle', show.id)"
      >
        <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
          <path d="M8.59 16.59L13.17 12 8.59 7.41 10 6l6 6-6 6-1.41-1.41z"/>
        </svg>
      </button>

      <!-- Poster Thumbnail with TV Spec Overlay -->
      <div class="thumb-box">
        <img
          v-if="show.posterUrl || detail?.metadata?.posterUrl"
          :src="show.posterUrl || detail?.metadata?.posterUrl"
          :alt="show.titleHint"
          class="thumb-img"
          loading="lazy"
        />
        <div v-else class="thumb-placeholder">
          <svg viewBox="0 0 24 24" fill="currentColor" width="18" height="18" opacity="0.35">
            <path d="M21 3H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h5v2h8v-2h5c1.1 0 1.99-.9 1.99-2L23 5c0-1.1-.9-2-2-2zm0 14H3V5h18v12z"/>
          </svg>
        </div>
        <!-- TV Spec Overlay Badge -->
        <span class="res-badge font-code">TV</span>
      </div>

      <!-- TV Show Info -->
      <div class="row-info">
        <div class="title-primary" :title="show.titleHint">
          {{ show.titleHint }}
        </div>
        <div class="title-sub" :title="show.relativePath">
          {{ show.relativePath }}
        </div>
        <div class="row-meta">
          <span class="year-txt font-code">{{ show.yearHint ?? '—' }}</span>
          <div class="spec-pills-wrap">
            <span class="spec-badge font-code">{{ show.seasonCount }} {{ labels.seasons || '季' }}</span>
            <span class="spec-badge font-code">{{ show.episodeCount }} {{ labels.episodes || '集' }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Tree Children (Smooth, borderless hierarchy with guideline) -->
    <div v-if="expanded" class="tree-children">
      <!-- Loading State -->
      <div v-if="!detail" class="tree-loading">
        <span class="tree-spinner"></span>
        <span>{{ labels.loading || '正在加载季集...' }}</span>
      </div>

      <!-- Seasons & Episodes List -->
      <template v-else>
        <div
          v-for="[seasonNumber, episodes] in seasons"
          :key="seasonNumber"
          class="season-group"
        >
          <!-- Season Row -->
          <div
            class="season-row"
            :class="{ 'season-row--active': isSeasonActive(seasonNumber) }"
            @click="emit('select', { kind: 'season', showId: show.id, seasonNumber })"
          >
            <span class="season-name font-code">
              {{ labels.season || '第' }} {{ seasonNumber }} {{ labels.seasons || '季' }}
            </span>
            <span class="season-ep-count font-code">{{ episodes.length }} {{ labels.episodes || '集' }}</span>
          </div>

          <!-- Episode Rows -->
          <div class="episodes-list">
            <div
              v-for="episode in episodes"
              :key="episode.id"
              class="episode-row"
              :class="{ 'episode-row--active': isEpisodeActive(seasonNumber, episode.id) }"
              @click="emit('select', { kind: 'episode', showId: show.id, seasonNumber, episodeId: episode.id })"
            >
              <!-- S01E01 Badge -->
              <span class="episode-code font-code">{{ episodeCode(episode) }}</span>

              <!-- Episode Title -->
              <span class="episode-title" :title="episodeTitle(episode)">
                {{ episodeTitle(episode) }}
              </span>

              <!-- Sidecar Dot Indicator -->
              <span
                class="dot"
                :class="episode.sidecars.some(s => s.kind === 'nfo') ? 'dot-ok' : 'dot-off'"
                :title="episode.sidecars.some(s => s.kind === 'nfo') ? (labels.nfoReady || 'NFO Complete') : (labels.nfoMissing || 'NFO Missing')"
              ></span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.tv-tree-node {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

/* ── Show Main Row Card (Matched exactly with MediaCatalog.vue) ─────── */
.media-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border-radius: var(--radius-sm, 0.25rem);
  border-left: 2px solid transparent;
  background: transparent;
  cursor: pointer;
  transition: all 0.12s ease;
  user-select: none;
}

.media-row:hover {
  background: var(--surface-container-high, #23293c);
}

.media-row--active {
  background: var(--surface-container-highest, #2e3447) !important;
  border-left-color: var(--primary, #c0c1ff) !important;
}

.tree-chevron-btn {
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  color: var(--on-surface-variant, #c7c4d7);
  cursor: pointer;
  border-radius: 2px;
  transition: transform 0.15s ease, color 0.15s ease;
  flex-shrink: 0;
  padding: 0;
  margin-left: -2px;
}

.tree-chevron-btn:hover {
  color: var(--primary, #c0c1ff);
  background: rgba(192, 193, 255, 0.12);
}

.tree-chevron-btn--expanded {
  transform: rotate(90deg);
  color: var(--primary, #c0c1ff);
}

.thumb-box {
  position: relative;
  width: 40px;
  height: 56px;
  border-radius: var(--radius-sm, 0.25rem);
  overflow: hidden;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.thumb-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.thumb-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

.res-badge {
  position: absolute;
  bottom: 0;
  right: 0;
  background: var(--surface-bright, #33394c);
  color: var(--on-surface, #dce1fb);
  font-size: 8px;
  font-weight: 700;
  padding: 1px 3px;
  line-height: 1;
  border-top-left-radius: 2px;
  border-left: 1px solid var(--outline-variant, #2e3447);
  border-top: 1px solid var(--outline-variant, #2e3447);
}

.row-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.title-primary {
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.25;
}

.media-row--active .title-primary {
  color: var(--primary, #c0c1ff);
}

.title-sub {
  font-size: 11px;
  color: var(--on-surface-variant, #c7c4d7);
  opacity: 0.7;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 2px;
}

.year-txt {
  font-size: 11px;
  color: var(--on-surface-variant, #c7c4d7);
}

.spec-pills-wrap {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* ── Tree Children (Non-card, Clean Branching) ────────────────────── */
.tree-children {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-left: 16px;
  padding-left: 10px;
  border-left: 1px solid var(--outline-variant, #2e3447);
  margin-top: 2px;
  margin-bottom: 6px;
}

.tree-loading {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  font-size: 11px;
  color: var(--outline, #908fa0);
}

.tree-spinner {
  width: 10px;
  height: 10px;
  border: 2px solid var(--primary, #c0c1ff);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.season-group {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.season-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 8px;
  border-radius: var(--radius-sm, 0.25rem);
  cursor: pointer;
  transition: all 0.12s ease;
  user-select: none;
}

.season-row:hover {
  background: var(--surface-container-high, #23293c);
}

.season-row--active {
  background: var(--surface-container-highest, #2e3447);
}

.season-name {
  font-size: 11px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  letter-spacing: 0.02em;
}

.season-row--active .season-name {
  color: var(--primary, #c0c1ff);
}

.season-ep-count {
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.episodes-list {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.episode-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-radius: var(--radius-sm, 0.25rem);
  border-left: 2px solid transparent;
  cursor: pointer;
  transition: all 0.12s ease;
  user-select: none;
}

.episode-row:hover {
  background: var(--surface-container-high, #23293c);
}

.episode-row--active {
  background: var(--surface-container-highest, #2e3447) !important;
  border-left-color: var(--primary, #c0c1ff) !important;
}

.episode-code {
  font-size: 10px;
  font-weight: 600;
  color: var(--primary, #c0c1ff);
  width: 48px;
  flex-shrink: 0;
}

.episode-row--active .episode-code {
  font-weight: 700;
}

.episode-title {
  flex: 1;
  min-width: 0;
  font-size: 11px;
  color: var(--on-surface-variant, #c7c4d7);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.episode-row--active .episode-title {
  color: var(--on-surface, #dce1fb);
  font-weight: 600;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.dot-ok {
  background: var(--secondary, #4edea3);
  box-shadow: 0 0 4px rgba(78, 222, 163, 0.5);
}

.dot-off {
  background: var(--outline-variant, #464554);
}
</style>
