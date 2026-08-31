<script setup lang="ts">
import { shallowRef } from 'vue'

const props = defineProps<{
  setup: boolean
  error: string | null
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  submit: [username: string, password: string]
  toggleLocale: []
}>()

const username = shallowRef('admin')
const password = shallowRef('')

function submit() {
  emit('submit', username.value, password.value)
}
</script>

<template>
  <main class="auth-panel">
    <section class="auth-card">
      <button
        class="locale-btn btn btn-ghost btn-sm"
        type="button"
        @click="emit('toggleLocale')"
      >
        {{ props.labels.language }}
      </button>

      <div class="auth-header">
        <img
          src="/assets/logo-icon.png"
          alt="MediaGrap"
          class="auth-logo"
          width="28"
          height="28"
        />
        <p class="eyebrow">{{ props.labels.authEyebrow }}</p>
        <h1 class="auth-title">{{ props.setup ? props.labels.setup : props.labels.welcome }}</h1>
        <p class="auth-subtitle">
          {{ props.setup ? props.labels.setupIntro : props.labels.signInIntro }}
        </p>
      </div>

      <form class="auth-form" @submit.prevent="submit">
        <div class="field-group">
          <label class="field-label" for="auth-username">{{ props.labels.username }}</label>
          <input
            id="auth-username"
            v-model="username"
            class="auth-input"
            autocomplete="username"
            minlength="3"
            required
          />
        </div>

        <div class="field-group">
          <label class="field-label" for="auth-password">{{ props.labels.password }}</label>
          <input
            id="auth-password"
            v-model="password"
            type="password"
            class="auth-input"
            :autocomplete="props.setup ? 'new-password' : 'current-password'"
            minlength="12"
            required
          />
        </div>

        <p v-if="props.error" class="form-error" role="alert">{{ props.error }}</p>

        <button type="submit" class="btn btn-primary auth-submit">
          {{ props.setup ? props.labels.create : props.labels.signIn }}
        </button>
      </form>
    </section>
  </main>
</template>

<style scoped>
.auth-panel {
  display: grid;
  min-height: 100vh;
  place-items: center;
  padding: 1.5rem;
  background: var(--surface-base);
}

.auth-card {
  position: relative;
  width: min(100%, 28rem);
  padding: clamp(1.75rem, 5vw, 2.5rem);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xl);
  background: var(--surface-card);
}

.locale-btn {
  position: absolute;
  top: 1rem;
  right: 1rem;
  color: var(--text-secondary);
}

.locale-btn:hover {
  background: var(--surface-elevated);
  color: var(--text-primary);
}

.auth-header {
  text-align: center;
}

.auth-logo {
  display: block;
  width: 28px;
  height: 28px;
  margin: 0 auto 0.75rem;
  border-radius: var(--radius-sm);
}

.eyebrow {
  margin: 0;
  color: var(--text-muted);
  font: 500 0.65rem/1.4 var(--font-data);
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.auth-title {
  margin: 0.4rem 0 0.5rem;
  font-family: var(--font-display);
  font-size: clamp(1.6rem, 4vw, 2.1rem);
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: -0.03em;
  line-height: 1.15;
}

.auth-subtitle {
  margin: 0;
  color: var(--text-secondary);
  font-size: 0.875rem;
  line-height: 1.5;
}

.auth-form {
  display: grid;
  gap: 1.15rem;
  margin-top: 1.75rem;
}

.field-group {
  display: grid;
  gap: 0.35rem;
}

.field-label {
  font-weight: 600;
  font-size: 0.8125rem;
  color: var(--text-secondary);
}

.auth-input {
  width: 100%;
  min-height: 2.5rem;
  padding: 0.55rem 0.75rem;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-size: 0.875rem;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.auth-input:focus {
  outline: none;
  border-color: var(--border-focus);
  box-shadow: 0 0 0 2px rgba(128, 131, 255, 0.2);
}

.form-error {
  margin: 0;
  padding: 0.5rem 0.75rem;
  background: rgba(244, 63, 94, 0.1);
  border: 1px solid rgba(244, 63, 94, 0.25);
  border-radius: var(--radius-sm);
  color: var(--error);
  font-size: 0.8125rem;
  line-height: 1.4;
}

.auth-submit {
  width: 100%;
  min-height: 2.6rem;
  margin-top: 0.35rem;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
}
</style>
