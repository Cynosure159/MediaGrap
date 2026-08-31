<script setup lang="ts">
import type { Candidate } from '@/api/library'

defineProps<{ candidates: Candidate[]; busy: boolean; labels: Record<string, string> }>()
const emit = defineEmits<{
  select: [candidate: Candidate]
  close: []
}>()
</script>

<template>
  <section v-if="candidates.length || busy" class="candidates-panel card">
    <header class="candidates-header">
      <div class="header-info">
        <p class="eyebrow">{{ busy ? labels.tmdbSearch : labels.tmdbMatch }}</p>
        <h3 class="panel-title">{{ busy ? labels.searching : labels.matchCandidates }}</h3>
      </div>
      <div class="header-actions">
        <span v-if="busy" class="status-dot status-dot--warning" :title="labels.searching"></span>
        <span v-else-if="candidates.length" class="spec-badge">{{ candidates.length }} {{ labels.results }}</span>
        <button
          type="button"
          class="btn btn-ghost btn-icon btn-sm close-btn"
          :aria-label="labels.closeCandidates"
          @click="emit('close')"
        >
          ✕
        </button>
      </div>
    </header>

    <div v-if="busy" class="candidates-busy">
      <span class="status-dot status-dot--warning"></span>
      <span class="busy-text">{{ labels.searchingCandidates }}</span>
    </div>

    <ol v-else-if="candidates.length" class="candidates-list">
      <li
        v-for="candidate in candidates"
        :key="candidate.id"
        class="candidate-card"
        tabindex="0"
        @click="emit('select', candidate)"
        @keydown.enter="emit('select', candidate)"
        @keydown.space.prevent="emit('select', candidate)"
      >
        <div class="poster-frame">
          <img
            v-if="candidate.posterUrl"
            :src="candidate.posterUrl"
            :alt="candidate.title"
            class="poster-image"
            loading="lazy"
          />
          <div v-else class="poster-fallback">
            <span class="poster-icon">🎬</span>
          </div>
        </div>

        <div class="candidate-meta">
          <div class="title-row">
            <span class="candidate-title">{{ candidate.title }}</span>
            <span v-if="candidate.year" class="spec-badge">{{ candidate.year }}</span>
          </div>
          <p
            v-if="candidate.originalTitle && candidate.originalTitle !== candidate.title"
            class="candidate-original-title"
          >
            {{ candidate.originalTitle }}
          </p>
          <p v-if="candidate.overview" class="candidate-overview">
            {{ candidate.overview }}
          </p>
        </div>

        <div class="candidate-actions">
          <button
            type="button"
            class="btn btn-primary btn-sm select-btn"
            @click.stop="emit('select', candidate)"
          >
            {{ labels.select }}
          </button>
        </div>
      </li>
    </ol>

    <p v-else class="candidates-empty">
      {{ labels.noCandidates }}
    </p>
  </section>
</template>

<style scoped>
.candidates-panel {
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  overflow: hidden;
  margin: 0.75rem 0;
}

.candidates-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--surface-card);
}

.header-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.panel-title {
  margin: 0;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 0.95rem;
  color: var(--text-primary);
  letter-spacing: -0.01em;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.close-btn {
  color: var(--text-muted);
  font-size: 0.8rem;
  line-height: 1;
}

.close-btn:hover {
  color: var(--text-primary);
}

.candidates-busy {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 1.25rem 1rem;
  color: var(--text-muted);
  font-size: 0.8125rem;
}

.candidates-list {
  display: grid;
  gap: 0.5rem;
  margin: 0;
  padding: 0.75rem;
  list-style: none;
  max-height: 24rem;
  overflow-y: auto;
}

.candidate-card {
  display: grid;
  grid-template-columns: 3rem minmax(0, 1fr) auto;
  gap: 0.75rem;
  align-items: center;
  padding: 0.625rem 0.75rem;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
}

.candidate-card:hover {
  background: var(--surface-bright);
  border-color: var(--border-strong);
}

.candidate-card:focus-visible,
.candidate-card:active,
.candidate-card.selected {
  outline: none;
  border-color: var(--primary-bright);
  box-shadow: 0 0 0 2px var(--primary-bright), 0 0 12px rgba(128, 131, 255, 0.15);
  background: rgba(128, 131, 255, 0.05);
}

.poster-frame {
  width: 3rem;
  height: 4.25rem;
  flex-shrink: 0;
  border-radius: var(--radius-sm);
  overflow: hidden;
  border: 1px solid var(--border-subtle);
  background: var(--surface-dim);
}

.poster-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.poster-fallback {
  width: 100%;
  height: 100%;
  display: grid;
  place-items: center;
  background: var(--surface-dim);
  color: var(--text-disabled);
  font-size: 1rem;
}

.candidate-meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.candidate-title {
  color: var(--text-primary);
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 0.875rem;
  line-height: 1.25;
}

.candidate-original-title {
  margin: 0;
  color: var(--text-muted);
  font-size: 0.75rem;
  font-style: italic;
}

.candidate-overview {
  margin: 0;
  color: var(--text-muted);
  font-size: 0.775rem;
  line-height: 1.35;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-clamp: 2;
}

.candidate-actions {
  flex-shrink: 0;
}

.select-btn {
  font-size: 0.75rem;
  padding: 0.25rem 0.65rem;
}

.candidates-empty {
  margin: 0;
  padding: 1.5rem 1rem;
  color: var(--text-muted);
  font-size: 0.8125rem;
  text-align: center;
}

@media (max-width: 600px) {
  .candidate-card {
    grid-template-columns: 2.75rem minmax(0, 1fr);
    gap: 0.5rem;
  }
  .candidate-actions {
    grid-column: 1 / -1;
    display: flex;
    justify-content: flex-end;
  }
}
</style>
