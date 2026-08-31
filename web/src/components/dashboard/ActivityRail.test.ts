import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ActivityRail from './ActivityRail.vue'

describe('ActivityRail', () => {
  it('shows foundation status when no summary is available', () => {
    const wrapper = mount(ActivityRail, { props: { summary: null } })
    expect(wrapper.text()).toContain('Database migration')
    expect(wrapper.text()).toContain('foundation')
  })
})

