<script setup lang="ts">
defineProps<{
  posterUrl: string
  backdropUrl: string
  labels: Record<string, string>
}>()
</script>

<template>
  <section class="workshop-view">
    <div class="workshop-header">
      <h2>{{ labels.artworkGallery }}</h2>
      <div class="workshop-actions">
        <button class="btn btn-outline" type="button">{{ labels.uploadCustomImage }}</button>
        <button class="btn btn-primary" type="button">{{ labels.scrapeFanart }}</button>
      </div>
    </div>

    <div class="artwork-grid">
      <div class="art-slot-card">
        <div class="slot-header">
          <span class="slot-title">{{ labels.poster }}</span>
        </div>
        <div class="slot-preview poster-slot">
          <img v-if="posterUrl" :src="posterUrl" :alt="labels.poster" />
          <div v-else class="slot-empty">{{ labels.noPosterLoaded }}</div>
        </div>
      </div>

      <div class="art-slot-card">
        <div class="slot-header">
          <span class="slot-title">{{ labels.fanart }}</span>
        </div>
        <div class="slot-preview fanart-slot">
          <img v-if="backdropUrl" :src="backdropUrl" :alt="labels.fanart" />
          <div v-else class="slot-empty">{{ labels.fanartUnavailable }}</div>
        </div>
      </div>

      <div class="art-slot-card">
        <div class="slot-header">
          <span class="slot-title">{{ labels.clearLogo }}</span>
        </div>
        <div class="slot-preview logo-slot">
          <img src="/assets/logo-icon.png" :alt="labels.logo" class="logo-sample" />
        </div>
      </div>

      <div class="art-slot-card">
        <div class="slot-header">
          <span class="slot-title">{{ labels.banner }}</span>
          <span class="spec-badge">1000x185</span>
        </div>
        <div class="slot-preview banner-slot">
          <div class="banner-sample">{{ labels.bannerPreview }}</div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.workshop-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.workshop-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.workshop-header h2 {
  font-size: 16px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  margin: 0;
}

.workshop-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.artwork-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
}

.art-slot-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.5rem);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.slot-header {
  padding: 8px 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.slot-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--outline, #908fa0);
  letter-spacing: 0.04em;
}

.slot-preview {
  background: var(--surface-container-high, #23293c);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.poster-slot {
  aspect-ratio: 2 / 3;
}

.fanart-slot {
  aspect-ratio: 16 / 9;
}

.logo-slot,
.banner-slot {
  height: 120px;
}

.slot-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.logo-sample {
  width: 64px !important;
  height: 64px !important;
  object-fit: contain !important;
}

.banner-sample {
  font-size: 11px;
  color: var(--outline, #908fa0);
}

.slot-empty {
  font-size: 11px;
  color: var(--outline, #908fa0);
  text-align: center;
  padding: 8px;
}
</style>
