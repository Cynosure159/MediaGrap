import { computed, shallowRef } from 'vue'
import * as api from '@/api/library'

export function useLibrary(csrfToken: () => string) {
  const sourceItems = shallowRef<api.Source[]>([])
  const mediaItems = shallowRef<api.MediaItem[]>([])
  const tvShowItems = shallowRef<api.TVShow[]>([])
  const jobItems = shallowRef<api.Job[]>([])
  const error = shallowRef<string | null>(null)
  const isLoading = shallowRef(false)

  const hasSources = computed(() => sourceItems.value.length > 0)

  async function refresh(query = '') {
    isLoading.value = true
    error.value = null
    try {
      const [sourceResult, mediaResult, tvResult, jobResult] = await Promise.all([
        api.sources(),
        api.media(query),
        api.tvShows(query),
        api.jobs(),
      ])
      sourceItems.value = sourceResult.items
      mediaItems.value = mediaResult.items
      tvShowItems.value = tvResult.items
      jobItems.value = jobResult.items
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to load the library'
    } finally {
      isLoading.value = false
    }
  }

  async function createSource(name: string, rootPath: string) {
    await api.addSource(csrfToken(), name, rootPath)
    await refresh()
  }

  async function removeSource(id: number) {
    await api.deleteSource(csrfToken(), id)
    await refresh()
  }

  async function scan(id: number, query = '') {
    const job = await api.scanSource(csrfToken(), id)
    await refresh(query)

    // Scans run in the server worker. Keep the catalog in sync when this
    // particular scan reaches a terminal state instead of leaving stale rows
    // visible until the user manually reloads the page.
    for (let attempt = 0; attempt < 120; attempt += 1) {
      await new Promise(resolve => window.setTimeout(resolve, 500))
      await refresh(query)
      const currentJob = jobItems.value.find(item => item.id === job.id)
      if (currentJob && ['succeeded', 'failed', 'cancelled'].includes(currentJob.state)) return
    }
  }

  return {
    sourceItems,
    mediaItems,
    tvShowItems,
    jobItems,
    error,
    isLoading,
    hasSources,
    refresh,
    createSource,
    removeSource,
    scan,
  }
}
