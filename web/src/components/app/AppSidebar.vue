<script setup lang="ts">
export interface NavigationItem {
  id: string
  label: string
  icon: 'movie' | 'tv' | 'jobs' | 'settings'
  spinning?: boolean
  disabled?: boolean
}

defineProps<{
  items: NavigationItem[]
  activeSection: string
  username?: string
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  selectSection: [id: string]
  toggleLocale: []
}>()
</script>

<template>
  <nav class="nav-rail" :aria-label="labels.mainNavigation">
    <!-- Top Logo -->
    <div class="rail-logo-box">
      <img src="/logo-icon.svg" alt="MediaGrap" class="rail-logo-img" />
      <span class="rail-pulse-dot" :title="labels.serverConnected"></span>
    </div>

    <!-- Main Navigation Icons -->
    <div class="rail-nav-items">
      <button
        v-for="item in items"
        :key="item.id"
        class="rail-btn"
        :class="{ 'rail-btn--active': activeSection === item.id }"
        :disabled="item.disabled"
        :title="item.label"
        type="button"
        @click="emit('selectSection', item.id)"
      >
        <!-- Movies Icon -->
        <svg v-if="item.icon === 'movie'" class="rail-svg" viewBox="0 0 24 24" fill="currentColor">
          <path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z"/>
        </svg>

        <!-- TV Shows Icon -->
        <svg v-else-if="item.icon === 'tv'" class="rail-svg" viewBox="0 0 24 24" fill="currentColor">
          <path d="M21 3H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h5v2h8v-2h5c1.1 0 1.99-.9 1.99-2L23 5c0-1.1-.9-2-2-2zm0 14H3V5h18v12z"/>
        </svg>

        <!-- Background Jobs Icon -->
        <svg v-else-if="item.icon === 'jobs'" class="rail-svg" :class="{ 'spin-slow': item.spinning }" viewBox="0 0 24 24" fill="currentColor">
          <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
        </svg>

        <!-- Settings Icon -->
        <svg v-else-if="item.icon === 'settings'" class="rail-svg" viewBox="0 0 24 24" fill="currentColor">
          <path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.09.63-.09.94s.02.64.07.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"/>
        </svg>

        <span class="rail-btn-label">{{ item.label }}</span>

      </button>
    </div>

    <!-- Bottom Actions -->
    <div class="rail-bottom-actions">
      <!-- Language Switch -->
      <button class="rail-btn-sm" :title="labels.switchLanguage" type="button" @click="emit('toggleLocale')">
        <span class="lang-txt">{{ labels.language }}</span>
      </button>

      <!-- Admin Avatar -->
      <div class="rail-avatar" :title="username || labels.admin">
        <span class="avatar-letter">{{ (username || labels.admin).charAt(0).toUpperCase() }}</span>
      </div>
    </div>
  </nav>
</template>

<style scoped>
.nav-rail {
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  width: var(--sidebar-width, 64px);
  height: 100vh;
  background: var(--surface-container-lowest, #070d1f);
  border-right: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 0;
  z-index: 50;
}

.rail-logo-box {
  position: relative;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 0.5rem;
}

.rail-logo-img {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.rail-pulse-dot {
  position: absolute;
  bottom: 2px;
  right: 2px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--secondary, #4edea3);
  box-shadow: 0 0 0 2px var(--surface-container-lowest, #070d1f);
}

.rail-nav-items {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  flex: 1;
}

.rail-btn {
  position: relative;
  width: 48px;
  height: 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  border-radius: var(--radius-lg, 0.5rem);
  background: transparent;
  border: 1px solid transparent;
  color: var(--on-surface-variant, #c7c4d7);
  cursor: pointer;
  transition: all 0.15s ease;
  padding: 0;
}

.rail-btn:hover:not(:disabled) {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface, #dce1fb);
}

.rail-btn--active {
  background: var(--surface-container-low, #151b2d);
  color: var(--primary, #c0c1ff);
  border-left: 2px solid var(--primary, #c0c1ff);
}

.rail-svg {
  width: 20px;
  height: 20px;
}

.rail-btn-label {
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.02em;
  line-height: 1;
  max-width: 44px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rail-badge {
  position: absolute;
  top: 2px;
  right: 2px;
  background: var(--primary, #c0c1ff);
  color: var(--on-primary, #1000a9);
  font-family: var(--font-data);
  font-size: 9px;
  font-weight: 700;
  padding: 0 4px;
  border-radius: 9999px;
  line-height: 14px;
  border: 1px solid var(--surface-container-lowest, #070d1f);
}

.rail-bottom-actions {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
}

.rail-btn-sm {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-sm, 0.25rem);
  background: transparent;
  border: 1px solid var(--outline-variant, #2e3447);
  color: var(--on-surface-variant, #c7c4d7);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.rail-btn-sm:hover {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface, #dce1fb);
}

.lang-txt {
  font-size: 11px;
  font-weight: 700;
}

.rail-avatar {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 13px;
  color: var(--primary, #c0c1ff);
}

/* ── Mobile Layout ────────────────────────────────────────── */
@media (max-width: 768px) {
  .nav-rail {
    top: auto;
    bottom: 0;
    left: 0;
    right: 0;
    width: 100%;
    height: auto;
    min-height: 0;
    flex-direction: row;
    justify-content: space-around;
    padding: 0.35rem 0.5rem calc(0.35rem + env(safe-area-inset-bottom));
    border-right: none;
    border-top: 1px solid var(--outline-variant, #2e3447);
    z-index: 50;
  }

  .rail-logo-box,
  .rail-bottom-actions {
    display: none;
  }

  .rail-nav-items {
    flex-direction: row;
    justify-content: space-around;
    width: 100%;
  }

  .rail-btn {
    width: auto;
    min-width: 52px;
    height: 44px;
  }

  .rail-btn--active {
    border-left: none;
    border-bottom: 2px solid var(--primary, #c0c1ff);
  }
}
</style>
