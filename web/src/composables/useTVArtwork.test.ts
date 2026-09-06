import { afterEach, describe, expect, it, vi } from 'vitest'
import * as api from '@/api/library'
import { useTVArtwork } from './useTVArtwork'

vi.mock('@/api/library', () => ({
  tvArtworkCandidates: vi.fn(),
  scrapeTVArtworkCandidates: vi.fn(),
  previewTVArtworkSelection: vi.fn(),
  applyTVArtwork: vi.fn(),
}))

function candidate(id: string, kind: api.TVArtworkKind, seasonNumber?: number): api.TVArtworkCandidate {
  return {
    id,
    showId: 9,
    scope: seasonNumber === undefined ? 'show' : 'season',
    seasonNumber,
    provider: 'fanart.tv',
    providerAssetId: id,
    kind,
    sourceUrl: `https://assets.fanart.tv/fanart/tv/9/${id}.jpg`,
    previewUrl: `https://assets.fanart.tv/preview/tv/9/${id}.jpg`,
    language: 'en',
    likes: 1,
    width: 1000,
    height: 1500,
    mimeType: 'image/jpeg',
  }
}

describe('useTVArtwork', () => {
  afterEach(() => vi.clearAllMocks())

  it('groups one season and avoids preselecting existing artwork', async () => {
    vi.mocked(api.scrapeTVArtworkCandidates).mockResolvedValueOnce({ items: [candidate('poster', 'season_poster', 1), candidate('banner', 'season_banner', 1)] })
    const artwork = useTVArtwork(() => 9, () => ({ scope: 'season', seasonNumber: 1 }), () => 'csrf', kind => kind === 'season_poster')

    await artwork.scrape()

    expect(artwork.groups.value.map(group => group.kind)).toEqual(['season_poster', 'season_banner'])
    expect(artwork.selected.season_poster).toBeFalsy()
    expect(artwork.selected.season_banner).toBe('banner')
  })
})
