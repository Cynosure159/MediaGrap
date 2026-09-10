import { request } from './client'

export interface Webhook {
  id: string
  name: string
  url: string
  enabled: boolean
  eventTypes: string[]
  sourceIds: number[]
  version: number
  keyId: string
}

export interface APIToken {
  id: string
  name: string
  prefix: string
  scopes: string[]
  sourceIds: number[]
  expiresAt: number
  revokedAt: number | null
}

export interface Delivery {
  id: string
  eventId: string
  state: string
  attempts: number
  status: number
  errorCode: string
  createdAt: number
}

export interface IntegrationStatus {
  signingReady: boolean
  paused: boolean
  undispatched: number
  pendingDeliveries: number
  deadDeliveries: number
}

export interface IntegrationInput {
  name: string
  url: string
  sourceIds: number[]
  expiresAt: number
  eventTypes: string[]
  scopes: string[]
}

export interface AutomationPlanItem {
  kind: string
  source?: string
  target: string
  willReplace: boolean
  state: string
  content?: string
  candidateId?: string
  errorCode?: string
}

export interface AutomationPlan {
  id: string
  version: number
  digest: string
  tokenId: string
  sourceId: number
  mediaId: number
  kind: string
  state: string
  expiresAt: number
  recoverability: string
  items: AutomationPlanItem[]
}

export function mutate<T>(csrf: string, path: string, method: string, value?: unknown) {
  return request<T>(`/api/v1/${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': csrf,
    },
    body: value === undefined ? undefined : JSON.stringify(value),
  })
}

export const listWebhooks = () => request<{ items: Webhook[] }>('/api/v1/webhooks')
export const listTokens = () => request<{ items: APIToken[] }>('/api/v1/api-tokens')
export const integrationStatus = () => request<IntegrationStatus>('/api/v1/integrations/status')
export const deliveries = (id: string) => request<{ items: Delivery[] }>(`/api/v1/webhooks/${id}/deliveries`)
export const automationPlans = () => request<{ items: AutomationPlan[] }>('/api/v1/automation/plans')
