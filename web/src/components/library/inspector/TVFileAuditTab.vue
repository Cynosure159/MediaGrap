<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { MediaInspection, RenamePlan, TVEpisode, TVSelection } from '@/api/types'
import * as api from '@/api/library'

const props = withDefaults(defineProps<{
  showId?: number
  episodes: TVEpisode[]
  allEpisodes?: TVEpisode[]
  selection?: TVSelection | null
  inspection: MediaInspection | null
  inspectionLoading: boolean
  inspectionError: string | null
  labels: Record<string, string>
  csrfToken?: string
}>(), {
  showId: 0,
  allEpisodes: () => [],
  selection: null,
  csrfToken: '',
})

const availableTokens = [
  '${showTitle}',
  '${seasonNumber}',
  '${seasonNumberPad}',
  '${episodeNumber}',
  '${episodeNumberPad}',
  '${episodeTitle}',
  '${year}',
  '${resolution}',
  '${videoCodec}',
]

const presets = {
  kodi: '${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad}',
  kodiTitle: '${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}',
  plex: '${showTitle} (Season ${seasonNumberPad})/${showTitle} - s${seasonNumberPad}e${episodeNumberPad} - ${episodeTitle}',
  jellyfin: '${showTitle}/Season ${seasonNumberPad}/${showTitle} S${seasonNumberPad}E${episodeNumberPad} ${episodeTitle}',
  flat: '${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}',
  custom: '',
}

const selectedPreset = ref<keyof typeof presets>('kodiTitle')
const patternInput = ref(presets.kodiTitle)

function formatEpisodeCode(ep: TVEpisode): string {
  const s = String(ep.seasonNumber).padStart(2, '0')
  const e = String(ep.episodeStart).padStart(2, '0')
  return `S${s}E${e}`
}

const currentScopeInfo = computed(() => {
  const sel = props.selection
  if (sel?.kind === 'season' && typeof sel.seasonNumber === 'number') {
    return {
      type: 'season',
      label: `${props.labels.season || 'Season'} ${sel.seasonNumber} (${props.episodes.length} ${props.labels.episodes || 'episodes'})`,
      scope: { seasonNumber: sel.seasonNumber },
    }
  }
  if (sel?.kind === 'episode' && typeof sel.episodeId === 'number') {
    const ep = props.episodes.find(e => e.id === sel.episodeId) || props.episodes[0]
    const code = ep ? formatEpisodeCode(ep) : `Episode #${sel.episodeId}`
    return {
      type: 'episode',
      label: `${code} · ${props.labels.scopeSingleEpisode || 'Single episode'}`,
      scope: { episodeId: sel.episodeId },
    }
  }
  return {
    type: 'show',
    label: `${props.labels.scopeAllEpisodes || 'Whole show'} (${props.episodes.length} ${props.labels.episodes || 'episodes'})`,
    scope: {},
  }
})

function onPresetChange() {
  if (selectedPreset.value !== 'custom') {
    patternInput.value = presets[selectedPreset.value]
  }
}

function appendToken(token: string) {
  patternInput.value += token
  selectedPreset.value = 'custom'
}

const renamePlan = ref<RenamePlan | null>(null)
const previewLoading = ref(false)
const isApplying = ref(false)
const actionMessage = ref<string | null>(null)
const actionError = ref<string | null>(null)

const hasConflicts = computed(() => renamePlan.value?.hasConflicts ?? false)

// Reset preview when switching scope selection in the catalog
watch(() => props.selection, () => {
  renamePlan.value = null
  actionError.value = null
  actionMessage.value = null
})

async function runDryRun() {
  if (!props.showId) return
  previewLoading.value = true
  actionError.value = null
  actionMessage.value = null
  try {
    renamePlan.value = await api.previewTVRename(
      props.csrfToken,
      props.showId,
      patternInput.value,
      currentScopeInfo.value.scope,
    )
  } catch (err: any) {
    actionError.value = err.message || 'Failed to generate rename plan'
  } finally {
    previewLoading.value = false
  }
}

async function executeRename() {
  if (!renamePlan.value?.id || hasConflicts.value || isApplying.value) return
  isApplying.value = true
  actionError.value = null
  actionMessage.value = null
  try {
    await api.applyRenamePlan(props.csrfToken, renamePlan.value.id)
    actionMessage.value = props.labels.renameQueued || 'Rename job queued. Track progress in the Jobs panel.'
  } catch (err: any) {
    actionError.value = err.message || 'Failed to execute rename plan'
  } finally {
    isApplying.value = false
  }
}
</script>

<template>
  <section class="files-workshop-view">
    <div class="workshop-header">
      <div class="header-left">
        <h2>{{ labels.filesRenamePlanner || 'Files & Safe Rename Planner' }}</h2>
        <span class="sub-label">{{ labels.fileAuditReadOnly || 'Real filesystem state and preview-only naming; no files will be changed.' }}</span>
      </div>
    </div>

    <!-- Alert banners -->
    <div v-if="actionError" class="audit-alert error-banner font-code">
      {{ actionError }}
    </div>
    <div v-if="actionMessage" class="audit-alert success-banner font-code">
      {{ actionMessage }}
    </div>

    <!-- Naming pattern card -->
    <div class="rename-engine-card">
      <div class="card-header-bar">
        <div class="header-left-bar">
          <span class="header-title">{{ labels.namingPattern || 'Naming Pattern Engine' }}</span>
          <span class="scope-pill font-code">
            <span class="dot-ok"></span>
            {{ currentScopeInfo.label }}
          </span>
        </div>

        <div class="selectors-row">
          <!-- Preset selector -->
          <div class="select-wrapper">
            <label class="control-label font-code">{{ labels.patternPreset || 'Preset' }}:</label>
            <select v-model="selectedPreset" class="preset-select font-code" @change="onPresetChange">
              <option value="kodi">{{ labels.presetKodiTV || 'Kodi Standard' }}</option>
              <option value="kodiTitle">{{ labels.presetKodiTVWithTitle || 'Kodi (with Title)' }}</option>
              <option value="plex">{{ labels.presetPlexTV || 'Plex Standard' }}</option>
              <option value="jellyfin">{{ labels.presetJellyfinTV || 'Jellyfin / Emby' }}</option>
              <option value="flat">{{ labels.presetFlatTV || 'Flat (No folder)' }}</option>
              <option value="custom">{{ labels.presetCustom || 'Custom' }}</option>
            </select>
          </div>
        </div>
      </div>

      <div class="card-body">
        <label class="field-label font-code" for="tv-naming-pattern">{{ labels.patternTemplate || 'PATTERN TEMPLATE' }}</label>
        <div class="pattern-action-row">
          <input id="tv-naming-pattern" v-model="patternInput" type="text" class="pattern-input font-code" />
          <button class="btn btn-primary" :disabled="previewLoading" type="button" @click="runDryRun">
            {{ previewLoading ? (labels.previewing || 'Previewing…') : (labels.dryRunSimulation || 'Dry Run Simulation') }}
          </button>
        </div>

        <div class="tokens-list" :aria-label="labels.availableTokens">
          <button
            v-for="tok in availableTokens"
            :key="tok"
            class="token-pill font-code"
            type="button"
            @click="appendToken(tok)"
          >
            {{ tok }}
          </button>
        </div>
      </div>

      <!-- Dry run preview section -->
      <div v-if="renamePlan" class="rename-preview">
        <div class="preview-summary" :class="{ 'preview-summary--conflict': hasConflicts }">
          {{ hasConflicts ? (labels.conflictsDetected || 'Conflicts detected — cannot execute') : (labels.noConflictsDetected || 'No conflicts detected') }} ({{ renamePlan.items.length }} {{ labels.files || 'files' }})
        </div>
        <div v-for="entry in renamePlan.items" :key="entry.currentPath" class="rename-row">
          <span
            class="operation-tag font-code"
            :class="`operation-tag--${entry.operation.replace(' ', '_').toLowerCase()}`"
          >
            {{ entry.operation === 'rename' ? (labels.renameFile || 'RENAME FILE') : (entry.operation === 'rename_dir' ? (labels.renameDir || 'RENAME DIR') : entry.operation.toUpperCase()) }}
          </span>
          <div class="rename-paths font-code">
            <span class="current-path">{{ entry.currentPath }}</span>
            <span class="path-arrow">→</span>
            <span class="planned-path">{{ entry.plannedPath }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Current Structure Card -->
    <div class="structure-card">
      <div class="card-header-bar">
        <span class="header-title">{{ labels.currentFiles || 'Current File & Sidecar Assets' }}</span>
        <span class="item-count font-code">{{ episodes.length }} {{ labels.episodes || 'episodes' }}</span>
      </div>

      <div class="file-audit-list">
        <div v-for="ep in episodes" :key="ep.id" class="audit-item">
          <span class="spec-badge video-badge font-code">{{ formatEpisodeCode(ep) }}</span>
          <span class="audit-path font-code" :title="ep.relativePath">{{ ep.relativePath }}</span>
          <span class="audit-status text-ok font-code">{{ ep.sidecars?.length || 0 }} {{ labels.files || 'sidecars' }}</span>
        </div>
      </div>
    </div>

    <!-- Embedded Streams Probe Card -->
    <div class="probe-card">
      <div class="card-header-bar">
        <span class="header-title">{{ labels.mediaStreams || 'Embedded media streams' }}</span>
        <span v-if="inspection?.cached" class="item-count font-code">{{ labels.cachedProbe || 'CACHED' }}</span>
      </div>
      <div v-if="inspectionLoading" class="empty-row">{{ labels.inspectingMedia || 'Inspecting media…' }}</div>
      <div v-else-if="inspection?.probeStatus !== 'ready'" class="empty-row">
        {{ inspectionError || inspection?.probeError || labels.mediaInfoUnavailable || 'ffprobe media information is unavailable.' }}
      </div>
      <div v-else class="probe-summary font-code">
        <span v-for="stream in inspection.video" :key="`video-${stream.index}`" class="probe-pill">
          {{ labels.video || 'Video' }} {{ stream.codec.toUpperCase() }} {{ stream.width }}×{{ stream.height }}
        </span>
        <span v-for="stream in inspection.audio" :key="`audio-${stream.index}`" class="probe-pill">
          {{ labels.audio || 'Audio' }} {{ stream.codec.toUpperCase() }} {{ stream.channelLayout || `${stream.channels} ch` }}
        </span>
        <span v-if="!inspection.video.length && !inspection.audio.length" class="empty-row">{{ labels.mediaInfoUnavailable || 'No stream metadata.' }}</span>
      </div>
    </div>

    <!-- Sticky action bar -->
    <div v-if="renamePlan && !hasConflicts" class="sticky-action-bar">
      <div class="action-bar-left">
        <span class="spec-pill spec-pill--success"><span class="dot-ok"></span>{{ labels.noConflictsDetected || 'No conflicts detected' }} ({{ renamePlan.items.length }})</span>
      </div>
      <div class="action-bar-right">
        <button class="btn btn-outline" :disabled="previewLoading || isApplying" type="button" @click="runDryRun">
          {{ labels.dryRunSimulation || 'Dry Run Simulation' }}
        </button>
        <button class="btn btn-success" :disabled="isApplying" type="button" @click="executeRename">
          {{ isApplying ? (labels.loading || 'Applying…') : (labels.executeRename || 'Execute Safe Rename & Move') }}
        </button>
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

.empty-row {
  padding: 14px;
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 12px;
}

.rename-engine-card,
.structure-card,
.probe-card {
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
  gap: 12px;
  padding: 8px 14px;
  background: var(--surface-container-high, #23293c);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.header-left-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.header-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  white-space: nowrap;
}

.scope-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 2px 8px;
  color: var(--secondary, #4edea3);
  font-size: 10px;
  font-weight: 600;
}

.selectors-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.select-wrapper {
  display: flex;
  align-items: center;
  gap: 6px;
}

.control-label {
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.preset-select {
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: 4px;
  padding: 2px 8px;
  background: var(--surface-container-lowest, #070d1f);
  color: var(--on-surface, #dce1fb);
  font-size: 11px;
}

.item-count {
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.card-body {
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.field-label {
  font-size: 10px;
  font-weight: 700;
  color: var(--primary, #c0c1ff);
  letter-spacing: 0.05em;
}

.pattern-action-row {
  display: flex;
  gap: 8px;
}

.pattern-input {
  min-width: 0;
  flex: 1;
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
  padding: 3px 8px;
  border-radius: var(--radius-sm, 0.25rem);
  cursor: pointer;
  transition: all 0.15s ease;
}

.token-pill:hover {
  background: var(--surface-container-high, #23293c);
  border-color: var(--secondary, #4edea3);
}

.rename-preview {
  border-top: 1px solid var(--outline-variant, #2e3447);
}

.preview-summary {
  padding: 8px 14px;
  background: color-mix(in srgb, var(--secondary, #4edea3) 8%, transparent);
  color: var(--secondary, #4edea3);
  font-size: 11px;
}

.preview-summary--conflict {
  background: color-mix(in srgb, var(--error, #ffb4ab) 8%, transparent);
  color: var(--error, #ffb4ab);
}

.rename-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 9px 14px;
  border-top: 1px solid color-mix(in srgb, var(--outline-variant, #2e3447) 60%, transparent);
}

.operation-tag {
  min-width: 58px;
  border-radius: 4px;
  padding: 2px 6px;
  font-size: 9px;
  text-align: center;
  font-weight: 600;
}

.operation-tag--keep {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface-variant, #c7c4d7);
}

.operation-tag--rename,
.operation-tag--rename_file,
.operation-tag--rename_dir {
  background: color-mix(in srgb, var(--secondary, #4edea3) 10%, transparent);
  color: var(--secondary, #4edea3);
}

.operation-tag--conflict {
  background: color-mix(in srgb, var(--error, #ffb4ab) 10%, transparent);
  color: var(--error, #ffb4ab);
}

.rename-paths {
  display: grid;
  min-width: 0;
  flex: 1;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  gap: 8px;
  font-size: 10px;
}

.current-path,
.planned-path {
  overflow-wrap: anywhere;
}

.current-path {
  color: var(--on-surface-variant, #c7c4d7);
}

.planned-path {
  color: var(--on-surface, #dce1fb);
}

.path-arrow {
  color: var(--outline, #908fa0);
}

.file-audit-list {
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 280px;
  overflow-y: auto;
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

.probe-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 14px;
}

.probe-pill {
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 4px 8px;
  background: var(--surface-container-low, #151b2d);
  color: var(--on-surface, #dce1fb);
  font-size: 11px;
}

.audit-alert {
  border-radius: 4px;
  padding: 8px 12px;
  font-size: 12px;
}

.error-banner {
  border: 1px solid var(--error, #ffb4ab);
  color: var(--error, #ffb4ab);
  background: color-mix(in srgb, var(--error, #ffb4ab) 8%, transparent);
}

.success-banner {
  border: 1px solid var(--secondary, #4edea3);
  color: var(--secondary, #4edea3);
  background: color-mix(in srgb, var(--secondary, #4edea3) 8%, transparent);
}

.sticky-action-bar {
  position: sticky;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-container, #191f31);
  z-index: 10;
  margin-top: 8px;
  border-radius: var(--radius-lg, 0.5rem);
  box-shadow: 0 -4px 12px rgba(0, 0, 0, 0.3);
}

.action-bar-right {
  display: flex;
  gap: 8px;
}

@media (max-width: 700px) {
  .workshop-header,
  .pattern-action-row,
  .sticky-action-bar,
  .card-header-bar {
    align-items: stretch;
    flex-direction: column;
    gap: 12px;
  }
  .rename-paths {
    grid-template-columns: 1fr;
  }
  .path-arrow {
    transform: rotate(90deg);
  }
}
</style>
