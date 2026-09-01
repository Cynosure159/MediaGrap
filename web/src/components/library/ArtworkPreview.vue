<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import type { ArtworkPlan } from '@/api/library'

const props = defineProps<{ plan: ArtworkPlan | null; applying: boolean; labels: Record<string, string>; error?: string | null }>()
const emit = defineEmits<{ apply: []; close: [] }>()
const hasConflict = computed(() => props.plan?.assets.some((asset) => asset.conflict) ?? false)
const isPending = computed(() => props.plan?.state === 'previewed')

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') emit('close')
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', handleKeydown))

function kindLabel(kind: string, labels: Record<string, string>) {
  const key: Record<string, string> = { poster: 'poster', fanart: 'fanart', clearlogo: 'clearLogo', clearart: 'clearArt', discart: 'discArt', banner: 'banner', landscape: 'landscape' }
  return labels[key[kind]] || kind
}
</script>

<template>
  <Teleport to="body">
    <div v-if="plan" class="backdrop" @click.self="emit('close')" @keydown.esc="emit('close')">
    <section class="dialog" role="dialog" aria-modal="true" :aria-label="labels.artworkPreview" tabindex="-1">
      <header class="dialog-header">
        <div>
          <p class="eyebrow">{{ labels.artworkEyebrow }}</p>
          <h2>{{ labels.artworkPreview }}</h2>
        </div>
        <button class="close-btn" type="button" :aria-label="labels.closeArtworkPreview || labels.cancel || 'Close'" @click="emit('close')">×</button>
      </header>
      <div class="dialog-body">
        <div v-if="error" class="modal-error" role="alert">{{ error }}</div>
        <p class="dialog-help">{{ labels.artworkHelp }}</p>
        <ul class="asset-list">
          <li v-for="asset in plan.assets" :key="asset.kind" :class="{ conflict: asset.conflict }">
            <div class="asset-info">
              <strong>{{ kindLabel(asset.kind, labels) }}</strong>
              <small class="target-path" :title="asset.targetPath">{{ asset.targetPath }}</small>
              <small v-if="asset.width && asset.height">{{ labels.artworkDimensions }}: {{ asset.width }}×{{ asset.height }} · {{ asset.language || '00' }} · ♥ {{ asset.likes || 0 }}</small>
            </div>
            <span v-if="asset.conflict">{{ labels.artworkConflict }}</span>
            <span v-else-if="asset.willReplace">{{ labels.artworkReplace }}</span>
            <span v-else>{{ labels.artworkCreate }}</span>
          </li>
        </ul>
      </div>
      <footer class="dialog-footer">
        <span v-if="!isPending" class="queued-state">{{ labels.artworkQueued }}</span>
        <span v-else class="footer-spacer"></span>
        <div class="footer-actions">
          <button class="btn btn-ghost" type="button" @click="emit('close')">{{ labels.cancel }}</button>
          <button v-if="isPending" class="btn btn-primary" type="button" :disabled="hasConflict || applying" @click="emit('apply')">{{ applying ? labels.downloading : labels.downloadArtwork }}</button>
        </div>
      </footer>
    </section>
    </div>
  </Teleport>
</template>

<style scoped>
.backdrop { position: fixed; inset: 0; z-index: 210; display: flex; align-items: center; justify-content: center; padding: 20px; background: rgba(7, 13, 31, .86); backdrop-filter: blur(10px); }
.dialog { display: flex; flex-direction: column; width: min(100%, 720px); max-height: min(760px, calc(100vh - 40px)); overflow: hidden; border: 1px solid var(--outline-variant, #2e3447); border-radius: var(--radius-xl, .75rem); background: var(--surface-container, #191f31); box-shadow: 0 28px 80px rgba(0, 0, 0, .65); }
.dialog-header, .dialog-footer { display: flex; align-items: center; justify-content: space-between; gap: 1rem; flex-shrink: 0; }
.dialog-header { padding: 16px 20px; border-bottom: 1px solid var(--outline-variant, #2e3447); background: var(--surface-dim, #0c1324); }
.eyebrow { margin: 0 0 3px; color: var(--primary, #c0c1ff); font: 700 10px var(--font-data, monospace); letter-spacing: .08em; text-transform: uppercase; }
h2 { margin: 0; color: var(--on-surface, #dce1fb); font-size: 17px; }
.close-btn { display: grid; width: 32px; height: 32px; place-items: center; border: 1px solid transparent; border-radius: var(--radius-sm, .25rem); background: transparent; color: var(--on-surface-variant, #c7c4d7); font-size: 24px; line-height: 1; cursor: pointer; }
.close-btn:hover { border-color: var(--outline-variant, #2e3447); background: var(--surface-container-high, #23293c); color: var(--on-surface, #dce1fb); }
.dialog-body { min-height: 0; overflow: auto; padding: 16px 20px 20px; }
.dialog-help { margin: 0 0 14px; color: var(--on-surface-variant, #c7c4d7); font-size: 12px; line-height: 1.5; }
.modal-error { margin-bottom: 12px; padding: 9px 10px; border: 1px solid var(--error, #ffb4ab); border-radius: var(--radius-md, .375rem); background: color-mix(in srgb, var(--error, #ffb4ab) 12%, transparent); color: var(--error, #ffb4ab); font-size: 12px; line-height: 1.4; }
.asset-list { display: grid; gap: 8px; margin: 0; padding: 0; list-style: none; }
.asset-list li { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 12px; border: 1px solid var(--outline-variant, #2e3447); border-left: 3px solid var(--secondary, #4edea3); border-radius: var(--radius-md, .375rem); background: var(--surface-container-low, #151b2d); }
.asset-list li.conflict { border-left-color: var(--error-bright, #f43f5e); background: color-mix(in srgb, var(--error-bright, #f43f5e) 10%, var(--surface-container-low, #151b2d)); }
.asset-info { min-width: 0; }
.asset-info strong { display: block; color: var(--on-surface, #dce1fb); font-size: 13px; }
.asset-list small { display: block; margin-top: 4px; color: var(--outline, #908fa0); font-size: 11px; }
.target-path { overflow: hidden; font-family: var(--font-data, monospace); text-overflow: ellipsis; white-space: nowrap; }
.asset-list li > span { max-width: 15rem; color: var(--on-surface-variant, #c7c4d7); font-size: 11px; text-align: right; }
.dialog-footer { padding: 12px 20px; border-top: 1px solid var(--outline-variant, #2e3447); background: var(--surface-dim, #0c1324); }
.queued-state { color: var(--secondary, #4edea3); font: 11px var(--font-data, monospace); }
.footer-spacer { flex: 1; }
.footer-actions { display: flex; gap: 8px; }

@media (max-width: 700px) {
  .backdrop { align-items: flex-end; padding: 0; }
  .dialog { width: 100%; max-height: 94vh; border-radius: var(--radius-xl, .75rem) var(--radius-xl, .75rem) 0 0; }
  .dialog-header, .dialog-footer { padding-left: 16px; padding-right: 16px; }
  .dialog-body { padding: 14px 12px 16px; }
  .asset-list li { align-items: flex-start; flex-direction: column; gap: 8px; }
  .asset-list li > span { max-width: none; text-align: left; }
  .dialog-footer { align-items: stretch; flex-direction: column; padding-bottom: calc(12px + env(safe-area-inset-bottom)); }
  .footer-actions { justify-content: flex-end; }
}
</style>
