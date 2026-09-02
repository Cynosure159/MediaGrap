import { reactive, shallowRef } from 'vue'
import * as api from '@/api/library'

export function useSettings(csrfToken: () => string) {
  const settings = shallowRef<api.Settings | null>(null)
  const error = shallowRef<string | null>(null)
  const isLoading = shallowRef(false)
  const providerForm = reactive<api.SettingsUpdate>({ tmdbApiKey: '', clearTmdbApiKey: false, fanartTvApiKey: '', clearFanartTvApiKey: false, tmdbLanguage: 'en-US', fallbackLanguage: 'en-US', outboundProxy: '', clearOutboundProxy: false, noProxy: '', clearNoProxy: false, theme: 'dark', locale: 'en' })
	const connectionTests = reactive<Record<string, api.ConnectionTest | null>>({ tmdb: null, fanart_tv: null, proxy: null })
	const testingTarget = shallowRef<string | null>(null)

  function resetSensitiveFields() {
    providerForm.tmdbApiKey = ''
    providerForm.fanartTvApiKey = ''
    providerForm.outboundProxy = ''
    providerForm.noProxy = ''
    providerForm.clearTmdbApiKey = false
    providerForm.clearFanartTvApiKey = false
    providerForm.clearOutboundProxy = false
    providerForm.clearNoProxy = false
  }

  async function load() {
    isLoading.value = true
    error.value = null
    try {
      settings.value = await api.settings()
      providerForm.tmdbLanguage = settings.value.tmdbLanguage
      providerForm.fallbackLanguage = settings.value.fallbackLanguage
      providerForm.theme = settings.value.theme
      providerForm.locale = settings.value.locale
      resetSensitiveFields()
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
      resetSensitiveFields()
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to save settings'
      throw caught
    }
  }

	async function runConnectionTest(target: api.ConnectionTest['target']) {
		testingTarget.value = target
		try { connectionTests[target] = await api.testConnection(csrfToken(), target) }
		catch (caught) { error.value = caught instanceof Error ? caught.message : 'Connection test failed'; throw caught }
		finally { testingTarget.value = null }
	}

  return { settings, error, isLoading, providerForm, connectionTests, testingTarget, load, saveProvider, runConnectionTest }
}
