<script setup lang="ts">
import { computed, shallowReactive, shallowRef } from 'vue'
import * as api from '@/api/library'
import type { Job, TVSelection, TVShow, TVShowDetail } from '@/api/library'
import TVShowTreeItem from './TVShowTreeItem.vue'

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

const filteredItems = computed(() => {
  const normalized = query.value.trim().toLowerCase()
  return normalized
    ? props.items.filter(show =>
        show.titleHint.toLowerCase().includes(normalized) ||
        show.relativePath.toLowerCase().includes(normalized)
      )
    : props.items
})

async function toggleShow(showId: number) {
  if (expandedShowIds.has(showId)) {
    expandedShowIds.delete(showId)
    return
  }
  expandedShowIds.add(showId)
  if (detailsByShowId.has(showId) || loadingShowIds.has(showId)) return
  loadingShowIds.add(showId)
  try {
    detailsByShowId.set(showId, await api.tvShowDetail(showId))
  } finally {
    loadingShowIds.delete(showId)
  }
}
</script>

<template>
  <aside class="catalog-panel" :aria-label="labels.tvShowList || 'TV show list'">
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
          :placeholder="labels.searchShows || '搜索剧名、年份、路径...'"
          @keydown.enter.prevent="emit('search', query)"
        />
      </div>

      <!-- Count & Actions Bar -->
      <div class="catalog-toolbar">
        <div class="catalog-stats">
          <span class="count-main">{{ items.length }} {{ labels.tvShows || '电视剧' }}</span>
        </div>
        <div class="toolbar-actions">
          <button
            class="icon-action-btn"
            :title="labels.refresh || 'Scan Library'"
            type="button"
            @click="emit('scan')"
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
