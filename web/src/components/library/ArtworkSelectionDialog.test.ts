import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import ArtworkSelectionDialog from './ArtworkSelectionDialog.vue'

const labels = {
  artworkEyebrow: 'Fanart.tv artwork',
  artworkCandidates: 'Fanart.tv candidates',
  artworkDialogHelp: 'Choose one image per artwork type.',
  artworkSelectionHint: 'Files are unchanged until confirmation.',
  artworkProxyHint: 'Images use the server proxy.',
  artworkRefresh: 'Refresh candidates',
  artworkSelect: 'Select',
  artworkSelected: 'Selected',
  artworkPreview: 'Preview download',
  artworkGroups: 'groups',
  artworkNoCandidates: 'No artwork found.',
  closeArtworkCandidates: 'Close artwork candidates',
  cancel: 'Cancel',
}

const groups = [{
  kind: 'poster' as const,
  items: [{
    id: 'candidate-1', mediaItemId: 1, provider: 'fanart.tv', providerAssetId: 'asset-1',
    kind: 'poster' as const, sourceUrl: 'https://assets.fanart.tv/fanart/poster.png',
    previewUrl: 'https://assets.fanart.tv/preview/poster.png', language: 'en', likes: 2,
    width: 1000, height: 1500, mimeType: 'image/png',
  }],
}]

describe('ArtworkSelectionDialog', () => {
  it('emits close from the close button and backdrop', async () => {
    const wrapper = mount(ArtworkSelectionDialog, {
      props: { groups, selected: {}, labels, loading: false, writable: true },
    })

    document.body.querySelector<HTMLButtonElement>('.modal-close-btn')?.click()
    document.body.querySelector<HTMLElement>('.artwork-modal-backdrop')?.click()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()

    expect(wrapper.emitted('close')).toHaveLength(3)
    wrapper.unmount()
  })

  it('keeps preview disabled until an artwork is selected', async () => {
    const wrapper = mount(ArtworkSelectionDialog, {
      props: { groups, selected: {}, labels, loading: false, writable: true },
    })
    const previewButton = document.body.querySelector<HTMLButtonElement>('.artwork-modal-footer .btn-primary')

    expect(previewButton?.hasAttribute('disabled')).toBe(true)
    await wrapper.setProps({ selected: { poster: 'candidate-1' } })
    await nextTick()
    expect(document.body.querySelector<HTMLButtonElement>('.artwork-modal-footer .btn-primary')?.hasAttribute('disabled')).toBe(false)
    wrapper.unmount()
  })
})
