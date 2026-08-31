<script setup lang="ts">
import { ref } from 'vue'
import type { MediaItem } from '@/api/types'

const props = defineProps<{
  item: MediaItem
  labels: Record<string, string>
}>()

const patternInput = ref('${title} (${year})/${title} (${year}) - [${resolution}]')
const availableTokens = [
  '${title}',
  '${originalTitle}',
  '${year}',
  '${resolution}',
  '${videoCodec}',
  '${audioCodec}',
  '${imdbId}',
]
</script>

<template>
  <section class="files-workshop-view">
    <div class="workshop-header">
      <div class="header-left">
        <h2>{{ labels.filesRenamePlanner || 'Files & Rename Planner' }}</h2>
        <span class="sub-label">Preview safe rename operations, sidecars and directory structure</span>
      </div>
      <button class="btn btn-primary" type="button">
        <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
          <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
        </svg>
        {{ labels.runDryRun || 'Dry-Run Rename Plan' }}
      </button>
    </div>

    <!-- Template Pattern Engine Section -->
    <div class="rename-engine-card">
      <div class="card-header-bar">
        <span class="header-title">Naming Pattern Template Engine</span>
        <span class="preset-tag font-code">Kodi Standard</span>
      </div>
      <div class="card-body">
        <div class="pattern-field">
          <label class="field-label font-code">PATTERN TEMPLATE</label>
          <input v-model="patternInput" type="text" class="pattern-input font-code" />
        </div>
        <div class="tokens-row">
          <span class="tokens-label">AVAILABLE TOKENS:</span>
          <div class="tokens-list">
            <span
              v-for="tok in availableTokens"
              :key="tok"
              class="token-pill font-code"
              @click="patternInput += ' ' + tok"
            >
              {{ tok }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Current Structure & Sidecars List -->
    <div class="structure-card">
      <div class="card-header-bar">
        <span class="header-title">Current File & Sidecar Assets</span>
        <span class="item-count font-code">{{ 1 + (item.sidecars?.length || 0) }} files</span>
      </div>

      <div class="file-audit-list">
        <!-- Main Media File -->
        <div class="audit-item">
          <span class="spec-badge video-badge font-code">VIDEO</span>
          <span class="audit-path font-code">{{ item.relativePath }}</span>
          <span class="audit-status text-ok font-code">MATCHED</span>
        </div>

        <!-- Sidecars -->
        <template v-if="item.sidecars && item.sidecars.length > 0">
          <div v-for="sidecar in item.sidecars" :key="sidecar.relativePath" class="audit-item">
            <span class="spec-badge font-code">{{ sidecar.kind.toUpperCase() }}</span>
            <span class="audit-path font-code">{{ sidecar.relativePath }}</span>
            <span class="audit-status text-ok font-code">VALID SIDECAR</span>
          </div>
        </template>

        <template v-else>
          <div class="audit-item">
            <span class="spec-badge font-code">NFO</span>
            <span class="audit-path font-code">movie.nfo</span>
            <span class="audit-status text-ok font-code">VALID KODI XML</span>
          </div>
          <div class="audit-item">
            <span class="spec-badge font-code">POSTER</span>
            <span class="audit-path font-code">poster.jpg</span>
            <span class="audit-status text-ok font-code">1000x1500</span>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<style scoped>
.files-workshop-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}

.workshop-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 4px;
}

.header-left h2 {
  font-size: 18px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  margin: 0 0 2px 0;
  letter-spacing: -0.01em;
}

.sub-label {
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
}

.rename-engine-card,
.structure-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.card-header-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 14px;
  background: var(--surface-container-high, #23293c);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.header-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
}

.preset-tag,
.item-count {
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.card-body {
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.pattern-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.field-label {
  font-size: 10px;
  font-weight: 700;
  color: var(--primary, #c0c1ff);
  letter-spacing: 0.05em;
}

.pattern-input {
  width: 100%;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  padding: 8px 12px;
  font-size: 13px;
}

.pattern-input:focus {
  outline: none;
  border-color: var(--primary, #c0c1ff);
}

.tokens-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.tokens-label {
  font-size: 10px;
  font-weight: 700;
  color: var(--outline, #908fa0);
  letter-spacing: 0.05em;
}

.tokens-list {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.token-pill {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  color: var(--secondary, #4edea3);
  font-size: 11px;
  padding: 2px 8px;
  border-radius: var(--radius-sm, 0.25rem);
  cursor: pointer;
  transition: all 0.15s ease;
}

.token-pill:hover {
  background: var(--surface-container-high, #23293c);
  border-color: var(--secondary, #4edea3);
}

.file-audit-list {
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.audit-item {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 8px 12px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.video-badge {
  background: var(--primary-container, #8083ff) !important;
  color: #ffffff !important;
  border-color: var(--primary-container, #8083ff) !important;
}

.audit-path {
  flex: 1;
  font-size: 12px;
  color: var(--on-surface, #dce1fb);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.audit-status {
  font-size: 11px;
}

.text-ok {
  color: var(--secondary, #4edea3);
}
</style>
