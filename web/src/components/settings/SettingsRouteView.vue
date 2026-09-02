<script setup lang="ts">
import SettingsPage from './SettingsPage.vue'
import type { Locale } from '@/composables/useLocale'
import type { Theme } from '@/composables/useTheme'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  csrfToken: string
  labels: Record<string, string>
  locale: Locale
  theme: Theme
}>()

const emit = defineEmits<{
  changeLocale: [locale: Locale]
  changeTheme: [theme: Theme]
}>()
</script>

<template>
  <SettingsPage
    :csrf-token="props.csrfToken"
    :labels="props.labels"
    :locale="props.locale"
    :theme="props.theme"
    @change-locale="emit('changeLocale', $event)"
    @change-theme="emit('changeTheme', $event)"
  />
</template>
