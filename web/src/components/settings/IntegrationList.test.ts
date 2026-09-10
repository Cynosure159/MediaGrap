import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import IntegrationList from './IntegrationList.vue'

afterEach(() => { vi.useRealTimers() })
describe('token status', () => {
  it.each([false, true])('distinguishes expiry and revocation with zh=%s', async (zh) => {
    vi.useFakeTimers()
    vi.setSystemTime(10000)
    const token = { id: '1', name: 'Reader', prefix: 'mgp_', scopes: [], sourceIds: [], expiresAt: 10000, revokedAt: null }
    const wrapper = mount(IntegrationList, { props: { webhooks: [], tokens: [token], busy: false, zh } })
    expect(wrapper.find('.spec-pill').text()).toBe(zh ? '已过期' : 'Expired')
    await wrapper.setProps({ tokens: [{ ...token, expiresAt: 11000 }] })
    expect(wrapper.find('.spec-pill').text()).toBe(zh ? '有效' : 'Active')
    await vi.advanceTimersByTimeAsync(1000)
    expect(wrapper.find('.spec-pill').text()).toBe(zh ? '已过期' : 'Expired')
    await wrapper.setProps({ tokens: [{ ...token, revokedAt: 9000 }] })
    expect(wrapper.find('.spec-pill').text()).toBe(zh ? '已撤销' : 'Revoked')
    wrapper.unmount()
    expect(vi.getTimerCount()).toBe(0)
  })
})
