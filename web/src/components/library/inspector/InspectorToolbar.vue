<script setup lang="ts">
import type { InspectorTab } from '@/router'

export type { InspectorTab }

defineProps<{
  activeTab: InspectorTab
  hasDetail: boolean
  isWritable: boolean
  isEditing: boolean
  mutationsBlocked?: boolean
  isSaving: boolean
  isScraping: boolean
  isLocked: boolean
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  selectTab: [tab: InspectorTab]
  toggleEdit: []
  cancelEdit: []
  saveEdit: []
  scrape: []
  toggleLock: []
  close: []
}>()
</script>

<template>
  <header class="inspector-header-container">
    <!-- ── Row 1: Main Action Toolbar ─────────────────────────────── -->
    <div class="toolbar-top-row">
      <div class="top-row-left">
        <!-- Mobile/Compact Back to List Button -->
        <button
          class="mobile-back-btn"
          :title="labels.backToList || '返回列表'"
          type="button"
          @click="emit('close')"
        >
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/>
          </svg>
          <span class="back-txt">{{ labels.backToList || '列表' }}</span>
        </button>
      </div>

      <!-- Action Buttons (Right) -->
      <div v-if="hasDetail" class="top-row-actions">
        <!-- ── View Mode: Edit Button ── -->
        <template v-if="!isEditing">
          <button
            v-if="isWritable"
            class="btn btn-outline"
            type="button"
            @click="emit('toggleEdit')"
          >
            <svg viewBox="0 0 24 24" fill="currentColor" width="13" height="13">
              <path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/>
            </svg>
            <span>{{ labels.edit || '编辑' }}</span>
          </button>
        </template>

        <!-- ── Edit Mode: Cancel & Save Buttons ── -->
        <template v-else>
          <button
            class="btn btn-ghost"
            :disabled="isSaving"
            type="button"
            @click="emit('cancelEdit')"
          >
            <span>{{ labels.cancel || '取消' }}</span>
          </button>

          <button
            class="btn btn-save"
            :disabled="isSaving || !isWritable"
            type="button"
            @click="emit('saveEdit')"
          >
            <svg v-if="!isSaving" viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
              <path d="M17 3H5c-1.1 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V7l-4-4zm-5 16c-1.66 0-3-1.34-3-3s1.34-3 3-3 3 1.34 3 3-1.34 3-3 3zm3-10H5V5h10v4z"/>
            </svg>
            <span v-else class="toolbar-spinner"></span>
            <span>{{ isSaving ? (labels.saving || '保存中...') : (labels.save || '保存') }}</span>
          </button>
        </template>

        <!-- Scrape Button (Opens ScraperModal) -->
        <button
          class="btn btn-scrape"
          :disabled="isScraping || mutationsBlocked"
          type="button"
          @click="emit('scrape')"
        >
          <svg viewBox="0 0 24 24" fill="currentColor" width="13" height="13">
            <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
          </svg>
          <span>{{ labels.scrape || '刮削' }}</span>
        </button>

        <!-- Lock Metadata Button -->
        <button
          class="btn btn-outline btn-lock"
          :class="{ 'btn-lock--active': isLocked }"
          :title="isLocked ? (labels.metadataLocked || '元数据已锁定') : (labels.lockMetadata || '锁定元数据')"
          type="button"
          @click="emit('toggleLock')"
        >
          <svg viewBox="0 0 24 24" fill="currentColor" width="13" height="13">
            <path v-if="!isLocked" d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z"/>
            <path v-else d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6h2c0-1.66 1.34-3 3-3s3 1.34 3 3v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2z"/>
          </svg>
          <span>{{ isLocked ? (labels.locked || '已锁定') : (labels.lock || '锁定') }}</span>
        </button>
      </div>
    </div>

    <!-- ── Row 2: Workshop Tabs Navigation (Full Width) ───────────── -->
    <div class="toolbar-tabs-row">
      <nav class="workshop-tabs">
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': activeTab === 'overview' }"
          type="button"
          @click="emit('selectTab', 'overview')"
        >
          <svg class="tab-icon" viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"/>
          </svg>
          <span>{{ labels.overview || '概览' }}</span>
        </button>
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': activeTab === 'artwork' }"
          type="button"
          @click="emit('selectTab', 'artwork')"
        >
          <svg class="tab-icon" viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/>
          </svg>
          <span>{{ labels.artworkTab || '图片资源' }}</span>
        </button>
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': activeTab === 'cast' }"
          type="button"
          @click="emit('selectTab', 'cast')"
        >
          <svg class="tab-icon" viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/>
          </svg>
          <span>{{ labels.castTab || '演职人员' }}</span>
        </button>
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': activeTab === 'nfo' }"
          type="button"
          @click="emit('selectTab', 'nfo')"
        >
          <svg class="tab-icon" viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M9.4 16.6L4.8 12l4.6-4.6L8 6l-6 6 6 6 1.4-1.4zm5.2 0l4.6-4.6-4.6-4.6L16 6l6 6-6 6-1.4-1.4z"/>
          </svg>
          <span>{{ labels.nfoRaw || 'NFO 原文' }}</span>
        </button>
        <button
          class="tab-btn"
          :class="{ 'tab-btn--active': activeTab === 'files' }"
          type="button"
          @click="emit('selectTab', 'files')"
        >
          <svg class="tab-icon" viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"/>
          </svg>
          <span>{{ labels.fileAudit || '文件审计' }}</span>
        </button>
      </nav>
    </div>
  </header>
</template>

<style scoped>
.inspector-header-container {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  z-index: 20;
  width: 100%;
  max-width: 100%;
  overflow-x: hidden;
  box-sizing: border-box;
}

/* ── Row 1: Action Toolbar ────────────────────────────────────────── */
.toolbar-top-row {
  height: 42px;
  min-height: 42px;
  padding: 0 14px;
  background: var(--surface-dim, #0c1324);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  overflow-x: auto;
  scrollbar-width: none;
}

.toolbar-top-row::-webkit-scrollbar {
  display: none;
}

.top-row-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.mobile-back-btn {
  display: none;
  height: 28px;
  padding: 0 10px;
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  color: var(--on-surface, #dce1fb);
  align-items: center;
  gap: 5px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  transition: all 0.15s ease;
  flex-shrink: 0;
}

.mobile-back-btn:hover {
  background: var(--surface-container-highest, #2e3447);
  color: var(--primary, #c0c1ff);
}

.top-row-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
  flex-shrink: 0;
}

/* ── Row 2: Tabs Row (Full Width Spanning) ────────────────────────── */
.toolbar-tabs-row {
  height: 38px;
  min-height: 38px;
  padding: 0 8px;
  background: var(--surface-base, #0c1324);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  align-items: center;
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
}

.toolbar-tabs-row::-webkit-scrollbar {
  display: none;
}

.workshop-tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: max-content;
  width: 100%;
  height: 100%;
}

.tab-btn {
  flex: 1;
  height: 100%;
  padding: 0 12px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 12.5px;
  font-weight: 500;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  white-space: nowrap;
  transition: all 0.15s ease;
}

.tab-icon {
  opacity: 0.6;
  transition: opacity 0.15s ease;
  flex-shrink: 0;
}

.tab-btn:hover {
  color: var(--primary-fixed, #e1e0ff);
  background: rgba(255, 255, 255, 0.02);
}

.tab-btn:hover .tab-icon {
  opacity: 0.9;
}

.tab-btn--active {
  color: var(--primary, #c0c1ff);
  border-bottom-color: var(--primary, #c0c1ff);
  font-weight: 600;
  background: rgba(192, 193, 255, 0.04);
}

.tab-btn--active .tab-icon {
  opacity: 1;
  color: var(--primary, #c0c1ff);
}

/* ── Buttons ──────────────────────────────────────────────────────── */
.btn-save {
  height: 28px;
  padding: 0 12px;
  background: var(--secondary-container, #00a572);
  color: #ffffff;
  border: none;
  border-radius: var(--radius-sm, 0.25rem);
  font-size: 12px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.btn-save:hover:not(:disabled) {
  background: var(--secondary, #4edea3);
  color: var(--on-secondary-container, #00311f);
  box-shadow: 0 0 12px rgba(78, 222, 163, 0.25);
}

.btn-save:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-scrape {
  height: 28px;
  padding: 0 12px;
  background: var(--primary, #c0c1ff);
  color: var(--on-primary, #1000a9);
  border: none;
  border-radius: var(--radius-sm, 0.25rem);
  font-size: 12px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  white-space: nowrap;
}

.btn-scrape:hover:not(:disabled) {
  background: var(--primary-fixed, #e1e0ff);
  box-shadow: 0 0 12px rgba(192, 193, 255, 0.3);
}

.btn-scrape:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-lock {
  height: 28px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 500;
  gap: 5px;
  transition: all 0.15s ease;
}

.btn-lock--active {
  background: rgba(255, 185, 95, 0.12);
  border-color: var(--tertiary, #ffb95f);
  color: var(--tertiary, #ffb95f);
}

.toolbar-spinner {
  width: 12px;
  height: 12px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  display: inline-block;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 700px) {
  .mobile-back-btn {
    display: inline-flex;
  }
}
</style>
