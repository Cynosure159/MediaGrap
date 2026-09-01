import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MediaCatalog from './MediaCatalog.vue'

describe('MediaCatalog', () => {
  it('queues a scan when the refresh button is clicked', async () => {
    const wrapper = mount(MediaCatalog, {
      props: {
        items: [],
        selectedId: null,
        activeJob: undefined,
        labels: { refresh: 'Refresh', movies: 'Movies' },
      },
    })

    await wrapper.get('[title="Refresh"]').trigger('click')

    expect(wrapper.emitted('scan')).toHaveLength(1)
  })
})
