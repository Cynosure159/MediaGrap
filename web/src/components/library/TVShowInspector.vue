<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue'
import * as api from '@/api/library'

const props = defineProps<{
  showId: number | null
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  close: []
}>()

const detail = shallowRef<api.TVShowDetail | null>(null)
const isLoading = shallowRef(false)
const error = shallowRef<string | null>(null)
const selectedEpisodeId = shallowRef<number | null>(null)

async function loadDetail(id: number) {
  isLoading.value = true
  error.value = null
  try {
    detail.value = await api.tvShowDetail(id)
    if (detail.value?.episodes.length) {
      selectedEpisodeId.value = detail.value.episodes[0].id
    }
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : props.labels.errorLoadShow
  } finally {
    isLoading.value = false
  }
}

watch(() => props.showId, id => {
  if (id) loadDetail(id)
  else detail.value = null
}, { immediate: true })

const seasonsMap = computed(() => {
  if (!detail.value) return new Map<number, api.TVEpisode[]>()
  const map = new Map<number, api.TVEpisode[]>()
  for (const ep of detail.value.episodes) {
    const list = map.get(ep.seasonNumber) ?? []
    list.push(ep)
    map.set(ep.seasonNumber, list)
  }
  return map
})

function formatEpisodeCode(ep: api.TVEpisode): string {
  const s = String(ep.seasonNumber).padStart(2, '0')
  const e = String(ep.episodeStart).padStart(2, '0')
  return `S${s}E${e}`
}
</script>

<template>
  <main class="inspector-workspace" :aria-label="labels.tvShowDetails">
    <!-- Top Action Bar -->
    <header class="inspector-toolbar">
      <nav class="workshop-tabs">
        <button class="tab-btn tab-btn--active" type="button">{{ labels.episodesAndSeasons }}</button>
        <button class="tab-btn" type="button">{{ labels.artworkTab }}</button>
        <button class="tab-btn" type="button">{{ labels.castTab }}</button>
      </nav>

      <div v-if="detail" class="toolbar-actions">
        <!-- Close Mobile Button -->
        <button class="mobile-close-btn" :title="labels.backToList" type="button" @click="emit('close')">
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
            <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
          </svg>
        </button>
        <button class="btn btn-primary" type="button">
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
          </svg>
          {{ labels.batchScrapeAll }}
        </button>
      </div>
    </header>

    <!-- Error Alert -->
    <div v-if="error" class="inspector-error">{{ error }}</div>

    <!-- Empty State -->
    <div v-if="!showId" class="no-selection-state">
      <svg class="no-sel-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <rect x="2" y="4" width="20" height="16" rx="2" />
        <path d="M8 4v16M16 4v16M2 12h20" />
      </svg>
      <p>{{ labels.selectShow || 'Select a TV show to review its indexed episodes.' }}</p>
    </div>

    <!-- Loading State -->
    <div v-else-if="isLoading" class="loading-state">
      <div class="loading-spinner"></div>
      <p>{{ labels.loading }}</p>
    </div>

    <!-- Content Workspace -->
    <div v-else-if="detail" class="inspector-content">
      <!-- Show Hero Header -->
      <div class="hero-banner">
        <div class="title-cluster">
          <h1 class="main-title">{{ detail.show.titleHint }}</h1>
        </div>

        <div class="meta-strip">
          <span class="meta-item font-code">{{ detail.show.yearHint ? `${detail.show.yearHint}` : labels.tvSeries }}</span>
          <span class="meta-dot"></span>
          <span class="spec-pill font-code">{{ detail.show.seasonCount }} {{ labels.seasons }}</span>
          <span class="spec-pill font-code">{{ detail.show.episodeCount }} {{ labels.episodes }}</span>
          <span class="meta-dot"></span>
          <span class="meta-item font-code">{{ detail.show.relativePath }}</span>

          <div class="rating-badge">
            <span class="star-score">★ {{ labels.ratingUnavailable }}</span>
            <span class="provider-tag">TMDb</span>
          </div>
        </div>
      </div>

      <!-- Seasons & Episodes Accordion -->
      <div class="seasons-container">
        <section
          v-for="[seasonNum, episodes] in seasonsMap"
          :key="seasonNum"
          class="season-card"
        >
          <header class="season-header">
            <div class="season-title-box">
              <span class="season-badge">{{ labels.season }} {{ seasonNum }}</span>
              <span class="season-count">{{ episodes.length }} {{ labels.episodesCount }}</span>
            </div>
            <span class="spec-pill font-code">{{ labels.scrapedStatusUnavailable }}</span>
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
                @click="selectedEpisodeId = ep.id"
              >
                <td class="col-code font-code">{{ formatEpisodeCode(ep) }}</td>
                <td class="col-title">
                  <div class="ep-title-main">{{ ep.titleHint || ep.relativePath.split('/').pop() }}</div>
                  <div class="ep-path font-code">{{ ep.relativePath }}</div>
                </td>
                <td class="col-res">
                  <span class="spec-badge">{{ labels.qualityUnavailable }}</span>
                </td>
                <td class="col-status">
                  <span class="dot dot-off" :title="labels.nfoStatusUnavailable"></span>
                </td>
              </tr>
            </tbody>
          </table>
        </section>
      </div>
    </div>
  </main>
</template>

<style scoped>
.inspector-workspace {
  flex: 1;
  height: 100vh;
  background: var(--surface-base, #0c1324);
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.inspector-toolbar {
  height: var(--toolbar-height, 40px);
  padding: 0 var(--pane-padding, 12px);
  background: var(--surface-dim, #0c1324);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.workshop-tabs {
  display: flex;
  align-items: center;
  gap: 16px;
  height: 100%;
}

.tab-btn {
  height: 100%;
  padding: 0 4px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}

.tab-btn--active {
  color: var(--primary, #c0c1ff);
  border-bottom-color: var(--primary, #c0c1ff);
  font-weight: 700;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mobile-close-btn {
  display: none;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  background: transparent;
  border: 1px solid var(--outline-variant);
  color: var(--on-surface-variant);
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.inspector-error {
  padding: 8px 12px;
  background: var(--error-container, #93000a);
  color: var(--on-error-container, #ffdad6);
  font-size: 12px;
}

.no-selection-state,
.loading-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--outline, #908fa0);
  font-size: 13px;
}

.no-sel-icon {
  width: 48px;
  height: 48px;
  opacity: 0.3;
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--primary, #c0c1ff);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

.inspector-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--pane-padding, 12px);
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
}

.main-title {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.meta-strip {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  flex-wrap: wrap;
}

.font-code {
  font-family: var(--font-data);
}

.meta-dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--outline-variant, #2e3447);
}

.rating-badge {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm);
  font-size: 11px;
}

.star-score {
  font-weight: 700;
  color: var(--tertiary, #ffb95f);
}

.score-denom {
  font-size: 9px;
  color: var(--outline, #908fa0);
}

.provider-tag {
  font-family: var(--font-data);
  font-size: 9px;
  background: var(--primary-container, #6366f1);
  color: #fff;
  padding: 1px 4px;
  border-radius: 2px;
  margin-left: 4px;
}

.seasons-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.season-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md);
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
  color: var(--on-surface, #dce1fb);
}

.season-count {
  font-size: 11px;
  color: var(--outline, #908fa0);
}

.text-ok {
  color: var(--secondary, #4edea3);
}

.episode-table {
  width: 100%;
  border-collapse: collapse;
}

.episode-table th {
  text-align: left;
  font-size: 9px;
  font-weight: 700;
  color: var(--outline, #908fa0);
  padding: 6px 12px;
  background: var(--surface-container-lowest, #070d1f);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  letter-spacing: 0.05em;
}

.episode-table td {
  padding: 8px 12px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.ep-row {
  cursor: pointer;
  transition: all 0.12s ease;
}

.ep-row:hover {
  background: var(--surface-container-high, #23293c);
}

.ep-row--active {
  background: var(--surface-container-highest, #2e3447);
}

.col-code {
  width: 70px;
  font-size: 11px;
  font-weight: 700;
  color: var(--primary, #c0c1ff);
}

.col-title {
  min-width: 0;
}

.ep-title-main {
  font-size: 12px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
}

.ep-path {
  font-size: 10px;
  color: var(--outline, #908fa0);
  margin-top: 1px;
}

.col-res {
  width: 80px;
}

.col-status {
  width: 40px;
  text-align: center;
}

@media (max-width: 760px) {
  .mobile-close-btn {
    display: inline-flex;
  }
}
</style>
