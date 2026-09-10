import { mount, flushPromises } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import AutomationApprovals from './AutomationApprovals.vue'
import * as api from '@/api/integrations'
vi.mock('@/api/integrations', () => ({ automationPlans: vi.fn(), mutate: vi.fn() }))
describe('file approval', () => {
  it('requires review and sends only the exact displayed digest', async () => {
    const plan: api.AutomationPlan = { id: 'plan_one', version: 1, digest: 'frozen-digest', tokenId: 'token_one', sourceId: 1, mediaId: 2, kind: 'write', state: 'previewed', expiresAt: Date.now() + 60000, recoverability: 'atomic_replace_only', items: [{ kind: 'write', target: 'Film/movie.nfo', willReplace: true, state: 'pending', content: '<movie><title>Reviewed</title></movie>' }] }
    vi.mocked(api.automationPlans).mockResolvedValue({ items: [plan] })
    vi.mocked(api.mutate).mockResolvedValue(undefined)
    const wrapper = mount(AutomationApprovals, { props: { csrfToken: 'csrf', locale: 'en' } })
    await flushPromises()
    await wrapper.find('.plan-select').trigger('click')
    expect(wrapper.text()).toContain('Film/movie.nfo')
    expect(wrapper.text()).toContain('Replaces existing file')
    expect(wrapper.find('.btn-primary').attributes('disabled')).toBeDefined()
    expect(api.mutate).not.toHaveBeenCalled()
    await wrapper.find('input[type=checkbox]').setValue(true)
    await wrapper.find('.btn-primary').trigger('click')
    await flushPromises()
    expect(api.mutate).toHaveBeenCalledWith('csrf', 'automation/plans/plan_one/approve', 'POST', { version: 1, digest: 'frozen-digest' })
    wrapper.unmount()
  })
})
