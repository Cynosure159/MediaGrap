<script setup lang="ts">
import { computed, shallowRef, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import type { Job, MediaItem, CatalogOptions } from '@/api/library'
import { useDismissiblePopover } from '@/composables/useDismissiblePopover'
import SearchBar from '@/components/common/SearchBar.vue'

const props = defineProps<{
  items: MediaItem[]
  total?: number | null
  loading?: boolean
  hasMore?: boolean
  loadError?: string | null
  selectedId: number | null
  activeJob: Job | undefined
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  search: [query: string, options: CatalogOptions]
  loadMore: []
  select: [id: number]
  scan: []
}>()

const query = shallowRef('')
type FilterType = 'all' | 'unscraped' | 'nfo_missing' | 'poster_missing' | '4k' | '1080p'
const filterMode = shallowRef<FilterType>('all')
type SortType = 'title' | 'year' | 'size'
const sortMode = shallowRef<SortType>('title')
const { open: showFilterMenu, container: filterDropdownRef } = useDismissiblePopover()

const isFilterActive = computed(() => filterMode.value !== 'all' || sortMode.value !== 'title')

function submitSearch() {
  emit('search', query.value, { filter: filterMode.value, sort: sortMode.value })
}

function hasSidecar(item: MediaItem, kind: string): boolean {
  return item.sidecars.some(s => s.kind === kind || s.relativePath.toLowerCase().includes(kind))
}

function getResolution(item: MediaItem): string {
  const path = item.relativePath.toLowerCase()
  if (path.includes('2160p') || path.includes('4k') || path.includes('uhd')) return '4K'
  if (path.includes('1080p') || path.includes('fhd')) return '1080p'
  if (path.includes('720p')) return '720p'
  return 'HD'
}

// Keep DOM/image work bounded even after thousands of items have been fetched.
const listElement = shallowRef<HTMLElement | null>(null)
const scrollTop = shallowRef(0)
const viewportHeight = shallowRef(600)
const stride = 60 // 56px row + 4px spacing, per the catalog design tokens
const start = computed(() => Math.max(0, Math.floor(scrollTop.value / stride) - 5))
const end = computed(() => Math.min(props.items.length, Math.ceil((scrollTop.value + viewportHeight.value) / stride) + 5))
const visibleItems = computed(() => props.items.slice(start.value, end.value))
const count = computed(() => props.total === undefined ? props.items.length : props.total)
let resizeObserver: ResizeObserver | undefined

function checkMore() {
  const el = listElement.value
  if (!el || el.clientHeight <= 0) return
  if (!props.hasMore || props.loading || props.loadError) return

  const remainingScroll = el.scrollHeight - el.scrollTop - el.clientHeight
  if (remainingScroll < 300) {
    emit('loadMore')
  }
}

function onScroll() {
  scrollTop.value = listElement.value?.scrollTop || 0
  checkMore()
}

function resetScroll() {
  if (listElement.value) {
    listElement.value.scrollTop = 0
  }
  scrollTop.value = 0
}

watch([filterMode, sortMode], () => {
  resetScroll()
  submitSearch()
})

watch(() => props.items, async (items) => {
  if (!items.length) {
    resetScroll()
  } else if (props.selectedId !== null) {
    scrollToActiveItem()
  }
  await nextTick()
  checkMore()
})

function scrollToActiveItem() {
  if (props.selectedId === null || !listElement.value) return
  const index = props.items.findIndex(item => item.id === props.selectedId)
  if (index === -1) return
  const el = listElement.value
  const targetTop = index * stride
  // If the target item is outside the visible viewport, scroll it into view
  if (targetTop < el.scrollTop || targetTop + stride > el.scrollTop + el.clientHeight) {
    const desired = Math.max(0, targetTop - Math.floor(el.clientHeight / 2) + Math.floor(stride / 2))
    el.scrollTop = desired
    scrollTop.value = desired
  }
}

watch(() => props.selectedId, (id) => {
  if (id !== null) {
    nextTick(() => scrollToActiveItem())
  }
})

watch(() => props.loading, async () => {
  await nextTick()
  checkMore()
})

onMounted(() => {
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => {
      viewportHeight.value = listElement.value?.clientHeight || 600
      checkMore()
    })
    if (listElement.value) {
      resizeObserver.observe(listElement.value)
    }
  }
})

onBeforeUnmount(() => resizeObserver?.disconnect())

function setFilter(f: FilterType) {
  filterMode.value = f
  showFilterMenu.value = false
}

function setSort(s: SortType) {
  sortMode.value = s
  showFilterMenu.value = false
}
</script>

<template>
  <aside class="catalog-panel" :aria-label="labels.movieList || 'Movie List'">
    <!-- Header with Search and Stats -->
    <div class="catalog-header">
      <!-- Search Input -->
      <SearchBar
        v-model="query"
        :placeholder="labels.search || 'Search movies…'"
        @search="submitSearch"
      />

      <!-- Count & Actions Bar -->
      <div class="catalog-toolbar">
        <div class="catalog-stats">
          <span class="count-main">{{ count ?? '—' }} {{ labels.movies || '电影' }}</span>

        </div>
        <div class="toolbar-actions">
          <!-- Filter Menu Trigger -->
          <div ref="filterDropdownRef" class="filter-dropdown-wrap">
            <button
              class="icon-action-btn"
              :class="{ active: isFilterActive || showFilterMenu }"
              :title="labels.filterByStatus || 'Filter'"
              type="button"
              @click.stop="showFilterMenu = !showFilterMenu"
            >
              <svg viewBox="0 0 24 24" fill="currentColor" width="15" height="15">
                <path d="M10 18h4v-2h-4v2zM3 6v2h18V6H3zm3 7h12v-2H6v2z"/>
              </svg>
              <span v-if="isFilterActive" class="active-dot"></span>
            </button>

            <!-- Dropdown Popover with Animation -->
            <Transition name="dropdown-fade">
              <div v-if="showFilterMenu" class="filter-popover" @click.stop>
                <div class="popover-section">
                  <span class="popover-label">{{ labels.filterByStatus || 'Filter' }}</span>
                  <button
                    type="button"
                    class="popover-item"
                    :class="{ active: filterMode === 'all' }"
                    @click="setFilter('all')"
                  >
                    <span>{{ labels.filterAll || 'All' }}</span>
                  </button>
                  <button
                    type="button"
                    class="popover-item"
                    :class="{ active: filterMode === 'unscraped' }"
                    @click="setFilter('unscraped')"
                  >
                    <span>{{ labels.filterUnscraped || 'Unscraped' }}</span>

                  </button>
                  <button
                    type="button"
                    class="popover-item"
                    :class="{ active: filterMode === '4k' }"
                    @click="setFilter('4k')"
                  >
                    <span>{{ labels.filter4K || '4K UHD' }}</span>
                  </button>
                  <button
                    type="button"
                    class="popover-item"
                    :class="{ active: filterMode === '1080p' }"
                    @click="setFilter('1080p')"
                  >
                    <span>{{ labels.filter1080p || '1080p FHD' }}</span>
                  </button>
                </div>

                <div class="popover-divider"></div>

                <div class="popover-section">
                  <span class="popover-label">{{ labels.sortMovies || 'Sort by' }}</span>
                  <button
                    type="button"
                    class="popover-item"
                    :class="{ active: sortMode === 'title' }"
                    @click="setSort('title')"
                  >
                    <span>{{ labels.sortTitle || 'Title (A-Z)' }}</span>
                    <svg v-if="sortMode === 'title'" class="check-icon" viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
                      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                    </svg>
                  </button>
                  <button
                    type="button"
                    class="popover-item"
                    :class="{ active: sortMode === 'year' }"
                    @click="setSort('year')"
                  >
                    <span>{{ labels.sortYear || 'Year (Newest)' }}</span>
                    <svg v-if="sortMode === 'year'" class="check-icon" viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
                      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                    </svg>
                  </button>
                  <button
                    type="button"
                    class="popover-item"
                    :class="{ active: sortMode === 'size' }"
                    @click="setSort('size')"
                  >
                    <span>{{ labels.sortSize || 'File Size' }}</span>
                    <svg v-if="sortMode === 'size'" class="check-icon" viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
                      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                    </svg>
                  </button>
                </div>
              </div>
            </Transition>
          </div>

          <!-- Scan Button -->
          <button
            class="icon-action-btn"
            :title="labels.refresh || 'Scan Library'"
            type="button"
            @click="emit('scan')"
          >
            <svg viewBox="0 0 24 24" fill="currentColor" width="15" height="15">
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
    <div ref="listElement" class="catalog-list" :aria-busy="loading" @scroll.passive="onScroll">
      <div :style="{ height: `${start * stride}px` }" aria-hidden="true"></div>
      <div
        v-for="item in visibleItems"
        :key="item.id"
        class="media-row group"
        :class="{ 'media-row--active': selectedId === item.id }"
        role="button"
        tabindex="0"
        :aria-label="item.title || item.titleHint"
        @keydown.enter="emit('select', item.id)"
        @keydown.space.prevent="emit('select', item.id)"
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
            decoding="async"
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
            <span class="year-txt font-code">{{ item.year ?? item.yearHint ?? '—' }}</span>
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

      <div :style="{ height: `${Math.max(0, items.length - end) * stride}px` }" aria-hidden="true"></div>
      <div class="load-status" role="status">
        <template v-if="loading">{{ labels.loading }}</template>
        <button v-else-if="loadError" type="button" class="btn-secondary" @click="emit('loadMore')">{{ labels.catalogRetry || 'Retry loading' }}</button>
        <template v-else-if="count !== null">{{ labels.catalogLoaded || 'Loaded' }} {{ items.length }} / {{ count }}</template>
      </div>
      <!-- Empty State -->
      <div v-if="items.length === 0 && !loading && !loadError" class="catalog-empty">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="32" height="32" class="empty-icon">
          <rect x="2" y="4" width="20" height="16" rx="2" />
          <path d="M8 4v16M16 4v16M2 12h20" />
        </svg>
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

.filter-dropdown-wrap {
  position: relative;
}

.icon-action-btn {
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

.icon-action-btn:hover {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface, #dce1fb);
}

.icon-action-btn.active {
  background: var(--surface-container-high, #23293c);
  color: var(--primary, #c0c1ff);
  border-color: var(--primary, #c0c1ff);
}

.filter-popover {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  width: 180px;
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
  padding: 8px 0;
  z-index: 100;
  display: flex;
  flex-direction: column;
}

.popover-section {
  display: flex;
  flex-direction: column;
}

.popover-label {
  padding: 4px 12px;
  font-size: 9px;
  font-weight: 700;
  color: var(--outline, #908fa0);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.popover-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  background: transparent;
  border: none;
  color: var(--on-surface, #dce1fb);
  font-size: 12px;
  text-align: left;
  cursor: pointer;
  transition: background 0.1s ease;
}

.popover-item:hover {
  background: var(--surface-container-highest, #2e3447);
}

.popover-item.active {
  color: var(--primary, #c0c1ff);
  font-weight: 600;
  background: rgba(192, 193, 255, 0.08);
}

.popover-count {
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.popover-divider {
  height: 1px;
  background: var(--outline-variant, #2e3447);
  margin: 6px 0;
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
  min-height: 0;
  overflow-anchor: none;
}

.load-status { padding: 12px; text-align: center; font-size: 11px; color: var(--on-surface-variant); }

.media-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px;
  box-sizing: border-box;
  height: 56px;
  margin-bottom: 4px;
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
  width: 30px;
  height: 44px;
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
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 3rem 1rem;
  text-align: center;
  color: var(--outline, #908fa0);
  font-size: 13px;
}

.empty-icon {
  opacity: 0.4;
}

</style>
