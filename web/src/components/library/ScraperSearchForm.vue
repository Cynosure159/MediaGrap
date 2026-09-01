<script setup lang="ts">
defineProps<{
  modelValue: string
  isSearching: boolean
  placeholder?: string
  searchLabel?: string
  searchingLabel?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'search'): void
}>()
</script>

<template>
  <div class="search-bar-wrap">
    <div class="search-input-box">
      <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="8" />
        <line x1="21" y1="21" x2="16.65" y2="16.65" />
      </svg>
      <input
        :value="modelValue"
        type="text"
        class="modal-search-input"
        :placeholder="placeholder || '输入电影/剧集名称或关键词搜索...'"
        @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        @keydown.enter.prevent="emit('search')"
      />
    </div>
    <button
      class="btn btn-primary search-action-btn"
      :disabled="isSearching"
      type="button"
      @click="emit('search')"
    >
      <svg v-if="isSearching" viewBox="0 0 24 24" fill="currentColor" width="14" height="14" class="spin-slow">
        <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
      </svg>
      <span>{{ isSearching ? (searchingLabel || '搜索中...') : (searchLabel || '搜索') }}</span>
    </button>
  </div>
</template>

<style scoped>
.search-bar-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 20px;
  background: var(--surface-container-low, #151b2d);
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.search-input-box {
  position: relative;
  flex: 1;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  width: 16px;
  height: 16px;
  color: var(--outline, #908fa0);
  pointer-events: none;
}

.modal-search-input {
  width: 100%;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 8px 12px 8px 36px;
  font-size: 13px;
  color: var(--on-surface, #dce1fb);
  outline: none;
  transition: border-color 0.15s ease;
}

.modal-search-input:focus {
  border-color: var(--primary, #c0c1ff);
}

.search-action-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  padding: 8px 16px;
  white-space: nowrap;
}

.spin-slow {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
