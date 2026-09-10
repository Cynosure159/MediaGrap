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
  <section class="card approvals" aria-labelledby="approvals-title">
    <header class="approval-header">
      <h3 id="approvals-title">{{ zh ? '自动化文件审批' : 'Automation file approvals' }}</h3>
      <button class="btn-ghost" :disabled="busy" @click="refresh">
        {{ zh ? '刷新' : 'Refresh' }}
      </button>
    </header>

    <p>
      {{ zh
        ? '检查具体文件和覆盖内容后批准。批准有效期为 5 分钟，客户端随后才能提交执行。部分完成无法整体回滚。'
        : 'Review the files and replacements before approving. Approval lasts 5 minutes and allows the client to queue execution. Partial changes cannot be rolled back as a group.'
      }}
    </p>

    <p v-if="error" role="alert">{{ error }}</p>
    <p v-if="!plans.length">{{ zh ? '暂无计划。' : 'No plans yet.' }}</p>

    <div class="plan-list">
      <button
        v-for="plan in plans"
        :key="plan.id"
        class="btn-ghost plan-select"
        :aria-pressed="plan.id === selectedID"
        @click="select(plan)"
      >
        {{ plan.kind }} · {{ plan.mediaId }}
        <span class="spec-pill">{{ plan.state }}</span>
      </button>
    </div>

    <article v-if="selected" class="plan-detail">
      <p class="plan-code">{{ selected.tokenId }}<br />{{ selected.digest }}</p>
      <div v-for="(item, index) in selected.items" :key="index" class="plan-operation">
        <p class="plan-code">
          {{ item.source ? `${item.source} → ` : '' }}{{ item.target }}
        </p>
        <span class="spec-pill">{{ item.state }}</span>
        <strong v-if="item.willReplace">
          {{ zh ? '覆盖已有文件' : 'Replaces existing file' }}
        </strong>
        <span v-if="item.errorCode"> · {{ item.errorCode }}</span>
        <p v-if="item.candidateId" class="plan-code">{{ item.candidateId }}</p>
        <img
          v-if="item.candidateId"
          class="approval-artwork"
          :src="`/api/v1/media/${selected.mediaId}/artwork-preview/${encodeURIComponent(item.candidateId)}`"
          :alt="item.target"
        />
        <details v-if="item.content">
          <summary>{{ zh ? '查看 NFO 内容' : 'Inspect NFO content' }}</summary>
          <pre class="plan-content">{{ item.content }}</pre>
        </details>
      </div>

      <template v-if="selected.state === 'previewed' && selected.expiresAt > Date.now()">
        <label class="approval-check">
          <input v-model="confirmed" type="checkbox" :disabled="busy" />
          {{ zh ? '我已检查全部目标文件及覆盖内容，批准此计划。' : 'I reviewed every target and replacement and approve this exact plan.' }}
        </label>
        <button class="btn-primary" :disabled="busy || !confirmed" @click="approve">
          {{ zh ? '批准此计划' : 'Approve this plan' }}
        </button>
      </template>
    </article>
  </section>
</template>

<style scoped>
.approvals {
  margin-top: 1.5rem;
  padding: 1.25rem;
  font-size: 0.8125rem;
}
.approval-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.plan-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
.plan-select {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}
.plan-detail {
  margin-top: 1rem;
  background: var(--surface-container-high);
  border-radius: 0.5rem;
  padding: 1rem;
}
.plan-code {
  font-family: var(--font-data);
  font-size: 0.7rem;
  overflow-wrap: anywhere;
}
.plan-operation {
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--outline-variant);
}
.plan-content {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-family: var(--font-data);
  font-size: 0.75rem;
  max-height: 20rem;
  overflow-y: auto;
}
.approval-artwork {
  display: block;
  max-width: 180px;
  max-height: 240px;
  object-fit: contain;
  margin: 0.75rem 0;
}
.approval-check {
  display: flex;
  gap: 0.5rem;
  align-items: flex-start;
  margin: 1rem 0;
}
</style>
