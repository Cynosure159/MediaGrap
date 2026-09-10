import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsIntegrations from './SettingsIntegrations.vue'
import * as api from '@/api/integrations'
import type { Source } from '@/api/types'
vi.mock('@/api/integrations', () => ({ listWebhooks: vi.fn(), listTokens: vi.fn(), integrationStatus: vi.fn(), mutate: vi.fn(), deliveries: vi.fn() }))
const source = { id: 1, name: 'Movies' } as Source
beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listWebhooks).mockResolvedValue({ items: [] })
  vi.mocked(api.listTokens).mockResolvedValue({ items: [] })
  vi.mocked(api.integrationStatus).mockResolvedValue({ signingReady: true, paused: false, undispatched: 0, pendingDeliveries: 0, deadDeliveries: 0 })
})
describe('integrations settings', () => {
  it('requires explicit source selection and dismisses one-time secrets', async () => {
    vi.mocked(api.mutate).mockResolvedValue({ secret: 'one-time-secret' })
    const wrapper = mount(SettingsIntegrations, { props: { csrfToken: 'csrf', sources: [source], locale: 'en' } })
    await flushPromises()
    const form = wrapper.findAll('form')[1]!
    await form.find('input').setValue('Reader')
    expect(form.find('button').attributes('disabled')).toBeDefined()
    await form.find('input[type=checkbox]').setValue(true)
    await form.trigger('submit')
    await flushPromises()
    expect(api.mutate).toHaveBeenCalledWith('csrf', 'api-tokens', 'POST', expect.objectContaining({ name: 'Reader', sourceIds: [1], scopes: ['media:read', 'metadata:read', 'jobs:read'] }))
    expect(wrapper.text()).toContain('one-time-secret')
    await wrapper.findAll('button').find(button => button.text() === 'Saved, dismiss')!.trigger('click')
    expect(wrapper.text()).not.toContain('one-time-secret')
    wrapper.unmount()
  })
  it('blocks double submission and reports request failures', async () => {
    vi.mocked(api.mutate).mockRejectedValue(new Error('Invalid URL policy'))
    const wrapper = mount(SettingsIntegrations, { props: { csrfToken: 'csrf', sources: [source], locale: 'en' } })
    await flushPromises()
    const form = wrapper.findAll('form')[0]!
    await form.find('input').setValue('Notifications')
    await form.find('input[type=url]').setValue('https://example.com/hook')
    await form.find('input[type=checkbox]').setValue(true)
    await form.trigger('submit')
    await flushPromises()
    expect(wrapper.find('[role=alert]').text()).toContain('Invalid URL policy')
    wrapper.unmount()
  })
  it('revokes a token using the session CSRF credential', async () => {
    vi.mocked(api.listTokens).mockResolvedValue({ items: [{ id: 'tok_1', name: 'Reader', prefix: 'mgp_', scopes: ['media:read'], sourceIds: [1], expiresAt: Date.now() + 60000, revokedAt: null }] })
    vi.mocked(api.mutate).mockResolvedValue(undefined)
    const wrapper = mount(SettingsIntegrations, { props: { csrfToken: 'csrf', sources: [source], locale: 'en' } })
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'Revoke access')!.trigger('click')
    await flushPromises()
    expect(api.mutate).toHaveBeenCalledWith('csrf', 'api-tokens/tok_1', 'DELETE')
    wrapper.unmount()
  })
})
