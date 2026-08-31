export interface Session { user: { id: number; username: string }; csrfToken: string }
export interface Source { id: number; name: string; rootPath: string; enabled: boolean; itemCount: number; writable: boolean }
export interface MediaItem { id: number; sourceId: number; relativePath: string; titleHint: string; yearHint: number | null; fileSize: number; modifiedAt: string; sidecars: { relativePath: string; kind: string }[] }
export interface Job { id: number; state: string; progressCurrent: number; message: string; errorMessage: string }

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

