<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import type { Job, TVShow } from '@/api/library'

const props = defineProps<{
  items: TVShow[]
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

function submitSearch() {
  emit('search', query.value)
}

const filteredItems = computed(() => {
  if (!query.value.trim()) return props.items
  const q = query.value.toLowerCase()
  return props.items.filter(s =>
    s.titleHint.toLowerCase().includes(q) ||
    s.relativePath.toLowerCase().includes(q)
  )
})
</script>

<template>
  <aside class="catalog-panel" :aria-label="labels.tvShowList">
    <!-- Header with Search -->
    <div class="catalog-header">
      <div class="search-box">
        <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8" />
          <line x1="21" y1="21" x2="16.65" y2="16.65" />
        </svg>
        <input
          v-model="query"
          type="text"
          class="search-input"
          :placeholder="labels.searchShows"
          @keydown.enter.prevent="submitSearch"
        />
      </div>

      <div class="catalog-toolbar">
        <div class="catalog-stats">
          <span class="count-main">{{ items.length }} {{ labels.tvShows }}</span>
        </div>
        <div class="toolbar-actions">
          <button class="icon-action-btn" :title="labels.refresh" type="button" @click="emit('scan')">
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
        <p class="job-title">{{ labels.scanningTvLibrary }}</p>
        <p class="job-desc">{{ activeJob.message || labels.loading }}</p>
      </div>
    </div>

    <!-- TV Shows List -->
    <div class="catalog-list">
      <div
        v-for="show in filteredItems"
        :key="show.id"
        class="media-row"
        :class="{ 'media-row--active': selectedId === show.id }"
        @click="emit('select', show.id)"
      >
        <!-- Poster Thumbnail -->
        <div class="thumb-box">
          <svg viewBox="0 0 24 24" fill="currentColor" width="18" height="18" opacity="0.3">
            <path d="M21 3H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h5v2h8v-2h5c1.1 0 1.99-.9 1.99-2L23 5c0-1.1-.9-2-2-2zm0 14H3V5h18v12z"/>
          </svg>
          <span class="res-badge">TV</span>
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
            <span class="year-txt">{{ show.yearHint ?? '—' }}</span>
            <div class="spec-pills-wrap">
              <span class="spec-badge">{{ show.seasonCount }} {{ labels.seasons }}</span>
              <span class="spec-badge">{{ show.episodeCount }} {{ labels.episodes }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="filteredItems.length === 0" class="catalog-empty">
        <p>{{ labels.noShows }}</p>
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
  padding: 8px 10px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-dim, #0c1324);
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
  left: 8px;
  top: 50%;
  transform: translateY(-50%);
  width: 14px;
  height: 14px;
  color: var(--on-surface-variant, #c7c4d7);
  pointer-events: none;
}

.search-input {
  width: 100%;
  height: 30px;
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  font-size: 12px;
  padding: 0 8px 0 28px;
  transition: all 0.15s ease;
}

.search-input:focus {
  border-color: var(--primary, #c0c1ff);
  outline: none;
}

.catalog-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.catalog-stats {
  display: flex;
  align-items: baseline;
  gap: 4px;
}

.count-main {
  font-size: 12px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
}

.toolbar-actions {
  display: flex;
  gap: 2px;
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

.active-job-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: var(--surface-container-high, #23293c);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  font-size: 11px;
}

.job-spinner {
  width: 10px;
  height: 10px;
  border: 2px solid var(--tertiary, #ffb95f);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

.job-info { flex: 1; min-width: 0; }
.job-title { margin: 0; font-weight: 700; color: var(--tertiary, #ffb95f); }
.job-desc { margin: 0; color: var(--on-surface-variant, #c7c4d7); font-size: 10px; }

.catalog-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.media-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px;
  border-radius: var(--radius-sm, 0.25rem);
  border-left: 2px solid transparent;
  cursor: pointer;
  transition: all 0.12s ease;
}

.media-row:hover {
  background: var(--surface-container-high, #23293c);
}

.media-row--active {
  background: var(--surface-container-highest, #2e3447);
  border-left-color: var(--primary, #c0c1ff);
}

.thumb-box {
  position: relative;
  width: 36px;
  height: 50px;
  border-radius: 2px;
  overflow: hidden;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.res-badge {
  position: absolute;
  bottom: 0;
  right: 0;
  background: var(--surface-bright, #33394c);
  color: var(--on-surface, #dce1fb);
  font-family: var(--font-data);
  font-size: 7px;
  font-weight: 700;
  padding: 1px 2px;
}

.row-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.title-primary {
  font-size: 12px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.title-sub {
  font-size: 10px;
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
  font-family: var(--font-data);
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.spec-pills-wrap {
  display: flex;
  gap: 3px;
}

.catalog-empty {
  padding: 2rem 1rem;
  text-align: center;
  color: var(--outline, #908fa0);
  font-size: 12px;
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
