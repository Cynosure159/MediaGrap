<script setup lang="ts">
import { onMounted, shallowRef } from 'vue'
import * as api from '@/api/library'
import AuthPanel from '@/components/auth/AuthPanel.vue'
import LibraryWorkspace from '@/components/library/LibraryWorkspace.vue'
import { useLocale } from '@/composables/useLocale'
const mode = shallowRef<'loading' | 'setup' | 'login' | 'library'>('loading')
const session = shallowRef<api.Session | null>(null)
const error = shallowRef<string | null>(null)
const { t, toggleLocale } = useLocale()
async function initialize() { try { const status = await api.setupStatus(); if (status.needsSetup) { mode.value = 'setup'; return }; session.value = await api.session(); mode.value = 'library' } catch { mode.value = 'login' } }
async function authenticate(username: string, password: string) { error.value = null; try { session.value = mode.value === 'setup' ? await api.setup(username, password) : await api.signIn(username, password); mode.value = 'library' } catch (caught) { error.value = caught instanceof Error ? caught.message : 'Unable to sign in' } }
onMounted(initialize)
</script>
<template><div v-if="mode === 'loading'" class="loading-screen">{{ t.opening }}</div><AuthPanel v-else-if="mode === 'setup' || mode === 'login'" :setup="mode === 'setup'" :error="error" :labels="t" @submit="authenticate" @toggle-locale="toggleLocale" /><LibraryWorkspace v-else-if="session" :csrf-token="session.csrfToken" :username="session.user.username" :labels="t" @toggle-locale="toggleLocale" /></template>
<style scoped>.loading-screen{display:grid;min-height:100vh;place-items:center;background:var(--ink-900);color:var(--mist-100);font:1.2rem var(--font-display)}</style>
