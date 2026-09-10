<script setup lang="ts">
import type { JobFilter } from './jobFilters'
defineProps<{ labels: Record<string, string>; counts: Record<JobFilter, number> }>()
const activeFilter = defineModel<JobFilter>({ required: true })
</script>

<template>
  <div class="filter-tabs-bar" role="tablist">
    <div
      role="tab"
      tabindex="0"
      class="filter-tab"
      :class="{ active: activeFilter === 'all' }"
      @click="activeFilter = 'all'"
      @keydown.enter="activeFilter = 'all'"
    >
      <span>{{ labels.allJobs || 'All' }}</span>
      <span class="tab-count font-code">{{ counts.all }}</span>
    </div>
    <div
      role="tab"
      tabindex="0"
      class="filter-tab"
      :class="{ active: activeFilter === 'running' }"
      @click="activeFilter = 'running'"
      @keydown.enter="activeFilter = 'running'"
    >
      <span class="tab-dot tab-dot--running"></span>
      <span>{{ labels.runningJobs || 'Running' }}</span>
      <span v-if="counts.running" class="tab-count font-code">{{ counts.running }}</span>
    </div>
    <div
      role="tab"
      tabindex="0"
      class="filter-tab"
      :class="{ active: activeFilter === 'queued' }"
      @click="activeFilter = 'queued'"
      @keydown.enter="activeFilter = 'queued'"
    >
      <span>{{ labels.queuedJobs || 'Queued' }}</span>
      <span v-if="counts.queued" class="tab-count font-code">{{ counts.queued }}</span>
    </div>
    <div
      role="tab"
      tabindex="0"
      class="filter-tab"
      :class="{ active: activeFilter === 'succeeded' }"
      @click="activeFilter = 'succeeded'"
      @keydown.enter="activeFilter = 'succeeded'"
    >
      <span>{{ labels.succeededJobs || 'Succeeded' }}</span>
      <span v-if="counts.succeeded" class="tab-count font-code">{{ counts.succeeded }}</span>
    </div>
    <div
      role="tab"
      tabindex="0"
      class="filter-tab"
      :class="{ active: activeFilter === 'failed' }"
      @click="activeFilter = 'failed'"
      @keydown.enter="activeFilter = 'failed'"
    >
      <span class="tab-dot tab-dot--failed"></span>
      <span>{{ labels.failedJobs || 'Failed' }}</span>
      <span v-if="counts.failed" class="tab-count font-code">{{ counts.failed }}</span>
    </div>
  </div>
</template>

<style scoped>
.filter-tabs-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  overflow-x: auto;
  padding-bottom: 2px;
  scrollbar-width: none;
}

.filter-tabs-bar::-webkit-scrollbar {
  display: none;
}

.filter-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  border-radius: var(--radius-sm, 0.25rem);
  background: transparent;
  border: 1px solid transparent;
  color: var(--on-surface-variant, #c7c4d7);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s ease;
}

.filter-tab:hover {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface, #dce1fb);
}

.filter-tab.active {
  background: var(--surface-container-high, #23293c);
  border-color: var(--outline-variant, #2e3447);
  color: var(--primary, #c0c1ff);
  font-weight: 600;
}

.tab-count {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 999px;
  background: var(--surface-container-highest, #2e3447);
  color: var(--on-surface-variant, #c7c4d7);
}

.filter-tab.active .tab-count {
  background: rgba(192, 193, 255, 0.15);
  color: var(--primary, #c0c1ff);
}

.tab-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.tab-dot--running {
  background: var(--tertiary, #ffb95f);
}

.tab-dot--failed {
  background: var(--error, #ffb4ab);
}

</style>
