<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { MediaInspection, RenamePlan, TVEpisode, TVSelection } from '@/api/types'
import * as api from '@/api/library'

const props = withDefaults(defineProps<{
  showId?: number
  episodes: TVEpisode[]
  selection?: TVSelection | null
  inspection: MediaInspection | null
  inspectionLoading: boolean
  inspectionError: string | null
  labels: Record<string, string>
  csrfToken?: string
}>(), {
  showId: 0,
  selection: null,
  csrfToken: '',
})

const availableTokens = [
  '${showTitle}',
  '${originalTitle}',
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
  kodiTitle: '${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}',
  kodi: '${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad}',
  plex: '${showTitle}/Season ${seasonNumberPad}/${showTitle} - s${seasonNumberPad}e${episodeNumberPad} - ${episodeTitle}',
  jellyfin: '${showTitle}/Season ${seasonNumberPad}/${showTitle} S${seasonNumberPad}E${episodeNumberPad} ${episodeTitle}',
  flat: '${showTitle}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}',
  custom: '',
}

const selectedPreset = ref<keyof typeof presets>('kodiTitle')
const patternInput = ref(presets.kodiTitle)
const isPreviewCollapsed = ref(false)

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
  isPreviewCollapsed.value = false
})

function clearPreview() {
  renamePlan.value = null
  actionError.value = null
  actionMessage.value = null
}

async function runDryRun() {
  if (!props.showId) return
  previewLoading.value = true
  actionError.value = null
  actionMessage.value = null
  isPreviewCollapsed.value = false
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

    <!-- ── 1. Naming Pattern Engine ──────────────────────────────── -->
    <div class="rename-engine-card">
      <div class="card-header-bar">
        <div class="header-left-bar">
          <span class="header-title">{{ labels.namingPattern || '命名规则引擎' }}</span>
          <span class="scope-pill font-code">
            <span class="dot-ok"></span>
            {{ currentScopeInfo.label }}
          </span>
        </div>

        <div class="selectors-row">
          <div class="select-wrapper">
            <label class="control-label font-code">{{ labels.patternPreset || '预设' }}:</label>
            <select v-model="selectedPreset" class="preset-select font-code" @change="onPresetChange">
              <option value="kodiTitle">{{ labels.presetKodiTVWithTitle || 'Kodi 标准（带单集标题）' }}</option>
              <option value="kodi">{{ labels.presetKodiTV || 'Kodi 标准' }}</option>
              <option value="plex">{{ labels.presetPlexTV || 'Plex 标准' }}</option>
              <option value="jellyfin">{{ labels.presetJellyfinTV || 'Jellyfin / Emby' }}</option>
              <option value="flat">{{ labels.presetFlatTV || '平铺（无 Season 目录）' }}</option>
              <option value="custom">{{ labels.presetCustom || '自定义' }}</option>
            </select>
          </div>
        </div>
      </div>

      <div class="card-body">
        <label class="field-label font-code" for="tv-naming-pattern">{{ labels.patternTemplate || '文件名模板' }}</label>
        <div class="pattern-action-row">
          <input id="tv-naming-pattern" v-model="patternInput" type="text" class="pattern-input font-code" />
          <button class="btn btn-primary" :disabled="previewLoading" type="button" @click="runDryRun">
            <span v-if="previewLoading" class="btn-spinner"></span>
            <span>{{ previewLoading ? (labels.previewing || '正在计算…') : (labels.dryRunSimulation || '模拟预览') }}</span>
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
    </div>

    <!-- ── 2. Dry Run Simulation Result Card ──────────────────────── -->
    <div v-if="renamePlan" class="rename-plan-card">
      <div class="card-header-bar plan-header" :class="{ 'plan-header--conflict': hasConflicts }">
        <div class="plan-header-left">
          <span class="header-title">{{ labels.renamePlanPreview || '执行方案预览' }}</span>
          <span class="spec-pill" :class="hasConflicts ? 'spec-pill--error' : 'spec-pill--success'">
            <span class="dot-ok" :class="{ 'dot-conflict': hasConflicts }"></span>
            {{ hasConflicts ? (labels.conflictsDetected || '存在冲突 — 无法执行') : (labels.noConflictsDetected || '未发现冲突') }} ({{ renamePlan.items.length }} {{ labels.files || '个文件' }})
          </span>
        </div>
        <div class="plan-header-actions">
          <button class="btn-preview-action font-code" type="button" @click="isPreviewCollapsed = !isPreviewCollapsed">
            {{ isPreviewCollapsed ? (labels.expandPreview || '展开预览') : (labels.collapsePreview || '收起预览') }}
          </button>
          <button class="btn-preview-action font-code btn-dismiss" type="button" :title="labels.closePreview || '关闭预览'" @click="clearPreview">
            ✕ {{ labels.closePreview || '关闭预览' }}
          </button>
        </div>
      </div>

      <div v-show="!isPreviewCollapsed" class="preview-items-list">
        <div v-for="entry in renamePlan.items" :key="entry.currentPath" class="rename-row" :class="{ 'rename-row--conflict': entry.conflict }">
          <span
            class="operation-tag font-code"
            :class="`operation-tag--${entry.operation.replace(' ', '_').toLowerCase()}`"
          >
            {{ entry.operation === 'rename' ? (labels.renameFile || 'RENAME FILE') : (entry.operation === 'rename_dir' ? (labels.renameDir || 'RENAME DIR') : entry.operation.toUpperCase()) }}
          </span>
          <div class="rename-paths font-code">
            <span class="current-path" :title="entry.currentPath">{{ entry.currentPath }}</span>
            <span class="path-arrow">→</span>
            <span class="planned-path" :title="entry.plannedPath">{{ entry.plannedPath }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ── 3. Current Structure Card ──────────────────────────────── -->
    <div class="structure-card">
      <div class="card-header-bar">
        <span class="header-title">{{ labels.currentFiles || '当前文件与 Sidecar 审计' }}</span>
        <span class="item-count font-code">{{ episodes.length }} {{ labels.episodes || '集' }}</span>
      </div>

      <div class="file-audit-list">
        <div v-for="ep in episodes" :key="ep.id" class="audit-item">
          <span class="spec-badge video-badge font-code">{{ formatEpisodeCode(ep) }}</span>
          <span class="audit-path font-code" :title="ep.relativePath">{{ ep.relativePath }}</span>
          <span class="audit-status text-ok font-code">{{ ep.sidecars?.length || 0 }} {{ labels.files || 'sidecars' }}</span>
        </div>
      </div>
    </div>

    <!-- ── 4. Embedded Streams Probe Card ─────────────────────────── -->
    <div class="probe-card">
      <div class="card-header-bar">
        <span class="header-title">{{ labels.mediaStreams || '内嵌媒体流' }}</span>
        <span v-if="inspection?.cached" class="item-count font-code">{{ labels.cachedProbe || '已缓存' }}</span>
      </div>
      <div v-if="inspectionLoading" class="empty-row">{{ labels.inspectingMedia || '正在探测媒体…' }}</div>
      <div v-else-if="inspection?.probeStatus !== 'ready'" class="empty-row">
        {{ inspectionError || inspection?.probeError || labels.mediaInfoUnavailable || 'ffprobe 媒体信息不可用。' }}
      </div>
      <div v-else class="probe-summary font-code">
        <span v-for="stream in inspection.video" :key="`video-${stream.index}`" class="probe-pill">
          {{ labels.video || 'Video' }} {{ stream.codec.toUpperCase() }} {{ stream.width }}×{{ stream.height }}
        </span>
        <span v-for="stream in inspection.audio" :key="`audio-${stream.index}`" class="probe-pill">
          {{ labels.audio || 'Audio' }} {{ stream.codec.toUpperCase() }} {{ stream.channelLayout || `${stream.channels} ch` }}
        </span>
        <span v-if="!inspection.video.length && !inspection.audio.length" class="empty-row">{{ labels.mediaInfoUnavailable || '无媒体流信息。' }}</span>
      </div>
    </div>

    <!-- ── 5. Sticky Confirmation & Execution Action Bar ──────────── -->
    <div v-if="renamePlan && !hasConflicts" class="sticky-action-bar">
      <div class="action-bar-left">
        <span class="spec-pill spec-pill--success font-code">
          <span class="dot-ok"></span>
          {{ labels.noConflictsDetected || '未发现冲突' }} ({{ renamePlan.items.length }} {{ labels.files || '个文件' }})
        </span>
      </div>
      <div class="action-bar-right">
        <button class="btn btn-ghost" type="button" @click="clearPreview">
          {{ labels.closePreview || '关闭预览' }}
        </button>
        <button class="btn btn-outline" :disabled="previewLoading || isApplying" type="button" @click="runDryRun">
          {{ labels.dryRunSimulation || '重新模拟' }}
        </button>
        <button class="btn btn-success" :disabled="isApplying" type="button" @click="executeRename">
          <span v-if="isApplying" class="btn-spinner"></span>
          <span>{{ isApplying ? (labels.loading || '正在执行…') : (labels.executeRename || '执行安全重命名与移动') }}</span>
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.files-workshop-view {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}

.workshop-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 2px;
}

.header-left h2 {
  font-size: 16px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  margin: 0 0 2px 0;
  letter-spacing: -0.01em;
}

.sub-label {
  font-size: 11px;
  color: var(--on-surface-variant, #c7c4d7);
}

.empty-row {
  padding: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 11px;
}

.rename-engine-card,
.structure-card,
.probe-card,
.rename-plan-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.card-header-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 12px;
  background: var(--surface-container-high, #23293c);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.header-left-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.header-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  white-space: nowrap;
}

.scope-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 1px 6px;
  color: var(--secondary, #4edea3);
  font-size: 10px;
  font-weight: 600;
}

.selectors-row {
  display: flex;
  align-items: center;
  gap: 8px;
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
  padding: 2px 6px;
  background: var(--surface-container-lowest, #070d1f);
  color: var(--on-surface, #dce1fb);
  font-size: 10px;
}

.item-count {
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.card-body {
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
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
  padding: 6px 10px;
  font-size: 12px;
}

.pattern-input:focus {
  outline: none;
  border-color: var(--primary, #c0c1ff);
}

.tokens-list {
  display: flex;
  align-items: center;
  gap: 5px;
  flex-wrap: wrap;
}

.token-pill {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  color: var(--secondary, #4edea3);
  font-size: 10px;
  padding: 2px 6px;
  border-radius: var(--radius-sm, 0.25rem);
  cursor: pointer;
  transition: all 0.15s ease;
}

.token-pill:hover {
  background: var(--surface-container-high, #23293c);
  border-color: var(--secondary, #4edea3);
}

/* ── Rename Plan Card & Diff List ─────────────────────────────────── */
.rename-plan-card {
  border-color: color-mix(in srgb, var(--primary, #c0c1ff) 40%, var(--outline-variant, #2e3447));
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
}

.plan-header {
  background: var(--surface-container-high, #23293c);
}

.plan-header--conflict {
  background: color-mix(in srgb, var(--error, #ffb4ab) 12%, var(--surface-container-high, #23293c));
}

.plan-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.plan-header-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-preview-action {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: 3px;
  color: var(--on-surface, #dce1fb);
  padding: 3px 8px;
  font-size: 11px;
  cursor: pointer;
  transition: all 0.12s ease;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.btn-preview-action:hover {
  background: var(--surface-container-highest, #2d3347);
  color: var(--primary, #c0c1ff);
}

.btn-dismiss:hover {
  color: var(--error, #ffb4ab);
  border-color: color-mix(in srgb, var(--error, #ffb4ab) 40%, transparent);
}

.preview-items-list {
  display: flex;
  flex-direction: column;
  max-height: 320px;
  overflow-y: auto;
  background: var(--surface-container-lowest, #070d1f);
}

.rename-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-bottom: 1px solid color-mix(in srgb, var(--outline-variant, #2e3447) 40%, transparent);
  transition: background 0.1s ease;
}

.rename-row:hover {
  background: rgba(255, 255, 255, 0.02);
}

.rename-row--conflict {
  background: color-mix(in srgb, var(--error, #ffb4ab) 6%, transparent);
}

.operation-tag {
  min-width: 74px;
  border-radius: 3px;
  padding: 2px 6px;
  font-size: 9.5px;
  text-align: center;
  font-weight: 700;
  flex-shrink: 0;
}

.operation-tag--keep {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface-variant, #c7c4d7);
}

.operation-tag--rename,
.operation-tag--rename_file {
  background: color-mix(in srgb, var(--secondary, #4edea3) 15%, transparent);
  color: var(--secondary, #4edea3);
  border: 1px solid color-mix(in srgb, var(--secondary, #4edea3) 30%, transparent);
}

.operation-tag--rename_dir {
  background: color-mix(in srgb, var(--primary, #c0c1ff) 15%, transparent);
  color: var(--primary, #c0c1ff);
  border: 1px solid color-mix(in srgb, var(--primary, #c0c1ff) 30%, transparent);
}

.operation-tag--conflict {
  background: color-mix(in srgb, var(--error, #ffb4ab) 15%, transparent);
  color: var(--error, #ffb4ab);
  border: 1px solid color-mix(in srgb, var(--error, #ffb4ab) 30%, transparent);
}

.rename-paths {
  display: grid;
  min-width: 0;
  flex: 1;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1.1fr);
  gap: 8px;
  align-items: center;
  font-size: 11px;
}

.current-path {
  color: var(--on-surface-variant, #c7c4d7);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.planned-path {
  color: var(--secondary, #4edea3);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.path-arrow {
  color: var(--outline, #908fa0);
  font-weight: 700;
  flex-shrink: 0;
}

.dot-conflict {
  background: var(--error, #ffb4ab) !important;
  box-shadow: 0 0 6px var(--error, #ffb4ab) !important;
}

.file-audit-list {
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 280px;
  overflow-y: auto;
}

.audit-item {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 6px 10px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.video-badge {
  background: var(--primary-container, #8083ff) !important;
  color: #ffffff !important;
  border-color: var(--primary-container, #8083ff) !important;
  font-size: 10px;
  padding: 1px 5px;
}

.audit-path {
  flex: 1;
  font-size: 11px;
  color: var(--on-surface, #dce1fb);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.audit-status {
  font-size: 10px;
}

.text-ok {
  color: var(--secondary, #4edea3);
}

.probe-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 8px 12px;
}

.probe-pill {
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 3px 6px;
  background: var(--surface-container-low, #151b2d);
  color: var(--on-surface, #dce1fb);
  font-size: 10px;
}

.audit-alert {
  border-radius: 4px;
  padding: 6px 10px;
  font-size: 11px;
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
  padding: 10px 14px;
  border: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-container, #191f31);
  z-index: 25;
  margin-top: 8px;
  border-radius: var(--radius-md, 0.375rem);
  box-shadow: 0 -6px 20px rgba(0, 0, 0, 0.4);
}

.action-bar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-spinner {
  width: 12px;
  height: 12px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  display: inline-block;
  margin-right: 4px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 900px) {
  .stream-groups {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 700px) {
  .workshop-header,
  .pattern-action-row,
  .sticky-action-bar,
  .card-header-bar {
    align-items: stretch;
    flex-direction: column;
    gap: 8px;
  }
  .rename-paths {
    grid-template-columns: 1fr;
  }
  .path-arrow {
    transform: rotate(90deg);
    margin: 0 auto;
  }
}
</style>
