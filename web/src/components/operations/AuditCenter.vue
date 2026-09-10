<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import AuditEntryRow from './AuditEntryRow.vue'
import type { AuditEntry } from '@/api/types'

const props = defineProps<{ entries: ReadonlyArray<AuditEntry>; labels: Record<string, string> }>()

const searchQuery = shallowRef('')
const actionFilter = shallowRef<'all' | 'nfo' | 'rename'>('all')
const expandedEntryId = shallowRef<number | null>(null)

const filteredEntries = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return props.entries.filter(entry => {
    if (actionFilter.value === 'nfo' && !entry.action.toLowerCase().includes('nfo')) return false
    if (actionFilter.value === 'rename' && !entry.action.toLowerCase().includes('rename') && !entry.action.toLowerCase().includes('move')) return false
    if (!q) return true
    return (
      entry.action.toLowerCase().includes(q) ||
      (entry.target && entry.target.toLowerCase().includes(q)) ||
      entry.outcome.toLowerCase().includes(q) ||
      entry.detail.toLowerCase().includes(q)
    )
  })
})

function toggleExpand(id: number) {
  expandedEntryId.value = expandedEntryId.value === id ? null : id
}
</script>

<template>
  <section class="ops-card audit-card">
    <header class="card-header">
      <div class="header-main">
        <div class="title-group">
          <p class="eyebrow">{{ labels.fileSafety || 'FILESYSTEM SAFETY' }}</p>
          <h2>{{ labels.auditCenter || 'Audit & Recovery' }}</h2>
        </div>
        <span class="count-pill font-code">{{ entries.length }}</span>
      </div>

      <!-- Search & Category Filter Row -->
      <div class="audit-controls">
        <div class="search-input-wrap">
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14" class="search-ico">
            <path d="M15.5 14h-.79l-.28-.27A6.471 6.471 0 0 0 16 9.5 6.5 6.5 0 1 0 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            class="audit-search"
            :placeholder="labels.auditSearchPlaceholder || 'Search path, action…'"
          />
        </div>

        <div class="filter-btn-group">
          <button
            type="button"
            class="pill-btn"
            :class="{ active: actionFilter === 'all' }"
            @click="actionFilter = 'all'"
          >
            {{ labels.auditFilterAll || 'All' }}
          </button>
          <button
            type="button"
            class="pill-btn"
            :class="{ active: actionFilter === 'nfo' }"
            @click="actionFilter = 'nfo'"
          >
            {{ labels.auditFilterNfo || 'NFO' }}
          </button>
          <button
            type="button"
            class="pill-btn"
            :class="{ active: actionFilter === 'rename' }"
            @click="actionFilter = 'rename'"
          >
            {{ labels.auditFilterRename || 'Rename' }}
          </button>
        </div>
      </div>
    </header>

    <!-- Audit List -->
    <div class="audit-list">
      <AuditEntryRow v-for="entry in filteredEntries" :key="entry.id" :entry="entry" :labels="labels" :expanded="expandedEntryId === entry.id" @toggle="toggleExpand(entry.id)" />

      <div v-if="!filteredEntries.length" class="empty-state">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="24" height="24" class="empty-ico">
          <circle cx="12" cy="12" r="9"/>
          <path d="M12 8v4M12 16h.01"/>
        </svg>
        <p>{{ entries.length === 0 ? (labels.noAuditEntries || 'No write plans recorded yet.') : (labels.noAuditMatch || 'No matches found.') }}</p>
      </div>
    </div>

    <!-- Safety notice footer -->
    <footer class="safety-note">
      <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14" class="shield-ico">
        <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z"/>
      </svg>
      <span>{{ labels.auditRecoveryNotice }}</span>
    </footer>
  </section>
</template>

<style scoped>
.audit-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
  display: flex;
  flex-direction: column;
}

.card-header {
  padding: 16px 20px;
  background: var(--surface-container-low, #151b2d);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.header-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.title-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.eyebrow {
  margin: 0;
  color: var(--primary, #c0c1ff);
  font: 700 0.65rem/1 var(--font-data);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

h2 {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
}

.count-pill {
  border-radius: 999px;
  padding: 2px 8px;
  background: var(--surface-container-highest, #2e3447);
  color: var(--primary, #c0c1ff);
  font-size: 11px;
}

.audit-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.search-input-wrap {
  position: relative;
  flex: 1;
  min-width: 140px;
}

.search-ico {
  position: absolute;
  left: 8px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--outline, #908fa0);
  pointer-events: none;
}

.audit-search {
  width: 100%;
  height: 28px;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 0 8px 0 26px;
  font-size: 12px;
  color: var(--on-surface, #dce1fb);
  box-sizing: border-box;
}

.audit-search:focus {
  outline: none;
  border-color: var(--primary, #c0c1ff);
}

.filter-btn-group {
  display: flex;
  gap: 4px;
}

.pill-btn {
  height: 28px;
  padding: 0 10px;
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.12s ease;
}

.pill-btn:hover {
  color: var(--on-surface, #dce1fb);
}

.pill-btn.active {
  background: var(--primary, #c0c1ff);
  color: #0c1324;
  border-color: var(--primary, #c0c1ff);
  font-weight: 600;
}

.audit-list {
  max-height: 480px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px 20px;
  color: var(--outline, #908fa0);
  font-size: 12px;
}

.empty-ico {
  opacity: 0.4;
}

.safety-note {
  margin: 0;
  padding: 10px 16px;
  border-top: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-container-lowest, #070d1f);
  color: var(--tertiary, #ffb95f);
  font-size: 11px;
  line-height: 1.4;
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.shield-ico {
  flex-shrink: 0;
  margin-top: 2px;
}
</style>
