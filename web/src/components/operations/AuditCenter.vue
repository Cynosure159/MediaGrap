<script setup lang="ts">
import type { AuditEntry } from '@/api/types'

defineProps<{ entries: ReadonlyArray<AuditEntry>; labels: Record<string, string> }>()
</script>

<template>
  <section class="ops-card">
    <header class="section-header">
      <div><p class="eyebrow">{{ labels.fileSafety }}</p><h2>{{ labels.auditCenter }}</h2></div>
      <span class="count-pill font-code">{{ entries.length }}</span>
    </header>
    <div class="audit-list">
      <article v-for="entry in entries" :key="entry.id" class="audit-row">
        <div class="audit-row__head">
          <strong>{{ entry.action }}</strong>
          <span class="outcome" :class="`outcome--${entry.outcome}`">{{ labels[`auditOutcome_${entry.outcome}`] || entry.outcome }}</span>
        </div>
        <code>{{ entry.target || labels.noTarget }}</code>
        <p>{{ entry.detail }}</p>
        <div class="audit-meta"><span>{{ entry.createdAt }}</span><span>{{ labels[`recoverability_${entry.recoverability}`] || entry.recoverability }}</span></div>
      </article>
      <p v-if="!entries.length" class="empty-state">{{ labels.noAuditEntries }}</p>
    </div>
    <footer class="safety-note">{{ labels.auditRecoveryNotice }}</footer>
  </section>
</template>

<style scoped>
.ops-card { border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); background: var(--surface-container); overflow: hidden; }
.section-header { display: flex; align-items: center; justify-content: space-between; padding: 1rem 1.1rem; border-bottom: 1px solid var(--border-subtle); }
.eyebrow { margin: 0 0 .2rem; color: var(--primary); font: 700 .65rem/1 var(--font-data); letter-spacing: .08em; text-transform: uppercase; }
h2, p { margin: 0; } h2 { font-size: 1rem; color: var(--on-surface); }
.count-pill { border-radius: 999px; padding: .25rem .5rem; background: var(--surface-container-high); color: var(--primary); font-size: .68rem; }
.audit-list { max-height: 430px; overflow-y: auto; }
.audit-row { padding: .8rem 1.1rem; border-bottom: 1px solid var(--border-subtle); }
.audit-row__head { display: flex; justify-content: space-between; gap: .75rem; }
.audit-row strong { color: var(--on-surface); font: 700 .72rem/1.3 var(--font-data); }
.audit-row code { display: block; margin-top: .35rem; overflow: hidden; color: var(--primary); font-size: .68rem; text-overflow: ellipsis; white-space: nowrap; }
.audit-row p { margin-top: .3rem; color: var(--on-surface-variant); font-size: .72rem; }
.outcome { color: var(--outline); font: 700 .62rem/1 var(--font-data); text-transform: uppercase; } .outcome--applied { color: var(--secondary); } .outcome--previewed { color: var(--tertiary); }
.audit-meta { display: flex; justify-content: space-between; gap: .6rem; margin-top: .5rem; color: var(--outline); font: .62rem/1.3 var(--font-data); }
.empty-state, .safety-note { margin: 0; padding: 1rem 1.1rem; color: var(--outline); font-size: .72rem; }
.safety-note { border-top: 1px solid var(--border-subtle); background: var(--surface-container-low); color: var(--tertiary); }
</style>
