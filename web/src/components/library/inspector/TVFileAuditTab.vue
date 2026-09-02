<script setup lang="ts">
import { ref } from 'vue'
import type { MediaInspection, TVEpisode } from '@/api/types'

const props = defineProps<{
  episodes: TVEpisode[]
  inspection: MediaInspection | null
  inspectionLoading: boolean
  inspectionError: string | null
  labels: Record<string, string>
}>()

const patternInput = ref('${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}')
const availableTokens = [
  '${showTitle}',
  '${seasonNumber}',
  '${seasonNumberPad}',
  '${episodeNumber}',
  '${episodeNumberPad}',
  '${episodeTitle}',
  '${resolution}',
  '${videoCodec}',
]

function formatEpisodeCode(ep: TVEpisode): string {
  const s = String(ep.seasonNumber).padStart(2, '0')
  const e = String(ep.episodeStart).padStart(2, '0')
  return `S${s}E${e}`
}
</script>

<template>
  <section class="files-workshop-view">
    <div class="workshop-header">
      <div class="header-left">
        <h2>{{ labels.filesRenamePlanner || 'Files & Rename Planner' }}</h2>
        <span class="sub-label">Preview safe TV renaming patterns, season folders and sidecars</span>
      </div>
    </div>

    <div class="rename-engine-card">
      <div class="card-header-bar">
        <span class="header-title">Naming Pattern Template Engine</span>
        <span class="preset-tag font-code">Kodi TV Standard</span>
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

    <div class="structure-card">
      <div class="card-header-bar">
        <span class="header-title">Current File & Sidecar Assets</span>
        <span class="item-count font-code">{{ episodes.length }} episodes</span>
      </div>

      <div class="file-audit-list">
        <div v-for="ep in episodes" :key="ep.id" class="audit-item">
          <span class="spec-badge video-badge font-code">{{ formatEpisodeCode(ep) }}</span>
          <span class="audit-path font-code" :title="ep.relativePath">{{ ep.relativePath }}</span>
          <span class="audit-status text-ok font-code">{{ ep.sidecars?.length || 0 }} sidecars</span>
        </div>
      </div>
    </div>

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
</style>
