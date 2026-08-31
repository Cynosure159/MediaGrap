export interface Session { user: { id: number; username: string }; csrfToken: string }
export interface Source { id: number; name: string; rootPath: string; enabled: boolean; itemCount: number; writable: boolean }
export interface MediaItem { id: number; sourceId: number; relativePath: string; titleHint: string; yearHint: number | null; fileSize: number; modifiedAt: string; sidecars: { relativePath: string; kind: string }[] }
export interface Job { id: number; state: string; progressCurrent: number; message: string; errorMessage: string }
export interface Metadata { mediaItemId: number; provider: string; providerId: string; title: string; originalTitle: string; year: number | null; overview: string; runtimeMinutes: number | null; genres: string[]; posterUrl: string; backdropUrl: string; lockedFields: string[] }
export interface Candidate { id: string; title: string; originalTitle: string; year: number | null; overview: string; posterUrl: string }
export interface WritePlan { id: string; mediaItemId: number; targetPath: string; content: string; state: string; conflict: boolean }

async function request<T>(path: string, options: RequestInit = {}): Promise<T> { const response = await fetch(path, { credentials: 'same-origin', ...options }); if (!response.ok) { const data = await response.json().catch(() => null) as { error?: { message?: string } } | null; throw new Error(data?.error?.message ?? 'Request failed') }; return response.status === 204 ? undefined as T : response.json() as Promise<T> }
export const setupStatus = () => request<{ needsSetup: boolean }>('/api/v1/setup/status')
export const session = () => request<Session>('/api/v1/session')
export const signIn = (username: string, password: string) => request<Session>('/api/v1/session', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, password }) })
export const setup = (username: string, password: string) => request<Session>('/api/v1/setup', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, password }) })
export const sources = () => request<{ items: Source[] }>('/api/v1/sources')
export const addSource = (csrf: string, name: string, rootPath: string) => request<Source>('/api/v1/sources', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify({ name, rootPath }) })
export const scanSource = (csrf: string, id: number) => request<Job>(`/api/v1/sources/${id}/scans`, { method: 'POST', headers: { 'X-CSRF-Token': csrf } })
export const media = (q = '') => request<{ items: MediaItem[]; total: number }>(`/api/v1/media?page=1&pageSize=50&q=${encodeURIComponent(q)}`)
export const jobs = () => request<{ items: Job[] }>('/api/v1/jobs')
export const mediaDetail = (id: number) => request<{ item: MediaItem; metadata: Metadata; writable: boolean }>(`/api/v1/media/${id}`)
export const candidates = (id: number) => request<{ items: Candidate[] }>(`/api/v1/media/${id}/candidates`)
export const selectCandidate = (csrf: string, id: number, candidateId: string) => request<Metadata>(`/api/v1/media/${id}/select`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify({ candidateId }) })
export const saveMetadata = (csrf: string, id: number, metadata: Metadata) => request<Metadata>(`/api/v1/media/${id}/metadata`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify(metadata) })
export const previewNfo = (csrf: string, id: number, metadata: Metadata) => request<WritePlan>(`/api/v1/media/${id}/write-plans`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify(metadata) })
export const applyNfo = (csrf: string, id: string) => request<WritePlan>(`/api/v1/write-plans/${id}/apply`, { method: 'POST', headers: { 'X-CSRF-Token': csrf } })
