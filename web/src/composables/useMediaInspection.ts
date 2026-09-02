import { readonly, shallowRef, watch } from 'vue'
import * as api from '@/api/library'

export function useMediaInspection(itemId: () => number | null) {
  const inspection = shallowRef<api.MediaInspection | null>(null)
  const namingPreview = shallowRef<api.NamingPreview | null>(null)
  const isLoading = shallowRef(false)
  const isPreviewing = shallowRef(false)
  const error = shallowRef<string | null>(null)
  let generation = 0

  async function load(id: number, signal: AbortSignal, requestGeneration: number) {
    isLoading.value = true
    error.value = null
    try {
      const result = await api.mediaInspection(id, signal)
      if (requestGeneration === generation) inspection.value = result
    } catch (caught) {
      if (requestGeneration === generation && !(caught instanceof DOMException && caught.name === 'AbortError')) {
        inspection.value = null
        error.value = caught instanceof Error ? caught.message : 'Unable to inspect media'
      }
    } finally {
      if (requestGeneration === generation) isLoading.value = false
    }
  }

  async function previewNaming(pattern: string) {
    const id = itemId()
    if (!id) return
    const requestGeneration = generation
    isPreviewing.value = true
    error.value = null
    try {
      const result = await api.previewMediaNaming(id, pattern)
      if (requestGeneration === generation) namingPreview.value = result
    } catch (caught) {
      if (requestGeneration === generation) {
        namingPreview.value = null
        error.value = caught instanceof Error ? caught.message : 'Unable to preview naming pattern'
      }
    } finally {
      if (requestGeneration === generation) isPreviewing.value = false
    }
  }

  watch(itemId, (id, _, onCleanup) => {
    generation += 1
    const requestGeneration = generation
    const controller = new AbortController()
    onCleanup(() => controller.abort())
    namingPreview.value = null
    inspection.value = null
    error.value = null
    if (id) void load(id, controller.signal, requestGeneration)
  }, { immediate: true })

  return {
    inspection: readonly(inspection),
    namingPreview: readonly(namingPreview),
    isLoading: readonly(isLoading),
    isPreviewing: readonly(isPreviewing),
    error: readonly(error),
    previewNaming,
  }
}
