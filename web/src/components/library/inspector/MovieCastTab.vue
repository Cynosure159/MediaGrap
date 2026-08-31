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
  <section class="workshop-view">
    <div class="workshop-header">
      <h2>{{ labels.castAndCrew }}</h2>
      <button class="btn btn-outline" type="button">{{ labels.scrapeHeadshots }}</button>
    </div>

    <div v-if="cast.length === 0" class="cast-empty">
      <p>{{ labels.noCastFound || 'No cast and crew information available.' }}</p>
    </div>

    <div v-else class="cast-grid">
      <div v-for="person in cast" :key="person.id" class="cast-card">
        <div class="cast-avatar">
          <img v-if="person.avatar" :src="person.avatar" :alt="labels.avatarAlt" class="avatar-img" />
          <span v-else class="avatar-placeholder">{{ person.name.charAt(0) }}</span>
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
.workshop-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.workshop-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.workshop-header h2 {
  font-size: 16px;
  font-weight: 600;
  color: var(--on-surface, #dce1fb);
  margin: 0;
}

.cast-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 12px;
}

.cast-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-md, 0.5rem);
  padding: 10px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.cast-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--surface-container-high, #23293c);
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
  font-size: 14px;
  font-weight: 600;
  color: var(--primary, #c0c1ff);
}

.cast-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.cast-name {
  font-size: 12px;
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
  padding: 32px;
}
</style>
