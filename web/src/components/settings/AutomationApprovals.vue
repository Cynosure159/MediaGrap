<script setup lang="ts">
import { computed, onMounted, shallowRef } from 'vue'
import { automationPlans, mutate, type AutomationPlan } from '@/api/integrations'

const props = defineProps<{
  csrfToken: string
  locale: string
}>()

const zh = computed(() => props.locale === 'zh-CN')
const plans = shallowRef<AutomationPlan[]>([])
const selectedID = shallowRef(new URLSearchParams(window.location.search).get('approval') ?? '')
const selected = computed(() => plans.value.find(plan => plan.id === selectedID.value))
const busy = shallowRef(false)
const error = shallowRef('')
const confirmed = shallowRef(false)

async function refresh() {
  busy.value = true
  error.value = ''
  try {
    const res = await automationPlans()
    plans.value = res.items
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}

function select(plan: AutomationPlan) {
  selectedID.value = plan.id
  confirmed.value = false
}

async function approve() {
  if (!selected.value || !confirmed.value || busy.value) return
  const plan = selected.value
  busy.value = true
  error.value = ''
  try {
    await mutate(props.csrfToken, `automation/plans/${plan.id}/approve`, 'POST', {
      version: plan.version,
      digest: plan.digest,
    })
    confirmed.value = false
    const res = await automationPlans()
    plans.value = res.items
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}

onMounted(refresh)
</script>

<template>
  <section class="settings-card" aria-labelledby="approvals-title">
    <div class="card-header">
      <div class="card-header__info">
        <div class="header-tag">
          <svg class="header-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm-2 16l-4-4 1.41-1.41L10 14.17l6.59-6.59L18 9l-8 8z"/>
          </svg>
          <span class="eyebrow">{{ zh ? '安全审计与执行' : 'SECURITY & AUDIT' }}</span>
        </div>
        <h2 id="approvals-title" class="section-title">{{ zh ? '自动化文件审批' : 'Automation File Approvals' }}</h2>
        <p class="section-hint">
          {{ zh
            ? '审查由 MCP 智能体或外部脚本发起的写入修改计划。批准有效期为 5 分钟。'
            : 'Review and approve file write/replace plans from MCP agents or automation. Approval lasts 5 minutes.'
          }}
        </p>
      </div>
      <div class="card-header__actions">
        <button type="button" class="btn btn-outline btn-sm" :disabled="busy" @click="refresh">
          <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14" :class="{ 'spin-slow': busy }">
            <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
          </svg>
          <span>{{ zh ? '刷新' : 'Refresh' }}</span>
        </button>
      </div>
    </div>

    <div class="card-body">
      <div v-if="error" role="alert" class="approval-banner approval-banner--error">
        <span class="dot dot-err"></span>
        <span>{{ error }}</span>
      </div>

      <div v-if="!plans.length" class="empty-state">
        <svg viewBox="0 0 24 24" fill="currentColor" width="32" height="32" class="empty-icon">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
        </svg>
        <p class="empty-text">{{ zh ? '当前无待审批的文件变更计划。' : 'No pending file plans requiring approval.' }}</p>
      </div>

      <div v-else class="plans-container">
        <div class="plan-list">
          <button
            v-for="plan in plans"
            :key="plan.id"
            type="button"
            class="plan-select btn btn-outline"
            :class="{ 'plan-select--active': plan.id === selectedID }"
            :aria-pressed="plan.id === selectedID"
            @click="select(plan)"
          >
            <span class="plan-kind">{{ plan.kind }}</span>
            <span class="plan-media">#{{ plan.mediaId }}</span>
            <span class="spec-pill">{{ plan.state }}</span>
          </button>
        </div>

        <article v-if="selected" class="plan-detail-card">
          <div class="plan-detail__meta">
            <div class="meta-row">
              <span class="meta-label">TOKEN ID:</span>
              <code class="plan-code">{{ selected.tokenId }}</code>
            </div>
            <div class="meta-row">
              <span class="meta-label">DIGEST:</span>
              <code class="plan-code">{{ selected.digest }}</code>
            </div>
          </div>

          <div class="plan-operations-list">
            <div v-for="(item, index) in selected.items" :key="index" class="plan-operation-item">
              <div class="op-header">
                <div class="op-path-box">
                  <span v-if="item.source" class="op-source font-code">{{ item.source }} &rarr; </span>
                  <span class="op-target font-code">{{ item.target }}</span>
                </div>
                <div class="op-badges">
                  <span class="spec-pill">{{ item.state }}</span>
                  <span v-if="item.willReplace" class="spec-pill spec-pill--warn">
                    {{ zh ? '覆盖已有文件' : 'Replaces existing file' }}
                  </span>
                  <span v-if="item.errorCode" class="spec-pill spec-pill--err">{{ item.errorCode }}</span>
                </div>
              </div>

              <div v-if="item.candidateId" class="op-candidate">
                <span class="candidate-id font-code">{{ item.candidateId }}</span>
                <img
                  class="approval-artwork"
                  :src="`/api/v1/media/${selected.mediaId}/artwork-preview/${encodeURIComponent(item.candidateId)}`"
                  :alt="item.target"
                />
              </div>

              <details v-if="item.content" class="nfo-details">
                <summary class="nfo-summary">{{ zh ? '查看 NFO 源码内容' : 'Inspect NFO content' }}</summary>
                <pre class="plan-content font-code">{{ item.content }}</pre>
              </details>
            </div>
          </div>

          <div v-if="selected.state === 'previewed' && selected.expiresAt > Date.now()" class="approval-action-bar">
            <label class="approval-check">
              <input v-model="confirmed" type="checkbox" :disabled="busy" class="custom-check" />
              <span>{{ zh ? '我已确认审查全部目标文件及覆盖内容，批准此安全计划。' : 'I reviewed every target and replacement and approve this exact plan.' }}</span>
            </label>
            <button type="button" class="btn btn-primary" :disabled="busy || !confirmed" @click="approve">
              <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
                <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
              </svg>
              <span>{{ zh ? '批准此计划' : 'Approve this plan' }}</span>
            </button>
          </div>
        </article>
      </div>
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
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px 24px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-container-high, #23293c);
  gap: 16px;
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
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.approval-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-sm, 0.25rem);
  font-size: 12px;
}

.approval-banner--error {
  background: rgba(244, 63, 94, 0.1);
  border: 1px solid rgba(244, 63, 94, 0.3);
  color: var(--error, #ffb4ab);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px 0;
  gap: 8px;
}

.empty-icon {
  color: var(--outline-variant, #2e3447);
}

.empty-text {
  margin: 0;
  font-size: 13px;
  color: var(--outline, #908fa0);
}

.plans-container {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.plan-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.plan-select {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  font-size: 12px;
  border-radius: var(--radius-sm, 0.25rem);
}

.plan-select--active {
  background: var(--surface-container-highest, #2e3447);
  border-color: var(--primary, #c0c1ff);
  color: var(--primary, #c0c1ff);
}

.plan-detail-card {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-lg, 0.5rem);
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.plan-detail__meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
}

.meta-label {
  font-weight: 700;
  color: var(--outline, #908fa0);
}

.plan-code {
  font-family: var(--font-data, monospace);
  color: var(--on-surface, #dce1fb);
  overflow-wrap: anywhere;
}

.font-code {
  font-family: var(--font-data, monospace);
}

.plan-operations-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.plan-operation-item {
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.op-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.op-path-box {
  font-size: 12px;
  color: var(--on-surface, #dce1fb);
}

.op-source {
  color: var(--outline, #908fa0);
}

.op-target {
  font-weight: 600;
  color: var(--primary, #c0c1ff);
}

.op-badges {
  display: flex;
  align-items: center;
  gap: 6px;
}

.spec-pill--warn {
  background: rgba(255, 185, 95, 0.15);
  border-color: rgba(255, 185, 95, 0.3);
  color: var(--tertiary, #ffb95f);
}

.spec-pill--err {
  background: rgba(244, 63, 94, 0.15);
  border-color: rgba(244, 63, 94, 0.3);
  color: var(--error, #ffb4ab);
}

.approval-artwork {
  display: block;
  max-width: 140px;
  max-height: 200px;
  object-fit: contain;
  border-radius: var(--radius-sm, 0.25rem);
  border: 1px solid var(--outline-variant, #2e3447);
  margin-top: 6px;
}

.nfo-details {
  background: var(--surface-container-lowest, #070d1f);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 8px 12px;
}

.nfo-summary {
  font-size: 11px;
  font-weight: 600;
  color: var(--primary, #c0c1ff);
  cursor: pointer;
}

.plan-content {
  margin: 8px 0 0;
  font-size: 11px;
  color: var(--on-surface-variant, #c7c4d7);
  line-height: 1.4;
  white-space: pre-wrap;
  max-height: 200px;
  overflow-y: auto;
}

.approval-action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  padding-top: 10px;
  border-top: 1px solid var(--outline-variant, #2e3447);
}

.approval-check {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--on-surface, #dce1fb);
  cursor: pointer;
}

.custom-check {
  accent-color: var(--primary-container, #6366f1);
  cursor: pointer;
}
</style>
