<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { Candidate } from '@/api/types'
import * as api from '@/api/library'

const props = defineProps<{
  itemId: number
  itemTitle: string
  itemYear?: number | null
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  select: [candidate: Candidate]
  close: []
}>()

const searchQuery = ref(props.itemTitle || '')
const isSearching = ref(false)
const isApplying = ref(false)
const selectedCandidateId = ref<string | null>(null)
const candidatesList = ref<Candidate[]>([])
const searchError = ref<string | null>(null)

async function performSearch() {
  if (!props.itemId) return
  isSearching.value = true
  searchError.value = null
  try {
    const res = await api.candidates(props.itemId, searchQuery.value.trim())
    candidatesList.value = res.items || []
  } catch (err) {
    searchError.value = err instanceof Error ? err.message : (props.labels.errorSearchCandidates || 'Failed to search candidates')
  } finally {
    isSearching.value = false
  }
}

function handleChoose(candidate: Candidate) {
  if (isApplying.value) return
  selectedCandidateId.value = candidate.id
  isApplying.value = true
  emit('select', candidate)
}

onMounted(() => {
  performSearch()
})
</script>

<template>
  <div class="modal-backdrop" @click.self="emit('close')" @keydown.esc="emit('close')">
    <div class="scraper-modal" role="dialog" aria-modal="true">
      <!-- ── Modal Header ─────────────────────────────────────────── -->
      <header class="modal-header">
        <div class="header-left">
          <div class="header-icon-box">
            <svg viewBox="0 0 24 24" fill="currentColor" width="20" height="20">
              <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
            </svg>
          </div>
          <div>
            <h2 class="modal-title">{{ labels.scraperMatchTitle || '刮削匹配 (TMDb)' }}</h2>
            <p class="modal-subtitle" :title="itemTitle">{{ itemTitle }}</p>
          </div>
        </div>

        <div class="header-right">
          <span class="provider-badge font-code">TMDb</span>
          <button class="modal-close-btn" :title="labels.close || 'Close'" type="button" @click="emit('close')">
            <svg viewBox="0 0 24 24" fill="currentColor" width="18" height="18">
              <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
            </svg>
          </button>
        </div>
      </header>

      <!-- ── Search Box ───────────────────────────────────────────── -->
      <div class="search-bar-wrap">
        <div class="search-input-box">
          <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            class="modal-search-input"
            :placeholder="labels.searchKeyword || '输入电影名称或关键词搜索...'"
            @keydown.enter.prevent="performSearch"
          />
        </div>
        <button class="btn btn-primary search-action-btn" :disabled="isSearching" type="button" @click="performSearch">
          <svg v-if="isSearching" viewBox="0 0 24 24" fill="currentColor" width="14" height="14" class="spin-slow">
            <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
          </svg>
          <span>{{ isSearching ? (labels.searching || '搜索中...') : (labels.search || '搜索') }}</span>
        </button>
      </div>

      <!-- ── Error Alert ──────────────────────────────────────────── -->
      <div v-if="searchError" class="modal-error-alert">
        {{ searchError }}
      </div>

      <!-- ── Results List / Content Canvas ────────────────────────── -->
      <div class="modal-body-content">
        <!-- Applying State Overlay -->
        <div v-if="isApplying" class="applying-overlay">
          <div class="applying-box">
            <div class="applying-spinner"></div>
            <p class="applying-text">{{ labels.applyingCandidate || '正在抓取元数据并直接覆盖保存与写入 NFO...' }}</p>
          </div>
        </div>

        <!-- Searching Skeleton / Spinner -->
        <div v-if="isSearching" class="results-loading">
          <div class="loading-spinner"></div>
          <p class="loading-hint">{{ labels.searchingCandidates || '正在从 TMDb 查询匹配的影视作品...' }}</p>
        </div>

        <!-- Results List -->
        <div v-else-if="candidatesList.length > 0" class="candidates-container">
          <div class="results-count-bar">
            <span>{{ labels.foundMatches || '共找到' }} <strong>{{ candidatesList.length }}</strong> {{ labels.matchingItems || '个匹配结果，点击即可直接覆盖' }}</span>
          </div>

          <div class="candidates-grid">
            <div
              v-for="candidate in candidatesList"
              :key="candidate.id"
              class="candidate-card group"
              :class="{ 'candidate-card--selected': selectedCandidateId === candidate.id }"
              tabindex="0"
              @click="handleChoose(candidate)"
              @keydown.enter="handleChoose(candidate)"
            >
              <!-- Poster Thumbnail -->
              <div class="cand-poster-box">
                <img
                  v-if="candidate.posterUrl"
                  :src="candidate.posterUrl"
                  :alt="candidate.title"
                  class="cand-poster-img"
                  loading="lazy"
                />
                <div v-else class="cand-poster-fallback">
                  <svg viewBox="0 0 24 24" fill="currentColor" width="24" height="24" opacity="0.3">
                    <path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
                  </svg>
                </div>
              </div>

              <!-- Candidate Metadata -->
              <div class="cand-details">
                <div class="cand-header-row">
                  <span class="cand-title">{{ candidate.title }}</span>
                  <span v-if="candidate.year" class="cand-year-badge font-code">{{ candidate.year }}</span>
                </div>

                <div v-if="candidate.originalTitle && candidate.originalTitle !== candidate.title" class="cand-orig-title">
                  {{ candidate.originalTitle }}
                </div>

                <div class="cand-id-tag font-code">
                  TMDb #{{ candidate.id }}
                </div>

                <p v-if="candidate.overview" class="cand-overview">
                  {{ candidate.overview }}
                </p>
              </div>

              <!-- Select Action Button -->
              <div class="cand-action-col">
                <button
                  type="button"
                  class="btn btn-select"
                  :disabled="isApplying"
                  @click.stop="handleChoose(candidate)"
                >
                  <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
                    <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                  </svg>
                  <span>{{ labels.selectAndOverwrite || '选择并覆盖' }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Empty Results -->
        <div v-else class="results-empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="48" height="48" class="empty-icon">
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <p class="empty-title">{{ labels.noMatchesFound || '未找到匹配的电影' }}</p>
          <p class="empty-desc">{{ labels.tryRefineKeywords || '请尝试调整上方搜索框中的片名关键词或重新搜索。' }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(7, 13, 31, 0.85);
  backdrop-filter: blur(12px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  z-index: 100;
  animation: fadeIn 0.15s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.scraper-modal {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-xl, 0.75rem);
  width: 100%;
  max-width: 860px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7);
  overflow: hidden;
  position: relative;
}

/* ── Modal Header ─────────────────────────────────────────────────── */
.modal-header {
  padding: 14px 20px;
  background: var(--surface-dim, #0c1324);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.header-icon-box {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md, 0.375rem);
  background: var(--surface-container-high, #23293c);
  color: var(--primary, #c0c1ff);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.modal-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  line-height: 1.2;
}

.modal-subtitle {
  margin: 2px 0 0 0;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 450px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.provider-badge {
  background: #01b4e4;
  color: #ffffff;
  font-size: 10px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: var(--radius-sm, 0.25rem);
}

.modal-close-btn {
  background: transparent;
  border: none;
  color: var(--on-surface-variant, #c7c4d7);
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm, 0.25rem);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.15s ease;
}

.modal-close-btn:hover {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface, #dce1fb);
}

/* ── Search Bar ───────────────────────────────────────────────────── */
.search-bar-wrap {
  padding: 12px 20px;
  background: var(--surface-container-low, #151b2d);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.search-input-box {
  position: relative;
  flex: 1;
}

.search-icon {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  color: var(--on-surface-variant, #c7c4d7);
  pointer-events: none;
}

.modal-search-input {
  width: 100%;
  height: 36px;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  font-size: 13px;
  padding: 0 12px 0 34px;
}

.modal-search-input:focus {
  outline: none;
  border-color: var(--primary, #c0c1ff);
  box-shadow: 0 0 0 1px var(--primary, #c0c1ff);
}

.search-action-btn {
  height: 36px;
  padding: 0 18px;
  font-size: 13px;
}

.modal-error-alert {
  padding: 10px 20px;
  background: var(--error-container, #93000a);
  color: var(--on-error-container, #ffdad6);
  font-size: 12px;
}

/* ── Body Canvas ──────────────────────────────────────────────────── */
.modal-body-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  position: relative;
  min-height: 280px;
}

.results-count-bar {
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  margin-bottom: 12px;
}

.results-count-bar strong {
  color: var(--primary, #c0c1ff);
}

.candidates-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.candidate-card {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 10px 12px;
  display: flex;
  align-items: center;
  gap: 14px;
  cursor: pointer;
  transition: all 0.15s ease;
  user-select: none;
}

.candidate-card:hover {
  background: var(--surface-container-high, #23293c);
  border-color: var(--primary, #c0c1ff);
  transform: translateY(-1px);
}

.cand-poster-box {
  width: 52px;
  height: 76px;
  border-radius: var(--radius-sm, 0.25rem);
  overflow: hidden;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cand-poster-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.cand-details {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.cand-header-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cand-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  line-height: 1.2;
}

.cand-year-badge {
  background: var(--surface-container-highest, #2e3447);
  border: 1px solid var(--outline-variant, #464554);
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 10px;
  padding: 1px 5px;
  border-radius: var(--radius-sm, 0.25rem);
}

.cand-orig-title {
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  opacity: 0.8;
}

.cand-id-tag {
  font-size: 10px;
  color: var(--primary, #c0c1ff);
}

.cand-overview {
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  line-height: 1.4;
  margin: 2px 0 0 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.cand-action-col {
  flex-shrink: 0;
}

.btn-select {
  background: var(--secondary-container, #00a572);
  color: #ffffff;
  border: none;
  font-size: 12px;
  font-weight: 600;
  padding: 0 12px;
  height: 32px;
}

.btn-select:hover {
  background: var(--secondary, #4edea3);
  color: var(--on-secondary-container, #00311f);
}

/* ── Applying Overlay ─────────────────────────────────────────────── */
.applying-overlay {
  position: absolute;
  inset: 0;
  background: rgba(12, 19, 36, 0.9);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 20;
}

.applying-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  text-align: center;
}

.applying-spinner {
  width: 36px;
  height: 36px;
  border: 3px solid var(--secondary, #4edea3);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.applying-text {
  font-size: 14px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  margin: 0;
}

/* ── Loading / Empty States ───────────────────────────────────────── */
.results-loading,
.results-empty {
  padding: 48px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  text-align: center;
}

.loading-spinner {
  width: 30px;
  height: 30px;
  border: 3px solid var(--primary, #c0c1ff);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.loading-hint {
  color: var(--outline, #908fa0);
  font-size: 13px;
  margin: 0;
}

.empty-icon {
  color: var(--outline, #908fa0);
  opacity: 0.4;
}

.empty-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  margin: 0;
}

.empty-desc {
  font-size: 12px;
  color: var(--outline, #908fa0);
  margin: 0;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
