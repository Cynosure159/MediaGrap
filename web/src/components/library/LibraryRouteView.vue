<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter, type LocationQueryRaw } from 'vue-router'
import LibraryWorkspace from './LibraryWorkspace.vue'
import type { TVSelection } from '@/api/library'
import {
  movieRouteQuery,
  parseInspectorTab,
  parsePositiveQueryId,
  parseTVSelection,
  tvRouteQuery,
  type InspectorTab,
} from '@/router'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  mediaKind: 'movies' | 'shows'
  csrfToken: string
  username: string
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  toggleLocale: []
}>()

const route = useRoute()
const router = useRouter()
const routeName = computed(() => route.name === 'shows' ? 'shows' : 'movies')

const movieId = computed(() => routeName.value === 'movies' ? parsePositiveQueryId(route.query, 'movie') : null)
const tvSelection = computed(() => routeName.value === 'shows' ? parseTVSelection(route.query) : null)
const activeTab = computed(() => parseInspectorTab(route.query))

function canonicalQuery(): LocationQueryRaw {
  return routeName.value === 'movies'
    ? movieRouteQuery(movieId.value, activeTab.value)
    : tvRouteQuery(tvSelection.value, activeTab.value)
}

function canonicalizeQuery() {
  const target = router.resolve({ name: routeName.value, query: canonicalQuery() })
  if (target.fullPath !== route.fullPath) void router.replace(target)
}

watch(() => route.fullPath, canonicalizeQuery, { immediate: true })

function selectMovie(id: number) {
  void router.push({ name: 'movies', query: movieRouteQuery(id) })
}

function selectTV(selection: TVSelection) {
  void router.push({ name: 'shows', query: tvRouteQuery(selection) })
}

function selectTab(tab: InspectorTab) {
  if (routeName.value === 'movies' && movieId.value !== null) {
    void router.push({ name: 'movies', query: movieRouteQuery(movieId.value, tab) })
    return
  }
  if (routeName.value === 'shows' && tvSelection.value) {
    void router.push({ name: 'shows', query: tvRouteQuery(tvSelection.value, tab) })
  }
}

function closeSelection() {
  void router.push({ name: routeName.value, query: {} })
}
</script>

<template>
  <LibraryWorkspace
    :media-kind="props.mediaKind"
    :csrf-token="props.csrfToken"
    :username="props.username"
    :labels="props.labels"
    :selected-movie-id="movieId"
    :selected-tv-selection="tvSelection"
    :active-tab="activeTab"
    @select-movie="selectMovie"
    @select-tv-selection="selectTV"
    @select-tab="selectTab"
    @close-selection="closeSelection"
    @toggle-locale="emit('toggleLocale')"
  />
</template>
