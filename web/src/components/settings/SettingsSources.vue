<script setup lang="ts">
import type { Source } from '@/api/library'

defineProps<{ sources: Source[]; mediaRoots: string[]; labels: Record<string, string> }>()
const sourceName = defineModel<string>('sourceName', { required: true })
const sourcePath = defineModel<string>('sourcePath', { required: true })
const emit = defineEmits<{ add: []; scan: [id: number] }>()
</script>

<template>
  <section class="settings-card">
    <div class="section-heading"><p class="eyebrow">Library</p><h2>{{ labels.directorySettings }}</h2><p>{{ labels.directoryHelp }}</p></div>
    <form class="source-form" @submit.prevent="emit('add')"><label>{{ labels.source }}<input v-model="sourceName" required /></label><label>{{ labels.containerPath }}<input v-model="sourcePath" list="media-roots" required /></label><datalist id="media-roots"><option v-for="root in mediaRoots" :key="root" :value="root" /></datalist><button type="submit">{{ labels.addSource }}</button></form>
    <p v-if="sources.length === 0" class="empty-sources">{{ labels.noSources }}</p>
    <div v-else class="source-list"><article v-for="source in sources" :key="source.id"><div><strong>{{ source.name }}</strong><p>{{ source.rootPath }}</p></div><small>{{ source.itemCount }} {{ labels.indexed }}</small><button type="button" @click="emit('scan', source.id)">{{ labels.scan }}</button></article></div>
  </section>
</template>

<style scoped>
.settings-card{padding:clamp(1.2rem,3vw,2rem);border:1px solid var(--mist-300);border-radius:1rem;background:var(--paper)}.section-heading h2{margin:.35rem 0;font:500 2rem var(--font-display);letter-spacing:-.04em}.section-heading p{color:var(--ink-600)}.source-form{display:grid;grid-template-columns:1fr 1.3fr auto;gap:.7rem;align-items:end;margin-top:1.2rem}.source-form label{display:grid;gap:.35rem;font-size:.82rem;font-weight:700}.source-form input{min-height:2.6rem;padding:.5rem .7rem;border:1px solid var(--mist-300);border-radius:.4rem;background:white;font:inherit}.source-form button,.source-list button{min-height:2.6rem;padding:.5rem .8rem;border:1px solid var(--ink-700);border-radius:.45rem;background:var(--ink-900);color:#fff;font-weight:700;cursor:pointer}.empty-sources{color:var(--ink-600)}.source-list{display:grid;gap:.6rem;margin-top:1.2rem}.source-list article{display:grid;grid-template-columns:1fr auto auto;gap:1rem;align-items:center;padding:.85rem;border-top:1px solid var(--mist-300)}.source-list p{margin:.2rem 0 0;color:var(--ink-600);font: .75rem var(--font-data)}@media(max-width:700px){.source-form{grid-template-columns:1fr}.source-form button{justify-self:start}.source-list article{grid-template-columns:1fr auto}.source-list small{grid-column:1}.source-list button{grid-column:2;grid-row:1 / span 2}}
</style>
