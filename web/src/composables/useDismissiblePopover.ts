import { onMounted, onUnmounted, shallowRef } from 'vue'

export function useDismissiblePopover() {
  const open = shallowRef(false)
  const container = shallowRef<HTMLElement | null>(null)
  function closeOutside(event: MouseEvent) {
    if (container.value && !container.value.contains(event.target as Node)) open.value = false
  }
  onMounted(() => document.addEventListener('click', closeOutside))
  onUnmounted(() => document.removeEventListener('click', closeOutside))
  return { open, container }
}
