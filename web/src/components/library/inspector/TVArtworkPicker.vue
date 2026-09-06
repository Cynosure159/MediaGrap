<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue'
import * as api from '@/api/library'
import type { TVArtwork, TVSelection } from '@/api/types'
import ArtworkSelectionDialog from '../ArtworkSelectionDialog.vue'
import { useTVArtwork, type TVArtworkContext } from '@/composables/useTVArtwork'

const props = defineProps<{
  showId: number
  selection: TVSelection | null
  scopedArtwork: TVArtwork[]
  csrfToken: string
  writable: boolean
  labels: Record<string, string>
}>()

const emit = defineEmits<{ applied: [] }>()
const isCandidateDialogOpen = shallowRef(false)
const context = computed<TVArtworkContext | null>(() => {
  if (props.selection?.kind === 'season') return { scope: 'season', seasonNumber: props.selection.seasonNumber }
  if (!props.selection || props.selection.kind === 'show') return { scope: 'show' }
  return null
})

function hasExistingArtwork(kind: api.TVArtworkCandidate['kind']) {
  const keyword = kind.replace('season_', '')
  return props.scopedArtwork.some(asset => {
    const name = asset.relativePath.split('/').pop()?.toLowerCase() || ''
    return name.includes(keyword.replace('_', '')) || name.includes(keyword.replace('_', '-'))
  })
}

const artwork = useTVArtwork(
  () => props.showId,
  () => context.value ?? { scope: 'show' },
  () => props.csrfToken,
  hasExistingArtwork,
)

watch([() => props.showId, context], () => {
  if (context.value) artwork.load()
}, { immediate: true })

async function openCandidateDialog() {
  if (!context.value) return
  isCandidateDialogOpen.value = true
  await artwork.scrape()
}

async function previewSelectedArtwork() {
  await artwork.preview()
  if (artwork.plan.value) isCandidateDialogOpen.value = false
}

async function apply() {
  await artwork.apply()
  if (artwork.plan.value?.state === 'queued') emit('applied')
}
</script>

<template>
  <div class="tv-artwork-picker">
    <template v-if="context">
      <button class="btn btn-primary" type="button" :disabled="artwork.isLoading.value || !writable" @click="openCandidateDialog">
        {{ labels.scrapeFanart || 'Scrape Fanart.tv artwork' }}
      </button>
      <p v-if="artwork.error.value && !isCandidateDialogOpen && !artwork.plan.value" class="picker-error" role="alert">{{ artwork.error.value }}</p>
      <section v-if="artwork.plan.value" class="plan-summary">
        <p>{{ labels.artworkPreview || 'Artwork preview' }} · {{ artwork.plan.value.assets.length }}</p>
        <ul class="plan-list"><li v-for="asset in artwork.plan.value.assets" :key="asset.kind">{{ asset.targetPath }}</li></ul>
        <div class="plan-actions">
          <button class="btn btn-ghost" type="button" @click="artwork.closePlan">{{ labels.cancel }}</button>
          <button class="btn btn-primary" type="button" :disabled="artwork.isApplying.value" @click="apply">{{ labels.artworkApply || 'Queue download' }}</button>
        </div>
      </section>
      <ArtworkSelectionDialog
        v-if="isCandidateDialogOpen"
        :groups="artwork.groups.value"
        :selected="artwork.selected"
        :labels="labels"
        :loading="artwork.isLoading.value"
        :writable="writable"
        :error="artwork.error.value"
        :preview-url="candidate => api.tvArtworkPreviewUrl(showId, candidate.id)"
        @close="isCandidateDialogOpen = false"
        @scrape="artwork.scrape"
        @select="artwork.select"
        @preview="previewSelectedArtwork"
      />
    </template>
    <p v-else class="picker-hint">{{ labels.episodeArtwork || 'Fanart.tv artwork is available for shows and seasons.' }}</p>
  </div>
</template>

<style scoped>
.tv-artwork-picker { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.picker-error { margin: 0; color: var(--error, #ffb4ab); font-size: 12px; }
.picker-hint { margin: 0; color: var(--outline, #908fa0); font-size: 12px; }
.plan-summary { width: 100%; padding: 10px 12px; border: 1px solid var(--outline-variant, #2e3447); border-radius: var(--radius-md, .375rem); background: var(--surface-container-low, #151b2d); }
.plan-summary p { margin: 0; color: var(--on-surface, #dce1fb); font-size: 12px; }
.plan-list { max-height: 100px; margin: 8px 0; padding-left: 18px; overflow: auto; color: var(--outline, #908fa0); font: 10px var(--font-data, monospace); }
.plan-actions { display: flex; justify-content: flex-end; gap: 8px; }
</style>
