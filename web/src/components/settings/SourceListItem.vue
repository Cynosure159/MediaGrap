<script setup lang="ts">
import { reactive } from 'vue'
import type { Source } from '@/api/library'

const emit = defineEmits<{
  (e: 'scan', id: number): void
  (e: 'delete', id: number): void
	(e: 'savePolicy', id: number, policy: Pick<Source, 'scanMode' | 'scheduleEnabled' | 'scheduleIntervalMinutes'>): void
}>()

const props = defineProps<{ source: Source; labels: Record<string, string> }>()
const policy = reactive({ scanMode: props.source.scanMode, scheduleEnabled: props.source.scheduleEnabled, scheduleIntervalMinutes: props.source.scheduleIntervalMinutes })
</script>

<template>
  <div class="source-card">
    <!-- Left Status Indicator Bar -->
    <div
      class="source-accent-bar"
      :class="source.writable ? 'source-accent-bar--ok' : 'source-accent-bar--warn'"
    ></div>

    <!-- Main Body Content -->
    <div class="source-card__content">
      <!-- Source Type Icon Container -->
      <div class="source-icon-badge">
        <svg
          v-if="source.name.toLowerCase().includes('show') || source.name.toLowerCase().includes('tv')"
          class="source-icon"
          viewBox="0 0 24 24"
          fill="currentColor"
        >
          <path d="M21 3H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h5v2h8v-2h5c1.1 0 1.99-.9 1.99-2L23 5c0-1.1-.9-2-2-2zm0 14H3V5h18v12z"/>
        </svg>
        <svg
          v-else
          class="source-icon"
          viewBox="0 0 24 24"
          fill="currentColor"
        >
          <path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
        </svg>
      </div>

      <!-- Title, Path & Meta -->
      <div class="source-info">
        <div class="source-info__top">
          <h3 class="source-name">{{ source.name }}</h3>
          <span
            class="status-pill"
            :class="source.writable ? 'status-pill--ok' : 'status-pill--warn'"
            :title="source.writable ? labels.writable : labels.readOnly"
          >
            <span class="status-dot-inner" :class="source.writable ? 'dot-ok' : 'dot-warn'"></span>
            <span>{{ source.writable ? labels.writable : labels.readOnly }}</span>
          </span>
        </div>

        <div class="source-info__meta">
          <span class="source-path-code">{{ source.rootPath }}</span>
          <span class="meta-dot-sep">•</span>
          <span class="spec-pill">
            <svg class="spec-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M14 2H6c-1.1 0-1.99.9-1.99 2L4 20c0 1.1.89 2 1.99 2H18c1.1 0 2-.9 2-2V8l-6-6zm2 16H8v-2h8v2zm0-4H8v-2h8v2zm-3-5V3.5L18.5 9H13z"/>
            </svg>
            {{ source.itemCount }} {{ labels.indexed }}
          </span>
        </div>
		<div class="source-policy">
		  <label>{{ labels.scanStrategy }}
			<select v-model="policy.scanMode" class="policy-control"><option value="incremental">{{ labels.incrementalScan }}</option><option value="full">{{ labels.fullScan }}</option></select>
		  </label>
		  <label class="schedule-toggle"><input v-model="policy.scheduleEnabled" type="checkbox" /> {{ labels.scheduledScan }}</label>
		  <label v-if="policy.scheduleEnabled">{{ labels.scanInterval }}
			<input v-model.number="policy.scheduleIntervalMinutes" class="policy-control interval" type="number" min="15" max="10080" />
		  </label>
		</div>
      </div>

      <!-- Action Buttons (Scan + Remove) -->
      <div class="source-actions">
		<button type="button" class="btn btn-primary btn-sm action-btn" @click="emit('savePolicy', source.id, { ...policy })">{{ labels.savePolicy }}</button>
        <button
          type="button"
          class="btn btn-outline btn-sm action-btn"
          :title="labels.scan || 'Scan'"
          @click="emit('scan', source.id)"
        >
          <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
          </svg>
          <span>{{ labels.scan || '扫描' }}</span>
        </button>

        <button
          type="button"
          class="btn btn-outline btn-sm action-btn btn-danger-hover"
          :title="labels.delete || 'Delete'"
          @click="emit('delete', source.id)"
        >
          <svg class="btn-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
          </svg>
          <span>{{ labels.delete || '移除' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.source-card {
  position: relative;
  display: flex;
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  overflow: hidden;
  transition: all 0.2s ease;
}

.source-card:hover {
  border-color: var(--outline, #908fa0);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
  background: var(--surface-container, #191f31);
}

.source-accent-bar {
  width: 4px;
  flex-shrink: 0;
}

.source-accent-bar--ok {
  background: var(--secondary, #4edea3);
}

.source-accent-bar--warn {
  background: #faad14;
}

.source-card__content {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 18px;
  min-width: 0;
}

.source-icon-badge {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md, 0.375rem);
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary, #c0c1ff);
  flex-shrink: 0;
}

.source-icon {
  width: 20px;
  height: 20px;
}

.source-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.source-info__top {
  display: flex;
  align-items: center;
  gap: 10px;
}

.source-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 1px 8px;
  border-radius: 9999px;
  font-size: 11px;
  font-weight: 500;
}

.status-pill--ok {
  background: rgba(78, 222, 163, 0.1);
  color: var(--secondary, #4edea3);
  border: 1px solid rgba(78, 222, 163, 0.25);
}

.status-pill--warn {
  background: rgba(250, 173, 20, 0.1);
  color: #faad14;
  border: 1px solid rgba(250, 173, 20, 0.25);
}

.status-dot-inner {
  width: 5px;
  height: 5px;
  border-radius: 50%;
}

.source-info__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  overflow: hidden;
}

.source-path-code {
  font-family: var(--font-code, monospace);
  font-size: 11px;
  color: var(--outline, #908fa0);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.meta-dot-sep {
  color: var(--outline-variant, #2e3447);
  font-size: 10px;
}

.spec-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 1px 6px;
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-high, #23293c);
  font-size: 11px;
  color: var(--on-surface-variant, #c7c4d7);
  white-space: nowrap;
}

.spec-icon {
  width: 12px;
  height: 12px;
  color: var(--outline, #908fa0);
}

.source-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.source-policy { display: flex; flex-wrap: wrap; align-items: center; gap: 8px 12px; margin-top: 5px; }
.source-policy label { display: inline-flex; align-items: center; gap: 6px; color: var(--outline); font-size: 11px; }
.policy-control { height: 26px; border: 1px solid var(--outline-variant); border-radius: var(--radius-sm); background: var(--surface-container-lowest); color: var(--on-surface); padding: 0 7px; font: 11px var(--font-data); }
.policy-control.interval { width: 74px; }
.schedule-toggle { cursor: pointer; }

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  font-size: 12px;
}

.btn-icon {
  width: 14px;
  height: 14px;
}

.btn-danger-hover:hover {
  border-color: rgba(255, 77, 79, 0.5);
  color: #ff4d4f;
  background: rgba(255, 77, 79, 0.1);
}

@media (max-width: 768px) {
  .source-card__content {
    flex-direction: column;
    align-items: flex-start;
  }
  .source-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
