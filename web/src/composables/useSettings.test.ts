import { describe, expect, it, vi } from 'vitest'
import * as api from '@/api/library'
import { useSettings } from './useSettings'

vi.mock('@/api/library', () => ({
  settings: vi.fn(),
  saveSettings: vi.fn(),
  testConnection: vi.fn(),
}))

const view: api.Settings = {
  tmdbApiKeyConfigured: true,
  fanartTvApiKeyConfigured: false,
  outboundProxyConfigured: true,
  noProxyConfigured: true,
  tmdbLanguage: 'zh-CN',
  fallbackLanguage: 'en-US',
  theme: 'system',
  locale: 'zh-CN',
  mediaRoots: ['/media'],
}

describe('useSettings', () => {
  it('hydrates redacted settings and per-user preferences', async () => {
    vi.mocked(api.settings).mockResolvedValueOnce(view)
    const state = useSettings(() => 'csrf')
    await state.load()
    expect(state.providerForm.fallbackLanguage).toBe('en-US')
    expect(state.providerForm.theme).toBe('system')
    expect(state.providerForm.outboundProxy).toBe('')
  })

  it('runs a fixed-target connection test', async () => {
    vi.mocked(api.testConnection).mockResolvedValueOnce({ target: 'proxy', status: 'reachable', durationMs: 25, message: 'Connection succeeded' })
    const state = useSettings(() => 'csrf')
    await state.runConnectionTest('proxy')
    expect(api.testConnection).toHaveBeenCalledWith('csrf', 'proxy')
    expect(state.connectionTests.proxy?.status).toBe('reachable')
  })
})
