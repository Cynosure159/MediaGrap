<script setup lang="ts">
import { computed } from 'vue'
import * as api from '@/api/library'
import type { TVArtwork } from '@/api/types'

const props = defineProps<{
  showId: number
  assets: TVArtwork[]
  labels: Record<string, string>
}>()

const visibleAssets = computed(() => props.assets.map(asset => ({
  ...asset,
  url: api.tvArtworkUrl(props.showId, asset.id),
})))
</script>

<template>
  <section class="tv-artwork-panel" :aria-label="labels.artworkTab">
    <p v-if="visibleAssets.length === 0" class="empty-copy">{{ labels.noLocalArtwork }}</p>
    <div v-else class="artwork-grid">
      <figure v-for="asset in visibleAssets" :key="asset.id" class="artwork-card">
        <img :src="asset.url" :alt="asset.relativePath" class="artwork-image">
        <figcaption class="artwork-caption">
          <span class="spec-pill">{{ asset.kind }}</span>
          <span class="artwork-path font-code">{{ asset.relativePath }}</span>
        </figcaption>
      </figure>
    </div>
  </section>
</template>

<style scoped>
.tv-artwork-panel { padding: 20px; overflow: auto; }
.empty-copy { margin: 0; color: var(--outline); font-size: 0.8125rem; }
.artwork-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 14px; }
.artwork-card { margin: 0; overflow: hidden; border: 1px solid var(--outline-variant); border-radius: 8px; background: var(--surface-container-low); }
.artwork-image { display: block; width: 100%; aspect-ratio: 2 / 3; object-fit: cover; background: var(--surface-container-lowest); }
.artwork-caption { display: grid; gap: 7px; padding: 9px; }
.artwork-path { overflow: hidden; color: var(--on-surface-variant); font-size: 0.6875rem; text-overflow: ellipsis; white-space: nowrap; }
</style>
