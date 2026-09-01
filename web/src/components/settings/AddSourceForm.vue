<script setup lang="ts">
defineProps<{
  mediaRoots: string[]
  labels: Record<string, string>
}>()

const sourceName = defineModel<string>('sourceName', { required: true })
const sourcePath = defineModel<string>('sourcePath', { required: true })

const emit = defineEmits<{
  add: []
}>()
</script>

<template>
  <form class="source-form" @submit.prevent="emit('add')">
    <div class="form-grid">
      <div class="field-group">
        <label for="new-source-name" class="field-label">
          <svg class="field-label-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/>
          </svg>
          <span>{{ labels.source }}</span>
        </label>
        <input
          id="new-source-name"
          v-model="sourceName"
          class="input-control"
          required
          :placeholder="labels.sourceNamePlaceholder"
        />
      </div>

      <div class="field-group">
        <label for="new-source-path" class="field-label">
          <svg class="field-label-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 18c1.1 0 1.99-.9 1.99-2L22 6c0-1.1-.9-2-2-2H4c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"/>
          </svg>
          <span>{{ labels.containerPath }}</span>
        </label>
        <input
          id="new-source-path"
          v-model="sourcePath"
          class="input-control font-data"
          list="media-roots"
          required
          :placeholder="labels.sourcePathPlaceholder"
        />
        <datalist id="media-roots">
          <option v-for="root in mediaRoots" :key="root" :value="root" />
        </datalist>
      </div>

      <div class="form-action">
        <button type="submit" class="btn btn-primary add-button">
          <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
          </svg>
          <span>{{ labels.addSource }}</span>
        </button>
      </div>
    </div>
  </form>
</template>

<style scoped>
.source-form {
  margin-bottom: 24px;
  padding: 16px 20px;
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr auto;
  gap: 16px;
  align-items: flex-end;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--on-surface-variant, #c7c4d7);
}

.field-label-icon {
  width: 14px;
  height: 14px;
  color: var(--primary, #c0c1ff);
}

.input-control {
  width: 100%;
  height: 38px;
  padding: 0 12px;
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface, #dce1fb);
  font-size: 13px;
  outline: none;
  transition: all 0.15s ease;
}

.input-control:focus {
  border-color: var(--primary, #c0c1ff);
  box-shadow: 0 0 0 2px rgba(192, 193, 255, 0.15);
}

.font-data {
  font-family: var(--font-code, monospace);
}

.add-button {
  height: 38px;
  padding: 0 18px;
  font-weight: 600;
}

.btn-icon {
  width: 16px;
  height: 16px;
  margin-right: 6px;
}

@media (max-width: 768px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
