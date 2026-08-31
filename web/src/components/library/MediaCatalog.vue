<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import type { Job, MediaItem } from '@/api/library'

const props = defineProps<{
  items: MediaItem[]
  selectedId: number | null
  activeJob: Job | undefined
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  search: [query: string]
  select: [id: number]
  scan: []
}>()

const query = shallowRef('')
const filterMode = shallowRef<'all' | 'unscraped' | 'nfo_missing'>('all')

function submitSearch() {
  emit('search', query.value)
}

function hasSidecar(item: MediaItem, kind: string): boolean {
  return item.sidecars.some(s => s.kind === kind || s.relativePath.toLowerCase().includes(kind))
}

const filteredItems = computed(() => {
  if (filterMode.value === 'unscraped' || filterMode.value === 'nfo_missing') {
    return props.items.filter(item => !hasSidecar(item, 'nfo'))
  }
  return props.items
})

const unscrapedCount = computed(() => {
  return props.items.filter(item => !hasSidecar(item, 'nfo')).length
})

function getResolution(item: MediaItem): string {
  const path = item.relativePath.toLowerCase()
  if (path.includes('2160p') || path.includes('4k') || path.includes('uhd')) return '4K'
  if (path.includes('1080p') || path.includes('fhd')) return '1080p'
  if (path.includes('720p')) return '720p'
  return 'HD'
}
</script>

<template>
  <aside class="catalog-panel" :aria-label="labels.movieList || 'Movie List'">
    <!-- Header with Search and Stats -->
    <div class="catalog-header">
      <!-- Search Input -->
      <div class="search-box">
        <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8" />
          <line x1="21" y1="21" x2="16.65" y2="16.65" />
        </svg>
        <input
          v-model="query"
          type="text"
          class="search-input"
          :placeholder="labels.searchPlaceholder || '搜索片名、年份、IMDb ID...'"
          @keydown.enter.prevent="submitSearch"
        />
      </div>

      <!-- Count & Actions Bar -->
      <div class="catalog-toolbar">
        <div class="catalog-stats">
          <span class="count-main">{{ items.length }} {{ labels.movies || '电影' }}</span>
          <span v-if="unscrapedCount > 0" class="count-unscraped">({{ unscrapedCount }} {{ labels.unscraped || '未刮削' }})</span>
        </div>
        <div class="toolbar-actions">
          <button
            class="icon-action-btn"
            :class="{ active: filterMode !== 'all' }"
            :title="labels.filterByStatus || 'Filter'"
            type="button"
            @click="filterMode = filterMode === 'all' ? 'unscraped' : 'all'"
          >
            <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
              <path d="M10 18h4v-2h-4v2zM3 6v2h18V6H3zm3 7h12v-2H6v2z"/>
            </svg>
          </button>
          <button
            class="icon-action-btn"
            :title="labels.refresh || 'Scan Library'"
            type="button"
            @click="submitSearch"
          >
            <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
              <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Active Job Banner -->
    <div v-if="activeJob" class="active-job-banner">
      <span class="job-spinner"></span>
      <div class="job-info">
        <p class="job-title">{{ labels.activeJob || 'Task' }} #{{ activeJob.id }}: {{ activeJob.state }}</p>
        <p class="job-desc">{{ activeJob.message || labels.loading }}</p>
      </div>
    </div>

    <!-- Movie Items List -->
    <div class="catalog-list">
      <div
        v-for="item in filteredItems"
        :key="item.id"
        class="media-row group"
        :class="{ 'media-row--active': selectedId === item.id }"
        @click="emit('select', item.id)"
      >
        <!-- Poster Thumbnail with 4K badge -->
        <div class="thumb-box">
          <img
            v-if="item.posterUrl"
            :src="item.posterUrl"
            :alt="item.title || item.titleHint"
            class="thumb-img"
            loading="lazy"
          />
          <div v-else class="thumb-placeholder">
            <svg viewBox="0 0 24 24" fill="currentColor" width="18" height="18" opacity="0.35">
              <path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
            </svg>
          </div>
          <!-- 4K / HD Spec Overlay -->
          <span class="res-badge font-code" :class="{ 'res-4k': getResolution(item) === '4K' }">
            {{ getResolution(item) }}
          </span>
        </div>

        <!-- Movie Info -->
        <div class="row-info">
          <div class="title-primary" :title="item.title || item.titleHint">
            {{ item.title || item.titleHint }}
          </div>
          <div v-if="!item.title" class="title-sub" :title="item.relativePath">
            {{ item.relativePath.split('/').pop() }}
          </div>
          <div class="row-meta">
            <span class="year-txt font-code">{{ item.yearHint ?? '—' }}</span>
            <div class="status-dots">
              <!-- NFO Status Dot -->
              <span
                class="dot"
                :class="hasSidecar(item, 'nfo') ? 'dot-ok' : 'dot-warn'"
                :title="hasSidecar(item, 'nfo') ? (labels.nfoReady || 'NFO Complete') : (labels.nfoMissing || 'NFO Missing')"
              ></span>
              <!-- Poster Status Dot -->
              <span
                class="dot"
                :class="hasSidecar(item, 'poster') || hasSidecar(item, 'jpg') || hasSidecar(item, 'png') ? 'dot-ok' : 'dot-warn'"
                :title="hasSidecar(item, 'poster') ? (labels.posterReady || 'Poster Complete') : (labels.posterMissing || 'Poster Missing')"
              ></span>
              <!-- Fanart Status Dot -->
              <span
                class="dot"
                :class="hasSidecar(item, 'fanart') ? 'dot-ok' : 'dot-off'"
                :title="hasSidecar(item, 'fanart') ? (labels.fanartReady || 'Fanart Complete') : (labels.fanartMissing || 'Fanart Missing')"
              ></span>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="filteredItems.length === 0" class="catalog-empty">
        <p>{{ labels.noFilms || 'No media items found.' }}</p>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.catalog-panel {
  width: var(--catalog-width, 320px);
  min-width: var(--catalog-width, 320px);
  height: 100vh;
  background: var(--surface-container, #191f31);
  border-right: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  z-index: 40;
}

.catalog-header {
  padding: 10px 12px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-base, #0c1324);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.search-box {
  position: relative;
  width: 100%;
}

.search-icon {
  position: absolute;
  left: 9px;
  top: 50%;
  transform: translateY(-50%);
  width: 15px;
  height: 15px;
  color: var(--on-surface-variant, #c7c4d7);
  pointer-events: none;
}

.search-input {
  width: 100%;
  height: 32px;
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  font-size: 13px;
  padding: 0 10px 0 32px;
  transition: all 0.15s ease;
}

.search-input:focus {
  border-color: var(--primary, #c0c1ff);
  outline: none;
  box-shadow: 0 0 0 1px var(--primary, #c0c1ff);
}

.search-input::placeholder {
  color: rgba(199, 196, 215, 0.5);
}

.catalog-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.catalog-stats {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.count-main {
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
}

.count-unscraped {
  font-size: 11px;
  color: var(--tertiary, #ffb95f);
  font-family: var(--font-data);
}

.toolbar-actions {
  display: flex;
  gap: 4px;
}

.icon-action-btn {
  width: 26px;
  height: 26px;
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

.icon-action-btn:hover {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface, #dce1fb);
}

.icon-action-btn.active {
  background: var(--surface-container-high, #23293c);
  color: var(--primary, #c0c1ff);
  border-color: var(--primary, #c0c1ff);
}

.active-job-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: var(--surface-container-high, #23293c);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  font-size: 11px;
}

.job-spinner {
  width: 12px;
  height: 12px;
  border: 2px solid var(--tertiary, #ffb95f);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.job-info {
  flex: 1;
  min-width: 0;
}

.job-title {
  margin: 0;
  font-weight: 700;
  color: var(--tertiary, #ffb95f);
}

.job-desc {
  margin: 0;
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 10px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.catalog-list {
  flex: 1;
  overflow-y: auto;
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.media-row {
  display: flex;
  align-items: center;
  gap: 10px;
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

.res-4k {
  background: var(--secondary-container, #00a572);
  color: #ffffff;
  border: none;
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

.status-dots {
  display: flex;
  align-items: center;
  gap: 4px;
}

.catalog-empty {
  padding: 3rem 1rem;
  text-align: center;
  color: var(--outline, #908fa0);
  font-size: 13px;
}

@media (max-width: 700px) {
  .catalog-panel {
    width: 100%;
    min-width: 100%;
    height: auto;
    border-right: none;
  }
}
</style>
