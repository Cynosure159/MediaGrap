import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TVShowCatalog from './TVShowCatalog.vue'

describe('TVShowCatalog', () => {
  it('queues a scan when the refresh button is clicked', async () => {
    const wrapper = mount(TVShowCatalog, {
      props: {
        items: [],
        selectedId: null,
        activeJob: undefined,
        labels: { refresh: 'Refresh', tvShows: 'TV shows' },
      },
    })

    await wrapper.get('[title="Refresh"]').trigger('click')

    expect(wrapper.emitted('scan')).toHaveLength(1)
  })
})
