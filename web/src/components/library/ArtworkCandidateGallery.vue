<script setup lang="ts">
import * as api from '@/api/library'
import type { ArtworkCandidate } from '@/api/library'

defineProps<{
  groups: { kind: ArtworkCandidate['kind']; items: ArtworkCandidate[] }[]
  selected: Record<string, string>
  labels: Record<string, string>
  existingKinds?: Partial<Record<ArtworkCandidate['kind'], boolean>>
}>()
const emit = defineEmits<{ select: [candidate: ArtworkCandidate] }>()

function labelFor(kind: ArtworkCandidate['kind'], labels: Record<string, string>) {
  const keys: Record<ArtworkCandidate['kind'], string> = { poster: 'poster', fanart: 'fanart', clearlogo: 'clearLogo', clearart: 'clearArt', discart: 'discArt', banner: 'banner', landscape: 'landscape' }
  return labels[keys[kind]] || kind
}
</script>

<template>
  <section class="candidate-gallery" :aria-label="labels.artworkCandidates">
    <header class="gallery-header">
      <div>
        <p class="eyebrow">{{ labels.artworkEyebrow }}</p>
        <h3>{{ labels.artworkCandidates }}</h3>
      </div>
      <span class="gallery-hint">{{ labels.artworkSelect }}</span>
    </header>
    <p v-if="groups.length === 0" class="gallery-empty">{{ labels.artworkNoCandidates }}</p>
    <div v-for="group in groups" :key="group.kind" class="candidate-group">
      <div class="group-title">
        <strong>{{ labelFor(group.kind, labels) }}</strong>
        <span class="group-meta">
          <span v-if="existingKinds?.[group.kind]" class="existing-badge">{{ labels.artworkExisting || 'Already on disk' }}</span>
          <span>{{ group.items.length }}</span>
        </span>
      </div>
      <div class="candidate-grid">
        <button v-for="candidate in group.items" :key="candidate.id" type="button" class="candidate-card" :class="{ selected: selected[group.kind] === candidate.id }" @click="emit('select', candidate)">
          <img :src="api.artworkPreviewUrl(candidate.mediaItemId, candidate.id)" :alt="labelFor(group.kind, labels)" loading="lazy">
          <span class="candidate-meta"><b>{{ selected[group.kind] === candidate.id ? labels.artworkSelected : labels.artworkSelect }}</b><small>{{ candidate.width }}×{{ candidate.height }} · {{ candidate.language || '00' }} · ♥ {{ candidate.likes }}</small></span>
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.candidate-gallery { padding: 12px; border: 1px solid var(--outline-variant, #2e3447); border-radius: var(--radius-lg, .5rem); background: var(--surface-container, #191f31); }
.gallery-header, .group-title { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.gallery-header h3 { margin: 2px 0 0; color: var(--on-surface, #dce1fb); font-size: 15px; }
.eyebrow { margin: 0; color: var(--primary, #c0c1ff); font: 700 10px var(--font-data, monospace); text-transform: uppercase; letter-spacing: .08em; }
.gallery-hint, .group-title span { color: var(--outline, #908fa0); font-size: 11px; }
.group-meta { display: flex; align-items: center; gap: 8px; }
.existing-badge { color: var(--secondary, #4edea3) !important; font: 9px var(--font-data, monospace); }
.gallery-empty { color: var(--on-surface-variant, #c7c4d7); font-size: 12px; }
.candidate-group { margin-top: 14px; }
.group-title { margin-bottom: 7px; color: var(--on-surface-variant, #c7c4d7); font-size: 12px; }
.candidate-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(112px, 1fr)); gap: 8px; }
.candidate-card { min-width: 0; padding: 0; overflow: hidden; border: 1px solid var(--outline-variant, #2e3447); border-radius: 5px; background: var(--surface-container-low, #151b2d); color: inherit; text-align: left; cursor: pointer; }
.candidate-card:hover, .candidate-card.selected { border-color: var(--primary, #c0c1ff); box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary, #c0c1ff) 30%, transparent); }
.candidate-card img { display: block; width: 100%; height: 94px; object-fit: cover; background: var(--surface-container-lowest, #070d1f); }
.candidate-meta { display: grid; gap: 3px; padding: 6px; }
.candidate-meta b { color: var(--on-surface, #dce1fb); font-size: 10px; }
.candidate-meta small { overflow: hidden; color: var(--outline, #908fa0); font: 9px var(--font-data, monospace); text-overflow: ellipsis; white-space: nowrap; }
</style>
