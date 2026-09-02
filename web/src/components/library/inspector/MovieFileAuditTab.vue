<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import type { MediaInspection, MediaItem, NamingPreview } from '@/api/types'

const props = defineProps<{
  item: MediaItem
  labels: Record<string, string>
  inspection: MediaInspection | null
  inspectionLoading: boolean
  inspectionError: string | null
  namingPreview: NamingPreview | null
  previewLoading: boolean
}>()

const emit = defineEmits<{ previewNaming: [pattern: string] }>()
const patternInput = shallowRef('${title} (${year})')
const availableTokens = ['${title}', '${originalTitle}', '${year}', '${resolution}', '${videoCodec}', '${audioCodec}']
const hasConflicts = computed(() => props.namingPreview?.items.some(item => item.conflict) ?? false)

function appendToken(token: string) { patternInput.value += token }

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
        <h2>{{ labels.filesRenamePlanner }}</h2>
        <span class="sub-label">{{ labels.fileAuditReadOnly }}</span>
      </div>
      <span class="readonly-badge font-code">{{ labels.previewOnly }}</span>
    </div>

    <div v-if="inspectionError" class="audit-alert">{{ inspectionError }}</div>

    <div class="structure-card">
      <div class="card-header-bar">
        <span class="header-title">{{ labels.currentFiles }}</span>
        <span class="item-count font-code">{{ inspection?.files.length ?? 0 }} {{ labels.files }}</span>
      </div>
      <div v-if="inspectionLoading" class="empty-row">{{ labels.inspectingMedia }}</div>
      <div v-else-if="!inspection?.files.length" class="empty-row">{{ labels.fileAuditUnavailable }}</div>
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
                <span v-if="!file.warnings.length" class="status-ok">{{ labels.valid }}</span>
                <template v-else><span v-for="warning in file.warnings" :key="warning" class="warning-chip">{{ warningLabel(warning) }}</span></template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="streams-card">
      <div class="card-header-bar">
        <span class="header-title">{{ labels.mediaStreams }}</span>
        <span v-if="inspection?.cached" class="cache-tag font-code">{{ labels.cachedProbe }}</span>
      </div>
      <div v-if="inspection?.probeStatus !== 'ready'" class="empty-row">{{ inspectionLoading ? labels.inspectingMedia : (inspection?.probeError || labels.mediaInfoUnavailable) }}</div>
      <div v-else class="stream-groups">
        <div class="stream-group">
          <h3>{{ labels.videoStreams }} ({{ inspection.video.length }})</h3>
          <div v-for="stream in inspection.video" :key="stream.index" class="stream-row font-code">
            <span>#{{ stream.index }}</span><span>{{ stream.codec.toUpperCase() }}</span><span>{{ stream.width }}×{{ stream.height }}</span><span v-if="stream.bitDepth">{{ stream.bitDepth }}-bit</span><span v-if="stream.hdr" class="stream-accent">{{ stream.hdr }}</span><span v-if="stream.language">{{ stream.language }}</span>
          </div>
        </div>
        <div class="stream-group">
          <h3>{{ labels.audioStreams }} ({{ inspection.audio.length }})</h3>
          <div v-for="stream in inspection.audio" :key="stream.index" class="stream-row font-code">
            <span>#{{ stream.index }}</span><span>{{ stream.codec.toUpperCase() }}</span><span>{{ stream.channelLayout || `${stream.channels} ch` }}</span><span v-if="stream.language">{{ stream.language }}</span><span v-if="stream.title">{{ stream.title }}</span>
          </div>
        </div>
        <div class="stream-group">
          <h3>{{ labels.subtitleStreams }} ({{ inspection.subtitles.length }})</h3>
          <div v-if="!inspection.subtitles.length" class="stream-empty">{{ labels.noEmbeddedSubtitles }}</div>
          <div v-for="stream in inspection.subtitles" :key="stream.index" class="stream-row font-code">
            <span>#{{ stream.index }}</span><span>{{ stream.codec.toUpperCase() }}</span><span>{{ stream.language || labels.languageUnknown }}</span><span v-if="stream.title">{{ stream.title }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="rename-engine-card">
      <div class="card-header-bar"><span class="header-title">{{ labels.namingPattern }}</span><span class="preset-tag font-code">{{ labels.previewOnly }}</span></div>
      <div class="card-body">
        <label class="field-label font-code" for="movie-naming-pattern">{{ labels.patternTemplate }}</label>
        <div class="pattern-action-row">
          <input id="movie-naming-pattern" v-model="patternInput" type="text" class="pattern-input font-code" />
          <button class="btn btn-primary" :disabled="previewLoading" type="button" @click="emit('previewNaming', patternInput)">{{ previewLoading ? labels.previewing : labels.runDryRun }}</button>
        </div>
        <div class="tokens-list" :aria-label="labels.availableTokens">
          <button v-for="token in availableTokens" :key="token" class="token-pill font-code" type="button" @click="appendToken(token)">{{ token }}</button>
        </div>
      </div>
      <div v-if="namingPreview" class="rename-preview">
        <div class="preview-summary" :class="{ 'preview-summary--conflict': hasConflicts }">{{ hasConflicts ? labels.namingConflictsFound : labels.namingPreviewSafe }}</div>
        <div v-for="entry in namingPreview.items" :key="entry.currentPath" class="rename-row">
          <span class="operation-tag font-code" :class="`operation-tag--${entry.operation}`">{{ entry.operation.toUpperCase() }}</span>
          <div class="rename-paths font-code"><span class="current-path">{{ entry.currentPath }}</span><span class="path-arrow">→</span><span class="planned-path">{{ entry.plannedPath }}</span></div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.files-workshop-view { display: flex; flex-direction: column; gap: 16px; width: 100%; }
.workshop-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.header-left h2 { margin: 0 0 2px; color: var(--on-surface); font-size: 18px; }
.sub-label, .empty-row, .stream-empty { color: var(--on-surface-variant); font-size: 12px; }
.readonly-badge, .cache-tag { color: var(--secondary); border: 1px solid color-mix(in srgb, var(--secondary) 35%, transparent); border-radius: 4px; padding: 3px 7px; font-size: 10px; }
.structure-card, .streams-card, .rename-engine-card { overflow: hidden; border: 1px solid var(--outline-variant); border-radius: var(--radius-lg); background: var(--surface-container); }
.card-header-bar { display: flex; align-items: center; justify-content: space-between; padding: 8px 14px; border-bottom: 1px solid var(--outline-variant); background: var(--surface-container-high); }
.header-title { color: var(--on-surface); font-size: 13px; font-weight: 600; }
.item-count, .preset-tag { color: var(--outline); font-size: 10px; }
.empty-row { padding: 18px 14px; }
.audit-table-wrap { overflow-x: auto; }
.audit-table { width: 100%; border-collapse: collapse; font-size: 11px; text-align: left; }
.audit-table th { padding: 7px 10px; background: var(--surface-container-low); color: var(--outline); font-weight: 600; }
.audit-table td { padding: 8px 10px; border-top: 1px solid color-mix(in srgb, var(--outline-variant) 60%, transparent); color: var(--on-surface-variant); }
.audit-row--warning { border-left: 2px solid var(--tertiary); }
.audit-path { min-width: 220px; max-width: 420px; overflow: hidden; color: var(--on-surface) !important; text-overflow: ellipsis; white-space: nowrap; }
.status-ok { color: var(--secondary); }
.warning-chip { display: inline-block; margin: 1px 4px 1px 0; border: 1px solid color-mix(in srgb, var(--tertiary) 35%, transparent); border-radius: 3px; padding: 1px 5px; color: var(--tertiary); }
.stream-groups { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 1px; background: var(--outline-variant); }
.stream-group { min-width: 0; padding: 12px; background: var(--surface-container); }
.stream-group h3 { margin: 0 0 8px; color: var(--on-surface); font-size: 11px; text-transform: uppercase; }
.stream-row { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 5px; border-radius: 4px; padding: 6px 8px; background: var(--surface-container-low); color: var(--on-surface-variant); font-size: 10px; }
.stream-accent { color: var(--tertiary); }
.card-body { display: flex; flex-direction: column; gap: 8px; padding: 14px; }
.field-label { color: var(--primary); font-size: 10px; font-weight: 700; }
.pattern-action-row { display: flex; gap: 8px; }
.pattern-input { min-width: 0; flex: 1; border: 1px solid var(--outline-variant); border-radius: 4px; padding: 8px 10px; background: var(--surface-container-lowest); color: var(--on-surface); }
.tokens-list { display: flex; flex-wrap: wrap; gap: 6px; }
.token-pill { border: 1px solid var(--outline-variant); border-radius: 4px; padding: 3px 7px; background: var(--surface-container-low); color: var(--secondary); cursor: pointer; }
.rename-preview { border-top: 1px solid var(--outline-variant); }
.preview-summary { padding: 8px 14px; background: color-mix(in srgb, var(--secondary) 8%, transparent); color: var(--secondary); font-size: 11px; }
.preview-summary--conflict { background: color-mix(in srgb, var(--error) 8%, transparent); color: var(--error); }
.rename-row { display: flex; align-items: flex-start; gap: 10px; padding: 9px 14px; border-top: 1px solid color-mix(in srgb, var(--outline-variant) 60%, transparent); }
.operation-tag { min-width: 54px; color: var(--outline); font-size: 9px; }.operation-tag--rename { color: var(--secondary); }.operation-tag--conflict { color: var(--error); }
.rename-paths { display: grid; min-width: 0; flex: 1; grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr); gap: 8px; font-size: 10px; }
.current-path, .planned-path { overflow-wrap: anywhere; }.current-path { color: var(--on-surface-variant); }.planned-path { color: var(--on-surface); }.path-arrow { color: var(--outline); }
.audit-alert { border: 1px solid var(--error); border-radius: 4px; padding: 8px 12px; color: var(--error); font-size: 12px; }
@media (max-width: 900px) { .stream-groups { grid-template-columns: 1fr; } }
@media (max-width: 700px) { .workshop-header, .pattern-action-row { align-items: stretch; flex-direction: column; }.rename-paths { grid-template-columns: 1fr; }.path-arrow { transform: rotate(90deg); } }
</style>
