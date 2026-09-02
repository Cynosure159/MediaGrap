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

  async function refreshSources() {
    try {
      const res = await api.sources()
      sourceItems.value = res.items
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to load sources'
    }
  }

  async function refreshMedia(query = '') {
    try {
      const res = await api.media(query)
      mediaItems.value = res.items
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to load media items'
    }
  }

  async function refreshTVShows(query = '') {
    try {
      const res = await api.tvShows(query)
      tvShowItems.value = res.items
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to load TV shows'
    }
  }

  async function refreshJobs() {
    try {
      const res = await api.jobs()
      jobItems.value = res.items
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to load jobs'
    }
  }

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
    await refreshSources()
  }

  async function removeSource(id: number) {
    await api.deleteSource(csrfToken(), id)
    await refreshSources()
  }

  async function saveSourcePolicy(id: number, policy: Pick<api.Source, 'scanMode' | 'scheduleEnabled' | 'scheduleIntervalMinutes'>) {
	await api.updateSourcePolicy(csrfToken(), id, policy)
	await refreshSources()
  }

  async function scan(id: number, query = '') {
    const job = await api.scanSource(csrfToken(), id)
    await refreshJobs()

    // Scans run in the server worker. Poll job status efficiently.
    for (let attempt = 0; attempt < 120; attempt += 1) {
      await new Promise(resolve => window.setTimeout(resolve, 500))
      await refreshJobs()
      const currentJob = jobItems.value.find(item => item.id === job.id)
      if (currentJob && ['succeeded', 'failed', 'cancelled'].includes(currentJob.state)) {
        await Promise.all([refreshMedia(query), refreshTVShows(query), refreshSources()])
        return
      }
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
    refreshSources,
    refreshMedia,
    refreshTVShows,
    refreshJobs,
    createSource,
    removeSource,
	saveSourcePolicy,
    scan,
  }
}
