import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { ConnectionTest, Settings } from '@/api/library'
import SettingsStatusSidebar from './SettingsStatusSidebar.vue'

describe('connection sidebar', () => {
  it.each(['reachable', 'failed', 'not_configured'] as const)('shows %s results for both configured connections', (status) => {
    const test = (target: ConnectionTest['target']): ConnectionTest => ({ target, status, durationMs: 42, message: '' })
    const wrapper = mount(SettingsStatusSidebar, { props: {
      settings: { tmdbApiKeyConfigured: true, outboundProxyConfigured: true } as Settings,
      sources: [], connectionTests: { tmdb: test('tmdb'), proxy: test('proxy') }, testingTarget: null,
      locale: 'en', labels: { connection_reachable: 'Reachable', connection_failed: 'Failed', connection_not_configured: 'Not configured' },
    } })
    const results = wrapper.findAll('[role="status"]')
    expect(results).toHaveLength(2)
    for (const result of results) {
      expect(result.text()).toBe(`${{ reachable: 'Reachable', failed: 'Failed', not_configured: 'Not configured' }[status]} · 42 ms`)
      expect(result.classes().includes('spec-pill--ok')).toBe(status === 'reachable')
    }
    wrapper.unmount()
  })
})
