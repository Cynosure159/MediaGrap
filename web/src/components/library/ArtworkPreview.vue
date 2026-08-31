<script setup lang="ts">
import { computed } from 'vue'
import type { ArtworkPlan } from '@/api/library'

const props = defineProps<{ plan: ArtworkPlan | null; applying: boolean; labels: Record<string, string> }>()
const emit = defineEmits<{ apply: []; close: [] }>()
const hasConflict = computed(() => props.plan?.assets.some((asset) => asset.conflict) ?? false)
</script>

<template>
  <div v-if="plan" class="backdrop">
    <section class="dialog" role="dialog" aria-modal="true" :aria-label="labels.artworkPreview">
      <header class="dialog-header">
        <div>
          <p class="eyebrow">TMDb artwork</p>
          <strong>{{ labels.artworkPreview }}</strong>
        </div>
        <button type="button" @click="emit('close')">×</button>
      </header>
      <p class="dialog-help">{{ labels.artworkHelp }}</p>
      <ul class="asset-list">
        <li v-for="asset in plan.assets" :key="asset.kind" :class="{ conflict: asset.conflict }">
          <div>
            <strong>{{ asset.kind === 'poster' ? labels.poster : labels.fanart }}</strong>
            <small>{{ asset.targetPath }}</small>
          </div>
          <span v-if="asset.conflict">{{ labels.artworkConflict }}</span>
          <span v-else-if="asset.willReplace">{{ labels.artworkReplace }}</span>
          <span v-else>{{ labels.artworkCreate }}</span>
        </li>
      </ul>
      <footer class="dialog-footer">
        <button type="button" @click="emit('close')">{{ labels.cancel }}</button>
        <button type="button" :disabled="hasConflict || applying" @click="emit('apply')">{{ applying ? labels.downloading : labels.downloadArtwork }}</button>
      </footer>
    </section>
  </div>
</template>

<style scoped>
.backdrop { position: fixed; inset: 0; z-index: 10; display: grid; place-items: end center; padding: 1rem; background: #102b3e99; }
.dialog { width: min(100%, 42rem); padding: 1.2rem; border-radius: 1rem; background: var(--paper); box-shadow: 0 1.5rem 4rem #102b3e66; }
.dialog-header, .dialog-footer, .asset-list li { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
.dialog-header button, .dialog-footer button { min-height: 2.5rem; padding: .5rem .8rem; border: 1px solid var(--ink-700); border-radius: .4rem; background: var(--ink-900); color: white; cursor: pointer; }
.dialog-header button, .dialog-footer button:first-child { background: var(--paper); color: var(--ink-900); }
.dialog-help { color: var(--ink-600); }
.asset-list { display: grid; gap: .55rem; margin: 1rem 0; padding: 0; list-style: none; }
.asset-list li { padding: .75rem; border: 1px solid var(--mist-300); border-left: 3px solid var(--water-600); border-radius: .45rem; }
.asset-list li.conflict { border-left-color: #b44b27; background: #f9e1d6; }
.asset-list small { display: block; margin-top: .2rem; color: var(--ink-600); overflow-wrap: anywhere; }
.asset-list span { max-width: 12rem; color: var(--ink-600); font-size: .78rem; text-align: right; }
@media (min-width: 700px) { .backdrop { place-items: center; } }
</style>
