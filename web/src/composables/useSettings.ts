import { reactive, shallowRef } from 'vue'
import * as api from '@/api/library'

export function useSettings(csrfToken: () => string) {
  const settings = shallowRef<api.Settings | null>(null)
  const error = shallowRef<string | null>(null)
  const isLoading = shallowRef(false)
  const providerForm = reactive<api.SettingsUpdate>({ tmdbApiKey: '', clearTmdbApiKey: false, tmdbLanguage: 'en-US', outboundProxy: '', clearOutboundProxy: false })

  async function load() {
    isLoading.value = true
    error.value = null
    try {
      settings.value = await api.settings()
      providerForm.tmdbLanguage = settings.value.tmdbLanguage
      providerForm.tmdbApiKey = ''
      providerForm.outboundProxy = ''
      providerForm.clearTmdbApiKey = false
      providerForm.clearOutboundProxy = false
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to load settings'
    } finally {
      isLoading.value = false
    }
  }

  async function saveProvider() {
    error.value = null
    try {
      settings.value = await api.saveSettings(csrfToken(), { ...providerForm })
      providerForm.tmdbApiKey = ''
      providerForm.outboundProxy = ''
      providerForm.clearTmdbApiKey = false
      providerForm.clearOutboundProxy = false
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to save settings'
      throw caught
    }
  }

  return { settings, error, isLoading, providerForm, load, saveProvider }
}
