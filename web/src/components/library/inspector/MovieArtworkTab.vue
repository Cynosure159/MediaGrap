<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import * as api from '@/api/library'
import type { ArtworkCandidate, SidecarAsset } from '@/api/types'
import ArtworkSelectionDialog from '../ArtworkSelectionDialog.vue'
import ArtworkPreview from '../ArtworkPreview.vue'
import { useArtwork } from '@/composables/useArtwork'

const props = defineProps<{
  itemId: number
  sidecars: SidecarAsset[]
  posterUrl: string
  backdropUrl: string
  csrfToken: string
  writable: boolean
  labels: Record<string, string>
}>()
const emit = defineEmits<{ applied: [] }>()

interface ResolvedAsset extends SidecarAsset {
  url: string
  filename: string
}

const localArtwork = computed<ResolvedAsset[]>(() =>
  props.sidecars
    .filter(asset => asset.kind === 'image')
    .map(asset => {
      const filename = asset.relativePath.split('/').pop() || ''
      return {
        ...asset,
        filename,
        url: api.mediaArtworkUrl(props.itemId, asset.relativePath),
      }
    })
)

const artworkKinds: ArtworkCandidate['kind'][] = ['poster', 'fanart', 'clearlogo', 'clearart', 'discart', 'banner', 'landscape']
const artworkKeywords: Record<ArtworkCandidate['kind'], string[]> = {
  poster: ['poster', 'cover', 'folder'],
  fanart: ['fanart', 'backdrop', 'background'],
  clearlogo: ['clearlogo', 'logo'],
  clearart: ['clearart'],
  discart: ['discart', 'disc'],
  banner: ['banner'],
  landscape: ['landscape', 'keyart'],
}

function artworkNameMatches(filename: string, keywords: string[]) {
  const name = filename.replace(/\.[^/.]+$/, '').toLowerCase()
  return keywords.some(keyword => name === keyword || name.includes(keyword))
}

const existingArtworkKinds = computed<Partial<Record<ArtworkCandidate['kind'], boolean>>>(() =>
  Object.fromEntries(artworkKinds.map(kind => [kind, localArtwork.value.some(asset => artworkNameMatches(asset.filename, artworkKeywords[kind]))]))
)

const { groups, selected, plan, error, isLoading, isApplying, load, scrape, select, preview, apply, closePlan } = useArtwork(
  () => props.itemId,
  () => props.csrfToken,
  kind => existingArtworkKinds.value[kind] === true,
)
const isCandidateDialogOpen = ref(false)
onMounted(() => load())
watch(() => props.itemId, () => load())

async function openCandidateDialog() {
  if (!props.writable) return
  isCandidateDialogOpen.value = true
  await scrape()
}

async function previewSelectedArtwork() {
  if (!props.writable) return
  await preview()
  if (plan.value) isCandidateDialogOpen.value = false
}

async function handleApply() {
  if (!props.writable) return
  await apply()
	if (plan.value?.state === 'queued') {
    emit('applied')
  }
}

function findArtworkUrl(keywords: string[]): string | undefined {
  const match = localArtwork.value.find(asset => {
    const nameWithoutExt = asset.filename.replace(/\.[^/.]+$/, '').toLowerCase()
    return keywords.some(k => nameWithoutExt === k.toLowerCase() || nameWithoutExt.includes(k.toLowerCase()))
  })
  return match?.url
}

const resolvedPosterUrl = computed(() => {
  return findArtworkUrl(['poster', 'cover', 'folder']) || props.posterUrl || ''
})

const resolvedBackdropUrl = computed(() => {
  return findArtworkUrl(['fanart', 'backdrop', 'background', 'keyart']) || props.backdropUrl || ''
})

const resolvedLogoUrl = computed(() => {
  return findArtworkUrl(['clearlogo', 'logo', 'clearart']) || ''
})

const resolvedBannerUrl = computed(() => {
  return findArtworkUrl(['banner']) || ''
})
</script>

<template>
  <section class="artwork-workshop-view">
    <!-- Top Action Bar -->
    <div class="workshop-header">
      <div class="header-left">
        <h2>{{ labels.artworkGallery || 'Artwork Gallery' }}</h2>
        <span class="sub-label">Manage poster, fanart, logo and banner assets</span>
      </div>
      <div class="workshop-actions">
        <button class="btn btn-outline" type="button">
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM14 13v4h-4v-4H7l5-5 5 5h-3z"/>
          </svg>
          {{ labels.uploadCustomImage || 'Upload Custom' }}
        </button>
        <button class="btn btn-primary" type="button" :disabled="isLoading || !props.writable" @click="openCandidateDialog">
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
            <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
          </svg>
          {{ labels.scrapeFanart || 'Scrape All Artwork' }}
        </button>
      </div>
    </div>

    <div v-if="error && !isCandidateDialogOpen && !plan" class="artwork-error" role="alert">{{ error }}</div>
    <div class="artwork-entry-card">
      <div>
        <strong>{{ labels.artworkCandidates }}</strong>
        <span>{{ groups.length ? `${groups.length} ${labels.artworkGroups || 'groups'} · ${Object.values(selected).filter(Boolean).length} ${labels.artworkSelected}` : (labels.artworkNoCandidates || 'No artwork candidates loaded.') }}</span>
      </div>
      <button class="btn btn-outline" type="button" :disabled="isLoading || !props.writable" @click="openCandidateDialog">
        {{ labels.artworkRefresh || labels.scrapeFanart || 'Open artwork picker' }}
      </button>
    </div>

    <!-- Artwork Bento Grid -->
    <div class="artwork-layout-grid">
      <!-- Left Column: Primary Art (Poster & Logo) -->
      <div class="art-col-primary">
        <!-- Poster Section -->
        <div class="art-card">
          <div class="art-card-header">
            <div class="header-title">
              <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="title-icon">
                <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14zm-5.04-6.71l-2.75 3.54-1.96-2.36L6.5 17h11l-3.54-4.71z"/>
              </svg>
              <span>{{ labels.poster || 'Poster' }}</span>
            </div>
            <span class="ratio-pill font-code">2:3 RATIO</span>
          </div>

          <div class="poster-preview-box group">
            <img v-if="resolvedPosterUrl" :src="resolvedPosterUrl" :alt="labels.poster || 'Poster'" class="preview-img" />
            <div v-else class="preview-empty">
              <svg viewBox="0 0 24 24" fill="currentColor" width="40" height="40" opacity="0.3">
                <path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
              </svg>
              <span>{{ labels.noPosterLoaded || 'No Poster' }}</span>
            </div>

            <div v-if="resolvedPosterUrl" class="art-status-overlay">
              <span class="art-dim-badge font-code">1000x1500</span>
              <span class="art-active-badge font-code">ACTIVE</span>
            </div>
          </div>
        </div>

        <!-- Clear Logo Section -->
        <div class="art-card">
          <div class="art-card-header">
            <div class="header-title">
              <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="title-icon">
                <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z"/>
              </svg>
              <span>{{ labels.clearLogo || 'Clear Logo' }}</span>
            </div>
            <span class="ratio-pill font-code">PNG</span>
          </div>

          <div class="logo-preview-box checkerboard">
            <img v-if="resolvedLogoUrl" :src="resolvedLogoUrl" :alt="labels.logo || 'Logo'" class="logo-img-preview" />
            <div v-else class="preview-empty">
              <svg viewBox="0 0 24 24" fill="currentColor" width="36" height="36" opacity="0.3">
                <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z"/>
              </svg>
              <span>{{ labels.noLogoLoaded || 'No Clear Logo' }}</span>
            </div>
            <div v-if="resolvedLogoUrl" class="art-status-overlay">
              <span class="art-dim-badge font-code">LOGO</span>
              <span class="art-active-badge font-code">ACTIVE</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Right Column: Wide Art (Backdrop & Banner) -->
      <div class="art-col-wide">
        <!-- Backdrop / Fanart Section -->
        <div class="art-card">
          <div class="art-card-header">
            <div class="header-title">
              <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="title-icon">
                <path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/>
              </svg>
              <span>{{ labels.fanart || 'Fanart / Backdrop' }}</span>
            </div>
            <span class="ratio-pill font-code">16:9 HD</span>
          </div>

          <div class="fanart-preview-box">
            <img v-if="resolvedBackdropUrl" :src="resolvedBackdropUrl" :alt="labels.fanart || 'Fanart'" class="preview-img" />
            <div v-else class="preview-empty">
              <svg viewBox="0 0 24 24" fill="currentColor" width="40" height="40" opacity="0.3">
                <path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/>
              </svg>
              <span>{{ labels.fanartUnavailable || 'FANART MISSING' }}</span>
            </div>

            <div v-if="resolvedBackdropUrl" class="art-status-overlay">
              <span class="art-dim-badge font-code">1920x1080</span>
              <span class="art-active-badge font-code">ACTIVE</span>
            </div>
          </div>
        </div>

        <!-- Banner Section -->
        <div class="art-card">
          <div class="art-card-header">
            <div class="header-title">
              <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="title-icon">
                <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14z"/>
              </svg>
              <span>{{ labels.banner || 'Banner' }}</span>
            </div>
            <span class="ratio-pill font-code">1000x185</span>
          </div>

          <div class="banner-preview-box">
            <img v-if="resolvedBannerUrl" :src="resolvedBannerUrl" :alt="labels.banner || 'Banner'" class="banner-img-preview" />
            <div v-else class="banner-sample-txt font-code">{{ labels.bannerPreview || 'No Banner Found' }}</div>
            <div v-if="resolvedBannerUrl" class="art-status-overlay">
              <span class="art-dim-badge font-code">BANNER</span>
              <span class="art-active-badge font-code">ACTIVE</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Local Artwork Gallery (All 9+ files categorized) -->
    <section class="local-artwork-card">
      <div class="art-card-header">
        <div class="header-title"><span>{{ labels.localArtwork || 'Local artwork' }}</span></div>
        <span class="ratio-pill font-code">{{ localArtwork.length }} FILES</span>
      </div>
      <p v-if="localArtwork.length === 0" class="artwork-empty">{{ labels.noLocalArtwork || 'No local artwork found.' }}</p>
      <div v-else class="local-artwork-grid">
        <figure v-for="asset in localArtwork" :key="asset.relativePath" class="local-artwork-item">
          <img :src="asset.url" :alt="asset.relativePath" class="local-artwork-image" loading="lazy">
          <figcaption class="font-code">{{ asset.filename }}</figcaption>
        </figure>
      </div>
    </section>
    <ArtworkPreview
      v-if="plan"
      :plan="plan"
      :applying="isApplying || !writable"
      :error="error"
      :labels="labels"
      @apply="handleApply"
      @close="closePlan"
    />
    <ArtworkSelectionDialog
      v-if="isCandidateDialogOpen"
      :groups="groups"
      :selected="selected"
      :labels="labels"
      :loading="isLoading"
      :writable="writable"
      :error="error"
      :existing-kinds="existingArtworkKinds"
	  :preview-url="candidate => api.artworkPreviewUrl(props.itemId, candidate.id)"
      @close="isCandidateDialogOpen = false"
      @scrape="scrape"
      @select="select"
      @preview="previewSelectedArtwork"
    />
  </section>
</template>

<style scoped>
.artwork-workshop-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}
.artwork-error { padding: 8px 10px; border: 1px solid var(--error, #ffb4ab); border-radius: 5px; background: color-mix(in srgb, var(--error, #ffb4ab) 12%, transparent); color: var(--error, #ffb4ab); font-size: 12px; }
.artwork-entry-card { display: flex; align-items: center; justify-content: space-between; gap: 14px; padding: 12px 14px; border: 1px solid var(--outline-variant, #2e3447); border-radius: var(--radius-lg, .5rem); background: var(--surface-container, #191f31); }
.artwork-entry-card > div { display: grid; gap: 4px; min-width: 0; }
.artwork-entry-card strong { color: var(--on-surface, #dce1fb); font-size: 13px; }
.artwork-entry-card span { overflow: hidden; color: var(--on-surface-variant, #c7c4d7); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }

@media (max-width: 700px) {
  .artwork-entry-card { align-items: stretch; flex-direction: column; }
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

.workshop-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.artwork-layout-grid {
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: 18px;
}

.local-artwork-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  padding: 12px;
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

.art-col-primary,
.art-col-wide {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.art-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.art-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
}

.title-icon {
  color: var(--primary, #c0c1ff);
}

.ratio-pill {
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  color: var(--outline, #908fa0);
  font-size: 9px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: var(--radius-sm, 0.25rem);
}

.poster-preview-box {
  position: relative;
  width: 100%;
  aspect-ratio: 2 / 3;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  overflow: hidden;
}

.fanart-preview-box {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 9;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  overflow: hidden;
}

.logo-preview-box {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 9;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px;
}

.logo-img-preview {
  max-width: 85%;
  max-height: 80%;
  object-fit: contain;
  filter: drop-shadow(0 2px 8px rgba(0, 0, 0, 0.6));
}

.banner-preview-box {
  position: relative;
  width: 100%;
  height: 90px;
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.banner-img-preview {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.banner-sample-txt {
  font-size: 11px;
  color: var(--outline, #908fa0);
}

.preview-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.checkerboard {
  background-image: linear-gradient(45deg, #151b2d 25%, transparent 25%),
    linear-gradient(-45deg, #151b2d 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, #151b2d 75%),
    linear-gradient(-45deg, transparent 75%, #151b2d 75%);
  background-size: 16px 16px;
  background-position: 0 0, 0 8px, 8px -8px, -8px 0px;
}

.preview-empty {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--outline, #908fa0);
  font-size: 11px;
}

.art-status-overlay {
  position: absolute;
  bottom: 8px;
  left: 8px;
  right: 8px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  pointer-events: none;
}

.art-dim-badge {
  background: rgba(12, 19, 36, 0.85);
  backdrop-filter: blur(4px);
  border: 1px solid var(--outline-variant, #2e3447);
  color: var(--on-surface, #dce1fb);
  font-size: 10px;
  padding: 2px 6px;
  border-radius: var(--radius-sm, 0.25rem);
}

.art-active-badge {
  background: rgba(0, 165, 114, 0.9);
  color: #ffffff;
  font-size: 9px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: var(--radius-sm, 0.25rem);
  letter-spacing: 0.05em;
}

@media (max-width: 900px) {
  .artwork-layout-grid {
    grid-template-columns: 1fr;
  }
}
</style>
