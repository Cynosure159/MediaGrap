export interface User {
  id: number
  username: string
}

export interface Session {
  user: User
  csrfToken: string
}

export interface Source {
  id: number
  name: string
  rootPath: string
  enabled: boolean
  itemCount: number
  writable: boolean
}

export interface SidecarAsset {
  relativePath: string
  kind: string
}

export interface MediaItem {
  id: number
  sourceId: number
  relativePath: string
  titleHint: string
  yearHint: number | null
  title?: string
  posterUrl?: string
  fileSize: number
  modifiedAt: string
  sidecars: SidecarAsset[]
}

export interface TVShow {
  id: number
  sourceId: number
  relativePath: string
  titleHint: string
  yearHint: number | null
  episodeCount: number
  seasonCount: number
  posterUrl?: string
}

export interface TVEpisode extends MediaItem {
  seasonNumber: number
  episodeStart: number
  episodeEnd: number
}

export type TVSelection =
  | { kind: 'show'; showId: number }
  | { kind: 'season'; showId: number; seasonNumber: number }
  | { kind: 'episode'; showId: number; seasonNumber: number; episodeId: number }

export interface TVShowDetail {
  show: TVShow
  episodes: TVEpisode[]
  artwork: TVArtwork[]
  writable: boolean
  metadata: TVMetadata
  metadataOrigin: 'draft' | 'nfo' | 'empty'
}

export interface TVArtwork {
  id: string
  kind: 'poster' | 'fanart' | 'image'
  relativePath: string
}

export interface TVMetadata {
  showId: number
  provider: string
  providerId: string
  title: string
  originalTitle: string
  year: number | null
  overview: string
  genres: string[]
  posterUrl: string
  backdropUrl: string
  rating: number | null
  votes: number | null
  status: string
  network: string
  cast: CastMember[]
  episodes: TVEpisodeMetadata[]
  updatedAt?: string
}

export interface TVEpisodeMetadata {
  seasonNumber: number
  episodeNumber: number
  title: string
  overview: string
  airDate: string
  runtimeMinutes: number | null
  stillUrl: string
}

export interface Job {
  id: number
  kind?: string
  sourceId?: number
  state: 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled' | string
  progressCurrent: number
  progressTotal?: number
  message: string
  errorMessage: string
  createdAt?: string
}

export interface Metadata {
  mediaItemId: number
  provider: string
  providerId: string
  title: string
  originalTitle: string
  year: number | null
  overview: string
  runtimeMinutes: number | null
  genres: string[]
  posterUrl: string
  backdropUrl: string
  rating: number | null
  votes: number | null
  contentRating: string
  directors: string[]
  writers: string[]
  studios: string[]
  cast: CastMember[]
  lockedFields: string[]
  updatedAt?: string
}

export interface CastMember {
  name: string
  role: string
  profileUrl: string
}

export interface Candidate {
  id: string
  title: string
  originalTitle: string
  year: number | null
  overview: string
  posterUrl: string
}

export interface WritePlan {
  id: string
  mediaItemId: number
  targetPath: string
  content: string
  state: string
  conflict: boolean
  willReplace: boolean
  createdAt?: string
}

export interface ArtworkAsset {
  kind: 'poster' | 'fanart' | 'clearlogo' | 'clearart' | 'discart' | 'banner' | 'landscape'
  candidateId?: string
  provider?: string
  providerAssetId?: string
  sourceUrl: string
  previewUrl?: string
  language?: string
  likes?: number
  width?: number
  height?: number
  mimeType?: string
  targetPath: string
  conflict: boolean
  willReplace: boolean
}

export interface ArtworkCandidate {
  id: string
  mediaItemId: number
  provider: string
  providerAssetId: string
  kind: ArtworkAsset['kind']
  sourceUrl: string
  previewUrl: string
  language: string
  likes: number
  width: number
  height: number
  mimeType: string
}

export interface ArtworkPlan {
  id: string
  mediaItemId: number
  state: string
  createdAt: string
  assets: ArtworkAsset[]
}

export interface Settings {
  tmdbApiKeyConfigured: boolean
  fanartTvApiKeyConfigured: boolean
  outboundProxyConfigured: boolean
  tmdbLanguage: string
  mediaRoots: string[]
}

export interface SettingsUpdate {
  tmdbApiKey: string
  clearTmdbApiKey: boolean
  fanartTvApiKey: string
  clearFanartTvApiKey: boolean
  tmdbLanguage: string
  outboundProxy: string
  clearOutboundProxy: boolean
}
