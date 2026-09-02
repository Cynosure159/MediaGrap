import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import TVFileAuditTab from './TVFileAuditTab.vue'
import type { MediaInspection, TVEpisode } from '@/api/types'
import * as api from '@/api/library'

vi.mock('@/api/library', () => ({
  previewTVRename: vi.fn(),
  applyRenamePlan: vi.fn(),
}))

const labels = new Proxy<Record<string, string>>({}, { get: (_, key) => String(key) })

const episodes: TVEpisode[] = [
  {
    id: 101,
    sourceId: 1,
    relativePath: 'Breaking Bad/Season 1/Breaking.Bad.S01E01.mkv',
    titleHint: 'Pilot',
    yearHint: 2008,
    fileSize: 100000,
    modifiedAt: '2026-09-01T00:00:00Z',
    seasonNumber: 1,
    episodeStart: 1,
    episodeEnd: 1,
    sidecars: [
      { relativePath: 'Breaking Bad/Season 1/Breaking.Bad.S01E01.nfo', kind: 'nfo' },
      { relativePath: 'Breaking Bad/Season 1/Breaking.Bad.S01E01.zh.srt', kind: 'subtitle' },
    ],
  },
  {
    id: 102,
    sourceId: 1,
    relativePath: 'Breaking Bad/Season 1/Breaking.Bad.S01E02.mkv',
    titleHint: 'Cat\'s in the Bag...',
    yearHint: 2008,
    fileSize: 100000,
    modifiedAt: '2026-09-01T00:00:00Z',
    seasonNumber: 1,
    episodeStart: 2,
    episodeEnd: 2,
    sidecars: [],
  },
]

const inspection: MediaInspection = {
  probeStatus: 'ready',
  cached: true,
  probedAt: '',
  format: { name: 'matroska', durationSeconds: 3000, bitRate: 5000 },
  video: [{ index: 0, codec: 'hevc', width: 1920, height: 1080, bitDepth: 10 }],
  audio: [{ index: 1, codec: 'eac3', channels: 6, channelLayout: '5.1', language: 'eng' }],
  subtitles: [],
  files: [],
}

describe('TVFileAuditTab', () => {
  it('renders TV episodes and automatically displays scope badge based on selection', () => {
    const wrapper = mount(TVFileAuditTab, {
      props: {
        showId: 1,
        episodes,
        allEpisodes: episodes,
        selection: { kind: 'season', showId: 1, seasonNumber: 1 },
        inspection,
        inspectionLoading: false,
        inspectionError: null,
        labels,
      },
    })

    expect(wrapper.text()).toContain('S01E01')
    expect(wrapper.text()).toContain('S01E02')
    expect(wrapper.find('.scope-pill').text()).toContain('1 (2 episodes)')
  })

  it('automatically derives season scope when simulating dry run', async () => {
    const mockPlan = {
      id: 'plan-tv-1',
      tvShowId: 1,
      pattern: '${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}',
      state: 'previewed' as const,
      hasConflicts: false,
      warnings: [],
      createdAt: '2026-09-02T16:00:00Z',
      items: [
        {
          kind: 'video',
          currentPath: 'Breaking Bad/Season 1/Breaking.Bad.S01E01.mkv',
          plannedPath: 'Breaking Bad/Season 1/Breaking Bad - S01E01 - Pilot.mkv',
          operation: 'rename' as const,
          conflict: false,
          status: 'pending',
        },
      ],
    }

    vi.mocked(api.previewTVRename).mockResolvedValueOnce(mockPlan)

    const wrapper = mount(TVFileAuditTab, {
      props: {
        showId: 1,
        episodes,
        allEpisodes: episodes,
        selection: { kind: 'season', showId: 1, seasonNumber: 1 },
        inspection,
        inspectionLoading: false,
        inspectionError: null,
        labels,
        csrfToken: 'csrf-123',
      },
    })

    await wrapper.find('.btn-primary').trigger('click')
    expect(api.previewTVRename).toHaveBeenCalledWith(
      'csrf-123',
      1,
      expect.stringContaining('${showTitle}'),
      { seasonNumber: 1 }
    )

    // Wait for promise resolution
    await wrapper.vm.$nextTick()
    await new Promise(r => setTimeout(r, 10))

    expect(wrapper.text()).toContain('Breaking Bad - S01E01 - Pilot.mkv')
    expect(wrapper.find('.sticky-action-bar').exists()).toBe(true)

    // Execute rename
    vi.mocked(api.applyRenamePlan).mockResolvedValueOnce({ status: 'queued' })
    await wrapper.find('.btn-success').trigger('click')
    expect(api.applyRenamePlan).toHaveBeenCalledWith('csrf-123', 'plan-tv-1')
  })

  it('automatically derives single episode scope when an episode is selected', async () => {
    vi.mocked(api.previewTVRename).mockResolvedValueOnce({
      id: 'plan-tv-ep',
      tvShowId: 1,
      pattern: '${showTitle} - S${seasonNumberPad}E${episodeNumberPad}',
      state: 'previewed' as const,
      hasConflicts: false,
      warnings: [],
      createdAt: '2026-09-02T16:00:00Z',
      items: [],
    })

    const wrapper = mount(TVFileAuditTab, {
      props: {
        showId: 1,
        episodes: [episodes[0]],
        allEpisodes: episodes,
        selection: { kind: 'episode', showId: 1, seasonNumber: 1, episodeId: 101 },
        inspection,
        inspectionLoading: false,
        inspectionError: null,
        labels,
        csrfToken: 'csrf-123',
      },
    })

    expect(wrapper.find('.scope-pill').text()).toContain('S01E01')

    await wrapper.find('.btn-primary').trigger('click')
    expect(api.previewTVRename).toHaveBeenCalledWith(
      'csrf-123',
      1,
      expect.any(String),
      { episodeId: 101 }
    )
  })

  it('changes template when preset selector changes', async () => {
    const wrapper = mount(TVFileAuditTab, {
      props: {
        showId: 1,
        episodes,
        allEpisodes: episodes,
        inspection,
        inspectionLoading: false,
        inspectionError: null,
        labels,
      },
    })

    const select = wrapper.find('.preset-select')
    await select.setValue('plex')
    expect((wrapper.find('.pattern-input').element as HTMLInputElement).value).toContain('s${seasonNumberPad}e${episodeNumberPad}')
  })
})
