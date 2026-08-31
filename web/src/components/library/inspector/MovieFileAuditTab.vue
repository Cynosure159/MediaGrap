<script setup lang="ts">
import type { MediaItem } from '@/api/types'

defineProps<{
  item: MediaItem
  labels: Record<string, string>
}>()
</script>

<template>
  <section class="workshop-view">
    <div class="workshop-header">
      <h2>{{ labels.filesRenamePlanner }}</h2>
      <button class="btn btn-primary" type="button">{{ labels.runDryRun }}</button>
    </div>

    <div class="file-audit-list">
      <div class="audit-item">
        <span class="spec-badge">{{ labels.video }}</span>
        <span class="audit-path font-code">{{ item.relativePath }}</span>
        <span class="audit-status text-ok">{{ labels.matched }}</span>
      </div>

      <!-- Sidecar assets if present -->
      <template v-if="item.sidecars && item.sidecars.length > 0">
        <div v-for="sidecar in item.sidecars" :key="sidecar.relativePath" class="audit-item">
          <span class="spec-badge">{{ sidecar.kind.toUpperCase() }}</span>
          <span class="audit-path font-code">{{ sidecar.relativePath }}</span>
          <span class="audit-status text-ok">{{ labels.validKodi }}</span>
        </div>
      </template>

      <template v-else>
        <div class="audit-item">
          <span class="spec-badge">NFO</span>
          <span class="audit-path font-code">movie.nfo</span>
          <span class="audit-status text-ok">{{ labels.validKodi }}</span>
        </div>
        <div class="audit-item">
          <span class="spec-badge">POSTER</span>
          <span class="audit-path font-code">poster.jpg</span>
          <span class="audit-status text-ok">{{ labels.fileResolutionUnavailable }}</span>
        </div>
      </template>
    </div>
  </section>
</template>

<style scoped>
.workshop-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.workshop-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.workshop-header h2 {
  font-size: 16px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  margin: 0;
}

.file-audit-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.audit-item {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.5rem);
  padding: 10px 14px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.audit-path {
  flex: 1;
  font-size: 12px;
  color: var(--on-surface, #dce1fb);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.audit-status {
  font-size: 12px;
  font-weight: 500;
}

.text-ok {
  color: #10b981;
}
</style>
