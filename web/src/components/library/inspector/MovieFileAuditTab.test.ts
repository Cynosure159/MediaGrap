import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MovieFileAuditTab from './MovieFileAuditTab.vue'
import type { MediaInspection, MediaItem } from '@/api/types'

const labels = new Proxy<Record<string, string>>({}, { get: (_, key) => String(key) })
const item: MediaItem = { id: 7, sourceId: 1, relativePath: 'Movie.mkv', titleHint: 'Movie', yearHint: 2024, fileSize: 100, modifiedAt: '', sidecars: [] }
const inspection: MediaInspection = {
  probeStatus: 'ready', cached: true, probedAt: '', format: { name: 'matroska', durationSeconds: 120, bitRate: 1000 },
  video: [{ index: 0, codec: 'hevc', width: 3840, height: 2160, bitDepth: 10, hdr: 'HDR10' }],
  audio: [{ index: 1, codec: 'eac3', channels: 6, channelLayout: '5.1', language: 'eng' }],
  subtitles: [{ index: 2, codec: 'subrip', language: 'zho' }],
  files: [{ relativePath: 'Movie.mkv', kind: 'video', size: 100, mimeType: 'video/x-matroska', modifiedAt: '', permissions: '-rw-r-----', writable: true, regular: true, symlink: false, valid: true, warnings: [] }],
}

describe('MovieFileAuditTab', () => {
  it('renders real file and stream values and emits a read-only preview request', async () => {
    const wrapper = mount(MovieFileAuditTab, { props: { item, labels, inspection, inspectionLoading: false, inspectionError: null, namingPreview: null, previewLoading: false } })
    expect(wrapper.text()).toContain('3840×2160')
    expect(wrapper.text()).toContain('video/x-matroska')
    expect(wrapper.text()).toContain('HDR10')

    await wrapper.find('.btn-primary').trigger('click')
    expect(wrapper.emitted('previewNaming')?.[0]).toEqual(['${title} (${year})'])
  })

  it('shows filesystem safety warnings', () => {
    const warned: MediaInspection = { ...inspection, files: [{ ...inspection.files[0], symlink: true, valid: false, warnings: ['symlink'] }] }
    const wrapper = mount(MovieFileAuditTab, { props: { item, labels, inspection: warned, inspectionLoading: false, inspectionError: null, namingPreview: null, previewLoading: false } })
    expect(wrapper.text()).toContain('audit_symlink')
    expect(wrapper.find('.audit-row--warning').exists()).toBe(true)
  })
})
