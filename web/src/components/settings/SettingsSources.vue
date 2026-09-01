<script setup lang="ts">
import type { Source } from '@/api/library'

defineProps<{
  sources: Source[]
  mediaRoots: string[]
  labels: Record<string, string>
  feedback: { kind: 'success' | 'error'; message: string } | null
}>()
const sourceName = defineModel<string>('sourceName', { required: true })
const sourcePath = defineModel<string>('sourcePath', { required: true })
const emit = defineEmits<{ add: []; scan: [id: number]; delete: [id: number] }>()
</script>

<template>
  <section class="settings-card">
    <div class="card-header">
      <p class="eyebrow">{{ labels.libraryEyebrow }}</p>
      <h2 class="section-title">{{ labels.directorySettings }}</h2>
      <p class="section-hint">{{ labels.directoryHelp }}</p>
    </div>

    <div class="card-body">
      <!-- Add Source Form -->
      <form class="source-form" @submit.prevent="emit('add')">
        <div class="field-group">
          <label for="new-source-name" class="field-label">{{ labels.source }}</label>
          <input
            id="new-source-name"
            v-model="sourceName"
            class="input-compact"
            required
            :placeholder="labels.sourceNamePlaceholder"
          />
        </div>
        <div class="field-group">
          <label for="new-source-path" class="field-label">{{ labels.containerPath }}</label>
          <input
            id="new-source-path"
            v-model="sourcePath"
            class="input-compact"
            list="media-roots"
            required
            :placeholder="labels.sourcePathPlaceholder"
          />
          <datalist id="media-roots">
            <option v-for="root in mediaRoots" :key="root" :value="root" />
          </datalist>
        </div>
        <div class="form-action">
          <button type="submit" class="btn btn-outline">
            {{ labels.addSource }}
          </button>
        </div>
      </form>
      <p v-if="feedback" class="source-feedback" :class="`source-feedback--${feedback.kind}`" role="status">
        {{ feedback.message }}
      </p>

      <!-- Empty State -->
      <p v-if="sources.length === 0" class="empty-sources">
        {{ labels.noSources }}
      </p>

      <!-- Source List -->
      <div v-else class="source-list">
        <article v-for="source in sources" :key="source.id" class="source-item">
          <div class="source-item__main">
            <div class="source-item__header">
              <span class="source-name">{{ source.name }}</span>
              <span
                class="writable-indicator"
                :title="source.writable ? labels.writable : labels.readOnly"
              >
                <span
                  class="status-dot"
                  :class="source.writable ? 'status-dot--ok' : 'status-dot--warning'"
                ></span>
                <span class="writable-text">{{ source.writable ? labels.writable : labels.readOnly }}</span>
              </span>
            </div>
            <p class="source-path">{{ source.rootPath }}</p>
          </div>

          <div class="source-item__meta">
            <span class="spec-badge">{{ source.itemCount }} {{ labels.indexed }}</span>
          </div>

          <div class="source-item__actions">
            <button
              type="button"
              class="btn btn-outline btn-sm"
              @click="emit('scan', source.id)"
            >
              {{ labels.scan }}
            </button>
            <button
              type="button"
              class="btn btn-danger btn-sm"
              @click="emit('delete', source.id)"
            >
              {{ labels.deleteSource }}
            </button>
          </div>
        </article>
      </div>
    </div>
  </section>
</template>

<style scoped>
.settings-card {
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  overflow: hidden;
}

.card-header {
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--surface-card);
}

.section-title {
  margin: 0.25rem 0 0.35rem;
  color: var(--text-primary);
  font-family: var(--font-display);
  font-size: 0.9375rem;
  font-weight: 600;
  line-height: 1.3;
  letter-spacing: -0.01em;
}

.section-hint {
  margin: 0;
  color: var(--text-muted);
  font-size: 0.75rem;
  line-height: 1.4;
}

.card-body {
  padding: 1.25rem;
  display: grid;
  gap: 1.25rem;
}

.source-form {
  display: grid;
  grid-template-columns: 1fr 1.4fr auto;
  gap: 0.75rem;
  align-items: flex-end;
  padding: 1rem;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
}

.field-group {
  display: grid;
  gap: 0.35rem;
}

.field-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.input-compact {
  width: 100%;
  min-height: 2.125rem;
  padding: 0.35rem 0.65rem;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-panel);
  color: var(--text-primary);
  font-size: 0.8125rem;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.input-compact:focus {
  outline: none;
  border-color: var(--border-focus);
  box-shadow: 0 0 0 2px rgba(128, 131, 255, 0.2);
}

.form-action {
  display: flex;
  align-items: flex-end;
}

.source-feedback {
  margin: -0.5rem 0 0;
  padding: 0.625rem 0.75rem;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  font-size: 0.8125rem;
}

.source-feedback--success {
  border-color: var(--secondary-container, #00a572);
  color: var(--secondary, #4edea3);
}

.source-feedback--error {
  border-color: var(--error-bright, #f43f5e);
  color: var(--error, #ffb4ab);
}

.empty-sources {
  margin: 0;
  padding: 1.5rem;
  text-align: center;
  color: var(--text-muted);
  font-size: 0.8125rem;
  background: var(--surface-panel);
  border-radius: var(--radius-md);
  border: 1px dashed var(--border-default);
}

.source-list {
  display: grid;
  gap: 0.625rem;
}

.source-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.875rem 1rem;
  background: var(--surface-panel);
  border: 1px solid var(--border-subtle);
  border-left: 3px solid var(--warning);
  border-radius: var(--radius-md);
  transition: background 0.15s, border-color 0.15s;
}

.source-item:hover {
  background: var(--surface-elevated);
}

.source-item__main {
  min-width: 0;
  flex: 1;
}

.source-item__header {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.source-name {
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--text-primary);
}

.writable-indicator {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.6875rem;
}

.writable-text {
  font-size: 0.6875rem;
  color: var(--text-muted);
}

.source-path {
  margin: 0.25rem 0 0;
  color: var(--text-muted);
  font: 0.75rem/1.3 var(--font-data);
  word-break: break-all;
}

.source-item__meta {
  flex-shrink: 0;
}

.source-item__actions {
  flex-shrink: 0;
}

@media (max-width: 700px) {
  .source-form {
    grid-template-columns: 1fr;
  }
  .source-form .form-action {
    margin-top: 0.25rem;
  }
  .source-item {
    flex-wrap: wrap;
    gap: 0.75rem;
  }
  .source-item__actions {
    width: 100%;
    display: flex;
    justify-content: flex-end;
  }
}
</style>
