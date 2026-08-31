<script setup lang="ts">
export interface CastMember {
  id: string
  name: string
  role: string
  avatar?: string
}

defineProps<{
  cast: CastMember[]
  labels: Record<string, string>
}>()
</script>

<template>
  <section class="cast-workshop-view">
    <div class="workshop-header">
      <div class="header-left">
        <h2>{{ labels.castAndCrew || 'Cast & Crew Workshop' }}</h2>
        <span class="sub-label">Directors, writers, production and starring actors</span>
      </div>
      <button class="btn btn-outline" type="button">
        <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 3c1.66 0 3 1.34 3 3s-1.34 3-3 3-3-1.34-3-3 1.34-3 3-3zm0 14.2c-2.5 0-4.71-1.28-6-3.22.03-1.99 4-3.08 6-3.08 1.99 0 5.97 1.09 6 3.08-1.29 1.94-3.5 3.22-6 3.22z"/>
        </svg>
        {{ labels.scrapeHeadshots || 'Scrape Headshots' }}
      </button>
    </div>

    <div v-if="cast.length === 0" class="cast-empty">
      <svg viewBox="0 0 24 24" fill="currentColor" width="48" height="48" opacity="0.2">
        <path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/>
      </svg>
      <p>{{ labels.noCastFound || 'No cast and crew information available.' }}</p>
    </div>

    <div v-else class="cast-grid">
      <div v-for="person in cast" :key="person.id" class="cast-card">
        <div class="cast-avatar">
          <img v-if="person.avatar" :src="person.avatar" :alt="person.name" class="avatar-img" />
          <span v-else class="avatar-placeholder font-code">{{ person.name.charAt(0) }}</span>
        </div>
        <div class="cast-info">
          <strong class="cast-name">{{ person.name }}</strong>
          <span class="cast-role">{{ person.role }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.cast-workshop-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}

.workshop-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 4px;
}

.header-left h2 {
  font-size: 18px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  margin: 0 0 2px 0;
  letter-spacing: -0.01em;
}

.sub-label {
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
}

.cast-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}

.cast-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  padding: 10px 12px;
  display: flex;
  align-items: center;
  gap: 12px;
  transition: border-color 0.15s ease;
}

.cast-card:hover {
  border-color: var(--outline, #908fa0);
}

.cast-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: var(--surface-container-high, #23293c);
  border: 1px solid var(--outline-variant, #2e3447);
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-placeholder {
  font-size: 16px;
  font-weight: 700;
  color: var(--primary, #c0c1ff);
}

.cast-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.cast-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.cast-role {
  font-size: 11px;
  color: var(--outline, #908fa0);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.cast-empty {
  color: var(--outline, #908fa0);
  font-size: 13px;
  text-align: center;
  padding: 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}
</style>
