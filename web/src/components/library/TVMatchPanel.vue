<script setup lang="ts">
import { onMounted, shallowRef } from 'vue'
import type { Candidate } from '@/api/types'
import * as api from '@/api/library'

const props = defineProps<{ showId: number; title: string; labels: Record<string, string> }>()
const emit = defineEmits<{ select: [candidate: Candidate]; close: [] }>()

const query = shallowRef(props.title)
const candidates = shallowRef<Candidate[]>([])
const isSearching = shallowRef(false)
const error = shallowRef<string | null>(null)

async function search() {
  isSearching.value = true
  error.value = null
  try {
    candidates.value = (await api.tvShowCandidates(props.showId, query.value.trim())).items
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : props.labels.errorSearchCandidates
  } finally {
    isSearching.value = false
  }
}

onMounted(search)
</script>

<template>
  <section class="match-panel">
    <header class="match-header">
      <div>
        <p class="eyebrow">TMDb</p>
        <h2 class="match-title">{{ labels.matchTvShow }}</h2>
      </div>
      <button class="btn btn-ghost btn-sm" type="button" @click="emit('close')">{{ labels.cancel }}</button>
    </header>
    <form class="match-search" @submit.prevent="search">
      <input v-model="query" class="match-input" :placeholder="labels.searchShows" />
      <button class="btn btn-primary btn-sm" :disabled="isSearching" type="submit">{{ isSearching ? labels.searching : labels.search }}</button>
    </form>
    <p v-if="error" class="match-error">{{ error }}</p>
    <p v-else-if="!candidates.length && !isSearching" class="match-hint">{{ labels.matchTvHint }}</p>
    <ol v-else class="match-list">
      <li v-for="candidate in candidates" :key="candidate.id" class="match-row">
        <img v-if="candidate.posterUrl" class="match-poster" :src="candidate.posterUrl" :alt="candidate.title" />
        <div v-else class="match-poster match-poster--empty">TV</div>
        <div class="match-copy">
          <strong>{{ candidate.title }}</strong>
          <span>{{ candidate.year ?? '—' }} · TMDb #{{ candidate.id }}</span>
          <p v-if="candidate.overview">{{ candidate.overview }}</p>
        </div>
        <button class="btn btn-primary btn-sm" type="button" @click="emit('select', candidate)">{{ labels.select }}</button>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.match-panel { margin: 1rem; padding: 1rem; border: 1px solid var(--border-default); border-radius: var(--radius-md); background: var(--surface-card); }
.match-header, .match-search, .match-row { display: flex; gap: .75rem; align-items: center; }
.match-header { justify-content: space-between; }
.match-title { margin: 0; font-size: 1rem; }
.eyebrow { margin: 0 0 .15rem; color: var(--text-muted); font-size: .75rem; }
.match-search { margin: 1rem 0; }
.match-input { flex: 1; min-width: 0; padding: .55rem .65rem; color: var(--text-primary); background: var(--surface-elevated); border: 1px solid var(--border-default); border-radius: var(--radius-sm); }
.match-list { margin: 0; padding: 0; list-style: none; max-height: 18rem; overflow: auto; }
.match-row { padding: .6rem 0; border-top: 1px solid var(--border-subtle); }
.match-poster { width: 2.4rem; height: 3.5rem; object-fit: cover; border-radius: var(--radius-sm); background: var(--surface-elevated); }
.match-poster--empty { display: grid; place-items: center; color: var(--text-muted); font-size: .7rem; }
.match-copy { flex: 1; min-width: 0; display: grid; gap: .15rem; }
.match-copy span, .match-copy p, .match-hint, .match-error { margin: 0; font-size: .78rem; color: var(--text-muted); }
.match-copy p { overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; }
.match-error { color: var(--danger, #ff8b8b); }
@media (max-width: 700px) { .match-row { align-items: flex-start; } .match-row .btn { flex-shrink: 0; } }
</style>
