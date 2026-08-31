<script setup lang="ts">
import { watch, shallowRef } from 'vue'
import * as api from '@/api/library'

const props = defineProps<{ showId: number | null; labels: Record<string, string> }>()
const emit = defineEmits<{ close: [] }>()
const detail = shallowRef<api.TVShowDetail | null>(null)
const error = shallowRef<string | null>(null)
const loading = shallowRef(false)

watch(() => props.showId, async (id) => {
  detail.value = null
  error.value = null
  if (id === null) return
  loading.value = true
  try { detail.value = await api.tvShowDetail(id) } catch (caught) { error.value = caught instanceof Error ? caught.message : 'Unable to load TV show' } finally { loading.value = false }
}, { immediate: true })
</script>

<template>
  <section class="show-inspector">
    <p v-if="!props.showId" class="inspector-empty">{{ props.labels.selectShow }}</p>
    <p v-else-if="loading" class="inspector-empty">{{ props.labels.loading }}</p>
    <p v-else-if="error" class="inspector-error">{{ error }}</p>
    <template v-else-if="detail">
      <header class="inspector-header"><div><p class="eyebrow">{{ props.labels.tvShows }}</p><h2>{{ detail.show.titleHint }}</h2><p>{{ detail.show.relativePath }}</p></div><button type="button" @click="emit('close')">{{ props.labels.cancel }}</button></header>
      <section class="show-facts"><span>{{ detail.show.seasonCount }} {{ props.labels.seasons }}</span><span>{{ detail.show.episodeCount }} {{ props.labels.episodes }}</span><span>{{ detail.writable ? props.labels.writable : props.labels.readOnly }}</span></section>
      <section class="episode-section"><h3>{{ props.labels.episodes }}</h3><ol class="episode-list"><li v-for="episode in detail.episodes" :key="episode.id"><strong>S{{ String(episode.seasonNumber).padStart(2, '0') }}E{{ String(episode.episodeStart).padStart(2, '0') }}<template v-if="episode.episodeEnd !== episode.episodeStart">–E{{ String(episode.episodeEnd).padStart(2, '0') }}</template></strong><span>{{ episode.titleHint || episode.relativePath }}</span></li></ol></section>
    </template>
  </section>
</template>

<style scoped>
.show-inspector { min-width:0; min-height:28rem; padding:clamp(1rem,3vw,2rem); border:1px solid var(--mist-300); border-radius:.9rem; background:var(--paper); }.inspector-empty,.inspector-error { margin:0; padding:2rem 0; color:var(--ink-600); }.inspector-error { color:#9b2c2c; }.inspector-header { display:flex; justify-content:space-between; gap:1rem; padding-bottom:1rem; border-bottom:1px solid var(--mist-200); }.inspector-header h2 { margin:.3rem 0; font:500 clamp(2rem,5vw,3.8rem)/.95 var(--font-display); letter-spacing:-.05em; }.inspector-header p { margin:0; color:var(--ink-600); overflow-wrap:anywhere; }.inspector-header button { align-self:start; min-height:2.4rem; padding:.45rem .7rem; border:1px solid var(--mist-300); border-radius:.4rem; background:transparent; cursor:pointer; }.show-facts { display:flex; flex-wrap:wrap; gap:.5rem; margin:1rem 0; }.show-facts span { padding:.4rem .6rem; border-radius:999px; background:var(--water-100); color:var(--ink-800); font-size:.8rem; }.episode-section h3 { margin:1.4rem 0 .6rem; font:500 1.25rem var(--font-display); }.episode-list { margin:0; padding:0; list-style:none; border-top:1px solid var(--mist-200); }.episode-list li { display:grid; grid-template-columns:8rem minmax(0,1fr); gap:.8rem; padding:.7rem .2rem; border-bottom:1px solid var(--mist-200); }.episode-list strong { color:var(--ink-800); font: .8rem var(--font-data); }.episode-list span { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; color:var(--ink-600); }
</style>
