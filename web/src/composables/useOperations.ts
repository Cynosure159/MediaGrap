import { shallowRef } from 'vue'
import * as api from '@/api/library'

export function useOperations(csrfToken: () => string) {
  const jobs = shallowRef<api.Job[]>([])
  const audits = shallowRef<api.AuditEntry[]>([])
  const status = shallowRef<api.OperationsStatus | null>(null)
  const error = shallowRef<string | null>(null)
  const isLoading = shallowRef(false)
  const streamState = shallowRef<'connecting' | 'connected' | 'reconnecting'>('connecting')
	const testingTarget = shallowRef<string | null>(null)
  let events: EventSource | null = null
  let refreshTimer: number | null = null

  async function refreshJobs() {
    jobs.value = (await api.jobs()).items
  }

  async function refreshAll() {
    isLoading.value = true
    error.value = null
    try {
      const [jobResult, auditResult, statusResult] = await Promise.all([
        api.jobs(), api.auditEntries(), api.operationsStatus(),
      ])
      jobs.value = jobResult.items
      audits.value = auditResult.items
      status.value = statusResult
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to load operations data'
    } finally {
      isLoading.value = false
    }
  }

  function connect() {
    disconnect()
    streamState.value = 'connecting'
    events = new EventSource('/api/v1/jobs/events')
    events.onopen = () => { streamState.value = 'connected' }
    events.onerror = () => { streamState.value = 'reconnecting' }
    events.addEventListener('job', () => { void refreshJobs() })
    refreshTimer = window.setInterval(() => { void refreshJobs() }, 15_000)
  }

  function disconnect() {
    events?.close()
    events = null
    if (refreshTimer !== null) window.clearInterval(refreshTimer)
    refreshTimer = null
  }

  async function cancel(id: number) {
    await api.cancelJob(csrfToken(), id)
    await refreshJobs()
  }

  async function retry(id: number) {
    await api.retryJob(csrfToken(), id)
    await refreshJobs()
  }

	async function testConnection(target: api.ConnectionTest['target']) {
		testingTarget.value = target
		try { await api.testConnection(csrfToken(), target); status.value = await api.operationsStatus() }
		finally { testingTarget.value = null }
	}

  return { jobs, audits, status, error, isLoading, streamState, testingTarget, refreshAll, refreshJobs, connect, disconnect, cancel, retry, testConnection }
}
