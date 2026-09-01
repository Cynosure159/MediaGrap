<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import type { ArtworkCandidate } from '@/api/library'
import ArtworkCandidateGallery from './ArtworkCandidateGallery.vue'

const props = defineProps<{
  groups: { kind: ArtworkCandidate['kind']; items: ArtworkCandidate[] }[]
  selected: Record<string, string>
  labels: Record<string, string>
  loading: boolean
  writable: boolean
  error?: string | null
  existingKinds?: Partial<Record<ArtworkCandidate['kind'], boolean>>
}>()

const emit = defineEmits<{
  close: []
  scrape: []
  select: [candidate: ArtworkCandidate]
  preview: []
}>()

const selectedCount = computed(() => Object.values(props.selected).filter(Boolean).length)
const canPreview = computed(() => selectedCount.value > 0 && props.writable && !props.loading)

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') emit('close')
}

onMounted(() => {
  document.body.classList.add('modal-open')
  window.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  document.body.classList.remove('modal-open')
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <Teleport to="body">
    <div class="artwork-modal-backdrop" @click.self="emit('close')">
      <section class="artwork-modal" role="dialog" aria-modal="true" :aria-label="labels.artworkCandidates" tabindex="-1">
        <header class="artwork-modal-header">
          <div class="artwork-modal-heading">
            <div class="artwork-modal-icon" aria-hidden="true">✦</div>
            <div>
              <p class="eyebrow">{{ labels.artworkEyebrow }}</p>
              <h2>{{ labels.artworkCandidates }}</h2>
              <p class="artwork-modal-help">{{ labels.artworkDialogHelp || 'Choose one image per artwork type, then preview the download plan.' }}</p>
            </div>
          </div>
          <button class="modal-close-btn" type="button" :aria-label="labels.closeArtworkCandidates || labels.cancel || 'Close'" @click="emit('close')">
            <span aria-hidden="true">×</span>
          </button>
        </header>

        <div class="artwork-modal-toolbar">
          <span class="artwork-count">
            <strong>{{ selectedCount }}</strong> {{ labels.artworkSelected }}
            <span class="toolbar-separator">·</span>
            {{ groups.length }} {{ labels.artworkGroups || 'groups' }}
          </span>
          <button class="btn btn-outline btn-sm" type="button" :disabled="loading || !writable" @click="emit('scrape')">
            <span aria-hidden="true">↻</span>
            {{ labels.artworkRefresh }}
          </button>
        </div>

        <div class="artwork-modal-body">
          <div v-if="error" class="modal-error" role="alert">{{ error }}</div>
          <div v-if="loading" class="artwork-loading" role="status">
            <span class="loading-spinner" aria-hidden="true"></span>
            <strong>{{ labels.scrapingArtwork || labels.loading || 'Loading artwork…' }}</strong>
            <span>{{ labels.artworkProxyHint || 'Images are requested through the server proxy.' }}</span>
          </div>
          <ArtworkCandidateGallery
            v-else
            :groups="groups"
            :selected="selected"
            :labels="labels"
            :existing-kinds="existingKinds"
            @select="emit('select', $event)"
          />
        </div>

        <footer class="artwork-modal-footer">
          <span class="footer-note" :class="{ 'footer-note-warning': !writable }">
            {{ writable ? (labels.artworkSelectionHint || 'Your files are not changed until download is confirmed.') : (labels.readOnly || 'This source is read-only.') }}
          </span>
          <div class="footer-actions">
            <button class="btn btn-ghost" type="button" @click="emit('close')">{{ labels.cancel }}</button>
            <button class="btn btn-primary" type="button" :disabled="!canPreview" @click="emit('preview')">
              {{ labels.artworkPreview }}
            </button>
          </div>
        </footer>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.artwork-modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(7, 13, 31, 0.86);
  backdrop-filter: blur(10px);
}

.artwork-modal {
  display: flex;
  flex-direction: column;
  width: min(100%, 1080px);
  max-height: min(900px, calc(100vh - 40px));
  overflow: hidden;
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-xl, .75rem);
  background: var(--surface-container, #191f31);
  box-shadow: 0 28px 80px rgba(0, 0, 0, .65), 0 0 0 1px rgba(192, 193, 255, .04);
}

.artwork-modal-header,
.artwork-modal-toolbar,
.artwork-modal-footer {
  flex-shrink: 0;
}

.artwork-modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px 16px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  background: linear-gradient(115deg, rgba(99, 102, 241, .18), transparent 42%), var(--surface-dim, #0c1324);
}

.artwork-modal-heading {
  display: flex;
  gap: 12px;
  min-width: 0;
}

.artwork-modal-icon {
  display: grid;
  flex: 0 0 36px;
  width: 36px;
  height: 36px;
  place-items: center;
  border: 1px solid rgba(192, 193, 255, .35);
  border-radius: var(--radius-md, .375rem);
  background: var(--surface-container-high, #23293c);
  color: var(--primary, #c0c1ff);
  font-size: 18px;
}

.eyebrow {
  margin: 0 0 3px;
  color: var(--primary, #c0c1ff);
  font: 700 10px var(--font-data, monospace);
  letter-spacing: .08em;
  text-transform: uppercase;
}

h2 {
  margin: 0;
  color: var(--on-surface, #dce1fb);
  font-size: 18px;
  line-height: 1.2;
}

.artwork-modal-help {
  margin: 5px 0 0;
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 12px;
}

.modal-close-btn {
  display: grid;
  flex: 0 0 32px;
  width: 32px;
  height: 32px;
  place-items: center;
  border: 1px solid transparent;
  border-radius: var(--radius-sm, .25rem);
  background: transparent;
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 25px;
  line-height: 1;
  cursor: pointer;
}

.modal-close-btn:hover {
  border-color: var(--outline-variant, #2e3447);
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface, #dce1fb);
}

.artwork-modal-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 20px;
  border-bottom: 1px solid var(--border-subtle, #1e2436);
  background: var(--surface-container-low, #151b2d);
}

.artwork-count {
  color: var(--on-surface-variant, #c7c4d7);
  font: 11px var(--font-data, monospace);
}

.artwork-count strong {
  color: var(--secondary, #4edea3);
  font-size: 14px;
}

.toolbar-separator {
  padding: 0 7px;
  color: var(--outline, #908fa0);
}

.artwork-modal-body {
  min-height: 0;
  overflow: auto;
  padding: 16px 20px 20px;
}

.modal-error { margin-bottom: 12px; padding: 9px 10px; border: 1px solid var(--error, #ffb4ab); border-radius: var(--radius-md, .375rem); background: color-mix(in srgb, var(--error, #ffb4ab) 12%, transparent); color: var(--error, #ffb4ab); font-size: 12px; line-height: 1.4; }

.artwork-loading {
  display: grid;
  min-height: 280px;
  place-items: center;
  align-content: center;
  gap: 8px;
  color: var(--on-surface-variant, #c7c4d7);
  text-align: center;
}

.artwork-loading strong {
  color: var(--on-surface, #dce1fb);
  font-size: 14px;
}

.artwork-loading span:last-child {
  font-size: 11px;
}

.loading-spinner {
  width: 28px;
  height: 28px;
  border: 2px solid var(--outline-variant, #2e3447);
  border-top-color: var(--primary, #c0c1ff);
  border-radius: 50%;
  animation: spin .8s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

.artwork-modal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 20px;
  border-top: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-dim, #0c1324);
}

.footer-note {
  color: var(--outline, #908fa0);
  font-size: 11px;
}

.footer-note-warning { color: var(--tertiary, #ffb95f); }

.footer-actions {
  display: flex;
  flex-shrink: 0;
  gap: 8px;
}

.btn-sm { height: 26px; padding: 0 .65rem; font-size: 11px; }

@media (max-width: 700px) {
  .artwork-modal-backdrop { align-items: flex-end; padding: 0; }
  .artwork-modal { width: 100%; max-height: 94vh; border-radius: var(--radius-xl, .75rem) var(--radius-xl, .75rem) 0 0; }
  .artwork-modal-header { padding: 15px 16px 13px; }
  .artwork-modal-toolbar { padding: 9px 16px; }
  .artwork-modal-body { padding: 12px 12px 16px; }
  .artwork-modal-footer { align-items: stretch; flex-direction: column; padding: 11px 16px calc(11px + env(safe-area-inset-bottom)); }
  .footer-actions { justify-content: flex-end; }
}

@media (prefers-reduced-motion: reduce) {
  .loading-spinner { animation: none; }
}
</style>
