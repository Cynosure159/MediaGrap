import { computed, reactive, shallowRef } from 'vue'
import * as api from '@/api/library'

export function useArtwork(
  itemId: () => number,
  csrfToken: () => string,
  hasExistingArtwork: (kind: api.ArtworkCandidate['kind']) => boolean = () => false,
) {
  const candidates = shallowRef<api.ArtworkCandidate[]>([])
  const plan = shallowRef<api.ArtworkPlan | null>(null)
  const error = shallowRef<string | null>(null)
  const isLoading = shallowRef(false)
  const isApplying = shallowRef(false)
  const selected = reactive<Record<string, string>>({})

  const groups = computed(() => {
    const order: api.ArtworkCandidate['kind'][] = ['poster', 'fanart', 'clearlogo', 'clearart', 'discart', 'banner', 'landscape']
    return order.map(kind => ({ kind, items: candidates.value.filter(candidate => candidate.kind === kind) })).filter(group => group.items.length > 0)
  })

  function selectDefaultsForMissingArtwork() {
    for (const key of Object.keys(selected)) delete selected[key]
    for (const group of groups.value) {
      if (!hasExistingArtwork(group.kind) && group.items[0]) {
        selected[group.kind] = group.items[0].id
      }
    }
  }

  async function load() {
    isLoading.value = true
    error.value = null
    try {
      candidates.value = (await api.artworkCandidates(itemId())).items
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to load artwork candidates'
    } finally {
      isLoading.value = false
    }
  }

  async function scrape() {
    isLoading.value = true
    error.value = null
    try {
      candidates.value = (await api.scrapeArtworkCandidates(csrfToken(), itemId())).items
      selectDefaultsForMissingArtwork()
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to scrape artwork candidates'
    } finally {
      isLoading.value = false
    }
  }

	function select(candidate: { id: string; kind: string }) {
		if (!groups.value.some(group => group.kind === candidate.kind)) return
    selected[candidate.kind] = selected[candidate.kind] === candidate.id ? '' : candidate.id
  }

  async function preview() {
    const selections = Object.entries(selected).filter(([, id]) => id).map(([kind, candidateId]) => ({ kind: kind as api.ArtworkCandidate['kind'], candidateId }))
    if (selections.length === 0) {
      error.value = 'Select at least one artwork candidate'
      return
    }
    error.value = null
    try {
      plan.value = await api.previewArtworkSelection(csrfToken(), itemId(), selections)
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to preview artwork'
    }
  }

  async function apply() {
    if (!plan.value) return
    isApplying.value = true
    error.value = null
    try {
      plan.value = await api.applyArtwork(csrfToken(), plan.value.id)
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Unable to queue artwork download'
    } finally {
      isApplying.value = false
    }
  }

  function closePlan() { plan.value = null }

  return { candidates, groups, selected, plan, error, isLoading, isApplying, load, scrape, select, preview, apply, closePlan }
}
