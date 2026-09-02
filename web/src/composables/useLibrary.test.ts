import { afterEach, describe, expect, it, vi } from 'vitest'
import * as api from '@/api/library'
import { useLibrary } from './useLibrary'

vi.mock('@/api/library', () => ({
  sources: vi.fn(),
  media: vi.fn(),
  tvShows: vi.fn(),
  jobs: vi.fn(),
  addSource: vi.fn(),
  deleteSource: vi.fn(),
  scanSource: vi.fn(),
}))

describe('useLibrary', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('loads sources, media, tvShows, and jobs on refresh', async () => {
    vi.mocked(api.sources).mockResolvedValueOnce({ items: [{ id: 1, name: 'Movies', rootPath: '/media', enabled: true, itemCount: 10, writable: true, scanMode: 'incremental', scheduleEnabled: false, scheduleIntervalMinutes: 1440 }] })
    vi.mocked(api.media).mockResolvedValueOnce({ items: [{ id: 101, sourceId: 1, relativePath: 'Movie.mkv', titleHint: 'Movie', yearHint: 2024, fileSize: 1000, modifiedAt: 'now', sidecars: [] }], total: 1 })
    vi.mocked(api.tvShows).mockResolvedValueOnce({ items: [{ id: 201, sourceId: 1, relativePath: 'Show', titleHint: 'Show', yearHint: 2024, episodeCount: 1, seasonCount: 1 }] })
    vi.mocked(api.jobs).mockResolvedValueOnce({ items: [] })

    const library = useLibrary(() => 'test-csrf')
    expect(library.hasSources.value).toBe(false)

    await library.refresh()

    expect(library.isLoading.value).toBe(false)
    expect(library.error.value).toBeNull()
    expect(library.sourceItems.value).toHaveLength(1)
    expect(library.mediaItems.value).toHaveLength(1)
    expect(library.tvShowItems.value).toHaveLength(1)
    expect(library.hasSources.value).toBe(true)
  })

  it('handles error when api fails during refresh', async () => {
    vi.mocked(api.sources).mockRejectedValueOnce(new Error('Network error'))
    vi.mocked(api.media).mockResolvedValueOnce({ items: [], total: 0 })
    vi.mocked(api.tvShows).mockResolvedValueOnce({ items: [] })
    vi.mocked(api.jobs).mockResolvedValueOnce({ items: [] })

    const library = useLibrary(() => 'test-csrf')
    await library.refresh()

    expect(library.error.value).toBe('Network error')
    expect(library.isLoading.value).toBe(false)
  })

  it('calls createSource and triggers refresh', async () => {
    vi.mocked(api.addSource).mockResolvedValueOnce({ id: 2, name: 'TV', rootPath: '/tv', enabled: true, itemCount: 0, writable: true, scanMode: 'incremental', scheduleEnabled: false, scheduleIntervalMinutes: 1440 })
    vi.mocked(api.sources).mockResolvedValueOnce({ items: [] })
    vi.mocked(api.media).mockResolvedValueOnce({ items: [], total: 0 })
    vi.mocked(api.tvShows).mockResolvedValueOnce({ items: [] })
    vi.mocked(api.jobs).mockResolvedValueOnce({ items: [] })

    const library = useLibrary(() => 'test-csrf')
    await library.createSource('TV', '/tv')

    expect(api.addSource).toHaveBeenCalledWith('test-csrf', 'TV', '/tv')
    expect(api.sources).toHaveBeenCalled()
  })

  it('calls removeSource and triggers refresh', async () => {
    vi.mocked(api.deleteSource).mockResolvedValueOnce(undefined)
    vi.mocked(api.sources).mockResolvedValueOnce({ items: [] })
    vi.mocked(api.media).mockResolvedValueOnce({ items: [], total: 0 })
    vi.mocked(api.tvShows).mockResolvedValueOnce({ items: [] })
    vi.mocked(api.jobs).mockResolvedValueOnce({ items: [] })

    const library = useLibrary(() => 'test-csrf')
    await library.removeSource(5)

    expect(api.deleteSource).toHaveBeenCalledWith('test-csrf', 5)
    expect(api.sources).toHaveBeenCalled()
  })
})
