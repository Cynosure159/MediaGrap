export interface SystemSummary {
  status: 'ready' | 'not_ready'
  phase: string
  library: {
    sources: number
    items: number
  }
  jobs: {
    queued: number
    running: number
    failed: number
  }
}

export async function getSystemSummary(signal?: AbortSignal): Promise<SystemSummary> {
  const response = await fetch('/api/v1/system/summary', { signal })
  if (!response.ok) {
    throw new Error(`System status request failed with ${response.status}`)
  }
  return response.json() as Promise<SystemSummary>
}

