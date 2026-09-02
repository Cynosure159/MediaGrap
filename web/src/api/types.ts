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
  scanMode: 'full' | 'incremental'
  scheduleEnabled: boolean
  scheduleIntervalMinutes: number
  nextScanAt?: string
  lastScanAt?: string
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

export interface ProbeFormat {
  name: string
  durationSeconds: number
  bitRate: number
}

export interface VideoStream {
  index: number
  codec: string
  profile?: string
  width: number
  height: number
  pixelFormat?: string
  bitDepth?: number
  hdr?: string
  language?: string
}

export interface AudioStream {
  index: number
  codec: string
  channels: number
  channelLayout?: string
  language?: string
  title?: string
}

export interface SubtitleStream {
  index: number
  codec: string
  language?: string
  title?: string
}

export interface FileAuditEntry {
  relativePath: string
  kind: string
  size: number
  mimeType: string
  modifiedAt: string
  permissions: string
  writable: boolean
  regular: boolean
  symlink: boolean
  valid: boolean
  warnings: ReadonlyArray<string>
}

export interface MediaInspection {
  probeStatus: 'ready' | 'unavailable' | 'failed'
  probeError?: string
  cached: boolean
  probedAt?: string
  format: ProbeFormat
  video: ReadonlyArray<VideoStream>
  audio: ReadonlyArray<AudioStream>
  subtitles: ReadonlyArray<SubtitleStream>
  files: ReadonlyArray<FileAuditEntry>
}

export interface NamingPreviewItem {
  kind: string
  currentPath: string
  plannedPath: string
  operation: 'keep' | 'rename' | 'conflict'
  conflict: boolean
}

export interface NamingPreview {
  pattern: string
  readOnly: true
  items: ReadonlyArray<NamingPreviewItem>
  warnings: ReadonlyArray<string>
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
  startedAt?: string
  completedAt?: string
  updatedAt?: string
  retryCount?: number
  maxRetries?: number
}

export interface AuditEntry {
  id: number
  action: string
  mediaItemId?: number
  target: string
  detail: string
  outcome: 'previewed' | 'applied' | 'recorded' | string
  backup: string
  recoverability: 'atomic_replace_only' | 'not_applicable' | string
  createdAt: string
}

export interface OperationsStatus {
  application: { name: string; version: string; commit: string; builtAt: string }
  database: { ready: boolean; latestMigration: string; sizeBytes: number; walMode: boolean; journalMode: string }
  cache: { path: string; available: boolean; writable: boolean; usedBytes: number }
  mounts: Array<{ id: number; name: string; available: boolean; writable: boolean; itemCount: number; scanMode: string; scheduleEnabled: boolean; nextScanAt?: string; lastScanAt?: string }>
  providers: Array<{ id: string; configured: boolean; status: string }>
  network: { proxyConfigured: boolean; noProxyConfigured: boolean }
  connectionTests: Record<string, ConnectionTest>
}

export interface ConnectionTest {
  target: 'tmdb' | 'fanart_tv' | 'proxy'
  status: 'reachable' | 'failed' | 'not_configured'
  httpStatus?: number
  durationMs: number
  message: string
  testedAt?: string
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
  fallbackLanguage: string
  noProxyConfigured: boolean
  theme: 'dark' | 'light' | 'system'
  locale: 'en' | 'zh-CN'
  mediaRoots: string[]
}

export interface SettingsUpdate {
  tmdbApiKey: string
  clearTmdbApiKey: boolean
  fanartTvApiKey: string
  clearFanartTvApiKey: boolean
  tmdbLanguage: string
  fallbackLanguage: string
  outboundProxy: string
  clearOutboundProxy: boolean
  noProxy: string
  clearNoProxy: boolean
  theme: 'dark' | 'light' | 'system'
  locale: 'en' | 'zh-CN'
}

export interface RenamePlanItem {
  kind: string
  currentPath: string
  plannedPath: string
  operation: 'keep' | 'rename' | 'rename_dir' | 'conflict'
  conflict: boolean
  status: string
}

export interface RenamePlan {
  id: string
  mediaItemId?: number
  tvShowId?: number
  pattern: string
  state: 'previewed' | 'applied' | 'partial' | 'failed' | 'cancelled'
  items: ReadonlyArray<RenamePlanItem>
  warnings: ReadonlyArray<string>
  hasConflicts: boolean
  createdAt: string
  appliedAt?: string
}
