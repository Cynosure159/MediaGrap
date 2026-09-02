import { readonly, shallowRef, watch } from 'vue'
import * as api from '@/api/library'

export function useMediaInspection(itemId: () => number | null, csrfToken: () => string = () => '') {
  const inspection = shallowRef<api.MediaInspection | null>(null)
  const namingPreview = shallowRef<api.NamingPreview | null>(null)
  const renamePlan = shallowRef<api.RenamePlan | null>(null)
  const isLoading = shallowRef(false)
  const isPreviewing = shallowRef(false)
  const isApplying = shallowRef(false)
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

  async function previewRename(pattern: string) {
    const id = itemId()
    if (!id) return
    isPreviewing.value = true
    try {
      renamePlan.value = await api.previewMediaRename(csrfToken(), id, pattern)
    } catch (e: any) {
      error.value = e.message
    } finally {
      isPreviewing.value = false
    }
  }

  async function applyRename() {
    if (!renamePlan.value?.id || renamePlan.value.hasConflicts) return
    isApplying.value = true
    try {
      await api.applyRenamePlan(csrfToken(), renamePlan.value.id)
      // Plan was queued as a job — the job center will track execution
    } catch (e: any) {
      error.value = e.message
    } finally {
      isApplying.value = false
    }
  }

  watch(itemId, (id, _, onCleanup) => {
    generation += 1
    const requestGeneration = generation
    const controller = new AbortController()
    onCleanup(() => controller.abort())
    namingPreview.value = null
    renamePlan.value = null
    inspection.value = null
    error.value = null
    if (id) void load(id, controller.signal, requestGeneration)
  }, { immediate: true })

  return {
    inspection: readonly(inspection),
    namingPreview: readonly(namingPreview),
    renamePlan: readonly(renamePlan),
    isLoading: readonly(isLoading),
    isPreviewing: readonly(isPreviewing),
    isApplying: readonly(isApplying),
    error: readonly(error),
    previewNaming,
    previewRename,
    applyRename,
  }
}
