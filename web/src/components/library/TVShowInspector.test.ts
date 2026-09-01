import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TVShowInspector from './TVShowInspector.vue'

describe('TVShowInspector', () => {
  it('renders no selection state when selection is null', () => {
    const wrapper = mount(TVShowInspector, {
      props: {
        selection: null,
        csrfToken: 'test-token',
        labels: { selectShow: 'Select a TV show to inspect' },
      },
    })

    expect(wrapper.text()).toContain('Select a TV show to inspect')
  })
})
