import { onUnmounted, shallowRef } from 'vue'

export type Theme = 'dark' | 'light' | 'system'

export function useTheme() {
  const stored = localStorage.getItem('mediagrap.theme')
  const theme = shallowRef<Theme>(stored === 'light' || stored === 'system' ? stored : 'dark')
  const media = window.matchMedia('(prefers-color-scheme: light)')

  function apply() {
    document.documentElement.dataset.theme = theme.value === 'system' ? (media.matches ? 'light' : 'dark') : theme.value
  }

  function setTheme(value: Theme) {
    theme.value = value
    localStorage.setItem('mediagrap.theme', value)
    apply()
  }

  const onSystemChange = () => { if (theme.value === 'system') apply() }
  media.addEventListener('change', onSystemChange)
  onUnmounted(() => media.removeEventListener('change', onSystemChange))
  apply()

  return { theme, setTheme }
}
