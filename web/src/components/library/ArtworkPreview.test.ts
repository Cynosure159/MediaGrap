import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ArtworkPreview from './ArtworkPreview.vue'

const labels = {
  artworkEyebrow: 'TMDb artwork',
  artworkPreview: 'Artwork download preview',
  artworkHelp: 'Downloads use the selected TMDb JPEG files.',
  poster: 'Poster',
  fanart: 'Fanart',
  artworkConflict: 'This target is not a regular image file.',
  artworkReplace: 'An existing local image will be atomically replaced.',
  artworkCreate: 'A new local image will be created.',
  cancel: 'Cancel',
  downloadArtwork: 'Download artwork',
  downloading: 'Downloading…',
}

describe('ArtworkPreview', () => {
  it('renders artwork asset list properly', () => {
    const wrapper = mount(ArtworkPreview, {
      props: {
        plan: {
          id: 'art-plan-1',
          mediaItemId: 1,
          state: 'previewed',
          createdAt: '2026-09-01T00:00:00Z',
          assets: [
            {
              kind: 'poster',
              sourceUrl: 'https://image.tmdb.org/t/p/original/poster.jpg',
              targetPath: '/media/poster.jpg',
              conflict: false,
              willReplace: false,
            },
            {
              kind: 'fanart',
              sourceUrl: 'https://image.tmdb.org/t/p/original/fanart.jpg',
              targetPath: '/media/fanart.jpg',
              conflict: false,
              willReplace: true,
            },
          ],
        },
        applying: false,
        labels,
      },
    })

    expect(wrapper.text()).toContain('/media/poster.jpg')
    expect(wrapper.text()).toContain('/media/fanart.jpg')
    expect(wrapper.text()).toContain(labels.artworkCreate)
    expect(wrapper.text()).toContain(labels.artworkReplace)
  })

  it('disables download button if any asset has conflict', () => {
    const wrapper = mount(ArtworkPreview, {
      props: {
        plan: {
          id: 'art-plan-conflict',
          mediaItemId: 1,
          state: 'previewed',
          createdAt: '2026-09-01T00:00:00Z',
          assets: [
            {
              kind: 'poster',
              sourceUrl: 'https://image.tmdb.org/t/p/original/poster.jpg',
              targetPath: '/media/symlink.jpg',
              conflict: true,
              willReplace: false,
            },
          ],
        },
        applying: false,
        labels,
      },
    })

    const downloadBtn = wrapper.findAll('button')[2]
    expect(downloadBtn.attributes('disabled')).toBeDefined()
  })

  it('emits apply when download button is clicked', async () => {
    const wrapper = mount(ArtworkPreview, {
      props: {
        plan: {
          id: 'art-plan-ok',
          mediaItemId: 1,
          state: 'previewed',
          createdAt: '2026-09-01T00:00:00Z',
          assets: [
            {
              kind: 'poster',
              sourceUrl: 'https://image.tmdb.org/t/p/original/poster.jpg',
              targetPath: '/media/poster.jpg',
              conflict: false,
              willReplace: false,
            },
          ],
        },
        applying: false,
        labels,
      },
    })

    const downloadBtn = wrapper.findAll('button')[2]
    await downloadBtn.trigger('click')
    expect(wrapper.emitted('apply')).toHaveLength(1)
  })
})
