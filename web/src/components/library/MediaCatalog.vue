<script setup lang="ts">
import { shallowRef } from 'vue'
import type { Job, MediaItem } from '@/api/library'

const props = defineProps<{ items: MediaItem[]; selectedId: number | null; activeJob: Job | undefined; labels: Record<string, string> }>()
const emit = defineEmits<{ search: [query: string]; select: [id: number] }>()
const query = shallowRef('')
function submitSearch() { emit('search', query.value) }
</script>

<template>
  <section class="media-catalog" aria-label="Movie list">
    <header class="catalog-header">
      <div>
        <p class="eyebrow">{{ labels.indexed }}</p>
        <strong class="catalog-count">{{ props.items.length }}</strong>
      </div>
      <form class="catalog-search" @submit.prevent="submitSearch">
        <input v-model="query" type="search" placeholder="Search" aria-label="Search library" />
        <button type="submit">{{ labels.search }}</button>
      </form>
    </header>
    <p v-if="props.activeJob" class="job-note">{{ props.activeJob.message }} · {{ props.activeJob.progressCurrent }}</p>
    <ol v-if="props.items.length" class="movie-list">
      <li v-for="item in props.items" :key="item.id">
        <button type="button" :class="{ selected: item.id === props.selectedId }" @click="emit('select', item.id)">
          <span class="film-mark">{{ item.titleHint.slice(0, 1) }}</span>
          <span class="movie-copy"><strong>{{ item.titleHint }}</strong><small>{{ item.relativePath }}</small></span>
          <span class="movie-year">{{ item.yearHint ?? '—' }}</span>
        </button>
      </li>
    </ol>
    <p v-else class="empty-state">{{ labels.noFilms }}</p>
  </section>
</template>

<style scoped>
.media-catalog { min-width: 0; border: 1px solid var(--mist-300); border-radius: .9rem; background: var(--paper); }
.catalog-header { display: grid; gap: .9rem; padding: 1.1rem; border-bottom: 1px solid var(--mist-200); }
.catalog-count { display: block; margin-top: .15rem; color: var(--ink-800); font: 500 2rem/1 var(--font-display); }
.catalog-search { display: flex; gap: .4rem; }
.catalog-search input { min-width: 0; width: 100%; min-height: 2.45rem; padding: .45rem .6rem; border: 1px solid var(--mist-300); border-radius: .4rem; }
.catalog-search button { min-height: 2.45rem; padding: .45rem .6rem; border: 1px solid var(--ink-700); border-radius: .4rem; background: var(--ink-900); color: white; font-weight: 700; cursor: pointer; }
.job-note { margin: .7rem 1rem; padding: .55rem .65rem; border-radius: .4rem; background: var(--water-100); color: var(--ink-800); font-size: .8rem; }
.movie-list { max-height: calc(100vh - 23rem); margin: 0; padding: .35rem; overflow: auto; list-style: none; }
.movie-list button { display: grid; grid-template-columns: 2.2rem minmax(0, 1fr) auto; width: 100%; gap: .7rem; align-items: center; padding: .65rem; border: 0; border-radius: .55rem; background: transparent; color: var(--ink-900); text-align: left; cursor: pointer; }
.movie-list button:hover { background: var(--mist-100); }.movie-list button.selected { background: var(--ink-900); color: var(--paper); }.movie-list button.selected small { color: var(--water-100); }
.film-mark { display: grid; width: 2.2rem; height: 2.2rem; place-items: center; border-radius: .4rem; background: var(--water-100); color: var(--ink-900); font: 1rem var(--font-display); }.selected .film-mark { background: var(--amber-400); }
.movie-copy { min-width: 0; }.movie-copy strong, .movie-copy small { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }.movie-copy small { margin-top: .2rem; color: var(--ink-600); font-size: .72rem; }.movie-year { font: .76rem var(--font-data); }
.empty-state { padding: 2rem 1rem; color: var(--ink-600); }.media-catalog:focus-within { border-color: var(--water-600); }
@media (max-width: 760px) { .movie-list { max-height: none; }.catalog-header { grid-template-columns: 1fr; } }
</style>
