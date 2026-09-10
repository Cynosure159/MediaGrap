import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import { useDismissiblePopover } from './useDismissiblePopover'

describe('dismissible popover', () => {
  it('keeps inside clicks open, dismisses outside clicks and cleans up on unmount', async () => {
    let state!: ReturnType<typeof useDismissiblePopover>
    const wrapper = mount(defineComponent({
      setup() { state = useDismissiblePopover(); return state },
      template: '<div ref="container"><button @click="open = true">Open</button><span v-if="open">Options</span></div>',
    }), { attachTo: document.body })
    await wrapper.get('button').trigger('click')
    expect(wrapper.text()).toContain('Options')
    await wrapper.get('span').trigger('click')
    expect(state.open.value).toBe(true)
    document.body.click()
    await nextTick()
    expect(wrapper.find('span').exists()).toBe(false)
    const container = wrapper.element as HTMLElement
    wrapper.unmount()
    state.container.value = container
    state.open.value = true
    document.body.click()
    expect(state.open.value).toBe(true)
  })
})
