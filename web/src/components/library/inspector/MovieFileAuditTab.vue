<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue'
import type { MediaInspection, MediaItem, RenamePlan } from '@/api/types'

const props = withDefaults(defineProps<{
  item: MediaItem
  labels: Record<string, string>
  inspection: MediaInspection | null
  inspectionLoading: boolean
  inspectionError: string | null
  renamePlan?: RenamePlan | null
  previewLoading?: boolean
  isApplying?: boolean
}>(), {
  renamePlan: null,
  previewLoading: false,
  isApplying: false,
})

const emit = defineEmits<{ 
  previewRename: [pattern: string]
  applyRename: []
  clearRename: []
}>()

const patternInput = shallowRef('${title} (${year})/${title} (${year})')
const selectedPreset = shallowRef('kodi')
const availableTokens = ['${title}', '${originalTitle}', '${year}', '${resolution}', '${videoCodec}', '${audioCodec}', '${edition}', '${imdbId}']
const hasConflicts = computed(() => props.renamePlan?.hasConflicts ?? false)

const isPreviewCollapsed = shallowRef(false)
const isDismissed = shallowRef(false)

// Reset dismissed state when new plan arrives
watch(() => props.renamePlan, (newPlan) => {
  if (newPlan) {
    isDismissed.value = false
    isPreviewCollapsed.value = false
  }
})

function triggerPreview() {
  isDismissed.value = false
  isPreviewCollapsed.value = false
  emit('previewRename', patternInput.value)
}

function clearPreview() {
  isDismissed.value = true
  emit('clearRename')
}

function onPresetChange() {
  if (selectedPreset.value === 'kodi' || selectedPreset.value === 'plex') patternInput.value = '${title} (${year})/${title} (${year})'
}

function appendToken(token: string) { 
  patternInput.value += token
  selectedPreset.value = 'custom'
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MB`
  return `${(bytes / 1024 ** 3).toFixed(2)} GB`
}

function warningLabel(warning: string): string {
  return props.labels[`audit_${warning}`] || warning.replaceAll('_', ' ')
}

function formatModifiedAt(value: string): string {
  if (!value) return '—'
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}
</script>

<template>
  <section class="files-workshop-view">
    <div class="workshop-header">
      <div class="header-left">
        <h2>{{ labels.filesRenamePlanner || '文件与安全重命名计划' }}</h2>
        <span class="sub-label">{{ labels.fileAuditReadOnly || '展示真实文件状态与只读命名预览；未确认前不会修改任何文件。' }}</span>
      </div>
      <span class="readonly-badge font-code">{{ labels.previewOnly || '仅预览' }}</span>
    </div>

    <div v-if="inspectionError" class="audit-alert font-code">{{ inspectionError }}</div>

    <!-- ── 1. Naming Pattern Engine ──────────────────────────────── -->
    <div class="rename-engine-card">
      <div class="card-header-bar">
        <span class="header-title">{{ labels.namingPattern || '命名规则引擎' }}</span>
        <div class="select-wrapper">
          <label class="control-label font-code">{{ labels.patternPreset || '预设' }}:</label>
          <select v-model="selectedPreset" class="preset-select font-code" @change="onPresetChange">
            <option value="kodi">{{ labels.presetKodi || 'Kodi 标准' }}</option>
            <option value="plex">{{ labels.presetPlex || 'Plex 标准' }}</option>
            <option value="custom">{{ labels.presetCustom || '自定义' }}</option>
          </select>
        </div>
      </div>

      <div class="card-body">
        <label class="field-label font-code" for="movie-naming-pattern">{{ labels.patternTemplate || '文件名模板' }}</label>
        <div class="pattern-action-row">
          <input id="movie-naming-pattern" v-model="patternInput" type="text" class="pattern-input font-code" />
          <button class="btn btn-primary" :disabled="previewLoading" type="button" @click="triggerPreview">
            <span v-if="previewLoading" class="btn-spinner"></span>
            <span>{{ previewLoading ? (labels.previewing || '正在计算…') : (labels.dryRunSimulation || '模拟预览') }}</span>
          </button>
        </div>
        <div class="tokens-list" :aria-label="labels.availableTokens">
          <button v-for="token in availableTokens" :key="token" class="token-pill font-code" type="button" @click="appendToken(token)">
            {{ token }}
          </button>
        </div>
      </div>
    </div>

    <!-- ── 2. Dry Run Simulation Result Card ──────────────────────── -->
    <div v-if="renamePlan && !isDismissed" class="rename-plan-card">
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
          <span class="operation-tag font-code" :class="`operation-tag--${entry.operation.replace(' ', '_').toLowerCase()}`">
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

    <!-- ── 3. Current Files Structure Card ────────────────────────── -->
    <div class="structure-card">
      <div class="card-header-bar">
        <span class="header-title">{{ labels.currentFiles || '当前文件与 Sidecar 审计' }}</span>
        <span class="item-count font-code">{{ inspection?.files.length ?? 0 }} {{ labels.files || '个文件' }}</span>
      </div>
      <div v-if="inspectionLoading" class="empty-row">{{ labels.inspectingMedia || '正在探测媒体…' }}</div>
      <div v-else-if="!inspection?.files.length" class="empty-row">{{ labels.fileAuditUnavailable || '无法读取文件审计信息。' }}</div>
      <div v-else class="audit-table-wrap">
        <table class="audit-table">
          <thead><tr><th>{{ labels.type }}</th><th>{{ labels.path }}</th><th>{{ labels.size }}</th><th>{{ labels.mimeType }}</th><th>{{ labels.modifiedAt }}</th><th>{{ labels.permissions }}</th><th>{{ labels.status }}</th></tr></thead>
          <tbody>
            <tr v-for="file in inspection.files" :key="file.relativePath" :class="{ 'audit-row--warning': file.warnings.length }">
              <td><span class="spec-badge font-code">{{ file.kind.toUpperCase() }}</span></td>
              <td class="audit-path font-code" :title="file.relativePath">{{ file.relativePath }}</td>
              <td class="font-code">{{ formatFileSize(file.size) }}</td>
              <td class="font-code">{{ file.mimeType || '—' }}</td>
              <td class="font-code">{{ formatModifiedAt(file.modifiedAt) }}</td>
              <td class="font-code">{{ file.permissions || '—' }}</td>
              <td>
                <span v-if="!file.warnings.length" class="status-ok font-code">{{ labels.valid || '有效' }}</span>
                <template v-else><span v-for="warning in file.warnings" :key="warning" class="warning-chip font-code">{{ warningLabel(warning) }}</span></template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- ── 4. Streams Probe Card ──────────────────────────────────── -->
    <div class="streams-card">
      <div class="card-header-bar">
        <span class="header-title">{{ labels.mediaStreams || '内嵌媒体流' }}</span>
        <span v-if="inspection?.cached" class="cache-tag font-code">{{ labels.cachedProbe || '已缓存' }}</span>
      </div>
      <div v-if="inspection?.probeStatus !== 'ready'" class="empty-row">{{ inspectionLoading ? labels.inspectingMedia : (inspection?.probeError || labels.mediaInfoUnavailable) }}</div>
      <div v-else class="stream-groups">
        <div class="stream-group">
          <h3>{{ labels.videoStreams || '视频流' }} ({{ inspection.video.length }})</h3>
          <div v-for="stream in inspection.video" :key="stream.index" class="stream-row font-code">
            <span>#{{ stream.index }}</span><span>{{ stream.codec.toUpperCase() }}</span><span>{{ stream.width }}×{{ stream.height }}</span><span v-if="stream.bitDepth">{{ stream.bitDepth }}-bit</span><span v-if="stream.hdr" class="stream-accent">{{ stream.hdr }}</span><span v-if="stream.language">{{ stream.language }}</span>
          </div>
        </div>
        <div class="stream-group">
          <h3>{{ labels.audioStreams || '音频流' }} ({{ inspection.audio.length }})</h3>
          <div v-for="stream in inspection.audio" :key="stream.index" class="stream-row font-code">
            <span>#{{ stream.index }}</span><span>{{ stream.codec.toUpperCase() }}</span><span>{{ stream.channelLayout || `${stream.channels} ch` }}</span><span v-if="stream.language">{{ stream.language }}</span><span v-if="stream.title">{{ stream.title }}</span>
          </div>
        </div>
        <div class="stream-group">
          <h3>{{ labels.subtitleStreams || '字幕流' }} ({{ inspection.subtitles.length }})</h3>
          <div v-if="!inspection.subtitles.length" class="stream-empty">{{ labels.noEmbeddedSubtitles || '未发现内嵌字幕轨。' }}</div>
          <div v-for="stream in inspection.subtitles" :key="stream.index" class="stream-row font-code">
            <span>#{{ stream.index }}</span><span>{{ stream.codec.toUpperCase() }}</span><span>{{ stream.language || labels.languageUnknown }}</span><span v-if="stream.title">{{ stream.title }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ── 5. Sticky Confirmation & Execution Action Bar ──────────── -->
    <div v-if="renamePlan && !hasConflicts && !isDismissed" class="sticky-action-bar">
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
        <button class="btn btn-outline" :disabled="previewLoading || isApplying" type="button" @click="triggerPreview">
          {{ labels.dryRunSimulation || '重新模拟' }}
        </button>
        <button class="btn btn-success" :disabled="isApplying" type="button" @click="emit('applyRename')">
          <span v-if="isApplying" class="btn-spinner"></span>
          <span>{{ isApplying ? (labels.loading || '正在执行…') : (labels.executeRename || '执行安全重命名与移动') }}</span>
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.files-workshop-view { display: flex; flex-direction: column; gap: 12px; width: 100%; }
.workshop-header { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.header-left h2 { margin: 0 0 2px; color: var(--on-surface); font-size: 16px; font-weight: 700; }
.sub-label, .empty-row, .stream-empty { color: var(--on-surface-variant); font-size: 11px; }
.readonly-badge, .cache-tag { color: var(--secondary); border: 1px solid color-mix(in srgb, var(--secondary) 35%, transparent); border-radius: 4px; padding: 2px 6px; font-size: 9px; font-weight: 600; }
.structure-card, .streams-card, .rename-engine-card, .rename-plan-card { overflow: hidden; border: 1px solid var(--outline-variant); border-radius: var(--radius-md, 0.375rem); background: var(--surface-container); }
.card-header-bar { display: flex; align-items: center; justify-content: space-between; padding: 8px 12px; border-bottom: 1px solid var(--outline-variant); background: var(--surface-container-high); }
.header-title { color: var(--on-surface); font-size: 12px; font-weight: 600; }
.item-count, .preset-tag { color: var(--outline); font-size: 10px; }
.empty-row { padding: 12px; }
.audit-table-wrap { overflow-x: auto; }
.audit-table { width: 100%; border-collapse: collapse; font-size: 11px; text-align: left; }
.audit-table th { padding: 6px 8px; background: var(--surface-container-low); color: var(--outline); font-weight: 600; font-size: 10px; }
.audit-table td { padding: 6px 8px; border-top: 1px solid color-mix(in srgb, var(--outline-variant) 60%, transparent); color: var(--on-surface-variant); }
.audit-row--warning { border-left: 2px solid var(--tertiary); }
.audit-path { min-width: 200px; max-width: 400px; overflow: hidden; color: var(--on-surface) !important; text-overflow: ellipsis; white-space: nowrap; }
.status-ok { color: var(--secondary); font-size: 10px; }
.warning-chip { display: inline-block; margin: 1px 3px 1px 0; border: 1px solid color-mix(in srgb, var(--tertiary) 35%, transparent); border-radius: 3px; padding: 1px 4px; color: var(--tertiary); font-size: 9px; }
.stream-groups { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 1px; background: var(--outline-variant); }
.stream-group { min-width: 0; padding: 8px 10px; background: var(--surface-container); }
.stream-group h3 { margin: 0 0 4px; color: var(--on-surface); font-size: 10px; text-transform: uppercase; font-weight: 700; }
.stream-row { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 4px; border-radius: 3px; padding: 4px 6px; background: var(--surface-container-low); color: var(--on-surface-variant); font-size: 10px; }
.stream-accent { color: var(--tertiary); }
.card-body { display: flex; flex-direction: column; gap: 8px; padding: 10px 12px; }
.field-label { color: var(--primary); font-size: 10px; font-weight: 700; }
.pattern-action-row { display: flex; gap: 8px; }
.pattern-input { min-width: 0; flex: 1; border: 1px solid var(--outline-variant); border-radius: 4px; padding: 6px 10px; background: var(--surface-container-lowest); color: var(--on-surface); font-size: 12px; }
.pattern-input:focus { outline: none; border-color: var(--primary); }
.tokens-list { display: flex; flex-wrap: wrap; gap: 5px; }
.token-pill { border: 1px solid var(--outline-variant); border-radius: 3px; padding: 2px 6px; background: var(--surface-container-low); color: var(--secondary); cursor: pointer; font-size: 10px; transition: all 0.15s ease; }
.token-pill:hover { background: var(--surface-container-high); border-color: var(--secondary); }

/* ── Rename Plan Card & Diff List ─────────────────────────────────── */
.rename-plan-card { border-color: color-mix(in srgb, var(--primary) 40%, var(--outline-variant)); box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25); }
.plan-header { background: var(--surface-container-high); }
.plan-header--conflict { background: color-mix(in srgb, var(--error) 12%, var(--surface-container-high)); }
.plan-header-left { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.plan-header-actions { display: flex; align-items: center; gap: 6px; }
.btn-preview-action { background: var(--surface-container-low); border: 1px solid var(--outline-variant); border-radius: 3px; color: var(--on-surface); padding: 3px 8px; font-size: 11px; cursor: pointer; transition: all 0.12s ease; display: inline-flex; align-items: center; gap: 4px; }
.btn-preview-action:hover { background: var(--surface-container-highest, #2d3347); color: var(--primary); }
.btn-dismiss:hover { color: var(--error); border-color: color-mix(in srgb, var(--error) 40%, transparent); }
.preview-items-list { display: flex; flex-direction: column; max-height: 320px; overflow-y: auto; background: var(--surface-container-lowest); }
.rename-row { display: flex; align-items: center; gap: 10px; padding: 8px 12px; border-bottom: 1px solid color-mix(in srgb, var(--outline-variant) 40%, transparent); transition: background 0.1s ease; }
.rename-row:hover { background: rgba(255, 255, 255, 0.02); }
.rename-row--conflict { background: color-mix(in srgb, var(--error) 6%, transparent); }
.operation-tag { min-width: 74px; border-radius: 3px; padding: 2px 6px; font-size: 9.5px; text-align: center; font-weight: 700; flex-shrink: 0; }
.operation-tag--keep { background: var(--surface-container-high); color: var(--on-surface-variant); }
.operation-tag--rename, .operation-tag--rename_file { background: color-mix(in srgb, var(--secondary) 15%, transparent); color: var(--secondary); border: 1px solid color-mix(in srgb, var(--secondary) 30%, transparent); }
.operation-tag--rename_dir { background: color-mix(in srgb, var(--primary) 15%, transparent); color: var(--primary); border: 1px solid color-mix(in srgb, var(--primary) 30%, transparent); }
.operation-tag--conflict { background: color-mix(in srgb, var(--error) 15%, transparent); color: var(--error); border: 1px solid color-mix(in srgb, var(--error) 30%, transparent); }
.rename-paths { display: grid; min-width: 0; flex: 1; grid-template-columns: minmax(0, 1fr) auto minmax(0, 1.1fr); gap: 8px; align-items: center; font-size: 11px; }
.current-path { color: var(--on-surface-variant); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.planned-path { color: var(--secondary); font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.path-arrow { color: var(--outline); font-weight: 700; flex-shrink: 0; }
.dot-conflict { background: var(--error) !important; box-shadow: 0 0 6px var(--error) !important; }

.audit-alert { border: 1px solid var(--error); border-radius: 4px; padding: 6px 10px; color: var(--error); font-size: 11px; }
.select-wrapper { display: flex; align-items: center; gap: 6px; }
.control-label { font-size: 10px; color: var(--outline); }
.preset-select { border: 1px solid var(--outline-variant); border-radius: 4px; padding: 2px 6px; background: var(--surface-container-lowest); color: var(--on-surface); font-size: 10px; }

/* ── Sticky Action Bar ────────────────────────────────────────────── */
.sticky-action-bar { position: sticky; bottom: 0; display: flex; align-items: center; justify-content: space-between; padding: 10px 14px; border: 1px solid var(--outline-variant); background: var(--surface-container); z-index: 25; margin-top: 8px; border-radius: var(--radius-md, 0.375rem); box-shadow: 0 -6px 20px rgba(0, 0, 0, 0.4); }
.action-bar-right { display: flex; align-items: center; gap: 8px; }

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

@media (max-width: 900px) { .stream-groups { grid-template-columns: 1fr; } }
@media (max-width: 700px) {
  .workshop-header, .pattern-action-row, .sticky-action-bar, .card-header-bar { align-items: stretch; flex-direction: column; gap: 8px; }
  .rename-paths { grid-template-columns: 1fr; }
  .path-arrow { transform: rotate(90deg); margin: 0 auto; }
}
</style>
