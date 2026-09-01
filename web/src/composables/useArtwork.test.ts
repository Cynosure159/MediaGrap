import { afterEach, describe, expect, it, vi } from 'vitest'
import * as api from '@/api/library'
import { useArtwork } from './useArtwork'

vi.mock('@/api/library', () => ({
  artworkCandidates: vi.fn(),
  scrapeArtworkCandidates: vi.fn(),
}))

const candidate = (id: string, kind: 'poster' | 'fanart'): api.ArtworkCandidate => ({
  id,
  mediaItemId: 21,
  provider: 'fanart.tv',
  providerAssetId: id,
  kind,
  sourceUrl: `https://assets.fanart.tv/fanart/${id}.jpg`,
  previewUrl: `https://assets.fanart.tv/preview/${id}.jpg`,
  language: 'en',
  likes: 1,
  width: 100,
  height: 100,
  mimeType: 'image/jpeg',
})

describe('useArtwork', () => {
  afterEach(() => vi.clearAllMocks())

  it('selects the first candidate only for artwork kinds without local files', async () => {
    vi.mocked(api.scrapeArtworkCandidates).mockResolvedValueOnce({ items: [candidate('poster-1', 'poster'), candidate('fanart-1', 'fanart')] })
    const artwork = useArtwork(() => 21, () => 'csrf', kind => kind === 'poster')

    await artwork.scrape()

    expect(artwork.selected.poster).toBeFalsy()
    expect(artwork.selected.fanart).toBe('fanart-1')
  })
})
