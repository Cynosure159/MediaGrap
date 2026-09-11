import { onMounted, onUnmounted, ref, watch } from 'vue'
import { settings } from '@/api/library'

export const movieRenameDefault = '${title} (${year})/${title} (${year})'
export const tvRenameDefault = '${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}'
export const movieRenameTokens = ['title', 'originalTitle', 'year', 'resolution', 'videoCodec', 'audioCodec', 'edition', 'imdbId']
export const tvRenameTokens = ['showTitle', 'showOriginalTitle', 'originalTitle', 'seasonNumber', 'seasonNumberPad', 'episodeNumber', 'episodeNumberPad', 'episodeTitle', 'year', 'resolution', 'videoCodec', 'audioCodec']

export function useRenamePattern(kind: 'movie' | 'tv') {
  const pattern = ref(kind === 'movie' ? movieRenameDefault : tvRenameDefault)
  const selectedPreset = ref('custom')
  let edited = false
  let disposed = false
  watch(pattern, () => { edited = true }, { flush: 'sync' })
  onUnmounted(() => { disposed = true })
  onMounted(async () => {
    try {
      const current = await settings()
      const saved = kind === 'movie' ? current.movieRenamePattern : current.tvRenamePattern
      if (!disposed && !edited && saved) pattern.value = saved
    } catch {
      // Keep the built-in preview template when settings cannot be loaded.
    }
  })
  return { pattern, selectedPreset }
}
