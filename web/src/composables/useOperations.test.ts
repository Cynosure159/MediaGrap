import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useOperations } from './useOperations'
import * as api from '@/api/library'

vi.mock('@/api/library', () => ({
  jobs: vi.fn(), auditEntries: vi.fn(), operationsStatus: vi.fn(), cancelJob: vi.fn(), retryJob: vi.fn(),
}))

describe('useOperations', () => {
  beforeEach(() => vi.clearAllMocks())

  it('loads real operation datasets together', async () => {
    vi.mocked(api.jobs).mockResolvedValue({ items: [] })
    vi.mocked(api.auditEntries).mockResolvedValue({ items: [] })
    vi.mocked(api.operationsStatus).mockResolvedValue({
      application: { name: 'MediaGrap', version: 'v1', commit: 'abc', builtAt: '' },
      database: { ready: true, latestMigration: '0012', sizeBytes: 1, walMode: true },
      cache: { path: 'cache', available: true, writable: true, usedBytes: 0 },
      mounts: [], providers: [], network: { proxyConfigured: false },
    })
    const operations = useOperations(() => 'csrf')
    await operations.refreshAll()
    expect(operations.status.value?.database.latestMigration).toBe('0012')
    expect(operations.error.value).toBeNull()
  })

  it('refreshes jobs after retry', async () => {
    vi.mocked(api.retryJob).mockResolvedValue({ id: 4, state: 'queued', progressCurrent: 0, message: '', errorMessage: '' })
    vi.mocked(api.jobs).mockResolvedValue({ items: [{ id: 4, state: 'queued', progressCurrent: 0, message: '', errorMessage: '' }] })
    const operations = useOperations(() => 'csrf')
    await operations.retry(4)
    expect(api.retryJob).toHaveBeenCalledWith('csrf', 4)
    expect(operations.jobs.value[0].state).toBe('queued')
  })
})
