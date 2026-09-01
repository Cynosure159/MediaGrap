<script setup lang="ts">
import { computed, reactive, shallowRef, watch } from 'vue'
import * as api from '@/api/library'
import type { InspectorTab } from './inspector/InspectorToolbar.vue'
import type { CastMember as DisplayCastMember } from './inspector/MovieCastTab.vue'
import InspectorToolbar from './inspector/InspectorToolbar.vue'
import ScraperModal from './ScraperModal.vue'
import TVArtworkPanel from './TVArtworkPanel.vue'

const props = defineProps<{
  selection: api.TVSelection | null
  csrfToken: string
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  close: []
  metadataSaved: []
}>()

const detail = shallowRef<api.TVShowDetail | null>(null)
const activeTab = shallowRef<InspectorTab>('overview')
const showScraperModal = shallowRef(false)
const selectedEpisodeId = shallowRef<number | null>(null)
const isEditing = shallowRef(false)
const isSaving = shallowRef(false)
const isLoading = shallowRef(false)
const error = shallowRef<string | null>(null)
const isLocked = shallowRef(false)
const xmlCopied = shallowRef(false)
const showId = computed(() => props.selection?.showId ?? null)
const selectedUnitEpisode = computed(() => {
  const selection = props.selection
  if (selection?.kind !== 'episode') return null
  return detail.value?.episodes.find(item => item.id === selection.episodeId) ?? null
})
const selectedUnitSeasonEpisodes = computed(() => props.selection?.kind === 'season' ? (seasonsMap.value.get(props.selection.seasonNumber) ?? []) : [])
const selectedUnitTitle = computed(() => {
  if (props.selection?.kind === 'season') return `${props.labels.season} ${props.selection.seasonNumber}`
  if (selectedUnitEpisode.value) return `${formatEpisodeCode(selectedUnitEpisode.value)} · ${episodeTitle(selectedUnitEpisode.value)}`
  return draft.title || detail.value?.show.titleHint || ''
})
const selectedUnitOverview = computed(() => selectedUnitEpisode.value ? (remoteEpisode(selectedUnitEpisode.value)?.overview || props.labels.noOverview || '') : draft.overview)
const selectedUnitPath = computed(() => selectedUnitEpisode.value?.relativePath || detail.value?.show.relativePath || '')
const selectedUnitSize = computed(() => selectedUnitEpisode.value?.fileSize ?? (props.selection?.kind === 'season' ? selectedUnitSeasonEpisodes.value.reduce((total, item) => total + item.fileSize, 0) : totalFileSize.value))
const scopedEpisodes = computed(() => {
  if (props.selection?.kind === 'season') return selectedUnitSeasonEpisodes.value
  if (selectedUnitEpisode.value) return [selectedUnitEpisode.value]
  return detail.value?.episodes ?? []
})
const scopedArtwork = computed(() => {
  const assets = detail.value?.artwork ?? []
  if (props.selection?.kind === 'show') return assets
  if (props.selection?.kind === 'season') {
    const directories = new Set(scopedEpisodes.value.map(item => item.relativePath.slice(0, item.relativePath.lastIndexOf('/'))))
    return assets.filter(asset => directories.has(asset.relativePath.slice(0, asset.relativePath.lastIndexOf('/'))))
  }
  const episode = selectedUnitEpisode.value
  if (!episode) return []
  const basePath = episode.relativePath.slice(0, episode.relativePath.lastIndexOf('.'))
  return assets.filter(asset => asset.relativePath.startsWith(`${basePath}.`))
})
const scopedCastList = computed<DisplayCastMember[]>(() => props.selection?.kind === 'show' ? castList.value : [])
const nfoRaw = shallowRef({ exists: false, targetPath: '', content: '' })

const patternInput = shallowRef('${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumber}E${episodeNumber} - [${resolution}]')
const availableTokens = [
  '${showTitle}',
  '${originalTitle}',
  '${seasonNumber}',
  '${episodeNumber}',
  '${year}',
  '${resolution}',
  '${videoCodec}',
  '${audioCodec}',
]

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
})

const genresInput = computed({
  get: () => draft.genres.join(', '),
  set: (val: string) => {
    draft.genres = val.split(',').map(s => s.trim()).filter(Boolean)
  },
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

function nfoState(ep: api.TVEpisode): string {
  return ep.sidecars.some(sidecar => sidecar.kind === 'nfo') ? props.labels.nfoReady : props.labels.nfoMissing
}

const totalFileSize = computed(() => {
  if (!detail.value?.episodes) return 0
  return detail.value.episodes.reduce((acc, ep) => acc + (ep.fileSize || 0), 0)
})

function formatFileSize(bytes: number): string {
  if (!bytes) return '—'
  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return `${gb.toFixed(1)} GB`
  return `${(bytes / (1024 * 1024)).toFixed(0)} MB`
}

const techSpecs = computed(() => {
  const specs: string[] = []
  const episodes = detail.value?.episodes || []
  const samplePath = (episodes[0]?.relativePath || detail.value?.show.relativePath || '').toUpperCase()

  if (samplePath.includes('2160P') || samplePath.includes('4K') || samplePath.includes('UHD')) specs.push('4K UHD')
  else if (samplePath.includes('1080P') || samplePath.includes('FHD')) specs.push('1080p FHD')
  else if (samplePath.includes('720P')) specs.push('720p HD')
  else specs.push('1080p FHD')

  if (samplePath.includes('HEVC') || samplePath.includes('X265') || samplePath.includes('H.265') || samplePath.includes('H265')) specs.push('HEVC 10-bit')
  else specs.push('AVC 8-bit')

  if (samplePath.includes('ATMOS')) specs.push('Dolby Atmos 7.1')
  else if (samplePath.includes('DDP5.1') || samplePath.includes('DD+5.1') || samplePath.includes('EAC3')) specs.push('E-AC3 5.1')
  else specs.push('AAC 2.0')

  if (samplePath.includes('HDR10+') || samplePath.includes('HDR10PLUS')) specs.push('HDR10+')
  else if (samplePath.includes('HDR')) specs.push('HDR10')
  else if (samplePath.includes('DV') || samplePath.includes('DOVI')) specs.push('Dolby Vision')

  specs.push('Chs/Eng Sub')
  return specs
})

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

async function copyXml() {
  try {
    await navigator.clipboard.writeText(nfoRaw.value.content || nfoXmlContent.value)
    xmlCopied.value = true
    setTimeout(() => { xmlCopied.value = false }, 2000)
  } catch {
    // fallback
  }
}

async function loadNfoRaw(selection = props.selection) {
  if (!selection) {
    nfoRaw.value = { exists: false, targetPath: '', content: '' }
    return
  }
  try {
    nfoRaw.value = await api.tvNfoRaw(selection.showId, selection)
  } catch {
    nfoRaw.value = { exists: false, targetPath: '', content: '' }
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
}

async function loadDetail(id: number) {
  isLoading.value = true
  error.value = null
  try {
    const result = await api.tvShowDetail(id)
    detail.value = result
    applyMetadataToDraft(result.metadata, result.show.titleHint, result.show.yearHint)
    if (result.episodes.length) {
      selectedEpisodeId.value = result.episodes[0].id
    }
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : props.labels.errorLoadShow
  } finally {
    isLoading.value = false
  }
}

watch(showId, id => {
  isEditing.value = false
  if (id) {
    activeTab.value = 'overview'
    loadDetail(id)
  } else {
    detail.value = null
  }
}, { immediate: true })

watch(() => props.selection, () => { void loadNfoRaw() }, { immediate: true })

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
      @select-tab="activeTab = $event"
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

    <div v-else-if="detail" class="inspector-content">
      <section v-if="activeTab === 'overview'" class="overview-view">
        <div class="hero-banner">
          <div class="title-cluster">
            <template v-if="!isEditing">
              <h1 class="main-title">{{ selectedUnitTitle }}</h1>
              <h2 v-if="draft.originalTitle && draft.originalTitle !== draft.title" class="sub-title">
                {{ draft.originalTitle }}
              </h2>
            </template>
            <template v-else>
              <input v-model="draft.title" type="text" class="edit-title-input" :placeholder="labels.titlePlaceholder || '电视剧名称'" />
              <input v-model="draft.originalTitle" type="text" class="edit-subtitle-input" :placeholder="labels.origTitlePlaceholder || '原始片名 / 英文名'" />
            </template>
          </div>

          <div class="meta-strip">
            <span class="meta-item font-code">{{ selectedUnitEpisode ? formatEpisodeCode(selectedUnitEpisode) : (draft.year ?? detail.show.yearHint ?? labels.tvSeries) }}</span>
            <span class="meta-dot"></span>
            <span class="spec-pill font-code">{{ detail.show.seasonCount }} {{ labels.seasons }}</span>
            <span class="spec-pill font-code">{{ detail.show.episodeCount }} {{ labels.episodes }}</span>
            
            <template v-if="draft.status">
              <span class="meta-dot"></span>
              <span class="spec-pill font-code">{{ draft.status }}</span>
            </template>

            <template v-if="draft.network">
              <span class="meta-dot"></span>
              <span class="meta-studio" :title="draft.network">{{ draft.network }}</span>
            </template>

            <div v-if="draft.rating !== null" class="rating-badge">
              <span class="star-score">★ {{ typeof draft.rating === 'number' ? draft.rating.toFixed(1) : draft.rating }}</span>
              <span class="score-denom">/10</span>
              <span v-if="draft.votes !== null" class="vote-count">({{ draft.votes }})</span>
              <span class="provider-tag">TMDb</span>
            </div>
          </div>

          <div class="spec-pills-row">
            <span v-for="spec in techSpecs" :key="spec" class="spec-pill">
              {{ spec }}
            </span>
          </div>
        </div>

        <div class="overview-body-layout">
          <div class="poster-container">
            <div class="compact-poster-card group">
              <img
                v-if="draft.posterUrl"
                :src="draft.posterUrl"
                :alt="draft.title || 'Poster'"
                class="poster-img"
              />
              <div v-else class="poster-empty">
                <svg viewBox="0 0 24 24" fill="currentColor" width="32" height="32" opacity="0.3">
                  <path d="M21 3H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h5v2h8v-2h5c1.1 0 1.99-.9 1.99-2L23 5c0-1.1-.9-2-2-2zm0 14H3V5h18v12z"/>
                </svg>
                <span>{{ labels.noPosterLoaded || '未加载海报' }}</span>
              </div>
              <div v-if="draft.posterUrl" class="poster-res-badge font-code">
                1000×1500
              </div>
            </div>
          </div>

          <div class="details-container">
            <div class="meta-blocks-grid">
              <div class="meta-card">
                <span class="card-label-caps">{{ labels.releaseDate || '首播年份' }}</span>
                <div v-if="!isEditing" class="meta-val font-code">
                  {{ draft.year || '—' }}
                </div>
                <div v-else class="meta-input-wrap">
                  <input v-model="draft.year" type="number" class="card-input font-code" :placeholder="labels.yearPlaceholder || 'YYYY'" />
                </div>
              </div>

              <div class="meta-card">
                <span class="card-label-caps">{{ labels.genres || '类型' }}</span>
                <div v-if="!isEditing" class="genres-pills-list">
                  <span v-for="g in draft.genres" :key="g" class="genre-tag-pill">
                    {{ g }}
                  </span>
                  <span v-if="draft.genres.length === 0" class="genre-empty-hint">
                    {{ labels.noGenres || '暂无分类' }}
                  </span>
                </div>
                <input
                  v-else
                  v-model="genresInput"
                  type="text"
                  class="card-input"
                  :placeholder="labels.genresPlaceholder || '剧情, 动作, 奇幻'"
                />
              </div>

              <div class="meta-card">
                <span class="card-label-caps">NETWORK / 平台</span>
                <div v-if="!isEditing" class="meta-val meta-val-highlight">
                  {{ draft.network || '—' }}
                </div>
                <input v-else v-model="draft.network" type="text" class="card-input" placeholder="HBO, Netflix..." />
              </div>

              <div class="meta-card">
                <span class="card-label-caps">STATUS / 连载状态</span>
                <div v-if="!isEditing" class="meta-val">
                  {{ draft.status || '—' }}
                </div>
                <input v-else v-model="draft.status" type="text" class="card-input" placeholder="Ended, Returning Series..." />
              </div>

              <div class="meta-card meta-card-full">
                <span class="card-label-caps">STORAGE PATH / 目录路径</span>
                <div class="meta-val font-code" :title="selectedUnitPath">
                  {{ selectedUnitPath }}
                </div>
              </div>
            </div>

            <div class="plot-card">
              <div class="plot-box">
                <div class="plot-header">
                  <span class="card-label-caps">{{ labels.plotSummary || '剧情简介' }}</span>
                </div>
                <p v-if="!isEditing" class="plot-paragraph">
                  {{ selectedUnitOverview || labels.noOverview || '暂无剧情简介。' }}
                </p>
                <textarea
                  v-else
                  v-model="draft.overview"
                  class="plot-textarea"
                  rows="4"
                  :placeholder="labels.enterPlot || '输入电视剧剧情简介...'"
                ></textarea>
              </div>
            </div>

            <div class="file-info-card">
              <div class="file-icon-box">
                <svg viewBox="0 0 24 24" fill="currentColor" width="18" height="18">
                  <path d="M14 2H6c-1.1 0-1.99.9-1.99 2L4 20c0 1.1.89 2 1.99 2H18c1.1 0 2-.9 2-2V8l-6-6zm2 16H8v-2h8v2zm0-4H8v-2h8v2zm-3-5V3.5L18.5 9H13z"/>
                </svg>
              </div>
              <div class="file-details">
                <p class="file-path font-code" :title="selectedUnitPath">{{ selectedUnitPath }}</p>
                <div class="file-specs">
                  <span class="dot dot-ok"></span>
                  <span class="status-txt">{{ selectedUnitEpisode ? formatEpisodeCode(selectedUnitEpisode) : (props.selection?.kind === 'season' ? `${selectedUnitSeasonEpisodes.length} ${labels.episodes}` : `${detail.show.seasonCount} ${labels.seasons} · ${detail.show.episodeCount} ${labels.episodes}`) }}</span>
                  <span class="meta-sep">|</span>
                  <span class="stream-summary font-code">Video: 1080p AVC • Audio: AAC 2.0</span>
                </div>
              </div>
              <div class="file-size-divider"></div>
              <div class="file-size-box">
                <span class="size-val font-code">{{ formatFileSize(selectedUnitSize) }}</span>
                <span class="size-lbl font-code">{{ labels.fileSize || 'TOTAL SIZE' }}</span>
              </div>
            </div>
          </div>
        </div>

        <div v-if="props.selection?.kind !== 'episode'" class="seasons-container">
          <div class="seasons-header-title">
            <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="title-icon">
              <path d="M4 6H2v14c0 1.1.9 2 2 2h14v-2H4V6zm16-4H8c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H8V4h12v12z"/>
            </svg>
            <h2>{{ labels.episodesAndSeasons || 'Seasons & Episodes' }}</h2>
          </div>

          <section
            v-for="[seasonNum, episodes] in (props.selection?.kind === 'season' ? [[props.selection.seasonNumber, selectedUnitSeasonEpisodes]] : seasonsMap)"
            :key="seasonNum"
            class="season-card"
          >
            <header class="season-header">
              <div class="season-title-box">
                <span class="season-badge">{{ labels.season }} {{ seasonNum }}</span>
                <span class="season-count">{{ episodes.length }} {{ labels.episodesCount }}</span>
              </div>
              <span class="spec-pill font-code">{{ detail.metadata.provider ? labels.metadataReady : labels.metadataEmpty }}</span>
            </header>

            <table class="episode-table">
              <thead>
                <tr>
                  <th class="col-code">{{ labels.code }}</th>
                  <th class="col-title">{{ labels.titleAndPath }}</th>
                  <th class="col-res">{{ labels.quality }}</th>
                  <th class="col-status">{{ labels.nfo }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="ep in episodes"
                  :key="ep.id"
                  class="ep-row"
                  :class="{ 'ep-row--active': selectedEpisodeId === ep.id }"
                  @click="selectedEpisodeId = ep.id"
                >
                  <td class="col-code font-code">{{ formatEpisodeCode(ep) }}</td>
                  <td class="col-title">
                    <div class="ep-title-main">{{ episodeTitle(ep) }}</div>
                    <div class="ep-path font-code">{{ ep.relativePath }}</div>
                  </td>
                  <td class="col-res">
                    <span class="spec-badge font-code">{{ remoteEpisode(ep)?.runtimeMinutes ? `${remoteEpisode(ep)?.runtimeMinutes} ${labels.runtimeMinutes || 'min'}` : (labels.qualityUnavailable || '1080p') }}</span>
                  </td>
                  <td class="col-status">
                    <span class="dot" :class="ep.sidecars.some(sidecar => sidecar.kind === 'nfo') ? 'dot-ok' : 'dot-off'" :title="nfoState(ep)"></span>
                  </td>
                </tr>
              </tbody>
            </table>
          </section>
        </div>
      </section>

      <section v-else-if="activeTab === 'artwork'" class="artwork-workshop-view">
        <div class="workshop-header">
          <div class="header-left">
            <h2>{{ labels.artworkGallery || 'Artwork & Fanart Gallery' }}</h2>
            <span class="sub-label">Manage TV series posters, backdrops, season art and episode stills</span>
          </div>
        </div>

        <div class="artwork-layout-grid">
          <div v-if="props.selection?.kind === 'show'" class="art-col-primary">
            <div v-if="props.selection?.kind === 'show'" class="art-card">
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
                <img v-if="draft.posterUrl" :src="draft.posterUrl" :alt="labels.poster || 'Poster'" class="preview-img" />
                <div v-else class="preview-empty">
                  <svg viewBox="0 0 24 24" fill="currentColor" width="40" height="40" opacity="0.3">
                    <path d="M21 3H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h5v2h8v-2h5c1.1 0 1.99-.9 1.99-2L23 5c0-1.1-.9-2-2-2zm0 14H3V5h18v12z"/>
                  </svg>
                  <span>{{ labels.noPosterLoaded || 'No Poster' }}</span>
                </div>
                <div v-if="draft.posterUrl" class="art-status-overlay">
                  <span class="art-dim-badge font-code">1000x1500</span>
                  <span class="art-active-badge font-code">ACTIVE</span>
                </div>
              </div>
            </div>

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
                <img src="/assets/logo-icon.png" :alt="labels.logo || 'Logo'" class="logo-img" />
                <div class="art-status-overlay">
                  <span class="art-dim-badge font-code">512x512</span>
                </div>
              </div>
            </div>
          </div>

          <div class="art-col-wide">
            <div v-if="props.selection?.kind === 'show'" class="art-card">
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
                <img v-if="draft.backdropUrl" :src="draft.backdropUrl" :alt="labels.fanart || 'Fanart'" class="preview-img" />
                <div v-else class="preview-empty">
                  <svg viewBox="0 0 24 24" fill="currentColor" width="40" height="40" opacity="0.3">
                    <path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/>
                  </svg>
                  <span>{{ labels.fanartUnavailable || 'FANART MISSING' }}</span>
                </div>
                <div v-if="draft.backdropUrl" class="art-status-overlay">
                  <span class="art-dim-badge font-code">1920x1080</span>
                  <span class="art-active-badge font-code">ACTIVE</span>
                </div>
              </div>
            </div>

            <div class="art-card">
              <div class="art-card-header">
                <div class="header-title">
                  <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="title-icon">
                    <path d="M4 6H2v14c0 1.1.9 2 2 2h14v-2H4V6zm16-4H8c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H8V4h12v12z"/>
                  </svg>
                  <span>Local Indexed Assets</span>
                </div>
                <span class="item-count font-code">{{ scopedArtwork.length }} files</span>
              </div>
              <TVArtworkPanel :show-id="detail.show.id" :assets="scopedArtwork" :labels="labels" />
            </div>
          </div>
        </div>
      </section>

      <section v-else-if="activeTab === 'cast'" class="cast-workshop-view">
        <div class="workshop-header">
          <div class="header-left">
            <h2>{{ labels.castAndCrew || 'Cast & Crew Workshop' }}</h2>
            <span class="sub-label">{{ props.selection?.kind === 'show' ? 'Series creators, regular cast and guest stars' : 'No cast fields exist in Kodi season and episode NFOs.' }}</span>
          </div>
        </div>

        <div v-if="scopedCastList.length === 0" class="cast-empty">
          <svg viewBox="0 0 24 24" fill="currentColor" width="48" height="48" opacity="0.2">
            <path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/>
          </svg>
          <p>{{ labels.noCastFound || 'No cast and crew information available.' }}</p>
        </div>

        <div v-else class="cast-grid">
          <div v-for="person in scopedCastList" :key="person.id" class="cast-card">
            <div class="cast-avatar">
              <img v-if="person.avatar" :src="person.avatar" :alt="person.name" class="avatar-img" />
              <span v-else class="avatar-placeholder font-code">{{ person.name.charAt(0) }}</span>
            </div>
            <div class="cast-info">
              <strong class="cast-name">{{ person.name }}</strong>
              <span class="cast-role">{{ person.role }}</span>
            </div>
          </div>
        </div>
      </section>

      <section v-else-if="activeTab === 'nfo'" class="nfo-workshop-view">
        <div class="workshop-header">
          <div class="header-left">
            <h2>{{ labels.nfoXmlEditor || 'Kodi NFO Raw XML' }}</h2>
            <span class="sub-label">{{ nfoRaw.exists ? nfoRaw.targetPath : 'No local NFO exists for the selected item.' }}</span>
          </div>
          <div class="workshop-actions">
            <button class="btn btn-outline" type="button" @click="copyXml">
              <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
                <path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
              </svg>
              {{ xmlCopied ? (labels.copied || 'Copied!') : (labels.copy || 'Copy XML') }}
            </button>
            <button class="btn btn-success" :disabled="!detail.writable || isSaving || props.selection?.kind !== 'show'" type="button" @click="handleSaveAndWrite">
              <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
                <path d="M17 3H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V7l-4-4zm-5 16c-1.66 0-3-1.34-3-3s1.34-3 3-3 3 1.34 3 3-1.34 3-3 3zm3-10H5V5h10v4z"/>
              </svg>
              {{ labels.saveToNfo || 'Save to tvshow.nfo' }}
            </button>
          </div>
        </div>

        <div class="xml-editor-canvas">
          <div class="xml-canvas-header">
            <div class="file-tag font-code">{{ nfoRaw.targetPath.split('/').pop() || 'NFO' }}</div>
            <div class="encoding-tag font-code">UTF-8 • XML v1.0 • Kodi v20/v21</div>
          </div>
          <pre class="xml-code font-code"><code>{{ nfoRaw.content || nfoXmlContent }}</code></pre>
        </div>
      </section>

      <section v-else-if="activeTab === 'files'" class="files-workshop-view">
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
            <span class="item-count font-code">{{ scopedEpisodes.length }} episodes</span>
          </div>

          <div class="file-audit-list">
            <div v-for="ep in scopedEpisodes" :key="ep.id" class="audit-item">
              <span class="spec-badge video-badge font-code">{{ formatEpisodeCode(ep) }}</span>
              <span class="audit-path font-code">{{ ep.relativePath }}</span>
              <span class="audit-status text-ok font-code">{{ ep.sidecars.length }} sidecars</span>
            </div>
          </div>
        </div>
      </section>
    </div>

    <ScraperModal
      v-if="showScraperModal && showId && detail"
      :item-id="showId"
      :item-title="draft.title || detail.show.titleHint"
      :item-year="draft.year || detail.show.yearHint"
      :media-type="'tv'"
      :labels="labels"
      @select="handleCandidateSelect"
      @close="showScraperModal = false"
    />
  </main>
</template>
<style scoped>
.inspector-workspace {
  flex: 1;
  height: 100vh;
  background: var(--surface-base, #0c1324);
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.inspector-error {
  padding: 8px 12px;
  background: var(--error-container, #93000a);
  color: var(--on-error-container, #ffdad6);
  font-size: 12px;
}

.read-only-banner {
  padding: 6px 12px;
  background: var(--tertiary-container, #ca8100);
  color: var(--on-tertiary-container, #3e2400);
  font-size: 12px;
  font-weight: 600;
}

.no-selection-state,
.loading-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--outline, #908fa0);
  font-size: 13px;
}

.no-sel-icon {
  width: 48px;
  height: 48px;
  opacity: 0.3;
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--primary, #c0c1ff);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.inspector-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--pane-padding, 12px);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ── 1. Overview Tab & Hero Banner ────────────────────────────────── */
.overview-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}

.hero-banner {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.title-cluster {
  display: flex;
  align-items: baseline;
  gap: 12px;
  flex-wrap: wrap;
}

.main-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  margin: 0;
  line-height: 1.2;
  letter-spacing: -0.02em;
}

.sub-title {
  font-size: 15px;
  font-weight: 400;
  color: var(--on-surface-variant, #c7c4d7);
  margin: 0;
}

.edit-title-input {
  font-size: 18px;
  font-weight: 700;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  padding: 4px 10px;
  width: 280px;
}

.edit-subtitle-input {
  font-size: 13px;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface-variant, #c7c4d7);
  padding: 4px 10px;
  width: 220px;
}

.meta-strip {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  flex-wrap: wrap;
}

.meta-dot {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--outline-variant, #464554);
}

.meta-studio {
  max-width: 260px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rating-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  font-size: 11px;
  margin-left: auto;
}

.star-score {
  color: #f59e0b;
  font-weight: 700;
}

.score-denom,
.vote-count {
  color: var(--outline, #908fa0);
  font-size: 10px;
}

.provider-tag {
  background: #01b4e4;
  color: #ffffff;
  font-size: 9px;
  font-weight: 700;
  padding: 1px 4px;
  border-radius: 2px;
  margin-left: 2px;
  font-family: var(--font-data);
}

.spec-pills-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 2px;
}

/* ── 2. Bento Layout ─────────────────────────────────────────────── */
.overview-body-layout {
  display: flex;
  gap: 20px;
  align-items: flex-start;
  width: 100%;
}

.poster-container {
  width: 180px;
  flex-shrink: 0;
}

.compact-poster-card {
  width: 180px;
  height: 270px;
  aspect-ratio: 2 / 3;
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  overflow: hidden;
  position: relative;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.poster-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.poster-empty {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  color: var(--outline, #908fa0);
  font-size: 11px;
}

.poster-res-badge {
  position: absolute;
  top: 6px;
  right: 6px;
  background: rgba(12, 19, 36, 0.85);
  backdrop-filter: blur(4px);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 1px 5px;
  font-size: 9px;
  color: var(--on-surface, #dce1fb);
}

.details-container {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.meta-blocks-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}

.meta-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 6px 10px;
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-height: 48px;
  justify-content: center;
}

.meta-card-full {
  grid-column: span 2;
}

.meta-val {
  font-size: 12px;
  font-weight: 500;
  color: var(--on-surface, #dce1fb);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.meta-val-highlight {
  color: var(--primary, #c0c1ff);
}

.meta-input-wrap {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
}

.card-input {
  width: 100%;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  font-family: inherit;
  font-size: 12px;
  font-weight: 500;
  padding: 3px 6px;
}

.card-input:focus {
  outline: none;
  border-color: var(--primary, #c0c1ff);
}

.genres-pills-list {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
  padding: 1px 0;
}

.genre-tag-pill {
  background: var(--surface-container-highest, #2e3447);
  border: 1px solid var(--outline-variant, #464554);
  color: var(--on-surface, #dce1fb);
  font-size: 11px;
  padding: 1px 6px;
  border-radius: var(--radius-sm, 0.25rem);
}

.genre-empty-hint {
  font-size: 11px;
  color: var(--outline, #908fa0);
}

.plot-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 10px 12px;
}

.plot-box {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.plot-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.card-label-caps {
  font-size: 9px;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--outline, #908fa0);
  letter-spacing: 0.06em;
}

.plot-paragraph {
  font-size: 12px;
  line-height: 1.65;
  color: var(--on-surface, #dce1fb);
  margin: 0;
  white-space: pre-wrap;
}

.plot-textarea {
  width: 100%;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  font-family: inherit;
  font-size: 12px;
  line-height: 1.6;
  padding: 8px 10px;
  resize: vertical;
  min-height: 80px;
}

.plot-textarea:focus {
  outline: none;
  border-color: var(--primary, #c0c1ff);
}

.file-info-card {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.375rem);
  padding: 8px 12px;
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: auto;
}

.file-icon-box {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-highest, #2e3447);
  color: var(--tertiary, #ffb95f);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.file-details {
  flex: 1;
  min-width: 0;
}

.file-path {
  font-size: 11px;
  color: var(--on-surface, #dce1fb);
  margin: 0 0 2px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-specs {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.status-txt {
  font-size: 9px;
  font-weight: 700;
  color: var(--secondary, #4edea3);
  letter-spacing: 0.04em;
}

.meta-sep {
  color: var(--outline-variant, #2e3447);
}

.stream-summary {
  font-size: 10px;
}

.file-size-divider {
  width: 1px;
  height: 20px;
  background: var(--outline-variant, #2e3447);
}

.file-size-box {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  flex-shrink: 0;
  min-width: 50px;
}

.size-val {
  font-size: 11px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.size-lbl {
  font-size: 8px;
  color: var(--outline, #908fa0);
  letter-spacing: 0.05em;
}

/* ── 3. Seasons & Episodes Table ─────────────────────────────────── */
.seasons-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 8px;
}

.seasons-header-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.seasons-header-title h2 {
  font-size: 15px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  margin: 0;
}

.season-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  overflow: hidden;
}

.season-header {
  padding: 10px 14px;
  background: var(--surface-container-high, #23293c);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.season-title-box {
  display: flex;
  align-items: center;
  gap: 8px;
}

.season-badge {
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.season-count {
  font-size: 11px;
  color: var(--outline, #908fa0);
}

.episode-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.episode-table th {
  text-align: left;
  padding: 8px 12px;
  font-size: 10px;
  font-weight: 700;
  color: var(--outline, #908fa0);
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-container-low, #151b2d);
}

.episode-table td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.ep-row {
  cursor: pointer;
  transition: background 0.12s ease;
}

.ep-row:hover {
  background: var(--surface-container-high, #23293c);
}

.ep-row--active {
  background: rgba(192, 193, 255, 0.06);
}

.col-code {
  width: 80px;
  color: var(--primary, #c0c1ff);
  font-weight: 700;
}

.col-title {
  min-width: 200px;
}

.ep-title-main {
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
}

.ep-path {
  font-size: 10px;
  color: var(--outline, #908fa0);
  margin-top: 2px;
}

.col-res {
  width: 100px;
}

.col-status {
  width: 60px;
  text-align: center;
}

/* ── Workshops (Artwork, Cast, NFO, Files) ────────────────────────── */
.artwork-workshop-view,
.cast-workshop-view,
.nfo-workshop-view,
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
}

.preview-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.logo-img {
  width: 60px;
  height: 60px;
  object-fit: contain;
}

.checkerboard {
  background-image: linear-gradient(45deg, #151b2d 25%, transparent 25%),
    linear-gradient(-45deg, #151b2d 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, #151b2d 75%),
    linear-gradient(-45deg, transparent 75%, #151b2d 75%);
  background-size: 16px 16px;
  background-position: 0 0, 0 8px, 8px -8px, -8px 0px;
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

/* Cast Grid */
.cast-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}

.cast-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  padding: 10px 12px;
  display: flex;
  align-items: center;
  gap: 12px;
  transition: border-color 0.15s ease;
}

.cast-card:hover {
  border-color: var(--outline, #908fa0);
}

.cast-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-placeholder {
  font-size: 16px;
  font-weight: 700;
  color: var(--primary, #c0c1ff);
}

.cast-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.cast-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.cast-role {
  font-size: 11px;
  color: var(--outline, #908fa0);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.cast-empty {
  color: var(--outline, #908fa0);
  font-size: 13px;
  text-align: center;
  padding: 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

/* NFO XML Canvas */
.xml-editor-canvas {
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.4);
}

.xml-canvas-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 14px;
  background: var(--surface-container-low, #151b2d);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.file-tag {
  font-size: 12px;
  font-weight: 700;
  color: var(--primary, #c0c1ff);
}

.encoding-tag {
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.xml-code {
  margin: 0;
  padding: 16px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--on-surface, #dce1fb);
  overflow-x: auto;
  max-height: 520px;
}

/* Files Workshop */
.rename-engine-card,
.structure-card {
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
  max-height: 400px;
  overflow-y: auto;
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

@media (max-width: 900px) {
  .artwork-layout-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .overview-body-layout {
    flex-direction: column;
    gap: 12px;
  }

  .poster-container {
    width: 100%;
    display: flex;
    justify-content: center;
  }

  .compact-poster-card {
    width: 130px;
    height: 195px;
  }
}
</style>
