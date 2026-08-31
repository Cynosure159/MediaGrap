<script setup lang="ts">
import { computed, onMounted, shallowRef } from 'vue'
import { useLibrary } from '@/composables/useLibrary'
import MediaCatalog from './MediaCatalog.vue'
import MovieInspector from './MovieInspector.vue'
import TVShowCatalog from './TVShowCatalog.vue'
import TVShowInspector from './TVShowInspector.vue'

const props = defineProps<{ csrfToken: string; username: string; labels: Record<string, string> }>()
const emit = defineEmits<{ toggleLocale: [] }>()
const query = shallowRef('')
const selectedID = shallowRef<number | null>(null)
const mediaKind = shallowRef<'movies' | 'shows'>('movies')
const { sourceItems, mediaItems, tvShowItems, jobItems, error, hasSources, refresh, scan: queueScan } = useLibrary(() => props.csrfToken)
const activeJobs = computed(() => jobItems.value.filter(job => job.state === 'queued' || job.state === 'running'))

async function scan(id: number) { await queueScan(id) }
async function search(value = '') { query.value = value; await refresh(query.value) }
function selectKind(value: 'movies' | 'shows') { mediaKind.value = value; selectedID.value = null }

onMounted(refresh)
</script>

<template>
  <main class="library-workspace">
    <header class="library-header">
      <div><p class="eyebrow">MediaGrap · library desk</p><h1>{{ labels.library }}</h1><p>{{ props.username }}</p></div>
      <div class="header-actions"><button type="button" @click="emit('toggleLocale')">{{ labels.language }}</button><button type="button" @click="refresh(query)">{{ labels.refresh }}</button></div>
    </header>
    <p v-if="error" class="library-error">{{ error }}</p>
    <section v-if="!hasSources" class="source-onboarding"><p class="eyebrow">{{ labels.source }}</p><h2>{{ labels.firstSource }}</h2><p>{{ labels.openSettingsForSource }}</p></section>
    <template v-else>
      <section class="source-strip"><article v-for="source in sourceItems" :key="source.id"><p>{{ source.name }}</p><strong>{{ source.itemCount }} {{ labels.indexed }}</strong><small>{{ source.rootPath }}</small><button type="button" @click="scan(source.id)">{{ labels.scan }}</button></article></section>
      <nav class="media-tabs" :aria-label="labels.library"><button type="button" :class="{ active: mediaKind === 'movies' }" @click="selectKind('movies')">{{ labels.movies }}</button><button type="button" :class="{ active: mediaKind === 'shows' }" @click="selectKind('shows')">{{ labels.tvShows }}</button></nav>
      <section class="library-desk" :class="{ 'is-mobile-detail': selectedID !== null }">
        <template v-if="mediaKind === 'movies'"><MediaCatalog :items="mediaItems" :selected-id="selectedID" :active-job="activeJobs[0]" :labels="labels" @search="search" @select="selectedID = $event"/><MovieInspector :item-id="selectedID" :csrf-token="csrfToken" :labels="labels" @close="selectedID = null" /></template>
        <template v-else><TVShowCatalog :items="tvShowItems" :selected-id="selectedID" :active-job="activeJobs[0]" :labels="labels" @search="search" @select="selectedID = $event"/><TVShowInspector :show-id="selectedID" :labels="labels" @close="selectedID = null" /></template>
      </section>
    </template>
  </main>
</template>

<style scoped>
.library-workspace{max-width:88rem;margin:auto;padding:clamp(1.25rem,3vw,3rem)}.library-header{display:flex;justify-content:space-between;gap:1rem;align-items:end;padding-bottom:1.6rem;border-bottom:1px solid var(--mist-300)}.library-header h1{margin:.35rem 0;font:500 clamp(2.5rem,6vw,5rem)/.9 var(--font-display);letter-spacing:-.07em}.library-header p{color:var(--ink-600)}.header-actions{display:flex;gap:.5rem}.header-actions button,.source-strip button,.source-onboarding button{min-height:2.6rem;padding:.5rem .8rem;border:1px solid var(--ink-700);border-radius:.45rem;background:var(--ink-900);color:white;font-weight:700;cursor:pointer}.source-onboarding{margin-top:2rem;padding:clamp(1.2rem,3vw,2.4rem);border:1px solid var(--mist-300);border-radius:1rem;background:var(--paper)}.source-onboarding h2{margin:.35rem 0;font:500 2rem var(--font-display)}.source-strip{display:grid;grid-template-columns:repeat(auto-fit,minmax(15rem,1fr));gap:1rem;margin:1.2rem 0}.source-strip article{display:grid;gap:.45rem;padding:1rem;border-left:3px solid var(--amber-400);border-radius:.2rem .75rem .75rem .2rem;background:var(--paper)}.source-strip p,.source-strip small{margin:0;color:var(--ink-600)}.source-strip button{justify-self:start;margin-top:.3rem}.media-tabs{display:flex;gap:.4rem;margin-bottom:1rem}.media-tabs button{min-height:2.35rem;padding:.4rem .75rem;border:1px solid var(--mist-300);border-radius:.4rem;background:var(--paper);color:var(--ink-800);font-weight:700;cursor:pointer}.media-tabs button.active{border-color:var(--ink-900);background:var(--ink-900);color:var(--paper)}.library-desk{display:grid;grid-template-columns:minmax(18rem,25rem) minmax(0,1fr);gap:1rem;align-items:start}.library-error{padding:.7rem;border-radius:.4rem;background:var(--water-100);color:var(--ink-800)}@media(max-width:760px){.library-header{align-items:stretch;flex-direction:column}.header-actions button{flex:1}.library-desk{display:block}.library-desk.is-mobile-detail :deep(.media-catalog),.library-desk.is-mobile-detail :deep(.show-catalog){display:none}.library-desk:not(.is-mobile-detail) :deep(.inspector),.library-desk:not(.is-mobile-detail) :deep(.show-inspector){display:none}}
</style>
