<script setup lang="ts">
export interface NavigationItem {
  id: string
  label: string
  detail: string
  disabled?: boolean
}

defineProps<{
  items: NavigationItem[]
  activeSection: string
}>()

const emit = defineEmits<{
  selectSection: [id: string]
}>()
</script>

<template>
  <aside class="sidebar">
    <div class="brand-lockup">
      <div class="brand-mark" aria-hidden="true"><span></span></div>
      <div>
        <p class="brand-name">MediaGrap</p>
        <p class="brand-subtitle">Self-hosted metadata</p>
      </div>
    </div>

    <nav class="navigation" aria-label="Application navigation">
      <button
        v-for="item in items"
        :key="item.id"
        class="navigation-item"
        :class="{ 'navigation-item--active': activeSection === item.id }"
        :disabled="item.disabled"
        type="button"
        @click="emit('selectSection', item.id)"
      >
        <span class="navigation-label">{{ item.label }}</span>
        <span class="navigation-detail">{{ item.detail }}</span>
      </button>
    </nav>

    <p class="sidebar-footnote">v0.1 · phase 0</p>
  </aside>
</template>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  gap: 3rem;
  min-width: 15rem;
  padding: 2rem 1.5rem;
  background: var(--ink-900);
  color: var(--mist-100);
}

.brand-lockup {
  display: flex;
  align-items: center;
  gap: 0.8rem;
}

.brand-mark {
  display: grid;
  width: 2.25rem;
  height: 2.25rem;
  place-items: center;
  border: 1px solid var(--water-400);
  border-radius: 0.7rem;
}

.brand-mark span {
  width: 1.1rem;
  height: 0.75rem;
  border: 2px solid var(--mist-100);
  border-left-width: 0.45rem;
  border-right-width: 0.45rem;
}

.brand-name,
.brand-subtitle,
.sidebar-footnote {
  margin: 0;
}

.brand-name { font-family: var(--font-display); font-size: 1.25rem; letter-spacing: -0.04em; }
.brand-subtitle, .sidebar-footnote { color: var(--mist-400); font: 0.72rem/1.4 var(--font-data); letter-spacing: 0.08em; text-transform: uppercase; }
.navigation { display: grid; gap: 0.45rem; }
.navigation-item { display: grid; gap: 0.16rem; padding: 0.85rem; border: 1px solid transparent; border-radius: 0.65rem; background: transparent; color: inherit; text-align: left; cursor: pointer; }
.navigation-item:hover:not(:disabled), .navigation-item:focus-visible { background: var(--ink-800); border-color: var(--ink-700); outline: none; }
.navigation-item--active { background: var(--ink-800); border-color: var(--water-700); box-shadow: inset 0.2rem 0 0 var(--amber-400); }
.navigation-item:disabled { color: var(--mist-500); cursor: not-allowed; }
.navigation-label { font-weight: 650; }
.navigation-detail { color: var(--mist-400); font-size: 0.78rem; }
.sidebar-footnote { margin-top: auto; }

@media (max-width: 700px) {
  .sidebar { position: fixed; z-index: 2; right: 0; bottom: 0; left: 0; min-width: 0; padding: 0.6rem 0.75rem calc(0.6rem + env(safe-area-inset-bottom)); gap: 0; border-top: 1px solid var(--ink-700); }
  .brand-lockup, .sidebar-footnote, .navigation-detail { display: none; }
  .navigation { grid-template-columns: repeat(auto-fit, minmax(0, 1fr)); gap: 0.3rem; }
  .navigation-item { min-height: 3.25rem; padding: 0.45rem; text-align: center; }
  .navigation-item--active { box-shadow: inset 0 -0.2rem 0 var(--amber-400); }
}
</style>
