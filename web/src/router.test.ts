import { describe, expect, it } from 'vitest'
import {
  movieRouteQuery,
  parseInspectorTab,
  parsePositiveQueryId,
  parseTVSelection,
  router,
  tvRouteQuery,
} from './router'

describe('URL route state', () => {
  it('serializes movie selection and workshop', () => {
    expect(movieRouteQuery(12)).toEqual({ movie: '12' })
    expect(movieRouteQuery(12, 'files')).toEqual({ movie: '12', tab: 'files' })
    expect(movieRouteQuery(null, 'artwork')).toEqual({})
  })

  it('serializes and parses each TV hierarchy level', () => {
    const show = { kind: 'show', showId: 20 } as const
    const season = { kind: 'season', showId: 20, seasonNumber: 0 } as const
    const episode = { kind: 'episode', showId: 20, seasonNumber: 2, episodeId: 456 } as const

    expect(parseTVSelection(tvRouteQuery(show))).toEqual(show)
    expect(parseTVSelection(tvRouteQuery(season))).toEqual(season)
    expect(parseTVSelection(tvRouteQuery(episode, 'nfo'))).toEqual(episode)
    expect(tvRouteQuery(episode, 'nfo')).toEqual({ show: '20', season: '2', episode: '456', tab: 'nfo' })
  })

  it('rejects unsafe or incomplete query values', () => {
    expect(parsePositiveQueryId({ movie: '0' }, 'movie')).toBeNull()
    expect(parsePositiveQueryId({ movie: '-1' }, 'movie')).toBeNull()
    expect(parsePositiveQueryId({ movie: '1.2' }, 'movie')).toBeNull()
    expect(parseTVSelection({ episode: '456', season: '2' })).toBeNull()
    expect(parseTVSelection({ show: '20', episode: '456' })).toEqual({ kind: 'show', showId: 20 })
  })

  it('falls back to overview for an unknown workshop', () => {
    expect(parseInspectorTab({ tab: 'unknown' })).toBe('overview')
    expect(parseInspectorTab({ tab: ['files', 'nfo'] })).toBe('files')
  })

  it('keeps the old sources bookmark pointed at settings', async () => {
    await router.push({ name: 'shows', query: tvRouteQuery({ kind: 'episode', showId: 20, seasonNumber: 2, episodeId: 456 }, 'nfo') })
    expect(router.currentRoute.value.fullPath).toBe('/shows?show=20&season=2&episode=456&tab=nfo')

    await router.push('/sources')
    expect(router.currentRoute.value.name).toBe('settings')
    expect(router.currentRoute.value.fullPath).toBe('/settings')

    await router.push('/unknown-page')
    expect(router.currentRoute.value.name).toBe('movies')
    expect(router.currentRoute.value.fullPath).toBe('/movies')
  })
})
