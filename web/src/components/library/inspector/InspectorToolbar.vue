<script setup lang="ts">
export type InspectorTab = 'overview' | 'artwork' | 'cast' | 'nfo' | 'files'

defineProps<{
  activeTab: InspectorTab
  hasDetail: boolean
  isWritable: boolean
  isSaving: boolean
  isScraping: boolean
  isLocked: boolean
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  selectTab: [tab: InspectorTab]
  saveNfo: []
  scrape: []
  toggleLock: []
  close: []
}>()
</script>

<template>
  <header class="inspector-toolbar">
    <!-- Tabs Navigation -->
    <nav class="workshop-tabs">
      <button
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === 'overview' }"
        type="button"
        @click="emit('selectTab', 'overview')"
      >
        {{ labels.overview }}
      </button>
      <button
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === 'artwork' }"
        type="button"
        @click="emit('selectTab', 'artwork')"
      >
        {{ labels.artworkTab }}
      </button>
      <button
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === 'cast' }"
        type="button"
        @click="emit('selectTab', 'cast')"
      >
        {{ labels.castTab }}
      </button>
      <button
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === 'nfo' }"
        type="button"
        @click="emit('selectTab', 'nfo')"
      >
        {{ labels.nfoRaw }}
      </button>
      <button
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === 'files' }"
        type="button"
        @click="emit('selectTab', 'files')"
      >
        {{ labels.fileAudit }}
      </button>
    </nav>

    <!-- Action Buttons -->
    <div v-if="hasDetail" class="toolbar-actions">
      <!-- Close Mobile Button -->
      <button class="mobile-close-btn" :title="labels.backToList" type="button" @click="emit('close')">
        <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
          <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
        </svg>
      </button>

      <button
        class="btn btn-success"
        :disabled="isSaving || !isWritable"
        type="button"
        @click="emit('saveNfo')"
      >
        <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
          <path d="M17 3H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V7l-4-4zm-5 16c-1.66 0-3-1.34-3-3s1.34-3 3-3 3 1.34 3 3-1.34 3-3 3zm3-10H5V5h10v4z"/>
        </svg>
        {{ labels.saveAndWriteNfo }}
      </button>

      <button
        class="btn btn-primary"
        :disabled="isScraping"
        type="button"
        @click="emit('scrape')"
      >
        <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
          <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
        </svg>
        {{ isScraping ? labels.searching : labels.scrape }}
      </button>

      <div class="divider-v"></div>

      <button
        class="icon-btn"
        :class="{ active: isLocked }"
        :title="isLocked ? labels.metadataLocked : labels.lockMetadata"
        type="button"
        @click="emit('toggleLock')"
      >
        <svg viewBox="0 0 24 24" fill="currentColor" width="15" height="15">
          <path d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z"/>
        </svg>
      </button>
    </div>
  </header>
</template>

<style scoped>
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
  transition: all 0.15s ease;
}

.tab-btn:hover {
  color: var(--primary, #c0c1ff);
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

.divider-v {
  width: 1px;
  height: 16px;
  background: var(--outline-variant, #2e3447);
  margin: 0 2px;
}

.icon-btn {
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

.icon-btn:hover {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface, #dce1fb);
}

.icon-btn.active {
  color: var(--tertiary, #ffb95f);
}

@media (max-width: 700px) {
  .mobile-close-btn {
    display: flex;
  }
}
</style>
