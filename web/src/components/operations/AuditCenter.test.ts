import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AuditCenter from './AuditCenter.vue'

describe('audit disclosure', () => {
  it('provides a native button linked to recovery details', async () => {
    const wrapper = mount(AuditCenter, { attachTo: document.body, props: { labels: {}, entries: [{ id: 1, action: 'write_nfo', target: 'movie.nfo', detail: 'Written', outcome: 'applied', backup: '', recoverability: 'atomic_replace_only', createdAt: '' }] } })
    const button = wrapper.get('button[aria-controls]')
    const details = wrapper.get(`#${button.attributes('aria-controls')}`)
    expect(button.attributes('type')).toBe('button')
    expect(button.attributes('aria-expanded')).toBe('false')
    expect(details.isVisible()).toBe(false)
    await button.trigger('click')
    expect(button.attributes('aria-expanded')).toBe('true')
    expect(details.isVisible()).toBe(true)
    expect(details.text()).toContain('atomic_replace_only')
    await button.trigger('click')
    expect(button.attributes('aria-expanded')).toBe('false')
    expect(details.isVisible()).toBe(false)
    wrapper.unmount()
  })
})
