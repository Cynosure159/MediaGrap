<script setup lang="ts">
import type { Candidate } from '@/api/library'
defineProps<{ candidates: Candidate[]; busy: boolean }>()
const emit = defineEmits<{ select: [candidate: Candidate] }>()
</script>
<template><section class="candidates"><p class="eyebrow">{{ busy ? 'TMDb…' : 'TMDb' }}</p><ol v-if="candidates.length"><li v-for="candidate in candidates" :key="candidate.id"><button type="button" @click="emit('select', candidate)"><img v-if="candidate.posterUrl" :src="candidate.posterUrl" alt="" /><span><strong>{{ candidate.title }}</strong><small>{{ candidate.year ?? '—' }} · {{ candidate.overview }}</small></span></button></li></ol><p v-else class="empty">No results yet.</p></section></template>
<style scoped>.candidates{min-width:0}.candidates ol{display:grid;gap:.45rem;padding:0;list-style:none}.candidates button{display:grid;grid-template-columns:2.5rem 1fr;width:100%;gap:.7rem;padding:.45rem;border:1px solid var(--mist-300);border-radius:.55rem;background:var(--paper);color:inherit;text-align:left;cursor:pointer}.candidates button:hover{border-color:var(--water-600)}img{width:2.5rem;height:3.5rem;object-fit:cover;background:var(--mist-200)}small{display:block;margin-top:.25rem;color:var(--ink-600);line-height:1.35}.empty{color:var(--ink-600)}</style>
