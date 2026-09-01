import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import NfoPreview from './NfoPreview.vue'

const labels = {
  nfoSavePreview: 'NFO save preview',
  conflict: 'Conflict',
  overwrite: 'Overwrite',
  newFile: 'New file',
  nfoPreview: 'NFO preview',
  closeNfoPreview: 'Close NFO preview',
  target: 'Target',
  nfoConflictHelp: 'The target is not a regular NFO file.',
  nfoReplaceHelp: 'An existing local NFO will be replaced atomically.',
  xmlContent: 'XML content',
  cancel: 'Cancel',
  apply: 'Apply',
}

describe('NfoPreview', () => {
  it('renders write plan details and xml content', () => {
    const wrapper = mount(NfoPreview, {
      props: {
        plan: {
          id: 'plan-123',
          mediaItemId: 1,
          targetPath: '/media/movie.nfo',
          content: '<?xml version="1.0"?><movie><title>Test</title></movie>',
          state: 'previewed',
          createdAt: '2026-09-01T00:00:00Z',
          conflict: false,
          willReplace: true,
        },
        applying: false,
        labels,
      },
    })

    expect(wrapper.text()).toContain('/media/movie.nfo')
    expect(wrapper.text()).toContain('<movie><title>Test</title></movie>')
    expect(wrapper.find('.badge-replace').exists()).toBe(true)
  })

  it('displays conflict banner and disables apply button on conflict', () => {
    const wrapper = mount(NfoPreview, {
      props: {
        plan: {
          id: 'plan-conflict',
          mediaItemId: 1,
          targetPath: '/media/symlink.nfo',
          content: '<movie/>',
          state: 'previewed',
          createdAt: '2026-09-01T00:00:00Z',
          conflict: true,
          willReplace: false,
        },
        applying: false,
        labels,
      },
    })

    expect(wrapper.find('.banner-conflict').exists()).toBe(true)
    const applyBtn = wrapper.find('.btn-success')
    expect(applyBtn.attributes('disabled')).toBeDefined()
  })

  it('emits apply event when apply button is clicked', async () => {
    const wrapper = mount(NfoPreview, {
      props: {
        plan: {
          id: 'plan-ok',
          mediaItemId: 1,
          targetPath: '/media/movie.nfo',
          content: '<movie/>',
          state: 'previewed',
          createdAt: '2026-09-01T00:00:00Z',
          conflict: false,
          willReplace: false,
        },
        applying: false,
        labels,
      },
    })

    await wrapper.find('.btn-success').trigger('click')
    expect(wrapper.emitted('apply')).toHaveLength(1)
  })

  it('emits close event when close button or cancel button is clicked', async () => {
    const wrapper = mount(NfoPreview, {
      props: {
        plan: {
          id: 'plan-ok',
          mediaItemId: 1,
          targetPath: '/media/movie.nfo',
          content: '<movie/>',
          state: 'previewed',
          createdAt: '2026-09-01T00:00:00Z',
          conflict: false,
          willReplace: false,
        },
        applying: false,
        labels,
      },
    })

    await wrapper.find('.close-btn').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)

    await wrapper.find('.dialog-footer .btn-ghost').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(2)
  })
})
