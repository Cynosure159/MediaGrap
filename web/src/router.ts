import {
  createRouter,
  createWebHistory,
  type LocationQuery,
  type LocationQueryRaw,
} from 'vue-router'
import LibraryRouteView from '@/components/library/LibraryRouteView.vue'
import OperationsRouteView from '@/components/operations/OperationsRouteView.vue'
import SettingsRouteView from '@/components/settings/SettingsRouteView.vue'
import type { TVSelection } from '@/api/library'

export type MainSection = 'movies' | 'shows' | 'jobs' | 'settings'
export type InspectorTab = 'overview' | 'artwork' | 'cast' | 'nfo' | 'files'

const inspectorTabs: readonly InspectorTab[] = ['overview', 'artwork', 'cast', 'nfo', 'files']
type Query = LocationQuery | LocationQueryRaw

function queryValue(query: Query, key: string): string | undefined {
  const value = query[key]
  const first = Array.isArray(value) ? value[0] : value
  if (first === null || first === undefined) return undefined
  return String(first)
}

function parseInteger(value: string | undefined, minimum: number): number | null {
  if (!value || !/^\d+$/.test(value)) return null
  const parsed = Number(value)
  return Number.isSafeInteger(parsed) && parsed >= minimum ? parsed : null
}

export function parsePositiveQueryId(query: Query, key: string): number | null {
  return parseInteger(queryValue(query, key), 1)
}

export function parseSeasonQuery(query: Query): number | null {
  return parseInteger(queryValue(query, 'season'), 0)
}

export function parseInspectorTab(query: Query): InspectorTab {
  const value = queryValue(query, 'tab')
  return inspectorTabs.includes(value as InspectorTab) ? value as InspectorTab : 'overview'
}

export function parseTVSelection(query: Query): TVSelection | null {
  const showId = parsePositiveQueryId(query, 'show')
  if (showId === null) return null

  const seasonNumber = parseSeasonQuery(query)
  const episodeId = parsePositiveQueryId(query, 'episode')
  if (episodeId !== null && seasonNumber !== null) {
    return { kind: 'episode', showId, seasonNumber, episodeId }
  }
  if (seasonNumber !== null) return { kind: 'season', showId, seasonNumber }
  return { kind: 'show', showId }
}

function addTab(query: LocationQueryRaw, tab: InspectorTab): LocationQueryRaw {
  if (tab !== 'overview') query.tab = tab
  return query
}

export function movieRouteQuery(movieId: number | null, tab: InspectorTab = 'overview'): LocationQueryRaw {
  if (movieId === null) return {}
  return addTab({ movie: String(movieId) }, tab)
}

export function tvRouteQuery(selection: TVSelection | null, tab: InspectorTab = 'overview'): LocationQueryRaw {
  if (!selection) return {}
  const query: LocationQueryRaw = { show: String(selection.showId) }
  if (selection.kind === 'season' || selection.kind === 'episode') query.season = String(selection.seasonNumber)
  if (selection.kind === 'episode') query.episode = String(selection.episodeId)
  return addTab(query, tab)
}

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', redirect: { name: 'movies' } },
    { path: '/movies', name: 'movies', component: LibraryRouteView, props: { mediaKind: 'movies' }, meta: { section: 'movies' } },
    { path: '/shows', name: 'shows', component: LibraryRouteView, props: { mediaKind: 'shows' }, meta: { section: 'shows' } },
    { path: '/jobs', name: 'jobs', component: OperationsRouteView, meta: { section: 'jobs' } },
    { path: '/settings', name: 'settings', component: SettingsRouteView, meta: { section: 'settings' } },
    // Keep old bookmarks usable after Sources was folded into Settings.
    { path: '/sources', redirect: '/settings' },
    { path: '/:pathMatch(.*)*', redirect: '/movies' },
  ],
})
