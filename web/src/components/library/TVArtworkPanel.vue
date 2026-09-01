<script setup lang="ts">
import { computed } from 'vue'
import * as api from '@/api/library'
import type { TVArtwork } from '@/api/types'

const props = defineProps<{
  showId: number
  assets: TVArtwork[]
  labels: Record<string, string>
}>()

const visibleAssets = computed(() => props.assets.map(asset => {
  const filename = asset.relativePath.split('/').pop() || ''
  return {
    ...asset,
    filename,
    url: api.tvArtworkUrl(props.showId, asset.id),
  }
}))
</script>

<template>
  <div class="tv-artwork-panel" :aria-label="labels.artworkTab">
    <p v-if="visibleAssets.length === 0" class="artwork-empty">{{ labels.noLocalArtwork || '暂无本地图片资源' }}</p>
    <div v-else class="local-artwork-grid">
      <figure v-for="asset in visibleAssets" :key="asset.id" class="local-artwork-item">
        <img :src="asset.url" :alt="asset.relativePath" class="local-artwork-image" loading="lazy">
        <figcaption class="font-code" :title="asset.relativePath">{{ asset.filename }}</figcaption>
      </figure>
    </div>
  </div>
</template>

<style scoped>
.tv-artwork-panel {
  width: 100%;
}

.local-artwork-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
  gap: 10px;
  margin-top: 10px;
}

.local-artwork-item {
  margin: 0;
  overflow: hidden;
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: 4px;
  background: var(--surface-container-low, #151b2d);
}

.local-artwork-image {
  display: block;
  width: 100%;
  aspect-ratio: 1.4;
  object-fit: cover;
  background: var(--surface-container-lowest, #070d1f);
}

.local-artwork-item figcaption {
  overflow: hidden;
  padding: 6px;
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.artwork-empty {
  margin: 10px 0 0;
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 12px;
}
</style>
