<script setup lang="ts">
defineProps<{
  specs: string[]
}>()
</script>

<template>
  <div class="spec-pills-row">
    <span v-for="spec in specs" :key="spec" class="spec-pill font-code">
      {{ spec }}
    </span>
  </div>
</template>

<style scoped>
.spec-pills-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.spec-pill {
  font-size: 11px;
  font-weight: 500;
  padding: 3px 8px;
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  color: var(--on-surface-variant, #c7c4d7);
  letter-spacing: 0.02em;
}
</style>
