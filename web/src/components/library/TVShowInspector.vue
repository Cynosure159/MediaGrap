<script setup lang="ts">
import { computed, reactive, shallowRef, watch } from 'vue'
import * as api from '@/api/library'
import InspectorToolbar, { type InspectorTab } from './inspector/InspectorToolbar.vue'
import TVOverviewTab from './inspector/TVOverviewTab.vue'
import TVArtworkTab from './inspector/TVArtworkTab.vue'
import TVFileAuditTab from './inspector/TVFileAuditTab.vue'
import MovieCastTab, { type CastMember as DisplayCastMember } from './inspector/MovieCastTab.vue'
import MovieNfoTab from './inspector/MovieNfoTab.vue'
import ScraperModal from './ScraperModal.vue'
import { useMediaInspection } from '@/composables/useMediaInspection'

const props = defineProps<{
  selection: api.TVSelection | null
  activeTab?: InspectorTab
  csrfToken: string
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  close: []
  selectTab: [tab: InspectorTab]
  metadataSaved: []
}>()

const detail = shallowRef<api.TVShowDetail | null>(null)
const localActiveTab = shallowRef<InspectorTab>('overview')
const activeTab = computed(() => props.activeTab ?? localActiveTab.value)
const showScraperModal = shallowRef(false)
const selectedEpisodeId = shallowRef<number | null>(null)
const isEditing = shallowRef(false)
const isSaving = shallowRef(false)
const isLoading = shallowRef(false)
const error = shallowRef<string | null>(null)
const isLocked = shallowRef(false)
let detailRequestSequence = 0
let nfoRequestSequence = 0

const showId = computed(() => props.selection?.showId ?? null)
const inspectionEpisodeId = computed(() =>
  props.selection?.kind === 'episode' ? props.selection.episodeId : null
)
const {
  inspection,
  isLoading: isInspectionLoading,
  error: inspectionError,
} = useMediaInspection(() => inspectionEpisodeId.value, () => props.csrfToken)

const selectedUnitEpisode = computed(() => {
  const selection = props.selection
  if (selection?.kind !== 'episode') return null
  return detail.value?.episodes.find(item => item.id === selection.episodeId) ?? null
})

const selectedUnitSeasonEpisodes = computed(() =>
  props.selection?.kind === 'season' ? (seasonsMap.value.get(props.selection.seasonNumber) ?? []) : []
)

const selectedUnitTitle = computed(() => {
  if (props.selection?.kind === 'season') return `${props.labels.season} ${props.selection.seasonNumber}`
  if (selectedUnitEpisode.value) return `${formatEpisodeCode(selectedUnitEpisode.value)} · ${episodeTitle(selectedUnitEpisode.value)}`
  return draft.title || detail.value?.show.titleHint || ''
})

const selectedUnitOverview = computed(() =>
  selectedUnitEpisode.value ? (remoteEpisode(selectedUnitEpisode.value)?.overview || props.labels.noOverview || '') : draft.overview
)

const selectedUnitPath = computed(() =>
  selectedUnitEpisode.value?.relativePath || detail.value?.show.relativePath || ''
)

const totalFileSize = computed(() => {
  if (!detail.value?.episodes) return 0
  return detail.value.episodes.reduce((acc, ep) => acc + (ep.fileSize || 0), 0)
})

const selectedUnitSize = computed(() =>
  selectedUnitEpisode.value?.fileSize ??
  (props.selection?.kind === 'season' ? selectedUnitSeasonEpisodes.value.reduce((total, item) => total + item.fileSize, 0) : totalFileSize.value)
)

const scopedEpisodes = computed(() => {
  if (props.selection?.kind === 'season') return selectedUnitSeasonEpisodes.value
  if (selectedUnitEpisode.value) return [selectedUnitEpisode.value]
  return detail.value?.episodes ?? []
})

const scopedArtwork = computed(() => {
  const assets = detail.value?.artwork ?? []
  if (!props.selection || props.selection.kind === 'show') {
    return assets.filter(asset => {
      const parts = asset.relativePath.split('/')
      return !parts.some(p => /^season\s*\d+/i.test(p) || /^specials$/i.test(p))
    })
  }

  if (props.selection.kind === 'season') {
    const sNum = props.selection.seasonNumber
    const sPadded = String(sNum).padStart(2, '0')
    const sRaw = String(sNum)
    const isSpecial = sNum === 0
    const seasonDirRegex = new RegExp(`^season\\s*(${sRaw}|${sPadded})$`, 'i')

    return assets.filter(asset => {
      const filename = asset.relativePath.split('/').pop()?.toLowerCase() || ''
      const parts = asset.relativePath.split('/')
      const inSeasonDir = parts.some(p => seasonDirRegex.test(p) || (isSpecial && /^specials$/i.test(p)))
      if (inSeasonDir) return true

      if (isSpecial) {
        return filename.includes('specials') || filename.includes('season00') || filename.includes('season-00')
      }
      return (
        filename.startsWith(`season${sPadded}`) ||
        filename.startsWith(`season${sRaw}`) ||
        filename.startsWith(`season-${sPadded}`) ||
        filename.startsWith(`season-${sRaw}`)
      )
    })
  }

  const episode = selectedUnitEpisode.value
  if (!episode) return []
  const episodeFilename = episode.relativePath.split('/').pop() || ''
  const episodeBase = episodeFilename.replace(/\.[^/.]+$/, '').toLowerCase()

  return assets.filter(asset => {
    const filename = (asset.relativePath.split('/').pop() || '').toLowerCase()
    return filename.startsWith(episodeBase)
  })
})

function findTVArtworkUrl(keywords: string[]): string | undefined {
  const assets = detail.value?.artwork ?? []
  const match = assets.find(asset => {
    const filename = asset.relativePath.split('/').pop()?.toLowerCase() || ''
    const nameWithoutExt = filename.replace(/\.[^/.]+$/, '')
    return keywords.some(k => nameWithoutExt === k.toLowerCase() || nameWithoutExt.includes(k.toLowerCase()))
  })
  if (match && detail.value) {
    return api.tvArtworkUrl(detail.value.show.id, match.id)
  }
  return undefined
}

const resolvedPosterUrl = computed(() => findTVArtworkUrl(['poster', 'cover', 'folder']) || draft.posterUrl || '')
const localBackdropUrl = computed(() => findTVArtworkUrl(['fanart', 'backdrop', 'background', 'keyart']) || '')
const resolvedBackdropUrl = computed(() => localBackdropUrl.value || draft.backdropUrl || '')
const resolvedLogoUrl = computed(() => findTVArtworkUrl(['clearlogo', 'logo', 'clearart']) || '')
const resolvedBannerUrl = computed(() => findTVArtworkUrl(['banner']) || '')

const seasonPosterUrl = computed(() => {
  const sNum = props.selection?.kind === 'episode' ? selectedUnitEpisode.value?.seasonNumber : props.selection?.kind === 'season' ? props.selection.seasonNumber : null
  if (sNum !== null && sNum !== undefined) {
    const sPadded = String(sNum).padStart(2, '0')
    const sRaw = String(sNum)
    const isSpecial = sNum === 0
    const seasonPosterAsset = (detail.value?.artwork ?? []).find(a => {
      const fn = (a.relativePath.split('/').pop() || '').toLowerCase()
      if (isSpecial) return (fn.includes('specials') || fn.includes('season00')) && fn.includes('poster')
      return (
        (fn.startsWith(`season${sPadded}`) || fn.startsWith(`season${sRaw}`) || fn.startsWith(`season-${sPadded}`) || fn.startsWith(`season-${sRaw}`)) &&
        (fn.includes('poster') || fn.includes('cover') || fn.endsWith('.jpg') || fn.endsWith('.png'))
      )
    })
    if (seasonPosterAsset && detail.value) {
      return api.tvArtworkUrl(detail.value.show.id, seasonPosterAsset.id)
    }
  }
  return resolvedPosterUrl.value
})

const currentContextPosterUrl = computed(() => {
  if (props.selection?.kind === 'episode') {
    const episode = selectedUnitEpisode.value
    if (episode) {
      const epBase = (episode.relativePath.split('/').pop() || '').replace(/\.[^/.]+$/, '').toLowerCase()
      const thumbAsset = (detail.value?.artwork ?? []).find(a => {
        const fn = (a.relativePath.split('/').pop() || '').toLowerCase()
        return fn.startsWith(epBase) && (fn.includes('thumb') || fn.includes('poster'))
      })
      if (thumbAsset && detail.value) {
        return api.tvArtworkUrl(detail.value.show.id, thumbAsset.id)
      }
      const remoteStill = remoteEpisode(episode)?.stillUrl
      if (remoteStill) return remoteStill
    }
    return seasonPosterUrl.value || resolvedPosterUrl.value
  }

  if (props.selection?.kind === 'season') {
    return seasonPosterUrl.value || resolvedPosterUrl.value
  }

  return resolvedPosterUrl.value
})

const currentContextBackdropUrl = computed(() => {
  if (props.selection?.kind === 'season') {
    const sNum = props.selection.seasonNumber
    const sPadded = String(sNum).padStart(2, '0')
    const seasonFanartAsset = (detail.value?.artwork ?? []).find(a => {
      const fn = (a.relativePath.split('/').pop() || '').toLowerCase()
      return (fn.startsWith(`season${sPadded}`) || fn.startsWith(`season${sNum}`)) && (fn.includes('fanart') || fn.includes('backdrop'))
    })
    if (seasonFanartAsset && detail.value) {
      return api.tvArtworkUrl(detail.value.show.id, seasonFanartAsset.id)
    }
  }
  return localBackdropUrl.value
})

const currentContextBannerUrl = computed(() => {
  if (props.selection?.kind === 'season') {
    const sNum = props.selection.seasonNumber
    const sPadded = String(sNum).padStart(2, '0')
    const seasonBannerAsset = (detail.value?.artwork ?? []).find(a => {
      const fn = (a.relativePath.split('/').pop() || '').toLowerCase()
      return (fn.startsWith(`season${sPadded}`) || fn.startsWith(`season${sNum}`)) && fn.includes('banner')
    })
    if (seasonBannerAsset && detail.value) {
      return api.tvArtworkUrl(detail.value.show.id, seasonBannerAsset.id)
    }
  }
  return resolvedBannerUrl.value
})

const nfoRaw = shallowRef({ exists: false, targetPath: '', content: '' })

const draft = reactive({
  title: '',
  originalTitle: '',
  year: null as number | null,
  overview: '',
  genres: [] as string[],
  posterUrl: '',
  backdropUrl: '',
  network: '',
  status: '',
  rating: null as number | null,
  votes: null as number | null,
  cast: [] as api.CastMember[],
  episodes: [] as {
    seasonNumber: number
    episodeNumber: number
    title: string
    overview: string
    airDate: string
    runtimeMinutes?: number | null
    stillUrl: string
  }[],
})

const castList = computed<DisplayCastMember[]>(() =>
  draft.cast.map((person, index) => ({
    id: `cast-${index}-${person.name}`,
    name: person.name,
    role: person.role,
    avatar: person.profileUrl,
  }))
)

const seasonsMap = computed(() => {
  if (!detail.value) return new Map<number, api.TVEpisode[]>()
  const map = new Map<number, api.TVEpisode[]>()
  for (const ep of detail.value.episodes) {
    const list = map.get(ep.seasonNumber) ?? []
    list.push(ep)
    map.set(ep.seasonNumber, list)
  }
  return map
})

const metadataByEpisode = computed(() => {
  const entries = detail.value?.metadata.episodes ?? []
  return new Map(entries.map(episode => [`${episode.seasonNumber}:${episode.episodeNumber}`, episode]))
})

function remoteEpisode(ep: api.TVEpisode) {
  return metadataByEpisode.value.get(`${ep.seasonNumber}:${ep.episodeStart}`)
}

function episodeTitle(ep: api.TVEpisode): string {
  return remoteEpisode(ep)?.title || ep.titleHint || ep.relativePath.split('/').pop() || ''
}

function formatEpisodeCode(ep: api.TVEpisode): string {
  const s = String(ep.seasonNumber).padStart(2, '0')
  const e = String(ep.episodeStart).padStart(2, '0')
  return `S${s}E${e}`
}

const nfoXmlContent = computed(() => {
  const genresXml = draft.genres.map(g => `    <genre>${g}</genre>`).join('\n')
  return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<tvshow>
    <title>${draft.title}</title>
    <originaltitle>${draft.originalTitle}</originaltitle>
    <year>${draft.year ?? ''}</year>
    <rating>${draft.rating ?? ''}</rating>
    <votes>${draft.votes ?? ''}</votes>
    <plot>${draft.overview}</plot>
    <status>${draft.status}</status>
    <studio>${draft.network}</studio>
    <uniqueid type="tmdb" default="true">${detail.value?.metadata.providerId || ''}</uniqueid>
${genresXml}
</tvshow>`
})

async function loadNfoRaw(selection = props.selection) {
  const requestSequence = ++nfoRequestSequence
  if (!selection) {
    nfoRaw.value = { exists: false, targetPath: '', content: '' }
    return
  }
  try {
    const result = await api.tvNfoRaw(selection.showId, selection)
    if (requestSequence === nfoRequestSequence) nfoRaw.value = result
  } catch {
    if (requestSequence === nfoRequestSequence) {
      nfoRaw.value = { exists: false, targetPath: '', content: '' }
    }
  }
}

function applyMetadataToDraft(meta: Partial<api.TVMetadata>, fallbackTitle = '', fallbackYear: number | null = null) {
  draft.title = meta.title || fallbackTitle
  draft.originalTitle = meta.originalTitle || fallbackTitle
  draft.year = meta.year ?? fallbackYear
  draft.overview = meta.overview || ''
  draft.genres = [...(meta.genres || [])]
  draft.posterUrl = meta.posterUrl || ''
  draft.backdropUrl = meta.backdropUrl || ''
  draft.network = meta.network || ''
  draft.status = meta.status || ''
  draft.rating = meta.rating ?? null
  draft.votes = meta.votes ?? null
  draft.cast = [...(meta.cast || [])]
  draft.episodes = (meta.episodes || []).map(ep => ({
    seasonNumber: ep.seasonNumber,
    episodeNumber: ep.episodeNumber,
    title: ep.title,
    overview: ep.overview,
    airDate: ep.airDate,
    runtimeMinutes: ep.runtimeMinutes,
    stillUrl: ep.stillUrl,
  }))
}

async function loadDetail(id: number) {
  const requestSequence = ++detailRequestSequence
  isLoading.value = true
  error.value = null
  try {
    const result = await api.tvShowDetail(id)
    if (requestSequence !== detailRequestSequence || props.selection?.showId !== id) return
    detail.value = result
    applyMetadataToDraft(result.metadata, result.show.titleHint, result.show.yearHint)
    if (result.episodes.length) {
      selectedEpisodeId.value = result.episodes[0].id
    }
  } catch (caught) {
    if (requestSequence !== detailRequestSequence) return
    error.value = caught instanceof Error ? caught.message : props.labels.errorLoadShow
  } finally {
    if (requestSequence === detailRequestSequence) isLoading.value = false
  }
}

watch(showId, id => {
  isEditing.value = false
  if (id) {
    if (props.activeTab === undefined) localActiveTab.value = 'overview'
    loadDetail(id)
  } else {
    detailRequestSequence += 1
    isLoading.value = false
    detail.value = null
  }
}, { immediate: true })

watch(() => props.selection, selection => {
  if (selection?.kind === 'episode') selectedEpisodeId.value = selection.episodeId
  void loadNfoRaw()
}, { immediate: true, deep: true })

function selectTab(tab: InspectorTab) {
  localActiveTab.value = tab
  emit('selectTab', tab)
}

function cancelEditing() {
  if (detail.value) {
    applyMetadataToDraft(detail.value.metadata, detail.value.show.titleHint, detail.value.show.yearHint)
  }
  isEditing.value = false
}

async function handleCandidateSelect(candidate: api.Candidate) {
  if (!showId.value) return
  try {
    const newMeta = await api.selectTVShowCandidate(props.csrfToken, showId.value, candidate.id)
    applyMetadataToDraft(newMeta)
    await loadDetail(showId.value)
    await loadNfoRaw()
    isEditing.value = false
    showScraperModal.value = false
    emit('metadataSaved')
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : props.labels.errorApplyCandidate
    showScraperModal.value = false
  }
}

async function handleScrape() {
  if (!props.selection || !showId.value) return
  if (props.selection.kind === 'show') {
    showScraperModal.value = true
    return
  }
  isSaving.value = true
  error.value = null
  try {
    if (props.selection.kind === 'season') {
      await api.scrapeTVSeason(props.csrfToken, showId.value, props.selection.seasonNumber)
    } else {
      await api.scrapeTVEpisode(props.csrfToken, showId.value, props.selection.seasonNumber, props.selection.episodeId)
    }
    await loadDetail(showId.value)
    await loadNfoRaw()
    emit('metadataSaved')
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : props.labels.errorApplyCandidate
  } finally {
    isSaving.value = false
  }
}

async function handleSaveAndWrite() {
  if (!showId.value || !detail.value) return
  isSaving.value = true
  error.value = null
  try {
    await api.previewTVNfoPlans(props.csrfToken, showId.value)
    await loadDetail(showId.value)
    isEditing.value = false
    emit('metadataSaved')
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : props.labels.errorSaveMetadata
  } finally {
    isSaving.value = false
  }
}
</script>

<template>
  <main class="inspector-workspace" :aria-label="labels.tvShowDetails">
    <InspectorToolbar
      :active-tab="activeTab"
      :has-detail="detail !== null"
      :is-writable="detail?.writable ?? false"
      :is-editing="isEditing"
      :is-saving="isSaving"
      :is-scraping="false"
      :is-locked="isLocked"
      :labels="labels"
      @select-tab="selectTab"
      @toggle-edit="isEditing = !isEditing"
      @cancel-edit="cancelEditing"
      @save-edit="handleSaveAndWrite"
      @scrape="handleScrape"
      @toggle-lock="isLocked = !isLocked"
      @close="emit('close')"
    />

    <div v-if="error" class="inspector-error">
      {{ error }}
    </div>

    <div v-if="detail && !detail.writable" class="read-only-banner">
      {{ labels.readOnly }}
    </div>

    <div v-if="!showId" class="no-selection-state">
      <svg class="no-sel-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <rect x="2" y="4" width="20" height="16" rx="2" />
        <path d="M8 4v16M16 4v16M2 12h20" />
      </svg>
      <p>{{ labels.selectShow }}</p>
    </div>

    <div v-else-if="isLoading" class="loading-state">
      <div class="loading-spinner"></div>
      <p>{{ labels.loading }}</p>
    </div>

    <div v-else-if="error && showId" class="error-state">
      <svg class="error-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="12" cy="12" r="9" />
        <path d="M12 8v4M12 16h.01" />
      </svg>
      <p>{{ labels.errorLoadShow || 'Unable to load TV show details.' }}</p>
    </div>

    <div v-else-if="detail" class="inspector-content">
      <!-- Ambient Backdrop Background Layer in parent container -->
      <div v-if="activeTab === 'overview' && currentContextBackdropUrl" class="inspector-backdrop-bg" aria-hidden="true">
        <img :src="currentContextBackdropUrl" alt="" class="backdrop-img" />
        <div class="backdrop-gradient"></div>
      </div>

      <TVOverviewTab
        v-if="activeTab === 'overview'"
        :selection="selection"
        :detail="detail"
        :draft="draft"
        :is-editing="isEditing"
        :resolved-poster-url="resolvedPosterUrl"
        :selected-unit-title="selectedUnitTitle"
        :selected-unit-episode="selectedUnitEpisode"
        :selected-unit-season-episodes="selectedUnitSeasonEpisodes"
        :selected-unit-overview="selectedUnitOverview"
        :selected-unit-path="selectedUnitPath"
        :selected-unit-size="selectedUnitSize"
        :seasons-map="seasonsMap"
        :selected-episode-id="selectedEpisodeId"
        :inspection="inspection"
        :inspection-loading="isInspectionLoading"
        :inspection-error="inspectionError"
        :labels="labels"
        @update:selected-episode-id="selectedEpisodeId = $event"
      />

      <TVArtworkTab
        v-else-if="activeTab === 'artwork'"
        :show-id="detail.show.id"
        :selection="selection"
        :poster-url="currentContextPosterUrl"
        :backdrop-url="currentContextBackdropUrl"
        :logo-url="resolvedLogoUrl"
        :banner-url="currentContextBannerUrl"
        :season-poster-url="seasonPosterUrl"
        :scoped-artwork="scopedArtwork"
        :labels="labels"
        :episode-title-text="selectedUnitEpisode ? `${formatEpisodeCode(selectedUnitEpisode)} - ${episodeTitle(selectedUnitEpisode)}` : ''"
      />

      <MovieCastTab
        v-else-if="activeTab === 'cast'"
        :cast="castList"
        :labels="labels"
      />

      <MovieNfoTab
        v-else-if="activeTab === 'nfo'"
        :content="nfoRaw.content || nfoXmlContent"
        :labels="labels"
        :filename="selection?.kind === 'episode' ? 'episode.nfo' : selection?.kind === 'season' ? 'season.nfo' : 'tvshow.nfo'"
        :header-label="labels.nfoXmlEditor || 'Kodi TV NFO Raw XML'"
        :action-label="labels.saveToNfo || 'Save to tvshow.nfo'"
        :read-only="!detail.writable"
        @save-to-nfo="handleSaveAndWrite"
      />

      <TVFileAuditTab
        v-else-if="activeTab === 'files'"
        :show-id="showId"
        :episodes="scopedEpisodes"
        :selection="selection"
        :inspection="inspection"
        :inspection-loading="isInspectionLoading"
        :inspection-error="inspectionError"
        :labels="labels"
        :csrf-token="csrfToken"
      />
    </div>

    <ScraperModal
      v-if="showScraperModal && showId && detail"
      :item-id="showId"
      :item-title="draft.title || detail.show.titleHint"
      :item-year="draft.year ?? detail.show.yearHint"
      :labels="labels"
      media-type="tv"
      @close="showScraperModal = false"
      @select="handleCandidateSelect"
    />
  </main>
</template>

<style scoped>
.inspector-workspace {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--surface-container-lowest, #070d1f);
  overflow-x: hidden;
  min-width: 0;
  max-width: 100%;
  box-sizing: border-box;
  position: relative;
}

.inspector-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 16px 20px;
  position: relative;
  min-width: 0;
  max-width: 100%;
  box-sizing: border-box;
}

.inspector-error {
  padding: 10px 16px;
  background: rgba(255, 77, 79, 0.15);
  border-bottom: 1px solid rgba(255, 77, 79, 0.3);
  color: #ff4d4f;
  font-size: 13px;
}

.read-only-banner {
  padding: 8px 16px;
  background: rgba(250, 173, 20, 0.15);
  border-bottom: 1px solid rgba(250, 173, 20, 0.3);
  color: #faad14;
  font-size: 12px;
}

.no-selection-state,
.loading-state,
.error-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  color: var(--outline, #908fa0);
}

.error-state {
  color: #ff8f8f;
  text-align: center;
  padding: 24px;
}

.error-icon {
  width: 42px;
  height: 42px;
  opacity: 0.8;
}

.no-sel-icon {
  width: 48px;
  height: 48px;
  opacity: 0.3;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--outline-variant, #2e3447);
  border-top-color: var(--primary, #c0c1ff);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Ambient Backdrop Background Layer in parent container */
.inspector-backdrop-bg {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  width: 100%;
  pointer-events: none;
  z-index: 0;
  -webkit-mask-image: linear-gradient(
    to bottom,
    rgba(0, 0, 0, 1) 0%,
    rgba(0, 0, 0, 0.9) 60%,
    rgba(0, 0, 0, 0.2) 85%,
    rgba(0, 0, 0, 0) 100%
  );
  mask-image: linear-gradient(
    to bottom,
    rgba(0, 0, 0, 1) 0%,
    rgba(0, 0, 0, 0.9) 60%,
    rgba(0, 0, 0, 0.2) 85%,
    rgba(0, 0, 0, 0) 100%
  );
}

.backdrop-img {
  display: block;
  width: 100%;
  height: auto;
  opacity: 0.35;
  filter: blur(0.5px);
}

.backdrop-gradient {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    to bottom,
    rgba(7, 13, 31, 0.05) 0%,
    rgba(7, 13, 31, 0.35) 50%,
    rgba(7, 13, 31, 0.85) 85%,
    var(--surface-container-lowest, #070d1f) 100%
  );
}
</style>
