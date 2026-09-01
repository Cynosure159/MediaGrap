<script setup lang="ts">
import { computed } from 'vue'
import type { MovieDraft } from './MovieOverviewTab.vue'

const props = defineProps<{
  draft: MovieDraft
  isEditing: boolean
  labels: Record<string, string>
}>()

const genresInput = computed({
  get: () => props.draft.genres.join(', '),
  set: (val: string) => {
    props.draft.genres = val.split(',').map(s => s.trim()).filter(Boolean)
  },
})
</script>

<template>
  <div class="meta-blocks-grid">
    <!-- Row 1, Col 1: Release Date -->
    <div class="meta-card">
      <span class="card-label-caps">{{ labels.releaseDate || '上映日期' }}</span>
      <div v-if="!isEditing" class="meta-val font-code">
        {{ draft.year || '—' }}
      </div>
      <div v-else class="meta-input-wrap">
        <input v-model="draft.year" type="number" class="card-input font-code" :placeholder="labels.yearPlaceholder || 'YYYY'" />
        <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14" class="meta-field-icon">
          <path d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM7 10h5v5H7z"/>
        </svg>
      </div>
    </div>

    <!-- Row 1, Col 2: Genres (Pill Badges) -->
    <div class="meta-card">
      <span class="card-label-caps">{{ labels.genres || '类型' }}</span>
      <div v-if="!isEditing" class="genres-pills-list">
        <span v-for="g in draft.genres" :key="g" class="genre-tag-pill">
          {{ g }}
        </span>
        <span v-if="draft.genres.length === 0" class="genre-empty-hint">
          {{ labels.noGenres || '暂无分类' }}
        </span>
      </div>
      <input
        v-else
        v-model="genresInput"
        type="text"
        class="card-input"
        :placeholder="labels.genresPlaceholder || '剧情, 动作, 历史'"
      />
    </div>

    <!-- Row 2, Col 1: Director -->
    <div class="meta-card">
      <span class="card-label-caps">{{ labels.director || '导演' }}</span>
      <div v-if="!isEditing" class="meta-val meta-val-highlight">
        {{ draft.director || '—' }}
      </div>
      <input v-else v-model="draft.director" type="text" class="card-input" :placeholder="labels.directorPlaceholder || '导演'" />
    </div>

    <!-- Row 2, Col 2: Writers -->
    <div class="meta-card">
      <span class="card-label-caps">{{ labels.writers || '编剧' }}</span>
      <div v-if="!isEditing" class="meta-val" :title="draft.writers">
        {{ draft.writers || '—' }}
      </div>
      <input v-else v-model="draft.writers" type="text" class="card-input" :placeholder="labels.writerPlaceholder || '编剧、剧本'" />
    </div>

    <!-- Row 3: Studio / Production (Full Width across 2 columns) -->
    <div class="meta-card meta-card-full">
      <span class="card-label-caps">{{ labels.studio || '制作公司' }}</span>
      <div v-if="!isEditing" class="meta-val" :title="draft.studio">
        {{ draft.studio || '—' }}
      </div>
      <input v-else v-model="draft.studio" type="text" class="card-input" :placeholder="labels.studioPlaceholder || '制作公司'" />
    </div>
  </div>
</template>

<style scoped>
.meta-blocks-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
}

.meta-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-height: 58px;
  justify-content: center;
  min-width: 0;
  max-width: 100%;
  box-sizing: border-box;
  overflow: hidden;
}

.meta-card-full {
  grid-column: 1 / -1;
  min-width: 0;
  max-width: 100%;
}

.card-label-caps {
  font-size: 10px;
  font-weight: 700;
  color: var(--primary, #c0c1ff);
  letter-spacing: 0.05em;
  line-height: 1;
}

.meta-val {
  font-size: 13px;
  font-weight: 500;
  color: var(--on-surface, #dce1fb);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta-val-highlight {
  font-weight: 600;
  color: var(--primary, #c0c1ff);
}

.meta-input-wrap {
  position: relative;
  display: flex;
  align-items: center;
}

.meta-field-icon {
  position: absolute;
  right: 8px;
  color: var(--outline, #908fa0);
  pointer-events: none;
}

.card-input {
  width: 100%;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  font-size: 12px;
  padding: 4px 8px;
  outline: none;
  transition: border-color 0.15s ease;
}

.card-input:focus {
  border-color: var(--primary, #c0c1ff);
}

.genres-pills-list {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.genre-tag-pill {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  color: var(--on-surface, #dce1fb);
}

.genre-empty-hint {
  font-size: 12px;
  color: var(--outline, #908fa0);
}

@media (max-width: 480px) {
  .meta-blocks-grid {
    grid-template-columns: 1fr;
    gap: 8px;
  }
}
</style>
