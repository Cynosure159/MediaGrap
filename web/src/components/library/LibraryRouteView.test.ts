import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import LibraryRouteView from './LibraryRouteView.vue'
import { router } from '@/router'
import * as api from '@/api/library'
import TVShowInspector from './TVShowInspector.vue'

describe('LibraryRouteView', () => {
  afterEach(async () => {
    vi.restoreAllMocks()
    await router.push('/movies')
  })

  it('routes TV selection events emitted with the kebab-case listener name', async () => {
    await router.push('/shows')
    const wrapper = mount(LibraryRouteView, {
      props: {
        mediaKind: 'shows',
        csrfToken: 'csrf',
        username: 'admin',
        labels: {},
      },
      global: {
        plugins: [router],
        stubs: {
          LibraryWorkspace: {
            emits: ['selectTvSelection'],
            template: '<button @click="$emit(\'selectTvSelection\', { kind: \'show\', showId: 42 })">select</button>',
          },
        },
      },
    })

    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe('/shows?show=42')
  })

  it('keeps a direct TV selection rendered through the real workspace', async () => {
    vi.spyOn(api, 'sources').mockResolvedValue({ items: [{ id: 1, name: 'TV', rootPath: '/tv', enabled: true, itemCount: 1, writable: true, scanMode: 'incremental', scheduleEnabled: false, scheduleIntervalMinutes: 1440 }] })
    vi.spyOn(api, 'media').mockResolvedValue({ items: [], total: 0 })
    vi.spyOn(api, 'tvShows').mockResolvedValue({ items: [{ id: 1, sourceId: 1, relativePath: 'Show', titleHint: 'Show', yearHint: 2024, episodeCount: 1, seasonCount: 1 }] })
    vi.spyOn(api, 'jobs').mockResolvedValue({ items: [] })
    vi.spyOn(api, 'tvShowDetail').mockResolvedValue({
      show: { id: 1, sourceId: 1, relativePath: 'Show', titleHint: 'Show', yearHint: 2024, episodeCount: 1, seasonCount: 1 },
      episodes: [{ id: 10, sourceId: 1, relativePath: 'Show/S01E01.mkv', titleHint: 'Episode', yearHint: null, fileSize: 1, modifiedAt: '', sidecars: [], seasonNumber: 1, episodeStart: 1, episodeEnd: 1 }],
      artwork: [],
      writable: true,
      metadata: { showId: 1, provider: '', providerId: '', title: 'Show', originalTitle: '', year: 2024, overview: '', genres: [], posterUrl: '', backdropUrl: '', rating: null, votes: null, status: '', network: '', cast: [], episodes: [] },
      metadataOrigin: 'empty',
    })
    vi.spyOn(api, 'tvNfoRaw').mockResolvedValue({ exists: false, targetPath: '', content: '' })

    await router.push('/shows?show=1&season=1&episode=10')
    expect(router.currentRoute.value.name).toBe('shows')
    expect(router.currentRoute.value.query).toEqual({ show: '1', season: '1', episode: '10' })
    const wrapper = mount(LibraryRouteView, {
      props: { mediaKind: 'shows', csrfToken: 'csrf', username: 'admin', labels: {} },
      global: { plugins: [router] },
    })
    await flushPromises()

    expect(wrapper.findComponent(TVShowInspector).props('selection')).toEqual({ kind: 'episode', showId: 1, seasonNumber: 1, episodeId: 10 })
    expect(wrapper.find('.inspector-workspace').exists()).toBe(true)
    expect(wrapper.find('.workspace-shell').attributes('selected-tv-selection')).toBeUndefined()
    wrapper.unmount()
  })

})
