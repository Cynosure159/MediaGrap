import { computed, onBeforeUnmount, shallowRef } from 'vue'
import { getSystemSummary, type SystemSummary } from '@/api/system'

export function useSystemSummary() {
  const summary = shallowRef<SystemSummary | null>(null)
  const isLoading = shallowRef(false)
  const error = shallowRef<string | null>(null)
  let controller: AbortController | undefined

  const connectionLabel = computed(() => {
    if (error.value) return 'Unavailable'
    return summary.value?.status === 'ready' ? 'Connected' : 'Checking'
  })

  async function refresh() {
    controller?.abort()
    controller = new AbortController()
    isLoading.value = true
    error.value = null
    try {
      summary.value = await getSystemSummary(controller.signal)
    } catch (caught) {
      if ((caught as DOMException).name !== 'AbortError') {
        error.value = 'The server status could not be loaded.'
      }
    } finally {
      isLoading.value = false
    }
  }

  onBeforeUnmount(() => controller?.abort())

  return { summary, isLoading, error, connectionLabel, refresh }
}

