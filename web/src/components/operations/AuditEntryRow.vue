<script setup lang="ts">
import type { AuditEntry } from '@/api/types'
defineProps<{ entry: AuditEntry; labels: Record<string, string>; expanded: boolean }>()
const emit = defineEmits<{ toggle: [] }>()
</script>

<template>
  <article
    class="audit-row"
    :class="{ 'audit-row--expanded': expanded }"
  >
    <div class="audit-row__head">
      <div class="action-tag font-code">{{ entry.action }}</div>
      <span class="outcome-badge" :class="`outcome-badge--${entry.outcome}`">
        {{ labels[`auditOutcome_${entry.outcome}`] || entry.outcome }}
      </span>
    </div>

    <div class="target-path font-code" :title="entry.target || labels.noTarget">
      <svg viewBox="0 0 24 24" fill="currentColor" width="12" height="12" class="path-icon">
        <path d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/>
      </svg>
      <span>{{ entry.target || labels.noTarget }}</span>
    </div>

    <p class="audit-detail">{{ entry.detail }}</p>

    <!-- Expanded details block -->
    <div v-show="expanded" :id="`audit-details-${entry.id}`" class="expanded-audit-body">
      <div class="meta-row font-code">
        <span class="meta-label">ID:</span>
        <span class="meta-val">#{{ entry.id }}</span>
      </div>
      <div class="meta-row font-code">
        <span class="meta-label">{{ labels.status || 'Status' }}:</span>
        <span class="meta-val">{{ labels[`auditOutcome_${entry.outcome}`] || entry.outcome }}</span>
      </div>
      <div class="meta-row font-code">
        <span class="meta-label">Recoverability:</span>
        <span class="meta-val">{{ labels[`recoverability_${entry.recoverability}`] || entry.recoverability }}</span>
      </div>
    </div>

    <div class="audit-meta font-code">
      <span class="timestamp">{{ entry.createdAt }}</span>
      <button
        type="button"
        class="expand-hint btn-ghost"
        :aria-expanded="expanded"
        :aria-controls="`audit-details-${entry.id}`"
        @click="emit('toggle')"
      >{{ expanded ? (labels.closeDiff || 'Close') : (labels.viewDiff || 'Details') }}</button>
    </div>
  </article>
</template>

<style scoped>
.audit-row {
  padding: 12px 20px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  transition: background 0.12s ease;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.audit-row:hover {
  background: var(--surface-container-high, #23293c);
}

.audit-row--expanded {
  background: var(--surface-container-high, #23293c);
}

.audit-row__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.action-tag {
  font-size: 12px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.outcome-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 4px;
  text-transform: uppercase;
  background: var(--surface-container-highest, #2e3447);
  color: var(--outline, #908fa0);
}

.outcome-badge--applied {
  background: rgba(78, 222, 163, 0.15);
  color: var(--secondary, #4edea3);
}

.outcome-badge--previewed {
  background: rgba(255, 185, 95, 0.15);
  color: var(--tertiary, #ffb95f);
}

.target-path {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  color: var(--primary, #c0c1ff);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.path-icon {
  flex-shrink: 0;
  opacity: 0.7;
}

.audit-detail {
  margin: 0;
  font-size: 11px;
  color: var(--on-surface-variant, #c7c4d7);
  line-height: 1.4;
}

.expanded-audit-body {
  margin-top: 4px;
  padding: 8px 10px;
  background: var(--surface-container-lowest, #070d1f);
  border-radius: var(--radius-sm, 0.25rem);
  border: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.meta-row {
  display: flex;
  justify-content: space-between;
  font-size: 10px;
}

.meta-label {
  color: var(--outline, #908fa0);
}

.meta-val {
  color: var(--on-surface, #dce1fb);
}

.audit-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 10px;
  color: var(--outline, #908fa0);
}

.expand-hint:focus-visible {
  outline: 2px solid var(--primary, #c0c1ff);
  outline-offset: 2px;
}

.expand-hint {
  color: var(--primary, #c0c1ff);
}

</style>
