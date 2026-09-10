<script setup lang="ts">
import type { ConnectionTest, Settings, SettingsUpdate } from '@/api/library'

defineProps<{
  settings: Settings | null
  labels: Record<string, string>
  saving: boolean
  connectionTests: Record<string, ConnectionTest | null>
  testingTarget: string | null
}>()

const model = defineModel<SettingsUpdate>('model', { required: true })
const emit = defineEmits<{ save: []; test: [target: ConnectionTest['target']] }>()
</script>

<template>
  <section class="settings-card">
    <div class="card-header">
      <div class="card-header__info">
        <div class="header-tag">
          <svg class="header-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20.2 5.9l.8-.8C19.6 3.7 17.8 3 16 3s-3.6.7-5 2.1l.8.8C13 4.8 14.5 4.2 16 4.2s3 .6 4.2 1.7zm2.8-2.8l.8-.8C21.6.8 18.9 0 16 0S10.4.8 8.2 2.3l.8.8C10.8 1.9 13.3 1.2 16 1.2s5.2.7 7 1.9zM19 13h-2V9h-2v4H5c-1.1 0-2 .9-2 2v4c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2v-4c0-1.1-.9-2-2-2zM6 18H4v-2h2v2zm3 0H7v-2h2v2zm3 0h-2v-2h2v2zm7 0h-2v-2h2v2z"/>
          </svg>
          <span class="eyebrow">NETWORK &amp; ROUTING</span>
        </div>
        <h2 class="section-title">{{ labels.outboundProxy || '出站网络与代理配置' }}</h2>
        <p class="section-hint">{{ labels.proxyHelp || '配置服务刮削 TMDb 与 Fanart.tv 图片时的 HTTP / SOCKS5 代理路由。' }}</p>
      </div>
    </div>

    <div class="card-body">
      <form class="settings-form" @submit.prevent="emit('save')">
        <!-- Outbound Proxy -->
        <div class="field-group">
          <div class="field-header">
            <label for="network-outbound-proxy" class="field-label">{{ labels.outboundProxy }}</label>
            <span
              class="status-pill"
              :class="settings?.outboundProxyConfigured ? 'status-pill--configured' : 'status-pill--unset'"
            >
              <span
                class="status-dot"
                :class="settings?.outboundProxyConfigured ? 'status-dot--ok' : 'status-dot--neutral'"
              ></span>
              {{ settings?.outboundProxyConfigured ? labels.configured : labels.notSet }}
            </span>
          </div>
          <input
            id="network-outbound-proxy"
            v-model="model.outboundProxy"
            type="text"
            autocomplete="off"
            class="input-control font-code"
            :placeholder="settings?.outboundProxyConfigured ? labels.proxyConfigured : labels.proxyPlaceholder"
          />
          <p class="field-hint">{{ labels.proxyPlaceholder }}</p>

          <label v-if="settings?.outboundProxyConfigured" class="custom-checkbox-row">
            <input v-model="model.clearOutboundProxy" type="checkbox" class="sr-only" />
            <span class="custom-checkbox-box" :class="{ 'custom-checkbox-box--checked': model.clearOutboundProxy }">
              <svg v-if="model.clearOutboundProxy" viewBox="0 0 24 24" fill="currentColor" width="11" height="11">
                <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
              </svg>
            </span>
            <span class="custom-checkbox-label" :class="{ 'custom-checkbox-label--checked': model.clearOutboundProxy }">
              {{ labels.clearProxy }}
            </span>
          </label>

          <div class="test-row">
            <button
              type="button"
              class="btn btn-outline btn-sm"
              :disabled="testingTarget !== null || !settings?.outboundProxyConfigured"
              @click="emit('test', 'proxy')"
            >
              {{ testingTarget === 'proxy' ? labels.testingConnection : labels.testProxy }}
            </button>
            <p v-if="connectionTests.proxy" class="test-result" :class="`test-result--${connectionTests.proxy.status}`">
              {{ labels[`connection_${connectionTests.proxy.status}`] }} · {{ connectionTests.proxy.durationMs }} ms
            </p>
          </div>
        </div>

        <!-- NO_PROXY -->
        <div class="field-group">
          <div class="field-header">
            <label for="network-no-proxy" class="field-label">NO_PROXY</label>
            <span class="status-pill" :class="settings?.noProxyConfigured ? 'status-pill--configured' : 'status-pill--unset'">
              {{ settings?.noProxyConfigured ? labels.configured : labels.notSet }}
            </span>
          </div>
          <input
            id="network-no-proxy"
            v-model="model.noProxy"
            type="text"
            autocomplete="off"
            class="input-control font-code"
            :placeholder="labels.noProxyPlaceholder"
          />
          <p class="field-hint">{{ labels.noProxyHelp }}</p>
          <label v-if="settings?.noProxyConfigured" class="custom-checkbox-row">
            <input v-model="model.clearNoProxy" type="checkbox" class="sr-only" />
            <span class="custom-checkbox-label">{{ labels.clearNoProxy }}</span>
          </label>
        </div>

        <!-- Submit Button -->
        <div class="form-actions">
          <button type="submit" class="btn btn-primary" :disabled="saving">
            <svg v-if="saving" viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="spin-slow">
              <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/>
            </svg>
            <span>{{ saving ? labels.saving : (labels.saveSettings || '保存网络设置') }}</span>
          </button>
        </div>
      </form>
    </div>
  </section>
</template>

<style scoped>
.settings-card {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-xl, 0.75rem);
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
}

.card-header {
  padding: 20px 24px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-container-high, #23293c);
}

.card-header__info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.header-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--primary, #c0c1ff);
}

.header-icon {
  width: 14px;
  height: 14px;
}

.eyebrow {
  margin: 0;
  font-family: var(--font-data, monospace);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--primary, #c0c1ff);
}

.section-title {
  margin: 0;
  color: var(--on-surface, #dce1fb);
  font-size: 17px;
  font-weight: 700;
  line-height: 1.3;
  letter-spacing: -0.01em;
}

.section-hint {
  margin: 0;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  line-height: 1.4;
}

.card-body {
  padding: 24px;
}

.settings-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.field-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--on-surface-variant, #c7c4d7);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.field-hint {
  margin: 0;
  font-size: 11px;
  color: var(--outline, #908fa0);
  line-height: 1.4;
}

.input-control {
  width: 100%;
  height: 38px;
  padding: 0 12px;
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-low, #151b2d);
  color: var(--on-surface, #dce1fb);
  font-size: 13px;
  outline: none;
  transition: all 0.15s ease;
}

.input-control:focus {
  border-color: var(--primary, #c0c1ff);
  box-shadow: 0 0 0 2px rgba(192, 193, 255, 0.15);
}

.font-code {
  font-family: var(--font-data, monospace);
  font-size: 12px;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 8px;
  border-radius: 9999px;
  font-size: 11px;
  font-weight: 600;
}

.status-pill--configured {
  background: rgba(78, 222, 163, 0.1);
  color: var(--secondary, #4edea3);
  border: 1px solid rgba(78, 222, 163, 0.25);
}

.status-pill--unset {
  background: var(--surface-container-high, #23293c);
  color: var(--outline, #908fa0);
  border: 1px solid var(--outline-variant, #2e3447);
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.status-dot--ok {
  background: var(--secondary, #4edea3);
}

.status-dot--neutral {
  background: var(--outline, #908fa0);
}

.custom-checkbox-row {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
  margin-top: 2px;
}

.custom-checkbox-box {
  width: 16px;
  height: 16px;
  border-radius: 3px;
  border: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-container-low, #151b2d);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--on-primary, #1000a9);
}

.custom-checkbox-box--checked {
  background: var(--primary, #c0c1ff);
  border-color: var(--primary, #c0c1ff);
}

.custom-checkbox-label {
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
}

.test-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 4px;
}

.test-result {
  margin: 0;
  font: 11px var(--font-data, monospace);
}

.test-result--reachable {
  color: var(--secondary, #4edea3);
}

.test-result--failed {
  color: var(--error, #ffb4ab);
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border-width: 0;
}

.form-actions {
  display: flex;
  justify-content: flex-start;
  padding-top: 6px;
}
</style>
