import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import JobCenter from './JobCenter.vue'

const labels = new Proxy({} as Record<string, string>, { get: (_target, key) => String(key) })

describe('JobCenter', () => {
  it('shows active progress and emits cancellation', async () => {
    const wrapper = mount(JobCenter, {
      props: {
        labels,
        streamState: 'connected',
        jobs: [{ id: 7, kind: 'scan', state: 'running', progressCurrent: 2, progressTotal: 4, message: 'Scanning', errorMessage: '' }],
      },
    })
    expect(wrapper.text()).toContain('Scanning')
    expect(wrapper.find('.progress-track span').attributes('style')).toContain('50%')
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('cancel')).toEqual([[7]])
  })

  it('offers retry only for retryable history', async () => {
    const wrapper = mount(JobCenter, {
      props: {
        labels,
        streamState: 'reconnecting',
        jobs: [
          { id: 8, kind: 'scan', state: 'failed', progressCurrent: 0, progressTotal: 0, message: '', errorMessage: 'disk error' },
          { id: 9, kind: 'scan', state: 'succeeded', progressCurrent: 1, progressTotal: 1, message: 'done', errorMessage: '' },
        ],
      },
    })
    const buttons = wrapper.findAll('button')
    expect(buttons).toHaveLength(1)
    await buttons[0].trigger('click')
    expect(wrapper.emitted('retry')).toEqual([[8]])
  })
})
