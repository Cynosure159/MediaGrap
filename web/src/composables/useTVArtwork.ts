import { computed, reactive, shallowRef } from 'vue'
import * as api from '@/api/library'

export interface TVArtworkContext {
  scope: 'show' | 'season'
  seasonNumber?: number
}

export function useTVArtwork(
  showId: () => number,
  context: () => TVArtworkContext,
  csrfToken: () => string,
  hasExistingArtwork: (kind: api.TVArtworkCandidate['kind']) => boolean = () => false,
) {
  const candidates = shallowRef<api.TVArtworkCandidate[]>([])
  const plan = shallowRef<api.TVArtworkPlan | null>(null)
  const error = shallowRef<string | null>(null)
  const isLoading = shallowRef(false)
  const isApplying = shallowRef(false)
  const selected = reactive<Record<string, string>>({})

  const groups = computed(() => {
    const order: api.TVArtworkCandidate['kind'][] = context().scope === 'season'
      ? ['season_poster', 'season_banner', 'season_landscape']
      : ['poster', 'fanart', 'clearlogo', 'logo', 'clearart', 'banner', 'landscape', 'character']
    return order.map(kind => ({ kind, items: candidates.value.filter(candidate => candidate.kind === kind) })).filter(group => group.items.length > 0)
  })

  function selectDefaultsForMissingArtwork() {
    for (const key of Object.keys(selected)) delete selected[key]
    for (const group of groups.value) {
      if (!hasExistingArtwork(group.kind) && group.items[0]) selected[group.kind] = group.items[0].id
    }
  }

  async function load() {
    isLoading.value = true
    error.value = null
    try {
      const current = context()
      candidates.value = (await api.tvArtworkCandidates(showId(), current.scope, current.seasonNumber)).items
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to load TV artwork candidates'
    } finally {
      isLoading.value = false
    }
  }

  async function scrape() {
    isLoading.value = true
    error.value = null
    try {
      const current = context()
      candidates.value = (await api.scrapeTVArtworkCandidates(csrfToken(), showId(), current.scope, current.seasonNumber)).items
      selectDefaultsForMissingArtwork()
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to scrape TV artwork candidates'
    } finally {
      isLoading.value = false
    }
  }

	function select(candidate: { id: string; kind: string }) {
		if (!groups.value.some(group => group.kind === candidate.kind)) return
    selected[candidate.kind] = selected[candidate.kind] === candidate.id ? '' : candidate.id
  }

  async function preview() {
    const selections = Object.entries(selected).filter(([, candidateId]) => candidateId).map(([kind, candidateId]) => ({ kind: kind as api.TVArtworkCandidate['kind'], candidateId }))
    if (selections.length === 0) {
      error.value = 'Select at least one artwork candidate'
      return
    }
    error.value = null
    try {
      const current = context()
      plan.value = await api.previewTVArtworkSelection(csrfToken(), showId(), current.scope, current.seasonNumber, selections)
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to preview TV artwork'
    }
  }

  async function apply() {
    if (!plan.value) return
    isApplying.value = true
    error.value = null
    try {
      plan.value = await api.applyTVArtwork(csrfToken(), plan.value.id)
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to queue TV artwork download'
    } finally {
      isApplying.value = false
    }
  }

  function closePlan() { plan.value = null }

  return { groups, selected, plan, error, isLoading, isApplying, load, scrape, select, preview, apply, closePlan }
}
