import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import SearchBar from './SearchBar.vue'

describe('SearchBar', () => {
  it('emits search event when enter key is pressed', async () => {
    const wrapper = mount(SearchBar, {
      props: {
        modelValue: 'Interstellar',
        'onUpdate:modelValue': (e: string) => wrapper.setProps({ modelValue: e }),
        placeholder: 'Search movies...',
      },
    })

    const input = wrapper.get('input')
    expect(input.attributes('placeholder')).toBe('Search movies...')
    await input.trigger('keydown.enter')

    expect(wrapper.emitted('search')).toBeTruthy()
    expect(wrapper.emitted('search')?.[0]).toEqual(['Interstellar'])
  })
})
