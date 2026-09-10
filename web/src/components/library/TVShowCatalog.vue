<script setup lang="ts">
import { computed, shallowReactive, shallowRef, watch } from 'vue'
import * as api from '@/api/library'
import type { Job, TVSelection, TVShow, TVShowDetail } from '@/api/library'
import TVShowTreeItem from './TVShowTreeItem.vue'
import { useDismissiblePopover } from '@/composables/useDismissiblePopover'
import SearchBar from '@/components/common/SearchBar.vue'

const props = defineProps<{
  items: TVShow[]
  selected: TVSelection | null
  activeJob: Job | undefined
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  search: [query: string]
  select: [selection: TVSelection]
  scan: []
}>()

const query = shallowRef('')
const expandedShowIds = shallowReactive(new Set<number>())
const detailsByShowId = shallowReactive(new Map<number, TVShowDetail>())
const loadingShowIds = shallowReactive(new Set<number>())

type FilterType = 'all' | 'multi_season'
const filterMode = shallowRef<FilterType>('all')
type SortType = 'title' | 'year' | 'seasons'
const sortMode = shallowRef<SortType>('title')
const { open: showFilterMenu, container: filterDropdownRef } = useDismissiblePopover()

const isFilterActive = computed(() => filterMode.value !== 'all' || sortMode.value !== 'title')

const filteredItems = computed(() => {
  const normalized = query.value.trim().toLowerCase()
  let list = props.items

  if (normalized) {
    list = list.filter(show =>
      show.titleHint.toLowerCase().includes(normalized) ||
      show.relativePath.toLowerCase().includes(normalized)
    )
  }

  if (filterMode.value === 'multi_season') {
    list = list.filter(show => show.seasonCount > 1)
  }

  if (sortMode.value === 'year') {
    return [...list].sort((a, b) => (b.yearHint || 0) - (a.yearHint || 0))
  }
  if (sortMode.value === 'seasons') {
    return [...list].sort((a, b) => b.seasonCount - a.seasonCount)
  }
  return [...list].sort((a, b) => (a.titleHint || '').localeCompare(b.titleHint || ''))
})



async function ensureShowExpanded(showId: number) {
  expandedShowIds.add(showId)
  if (detailsByShowId.has(showId) || loadingShowIds.has(showId)) return
  loadingShowIds.add(showId)
  try {
    detailsByShowId.set(showId, await api.tvShowDetail(showId))
  } catch {
    // The inspector owns the visible error state for a stale/deleted URL selection.
  } finally {
    loadingShowIds.delete(showId)
  }
}

async function toggleShow(showId: number) {
  if (expandedShowIds.has(showId)) {
    expandedShowIds.delete(showId)
    return
  }
  await ensureShowExpanded(showId)
}

watch(
  () => props.selected?.showId,
  showId => {
    if (showId !== undefined) void ensureShowExpanded(showId)
  },
  { immediate: true },
)
</script>

<template>
  <aside class="catalog-panel" :aria-label="labels.tvShowList || 'TV show list'">
    <!-- Header with Search and Stats -->
    <div class="catalog-header">
      <!-- Search Input -->
      <SearchBar
        v-model="query"
        :placeholder="labels.searchShows || '搜索剧名、年份、路径...'"
        @search="emit('search', query)"
      />

      <!-- Count & Actions Bar -->
      <div class="catalog-toolbar">
        <div class="catalog-stats">
          <span class="count-main">{{ filteredItems.length }} {{ labels.tvShows || '电视剧' }}</span>
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
                    @click="filterMode = 'all'; showFilterMenu = false"
                  >
                    <span>{{ labels.filterAll || 'All' }}</span>
                    <span class="popover-count font-code">{{ items.length }}</span>
                  </button>
                  <button
                    type="button"
                    class="popover-item"
                    :class="{ active: filterMode === 'multi_season' }"
                    @click="filterMode = 'multi_season'; showFilterMenu = false"
                  >
                    <span>多季剧集</span>
                  </button>
                </div>

                <div class="popover-divider"></div>

                <div class="popover-section">
                  <span class="popover-label">{{ labels.sortMovies || 'Sort by' }}</span>
                  <button
                    type="button"
                    class="popover-item"
                    :class="{ active: sortMode === 'title' }"
                    @click="sortMode = 'title'; showFilterMenu = false"
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
                    @click="sortMode = 'year'; showFilterMenu = false"
                  >
                    <span>{{ labels.sortYear || 'Year (Newest)' }}</span>
                    <svg v-if="sortMode === 'year'" class="check-icon" viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
                      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                    </svg>
                  </button>
                  <button
                    type="button"
                    class="popover-item"
                    :class="{ active: sortMode === 'seasons' }"
                    @click="sortMode = 'seasons'; showFilterMenu = false"
                  >
                    <span>按季数最多</span>
                    <svg v-if="sortMode === 'seasons'" class="check-icon" viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
                      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                    </svg>
                  </button>
                </div>
              </div>
            </Transition>
          </div>

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
        <p class="job-title">{{ labels.scanningTvLibrary || 'Scanning Library' }}</p>
        <p class="job-desc">{{ activeJob.message || labels.loading }}</p>
      </div>
    </div>

    <!-- TV Shows List / Tree -->
    <div class="catalog-list">
      <TVShowTreeItem
        v-for="show in filteredItems"
        :key="show.id"
        :show="show"
        :detail="detailsByShowId.get(show.id)"
        :expanded="expandedShowIds.has(show.id)"
        :selected="selected"
        :labels="labels"
        @toggle="toggleShow"
        @select="emit('select', $event)"
      />
      <div v-if="filteredItems.length === 0" class="catalog-empty">
        <p>{{ labels.noShows || 'No TV shows found.' }}</p>
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

.toolbar-actions {
  display: flex;
  gap: 4px;
}

.filter-dropdown-wrap {
  position: relative;
  display: inline-flex;
}

.icon-action-btn {
  position: relative;
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

.active-dot {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--primary, #c0c1ff);
  box-shadow: 0 0 6px var(--primary, #c0c1ff);
}

.filter-popover {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  width: 190px;
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5), 0 0 1px rgba(255, 255, 255, 0.1);
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
  transition: all 0.1s ease;
}

.popover-item:hover {
  background: var(--surface-container-highest, #2e3447);
  color: #ffffff;
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

.check-icon {
  color: var(--primary, #c0c1ff);
}

.popover-divider {
  height: 1px;
  background: var(--outline-variant, #2e3447);
  margin: 6px 0;
}

.dropdown-fade-enter-active,
.dropdown-fade-leave-active {
  transition: all 0.15s cubic-bezier(0.16, 1, 0.3, 1);
}

.dropdown-fade-enter-from,
.dropdown-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.97);
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

.catalog-empty {
  padding: 3rem 1rem;
  text-align: center;
  color: var(--outline, #908fa0);
  font-size: 13px;
}
</style>
