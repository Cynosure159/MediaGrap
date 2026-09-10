<script setup lang="ts">
defineProps<{
  placeholder?: string
}>()

const query = defineModel<string>({ required: true })

const emit = defineEmits<{
  search: [query: string]
}>()

function clear() {
  query.value = ''
  emit('search', '')
}
</script>

<template>
  <div class="search-box">
    <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="11" cy="11" r="8" />
      <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
    <input
      v-model="query"
      type="text"
      class="search-input"
      :placeholder="placeholder || 'Search...'"
      @keydown.enter.prevent="emit('search', query)"
    />
    <!-- Clear button when query is typed -->
    <button
      v-if="query"
      type="button"
      class="clear-btn"
      title="Clear"
      @click="clear"
    >
      <svg viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
        <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
      </svg>
    </button>
    <!-- Keyboard hint badge when empty -->
    <span v-else class="kbd-hint font-code">↵</span>
  </div>
</template>

<style scoped>
.search-box {
  position: relative;
  display: flex;
  align-items: center;
  width: 100%;
}

.search-icon {
  position: absolute;
  left: 10px;
  width: 14px;
  height: 14px;
  color: var(--outline, #908fa0);
  pointer-events: none;
  transition: color 0.15s ease;
}

.search-box:focus-within .search-icon {
  color: var(--primary, #c0c1ff);
}

.search-input {
  width: 100%;
  height: 32px;
  padding: 0 28px 0 32px;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  font-size: 12px;
  outline: none;
  transition: all 0.15s ease;
  box-sizing: border-box;
}

.search-input:focus {
  border-color: var(--primary, #c0c1ff);
  box-shadow: 0 0 0 2px rgba(192, 193, 255, 0.15);
  background: var(--surface-container-low, #151b2d);
}

.search-input::placeholder {
  color: rgba(144, 143, 160, 0.6);
  font-size: 12px;
}

.clear-btn {
  position: absolute;
  right: 6px;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-container-high, #23293c);
  border: none;
  border-radius: 50%;
  color: var(--outline, #908fa0);
  cursor: pointer;
  padding: 0;
  transition: all 0.12s ease;
}

.clear-btn:hover {
  background: var(--surface-container-highest, #2e3447);
  color: var(--on-surface, #dce1fb);
}

.kbd-hint {
  position: absolute;
  right: 8px;
  font-size: 10px;
  color: var(--outline, #908fa0);
  opacity: 0.5;
  pointer-events: none;
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: 3px;
  padding: 1px 4px;
  line-height: 1;
}
</style>
