<script setup lang="ts">
import type { TVArtwork, TVSelection } from '@/api/types'
import TVArtworkPanel from '../TVArtworkPanel.vue'

defineProps<{
  showId: number
  selection: TVSelection | null
  posterUrl?: string
  backdropUrl?: string
  logoUrl?: string
  bannerUrl?: string
  seasonPosterUrl?: string
  scopedArtwork: TVArtwork[]
  labels: Record<string, string>
  episodeTitleText?: string
}>()
</script>

<template>
  <section class="artwork-workshop-view">
    <div class="workshop-header">
      <div class="header-left">
        <h2>
          {{ selection?.kind === 'episode' ? (labels.episodeArtwork || '单集剧照与图片') : selection?.kind === 'season' ? (labels.seasonArtwork || '季海报与背景图') : (labels.artworkGallery || '剧集图片与背景图库') }}
        </h2>
        <span class="sub-label">
          {{ selection?.kind === 'episode' ? (episodeTitleText || '单集图片') : selection?.kind === 'season' ? `第 ${selection.seasonNumber} 季专属图片资源` : 'Manage TV series posters, backdrops, logos and banners' }}
        </span>
      </div>
    </div>

    <div class="artwork-layout-grid">
      <!-- Left Column: Primary Art (Poster & Logo / Season Art) -->
      <div class="art-col-primary">
        <!-- Poster / Episode Still Card -->
        <div class="art-card">
          <div class="art-card-header">
            <div class="header-title">
              <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="title-icon">
                <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14zm-5.04-6.71l-2.75 3.54-1.96-2.36L6.5 17h11l-3.54-4.71z"/>
              </svg>
              <span>{{ selection?.kind === 'episode' ? (labels.episodeThumb || '单集剧照 / 缩略图') : selection?.kind === 'season' ? (labels.seasonPoster || '季海报 (Season Poster)') : (labels.poster || '剧集海报 (Poster)') }}</span>
            </div>
            <span class="ratio-pill font-code">{{ selection?.kind === 'episode' ? '16:9 HD' : '2:3 RATIO' }}</span>
          </div>
          <div :class="selection?.kind === 'episode' ? 'fanart-preview-box' : 'poster-preview-box'" class="group">
            <img v-if="posterUrl" :src="posterUrl" :alt="labels.poster || 'Poster'" class="preview-img" />
            <div v-else class="preview-empty">
              <svg viewBox="0 0 24 24" fill="currentColor" width="40" height="40" opacity="0.3">
                <path d="M21 3H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h5v2h8v-2h5c1.1 0 1.99-.9 1.99-2L23 5c0-1.1-.9-2-2-2zm0 14H3V5h18v12z"/>
              </svg>
              <span>{{ labels.noPosterLoaded || '暂无图片' }}</span>
            </div>
            <div v-if="posterUrl" class="art-status-overlay">
              <span class="art-dim-badge font-code">{{ selection?.kind === 'episode' ? '1920x1080' : '1000x1500' }}</span>
              <span class="art-active-badge font-code">ACTIVE</span>
            </div>
          </div>
        </div>

        <!-- Clear Logo / Season Poster Card -->
        <div class="art-card">
          <div class="art-card-header">
            <div class="header-title">
              <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="title-icon">
                <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z"/>
              </svg>
              <span>{{ selection?.kind === 'episode' ? '所属季海报' : (labels.clearLogo || '透明标志 (Clear Logo)') }}</span>
            </div>
            <span class="ratio-pill font-code">{{ selection?.kind === 'episode' ? '2:3 RATIO' : 'PNG' }}</span>
          </div>

          <!-- When episode is selected: show season poster -->
          <div v-if="selection?.kind === 'episode'" class="poster-preview-box">
            <img v-if="seasonPosterUrl" :src="seasonPosterUrl" alt="Season Poster" class="preview-img" />
            <div v-else class="preview-empty"><span>暂无季海报</span></div>
            <div v-if="seasonPosterUrl" class="art-status-overlay">
              <span class="art-dim-badge font-code">SEASON</span>
              <span class="art-active-badge font-code">ACTIVE</span>
            </div>
          </div>
          <!-- When show or season is selected: show Clear Logo -->
          <div v-else class="logo-preview-box checkerboard">
            <img v-if="logoUrl" :src="logoUrl" :alt="labels.logo || 'Logo'" class="logo-img-preview" />
            <div v-else class="preview-empty">
              <svg viewBox="0 0 24 24" fill="currentColor" width="36" height="36" opacity="0.3">
                <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z"/>
              </svg>
              <span>{{ labels.noLogoLoaded || 'No Clear Logo' }}</span>
            </div>
            <div v-if="logoUrl" class="art-status-overlay">
              <span class="art-dim-badge font-code">LOGO</span>
              <span class="art-active-badge font-code">ACTIVE</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Right Column: Wide Art (Backdrop & Banner) -->
      <div class="art-col-wide">
        <!-- Backdrop / Fanart Card -->
        <div class="art-card">
          <div class="art-card-header">
            <div class="header-title">
              <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="title-icon">
                <path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/>
              </svg>
              <span>{{ selection?.kind === 'season' ? '季背景图 (Fanart)' : (labels.fanart || '背景图 / Fanart') }}</span>
            </div>
            <span class="ratio-pill font-code">16:9 HD</span>
          </div>
          <div class="fanart-preview-box">
            <img v-if="backdropUrl" :src="backdropUrl" :alt="labels.fanart || 'Fanart'" class="preview-img" />
            <div v-else class="preview-empty">
              <svg viewBox="0 0 24 24" fill="currentColor" width="40" height="40" opacity="0.3">
                <path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/>
              </svg>
              <span>{{ labels.fanartUnavailable || 'FANART MISSING' }}</span>
            </div>
            <div v-if="backdropUrl" class="art-status-overlay">
              <span class="art-dim-badge font-code">1920x1080</span>
              <span class="art-active-badge font-code">ACTIVE</span>
            </div>
          </div>
        </div>

        <!-- Banner Card -->
        <div class="art-card">
          <div class="art-card-header">
            <div class="header-title">
              <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="title-icon">
                <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14z"/>
              </svg>
              <span>{{ selection?.kind === 'season' ? '季横幅 (Banner)' : (labels.banner || '横幅 (Banner)') }}</span>
            </div>
            <span class="ratio-pill font-code">1000x185</span>
          </div>
          <div class="banner-preview-box">
            <img v-if="bannerUrl" :src="bannerUrl" alt="Banner" class="banner-img-preview" />
            <div v-else class="banner-sample-txt font-code">{{ labels.bannerPreview || '暂无横幅' }}</div>
            <div v-if="bannerUrl" class="art-status-overlay">
              <span class="art-dim-badge font-code">BANNER</span>
              <span class="art-active-badge font-code">ACTIVE</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Local Indexed Assets Card -->
    <section class="local-artwork-card">
      <div class="art-card-header">
        <div class="header-title">
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="title-icon">
            <path d="M4 6H2v14c0 1.1.9 2 2 2h14v-2H4V6zm16-4H8c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H8V4h12v12z"/>
          </svg>
          <span>{{ selection?.kind === 'episode' ? '单集本地图片' : selection?.kind === 'season' ? '本季本地图片' : '剧集本地图片' }}</span>
        </div>
        <span class="ratio-pill font-code">{{ scopedArtwork.length }} FILES</span>
      </div>
      <TVArtworkPanel :show-id="showId" :assets="scopedArtwork" :labels="labels" />
    </section>
  </section>
</template>

<style scoped>
.artwork-workshop-view {
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

.artwork-layout-grid {
  display: grid;
  grid-template-columns: 260px 1fr;
  gap: 16px;
}

.art-col-primary,
.art-col-wide {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.art-card,
.local-artwork-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.art-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--surface-container-high, #23293c);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.header-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
}

.title-icon {
  color: var(--primary, #c0c1ff);
}

.ratio-pill {
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.poster-preview-box {
  width: 100%;
  aspect-ratio: 2 / 3;
  background: var(--surface-container-lowest, #070d1f);
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.fanart-preview-box {
  width: 100%;
  aspect-ratio: 16 / 9;
  background: var(--surface-container-lowest, #070d1f);
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo-preview-box {
  width: 100%;
  height: 120px;
  background: var(--surface-container-lowest, #070d1f);
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px;
}

.banner-preview-box {
  width: 100%;
  height: 90px;
  background: var(--surface-container-lowest, #070d1f);
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.preview-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.logo-img-preview {
  max-width: 90%;
  max-height: 90%;
  object-fit: contain;
}

.banner-img-preview {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.preview-empty {
  color: var(--outline, #908fa0);
  font-size: 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.checkerboard {
  background-image: linear-gradient(45deg, rgba(255, 255, 255, 0.03) 25%, transparent 25%),
                    linear-gradient(-45deg, rgba(255, 255, 255, 0.03) 25%, transparent 25%),
                    linear-gradient(45deg, transparent 75%, rgba(255, 255, 255, 0.03) 75%),
                    linear-gradient(-45deg, transparent 75%, rgba(255, 255, 255, 0.03) 75%);
  background-size: 16px 16px;
  background-position: 0 0, 0 8px, 8px -8px, -8px 0px;
}

.art-status-overlay {
  position: absolute;
  bottom: 8px;
  right: 8px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.art-dim-badge,
.art-active-badge {
  font-size: 9px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: var(--radius-sm, 0.25rem);
}

.art-dim-badge {
  background: rgba(0, 0, 0, 0.7);
  color: #ffffff;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.art-active-badge {
  background: var(--secondary, #4edea3);
  color: #003822;
}

.local-artwork-card {
  padding: 12px 14px;
}

@media (max-width: 900px) {
  .artwork-layout-grid {
    grid-template-columns: 1fr;
  }
}
</style>
