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
      <div class="card-header__info">
        <div class="header-tag">
          <svg class="header-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"/>
          </svg>
          <span class="eyebrow">{{ labels.libraryEyebrow }}</span>
        </div>
        <h2 class="section-title">{{ labels.directorySettings }}</h2>
        <p class="section-hint">{{ labels.directoryHelp }}</p>
      </div>
    </div>

    <div class="card-body">
      <!-- Add Source Form -->
      <form class="source-form" @submit.prevent="emit('add')">
        <div class="form-grid">
          <div class="field-group">
            <label for="new-source-name" class="field-label">
              <svg class="field-label-icon" viewBox="0 0 24 24" fill="currentColor">
                <path d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/>
              </svg>
              <span>{{ labels.source }}</span>
            </label>
            <input
              id="new-source-name"
              v-model="sourceName"
              class="input-control"
              required
              :placeholder="labels.sourceNamePlaceholder"
            />
          </div>

          <div class="field-group">
            <label for="new-source-path" class="field-label">
              <svg class="field-label-icon" viewBox="0 0 24 24" fill="currentColor">
                <path d="M20 18c1.1 0 1.99-.9 1.99-2L22 6c0-1.1-.9-2-2-2H4c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"/>
              </svg>
              <span>{{ labels.containerPath }}</span>
            </label>
            <input
              id="new-source-path"
              v-model="sourcePath"
              class="input-control font-data"
              list="media-roots"
              required
              :placeholder="labels.sourcePathPlaceholder"
            />
            <datalist id="media-roots">
              <option v-for="root in mediaRoots" :key="root" :value="root" />
            </datalist>
          </div>

          <div class="form-action">
            <button type="submit" class="btn btn-primary add-button">
              <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                <path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
              </svg>
              <span>{{ labels.addSource }}</span>
            </button>
          </div>
        </div>
      </form>

      <!-- Feedback Alert -->
      <div
        v-if="feedback"
        class="source-feedback"
        :class="`source-feedback--${feedback.kind}`"
        role="status"
      >
        <span
          class="feedback-dot"
          :class="feedback.kind === 'success' ? 'dot-ok' : 'dot-err'"
        ></span>
        <span class="feedback-text">{{ feedback.message }}</span>
      </div>

      <!-- Empty State -->
      <div v-if="sources.length === 0" class="empty-sources">
        <div class="empty-icon-box">
          <svg class="empty-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"/>
          </svg>
        </div>
        <p class="empty-title">{{ labels.noSources }}</p>
        <p class="empty-desc">{{ labels.firstSource }}</p>
      </div>

      <!-- Configured Source List -->
      <div v-else class="source-list">
        <article
          v-for="source in sources"
          :key="source.id"
          class="source-card"
        >
          <!-- Left Status Indicator Bar -->
          <div
            class="source-accent-bar"
            :class="source.writable ? 'source-accent-bar--ok' : 'source-accent-bar--warn'"
          ></div>

          <!-- Main Body Content -->
          <div class="source-card__content">
            <!-- Source Type Icon Container -->
            <div class="source-icon-badge">
              <svg
                v-if="source.name.toLowerCase().includes('show') || source.name.toLowerCase().includes('tv')"
                class="source-icon"
                viewBox="0 0 24 24"
                fill="currentColor"
              >
                <path d="M21 3H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h5v2h8v-2h5c1.1 0 1.99-.9 1.99-2L23 5c0-1.1-.9-2-2-2zm0 14H3V5h18v12z"/>
              </svg>
              <svg
                v-else
                class="source-icon"
                viewBox="0 0 24 24"
                fill="currentColor"
              >
                <path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
              </svg>
            </div>

            <!-- Title, Path & Meta -->
            <div class="source-info">
              <div class="source-info__top">
                <h3 class="source-name">{{ source.name }}</h3>
                <span
                  class="status-pill"
                  :class="source.writable ? 'status-pill--ok' : 'status-pill--warn'"
                  :title="source.writable ? labels.writable : labels.readOnly"
                >
                  <span class="status-dot-inner" :class="source.writable ? 'dot-ok' : 'dot-warn'"></span>
                  <span>{{ source.writable ? labels.writable : labels.readOnly }}</span>
                </span>
              </div>

              <div class="source-info__meta">
                <span class="source-path-code">{{ source.rootPath }}</span>
                <span class="meta-dot-sep">•</span>
                <span class="spec-pill">
                  <svg class="spec-icon" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M14 2H6c-1.1 0-1.99.9-1.99 2L4 20c0 1.1.89 2 1.99 2H18c1.1 0 2-.9 2-2V8l-6-6zm2 16H8v-2h8v2zm0-4H8v-2h8v2zm-3-5V3.5L18.5 9H13z"/>
                  </svg>
                  {{ source.itemCount }} {{ labels.indexed }}
                </span>
              </div>
            </div>

            <!-- Action Buttons (Scan + Remove) -->
            <div class="source-actions">
              <button
                type="button"
                class="btn btn-outline btn-sm action-btn"
                :title="labels.scan || 'Scan'"
                @click="emit('scan', source.id)"
              >
                <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
                </svg>
                <span>{{ labels.scan || '扫描' }}</span>
              </button>

              <button
                type="button"
                class="btn btn-danger-subtle btn-sm action-btn"
                :title="labels.deleteSource || 'Remove'"
                @click="emit('delete', source.id)"
              >
                <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
                </svg>
                <span>{{ labels.deleteSource || '删除' }}</span>
              </button>
            </div>
          </div>
        </article>
      </div>
    </div>
  </section>
</template>

<style scoped>
.settings-card {
  background: var(--surface-container);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
}

.card-header {
  padding: 1.125rem 1.5rem;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--surface-container);
}

.card-header__info {
  display: grid;
  gap: 0.3rem;
}

.header-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--primary);
}

.header-icon {
  width: 14px;
  height: 14px;
}

.eyebrow {
  margin: 0;
  font-family: var(--font-data);
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--primary);
}

.section-title {
  margin: 0;
  color: var(--on-surface);
  font-family: var(--font-display);
  font-size: 1.0625rem;
  font-weight: 700;
  line-height: 1.3;
  letter-spacing: -0.01em;
}

.section-hint {
  margin: 0;
  color: var(--on-surface-variant);
  font-size: 0.8125rem;
  line-height: 1.45;
}

.card-body {
  padding: 1.5rem;
  display: grid;
  gap: 1.25rem;
}

/* ── Add Source Form ──────────────────────────────────────────────── */
.source-form {
  padding: 1.125rem 1.25rem;
  background: var(--surface-container-low);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1.5fr auto;
  gap: 1rem;
  align-items: flex-end;
}

.field-group {
  display: grid;
  gap: 0.4rem;
}

.field-label {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--on-surface-variant);
}

.field-label-icon {
  width: 14px;
  height: 14px;
  color: var(--outline);
}

.input-control {
  width: 100%;
  height: 34px;
  padding: 0.35rem 0.75rem;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-base);
  color: var(--on-surface);
  font-size: 0.8125rem;
  transition: all 0.15s ease;
}

.input-control.font-data {
  font-family: var(--font-data);
  font-size: 0.8125rem;
}

.input-control:focus {
  outline: none;
  border-color: var(--primary-bright);
  box-shadow: 0 0 0 2px rgba(128, 131, 255, 0.2);
  background: var(--surface-container-lowest);
}

.form-action {
  display: flex;
  align-items: flex-end;
}

.add-button {
  height: 34px;
  padding: 0 1rem;
  font-size: 0.8125rem;
  font-weight: 600;
  gap: 0.4rem;
}

.btn-icon {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
}

/* ── Feedback Alert ───────────────────────────────────────────────── */
.source-feedback {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 0.875rem;
  border-radius: var(--radius-sm);
  font-size: 0.8125rem;
}

.source-feedback--success {
  background: rgba(0, 165, 114, 0.15);
  border: 1px solid rgba(78, 222, 163, 0.3);
  color: var(--secondary);
}

.source-feedback--error {
  background: rgba(147, 0, 10, 0.15);
  border: 1px solid rgba(244, 63, 94, 0.3);
  color: var(--error);
}

.feedback-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.feedback-text {
  line-height: 1.4;
}

/* ── Empty State ─────────────────────────────────────────────────── */
.empty-sources {
  margin: 0;
  padding: 2.5rem 1.5rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  background: var(--surface-container-low);
  border: 1px dashed var(--border-default);
  border-radius: var(--radius-lg);
}

.empty-icon-box {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-lg);
  background: var(--surface-container-high);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 0.75rem;
  color: var(--outline);
}

.empty-icon {
  width: 24px;
  height: 24px;
}

.empty-title {
  margin: 0 0 0.25rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--on-surface);
}

.empty-desc {
  margin: 0;
  font-size: 0.75rem;
  color: var(--on-surface-variant);
}

/* ── Configured Sources List ──────────────────────────────────────── */
.source-list {
  display: grid;
  gap: 0.75rem;
}

.source-card {
  position: relative;
  display: flex;
  background: var(--surface-container-low);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  overflow: hidden;
  transition: all 0.15s ease;
}

.source-card:hover {
  background: var(--surface-container-high);
  border-color: var(--outline);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.source-accent-bar {
  width: 3px;
  flex-shrink: 0;
}

.source-accent-bar--ok {
  background: var(--secondary);
}

.source-accent-bar--warn {
  background: var(--tertiary);
}

.source-card__content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.875rem 1.125rem;
  flex: 1;
  min-width: 0;
}

.source-icon-badge {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: var(--surface-container-lowest);
  border: 1px solid var(--border-default);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--secondary);
  flex-shrink: 0;
}

.source-icon {
  width: 22px;
  height: 22px;
}

.source-info {
  display: grid;
  gap: 0.35rem;
  flex: 1;
  min-width: 0;
}

.source-info__top {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.source-name {
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--on-surface);
  line-height: 1.2;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.1rem 0.45rem;
  border-radius: var(--radius-sm);
  font-size: 0.6875rem;
  font-weight: 600;
  line-height: 1.2;
  letter-spacing: 0.02em;
}

.status-pill--ok {
  background: rgba(0, 165, 114, 0.18);
  color: var(--secondary);
  border: 1px solid rgba(78, 222, 163, 0.3);
}

.status-pill--warn {
  background: rgba(202, 129, 0, 0.18);
  color: var(--tertiary);
  border: 1px solid rgba(255, 185, 95, 0.3);
}

.status-dot-inner {
  width: 5px;
  height: 5px;
  border-radius: 50%;
}

.source-info__meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.source-path-code {
  display: inline-flex;
  align-items: center;
  padding: 0.15rem 0.5rem;
  background: var(--surface-container-lowest);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  color: var(--on-surface-variant);
  font-family: var(--font-data);
  font-size: 0.75rem;
  word-break: break-all;
}

.meta-dot-sep {
  color: var(--outline-variant);
  font-size: 0.75rem;
}

.spec-icon {
  width: 12px;
  height: 12px;
  color: var(--outline);
}

.source-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

/* ── Responsive Behavior ─────────────────────────────────────────── */
@media (max-width: 768px) {
  .form-grid {
    grid-template-columns: 1fr;
    gap: 0.75rem;
  }
  .form-action {
    margin-top: 0.25rem;
  }
  .add-button {
    width: 100%;
  }
  .source-card__content {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.75rem;
  }
  .source-actions {
    width: 100%;
    justify-content: flex-end;
    border-top: 1px solid var(--border-subtle);
    padding-top: 0.5rem;
  }
}
</style>

