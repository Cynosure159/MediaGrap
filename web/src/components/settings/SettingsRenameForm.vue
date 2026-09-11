<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { saveSettings } from '@/api/library'
import type { Settings } from '@/api/types'
import {
  movieRenameDefault,
  tvRenameDefault,
  movieRenameTokens,
  tvRenameTokens,
} from '@/composables/useRenamePattern'

const props = defineProps<{
  settings: Settings | null
  csrfToken: string
  locale: string
}>()

const emit = defineEmits<{
  saved: [settings: Settings]
}>()

const zh = computed(() => props.locale === 'zh-CN')

const movie = ref(movieRenameDefault)
const tv = ref(tvRenameDefault)
const busy = ref(false)
const error = ref('')
const saved = ref(false)

const movieInputRef = ref<HTMLInputElement | null>(null)
const tvInputRef = ref<HTMLInputElement | null>(null)

watch(
  () => props.settings,
  (value) => {
    movie.value = value?.movieRenamePattern ?? movieRenameDefault
    tv.value = value?.tvRenamePattern ?? tvRenameDefault
  },
  { immediate: true },
)

watch([movie, tv], () => {
  saved.value = false
})

const isMovieDefault = computed(() => movie.value === movieRenameDefault)
const isTvDefault = computed(() => tv.value === tvRenameDefault)

// Token documentation and descriptions
interface TokenMeta {
  token: string
  nameZh: string
  nameEn: string
  example: string
  category: 'id' | 'number' | 'spec' | 'other'
}

const movieTokensMeta: TokenMeta[] = [
  { token: 'title', nameZh: '中文/首选标题', nameEn: 'Primary Title', example: '盗梦空间', category: 'id' },
  { token: 'originalTitle', nameZh: '原始语言标题', nameEn: 'Original Title', example: 'Inception', category: 'id' },
  { token: 'year', nameZh: '发行年份', nameEn: 'Release Year', example: '2010', category: 'number' },
  { token: 'resolution', nameZh: '视频分辨率', nameEn: 'Resolution', example: '4K', category: 'spec' },
  { token: 'videoCodec', nameZh: '视频编码', nameEn: 'Video Codec', example: 'HEVC', category: 'spec' },
  { token: 'audioCodec', nameZh: '音频编码', nameEn: 'Audio Codec', example: 'TrueHD Atmos', category: 'spec' },
  { token: 'edition', nameZh: '版本（当前暂无可用值）', nameEn: 'Edition (currently unavailable)', example: 'Director Cut', category: 'other' },
  { token: 'imdbId', nameZh: 'IMDb 编号（当前暂无可用值）', nameEn: 'IMDb ID (currently unavailable)', example: 'tt1375666', category: 'other' },
]

const tvTokensMeta: TokenMeta[] = [
  { token: 'showTitle', nameZh: '剧集主标题', nameEn: 'Show Title', example: '绝命毒师', category: 'id' },
  { token: 'showOriginalTitle', nameZh: '剧集原始名称', nameEn: 'Original Show Title', example: 'Breaking Bad', category: 'id' },
  { token: 'episodeTitle', nameZh: '本集单集标题', nameEn: 'Episode Title', example: '化学老师', category: 'id' },
  { token: 'originalTitle', nameZh: '剧集原始名称（别名）', nameEn: 'Original Show Title (alias)', example: 'Breaking Bad', category: 'id' },
  { token: 'seasonNumber', nameZh: '季号 (不补零)', nameEn: 'Season Number', example: '1', category: 'number' },
  { token: 'seasonNumberPad', nameZh: '季号 (补零 01)', nameEn: 'Padded Season (01)', example: '01', category: 'number' },
  { token: 'episodeNumber', nameZh: '集号 (不补零)', nameEn: 'Episode Number', example: '1', category: 'number' },
  { token: 'episodeNumberPad', nameZh: '集号 (补零 01)', nameEn: 'Padded Episode (01)', example: '01', category: 'number' },
  { token: 'year', nameZh: '首播年份', nameEn: 'Release Year', example: '2008', category: 'number' },
  { token: 'resolution', nameZh: '视频分辨率', nameEn: 'Resolution', example: '1080p', category: 'spec' },
  { token: 'videoCodec', nameZh: '视频编码', nameEn: 'Video Codec', example: 'H264', category: 'spec' },
  { token: 'audioCodec', nameZh: '音频编码', nameEn: 'Audio Codec', example: 'DTS-HD', category: 'spec' },
]

// Preset patterns
const moviePresets = [
  {
    nameZh: 'Kodi / Plex 标准',
    nameEn: 'Kodi / Plex Standard',
    pattern: '${title} (${year})/${title} (${year})',
  },
  {
    nameZh: '包含规格标签',
    nameEn: 'With Specs Tag',
    pattern: '${title} (${year})/${title} (${year}) - [${resolution} ${videoCodec}]',
  },
  {
    nameZh: '仅文件名',
    nameEn: 'Filename Only',
    pattern: '${title} (${year})',
  },
]

const tvPresets = [
  {
    nameZh: 'Kodi / Plex 季目录分层',
    nameEn: 'Kodi / Plex Season Folder',
    pattern: '${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}',
  },
  {
    nameZh: '单层扁平结构',
    nameEn: 'Flat Season Format',
    pattern: '${showTitle}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}',
  },
  {
    nameZh: '带分辨率与编码',
    nameEn: 'With Resolution Tag',
    pattern: '${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle} [${resolution}]',
  },
]

// Interactive token insertion
function insertToken(kind: 'movie' | 'tv', tokenName: string) {
  const token = '${' + tokenName + '}'
  const input = kind === 'movie' ? movieInputRef.value : tvInputRef.value
  if (!input) {
    if (kind === 'movie') movie.value += token
    else tv.value += token
    return
  }

  const start = input.selectionStart ?? input.value.length
  const end = input.selectionEnd ?? input.value.length
  const currentVal = input.value
  const newVal = currentVal.substring(0, start) + token + currentVal.substring(end)

  if (kind === 'movie') movie.value = newVal
  else tv.value = newVal

  // Restore cursor position after inserted token
  setTimeout(() => {
    input.focus()
    const nextPos = start + token.length
    input.setSelectionRange(nextPos, nextPos)
  }, 0)
}

// Live simulated path evaluation
const sampleMovieData: Record<string, string> = {
  title: '盗梦空间',
  originalTitle: 'Inception',
  year: '2010',
  resolution: '4K',
  videoCodec: 'HEVC',
  audioCodec: 'TrueHD Atmos',
  edition: 'Remastered',
  imdbId: 'tt1375666',
}

const sampleTvData: Record<string, string> = {
  showTitle: '绝命毒师',
  showOriginalTitle: 'Breaking Bad',
  seasonNumber: '1',
  seasonNumberPad: '01',
  episodeNumber: '1',
  episodeNumberPad: '01',
  episodeTitle: '化学老师',
  originalTitle: 'Breaking Bad',
  year: '2008',
  resolution: '1080p',
  videoCodec: 'H264',
  audioCodec: 'DTS-HD',
}

function evaluatePattern(pattern: string, data: Record<string, string>): { dir: string; filename: string } {
  if (!pattern.trim()) {
    return { dir: '', filename: '—' }
  }
  let evaluated = pattern
  for (const [key, val] of Object.entries(data)) {
    evaluated = evaluated.replaceAll('${' + key + '}', val)
  }
  const parts = evaluated.split('/').filter(Boolean)
  if (parts.length === 0) return { dir: '', filename: evaluated + '.mkv' }
  if (parts.length === 1) return { dir: '', filename: parts[0] + '.mkv' }
  const filename = parts.pop() + '.mkv'
  const dir = parts.join('/') + '/'
  return { dir, filename }
}

const moviePreview = computed(() => evaluatePattern(movie.value, sampleMovieData))
const tvPreview = computed(() => evaluatePattern(tv.value, sampleTvData))

async function save() {
  if (busy.value) return
  busy.value = true
  error.value = ''
  saved.value = false
  try {
    const result = await saveSettings(props.csrfToken, {
      movieRenamePattern: movie.value,
      tvRenamePattern: tv.value,
    })
    emit('saved', result)
    saved.value = true
  } catch (caught) {
    error.value =
      caught instanceof Error
        ? caught.message
        : zh.value
          ? '保存失败'
          : 'Save failed'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="rename-page-container">
    <section class="settings-card">
      <!-- Card Header -->
      <div class="card-header">
        <div class="card-header__info">
          <div class="header-tag">
            <svg class="header-icon" viewBox="0 0 24 24" fill="currentColor">
              <path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z" />
            </svg>
            <span class="eyebrow">{{ zh ? '命名规则与模板' : 'NAMING PATTERNS & TEMPLATES' }}</span>
          </div>
          <h2 class="section-title">{{ zh ? '默认文件重命名格式' : 'Default Rename Patterns' }}</h2>
          <p class="section-hint">
            {{
              zh
                ? '配置电影与剧集新打开的文件预览所使用的默认规则。保存不会重命名文件，也不会替换已生成的方案。'
                : 'Set defaults for newly opened movie and TV file previews. Saving does not rename files or replace existing plans.'
            }}
          </p>
        </div>
      </div>

      <!-- Card Body -->
      <div class="card-body">
        <form class="settings-form" @submit.prevent="save">
          <fieldset :disabled="busy || !settings" class="form-fieldset">
            <!-- ── 1. Movie Renaming Pattern ────────────────────────── -->
            <div class="pattern-section">
              <div class="section-badge-header">
                <div class="badge-title-row">
                  <svg class="type-icon type-icon--movie" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M18 4l2 4h-3l-2-4h-2l2 4h-3l-2-4H8l2 4H7L5 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V4h-4z" />
                  </svg>
                  <label for="default-movie-pattern" class="pattern-heading">
                    {{ zh ? '电影默认重命名规则' : 'Movie Rename Pattern' }}
                  </label>
                </div>
                <div class="pattern-status-badges">
                  <span v-if="isMovieDefault" class="spec-badge spec-badge--default">
                    {{ zh ? '官方推荐标准' : 'Standard Default' }}
                  </span>
                  <button
                    type="button"
                    class="btn-reset-link"
                    :disabled="isMovieDefault"
                    :title="zh ? '恢复电影默认格式' : 'Reset to movie default'"
                    @click="movie = movieRenameDefault"
                  >
                    <svg viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
                      <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z" />
                    </svg>
                    <span>{{ zh ? '恢复电影默认格式' : 'Reset movie pattern' }}</span>
                  </button>
                </div>
              </div>

              <!-- Presets quick buttons -->
              <div class="presets-row">
                <span class="presets-label">{{ zh ? '快速预设' : 'Presets' }}:</span>
                <div class="preset-chips">
                  <button
                    v-for="p in moviePresets"
                    :key="p.nameEn"
                    type="button"
                    class="preset-chip"
                    :class="{ 'preset-chip--active': movie === p.pattern }"
                    @click="movie = p.pattern"
                  >
                    {{ zh ? p.nameZh : p.nameEn }}
                  </button>
                </div>
              </div>

              <!-- Input control -->
              <div class="input-wrapper">
                <input
                  id="default-movie-pattern"
                  ref="movieInputRef"
                  v-model="movie"
                  required
                  maxlength="1024"
                  class="input-control font-code pattern-input"
                  :placeholder="movieRenameDefault"
                  spellcheck="false"
                  autocomplete="off"
                />
              </div>

              <!-- Tokens List -->
              <div class="tokens-container">
                <div class="tokens-header">
                  <span class="tokens-title">{{ zh ? '可用占位符（点击快速插入）' : 'Available Tokens (Click to insert)' }}</span>
                </div>
                <div class="tokens-grid">
                  <button
                    v-for="item in movieTokensMeta"
                    :key="item.token"
                    type="button"
                    class="token-btn font-code"
                    :class="`token-btn--${item.category}`"
                    :title="`${zh ? item.nameZh : item.nameEn} (例: ${item.example})`"
                    @click="insertToken('movie', item.token)"
                  >
                    <span class="token-text">{{ '${' + item.token + '}' }}</span>
                    <span class="token-sub">{{ zh ? item.nameZh : item.nameEn }}</span>
                  </button>
                </div>
              </div>

              <!-- Real-time live simulation preview -->
              <div class="preview-box">
                <div class="preview-header">
                  <span class="preview-tag">
                    <svg viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
                      <path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/>
                    </svg>
                    {{ zh ? '实时路径模拟 (示例电影)' : 'Live Preview Simulation (Sample Movie)' }}
                  </span>
                </div>
                <div class="preview-content font-code">
                  <span v-if="moviePreview.dir" class="preview-dir">📁 /media/movies/{{ moviePreview.dir }}</span>
                  <span class="preview-file">📄 {{ moviePreview.filename }}</span>
                </div>
              </div>
            </div>

            <div class="section-divider"></div>

            <!-- ── 2. TV Show Renaming Pattern ──────────────────────── -->
            <div class="pattern-section">
              <div class="section-badge-header">
                <div class="badge-title-row">
                  <svg class="type-icon type-icon--tv" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M21 3H3c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h5v2h8v-2h5c1.1 0 1.99-.9 1.99-2L23 5c0-1.1-.9-2-2-2zm0 14H3V5h18v12z" />
                  </svg>
                  <label for="default-tv-pattern" class="pattern-heading">
                    {{ zh ? '剧集默认重命名规则' : 'TV Show Rename Pattern' }}
                  </label>
                </div>
                <div class="pattern-status-badges">
                  <span v-if="isTvDefault" class="spec-badge spec-badge--default">
                    {{ zh ? '官方推荐标准' : 'Standard Default' }}
                  </span>
                  <button
                    type="button"
                    class="btn-reset-link"
                    :disabled="isTvDefault"
                    :title="zh ? '恢复剧集默认格式' : 'Reset to TV default'"
                    @click="tv = tvRenameDefault"
                  >
                    <svg viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
                      <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z" />
                    </svg>
                    <span>{{ zh ? '恢复剧集默认格式' : 'Reset TV pattern' }}</span>
                  </button>
                </div>
              </div>

              <!-- Presets quick buttons -->
              <div class="presets-row">
                <span class="presets-label">{{ zh ? '快速预设' : 'Presets' }}:</span>
                <div class="preset-chips">
                  <button
                    v-for="p in tvPresets"
                    :key="p.nameEn"
                    type="button"
                    class="preset-chip"
                    :class="{ 'preset-chip--active': tv === p.pattern }"
                    @click="tv = p.pattern"
                  >
                    {{ zh ? p.nameZh : p.nameEn }}
                  </button>
                </div>
              </div>

              <!-- Input control -->
              <div class="input-wrapper">
                <input
                  id="default-tv-pattern"
                  ref="tvInputRef"
                  v-model="tv"
                  required
                  maxlength="1024"
                  class="input-control font-code pattern-input"
                  :placeholder="tvRenameDefault"
                  spellcheck="false"
                  autocomplete="off"
                />
              </div>

              <!-- Tokens List -->
              <div class="tokens-container">
                <div class="tokens-header">
                  <span class="tokens-title">{{ zh ? '可用占位符（点击快速插入）' : 'Available Tokens (Click to insert)' }}</span>
                </div>
                <div class="tokens-grid">
                  <button
                    v-for="item in tvTokensMeta"
                    :key="item.token"
                    type="button"
                    class="token-btn font-code"
                    :class="`token-btn--${item.category}`"
                    :title="`${zh ? item.nameZh : item.nameEn} (例: ${item.example})`"
                    @click="insertToken('tv', item.token)"
                  >
                    <span class="token-text">{{ '${' + item.token + '}' }}</span>
                    <span class="token-sub">{{ zh ? item.nameZh : item.nameEn }}</span>
                  </button>
                </div>
              </div>

              <!-- Real-time live simulation preview -->
              <div class="preview-box">
                <div class="preview-header">
                  <span class="preview-tag">
                    <svg viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
                      <path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/>
                    </svg>
                    {{ zh ? '实时路径模拟 (示例单集)' : 'Live Preview Simulation (Sample Episode)' }}
                  </span>
                </div>
                <div class="preview-content font-code">
                  <span v-if="tvPreview.dir" class="preview-dir">📁 /media/tv/{{ tvPreview.dir }}</span>
                  <span class="preview-file">📄 {{ tvPreview.filename }}</span>
                </div>
              </div>
            </div>

            <!-- ── 3. Safety Notice & Guidelines ───────────────────── -->
            <div class="safety-card">
              <div class="safety-icon-col">
                <svg viewBox="0 0 24 24" fill="currentColor" width="18" height="18">
                  <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm-1 6h2v2h-2V7zm0 4h2v6h-2v-6z"/>
                </svg>
              </div>
              <div class="safety-text-col">
                <h4 class="safety-title">{{ zh ? '安全机制与目录规则' : 'Safety Principles & Formatting Rules' }}</h4>
                <ul class="safety-list">
                  <li>
                    <strong>{{ zh ? '安全只读保证' : 'Safe Previews' }}:</strong>
                    {{
                      zh
                        ? '修改或保存此处的默认格式绝不会修改磁盘上的任何媒体文件。实际重命名操作需在电影或剧集详情页的「文件与安全重命名」工坊中执行 Dry-Run 预演并由您手动核准。'
                        : 'Saving default patterns will never modify files on disk. Actual renaming requires dry-run preview and manual approval in the File Audit workshop.'
                    }}
                  </li>
                  <li>
                    <strong>{{ zh ? '子目录划分' : 'Subdirectories' }}:</strong>
                    {{
                      zh
                        ? '使用正斜杠 / 分隔多级子文件夹（如 ${title} (${year})/${title}）。未使用的占位符若在媒体中缺失元数据时，预览会标记提示。'
                        : 'Use forward slashes / to define subdirectories. Missing metadata will be flagged during individual media preview.'
                    }}
                  </li>
                  <li>
                    <strong>{{ zh ? '文件扩展名' : 'Extensions' }}:</strong>
                    {{
                      zh
                        ? '主视频文件的扩展名（如 .mkv, .mp4）及同名伴随文件（.nfo, .srt, poster.jpg 等）将由重命名规划引擎自动继承与同步处理。'
                        : 'Original video extensions and companion files (.nfo, subtitles, posters) are preserved and moved automatically.'
                    }}
                  </li>
                </ul>
              </div>
            </div>

            <!-- ── 4. Submit & Action Buttons ───────────────────────── -->
            <div class="form-actions-row">
              <button
                type="submit"
                class="btn btn-primary save-btn"
                :disabled="!movie.trim() || !tv.trim() || busy"
              >
                <svg v-if="busy" viewBox="0 0 24 24" fill="currentColor" width="16" height="16" class="spin-slow">
                  <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z" />
                </svg>
                <svg v-else viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
                  <path d="M17 3H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V7l-4-4zm2 16H5V5h11.17L19 7.83V19zm-7-7c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3zM6 6h9v4H6z" />
                </svg>
                <span>{{ busy ? (zh ? '正在保存…' : 'Saving…') : (zh ? '保存默认格式' : 'Save Default Patterns') }}</span>
              </button>
            </div>
          </fieldset>
        </form>

        <!-- Status & Error Feedback -->
        <div v-if="error" class="feedback-alert feedback-alert--error" role="alert">
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
          </svg>
          <span>{{ error }}</span>
        </div>

        <div v-if="saved" class="feedback-alert feedback-alert--success" role="status">
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
            <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
          </svg>
          <span>{{ zh ? '默认重命名格式已成功保存' : 'Default rename patterns saved successfully' }}</span>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.rename-page-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
}

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
  gap: 6px;
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
  font-size: 18px;
  font-weight: 700;
  line-height: 1.3;
  letter-spacing: -0.01em;
}

.section-hint {
  margin: 0;
  font-size: 13px;
  color: var(--on-surface-variant, #c7c4d7);
  line-height: 1.5;
}

.card-body {
  padding: 24px;
}

.settings-form {
  display: flex;
  flex-direction: column;
  width: 100%;
}

.form-fieldset {
  border: 0;
  padding: 0;
  margin: 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.form-fieldset:disabled {
  opacity: 0.6;
  pointer-events: none;
}

/* ── Section & Headers ────────────────────────────────────────────── */
.pattern-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-divider {
  height: 1px;
  background: var(--outline-variant, #2e3447);
  margin: 4px 0;
}

.section-badge-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
}

.badge-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.type-icon {
  width: 18px;
  height: 18px;
}

.type-icon--movie {
  color: var(--primary, #c0c1ff);
}

.type-icon--tv {
  color: var(--secondary, #4edea3);
}

.pattern-heading {
  font-size: 14px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  margin: 0;
  letter-spacing: -0.01em;
  cursor: pointer;
}

.pattern-status-badges {
  display: flex;
  align-items: center;
  gap: 8px;
}

.spec-badge--default {
  background: rgba(78, 222, 163, 0.12);
  color: var(--secondary, #4edea3);
  border-color: rgba(78, 222, 163, 0.3);
  font-size: 10px;
  font-weight: 600;
  padding: 2px 6px;
}

.btn-reset-link {
  background: transparent;
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 3px 8px;
  font-size: 11px;
  color: var(--on-surface-variant, #c7c4d7);
  display: inline-flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-reset-link:hover:not(:disabled) {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface, #dce1fb);
  border-color: var(--outline, #908fa0);
}

.btn-reset-link:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* ── Presets ──────────────────────────────────────────────────────── */
.presets-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.presets-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--outline, #908fa0);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-family: var(--font-data, monospace);
}

.preset-chips {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.preset-chip {
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  color: var(--on-surface-variant, #c7c4d7);
  padding: 3px 8px;
  font-size: 11px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.preset-chip:hover {
  background: var(--surface-container-high, #23293c);
  color: var(--on-surface, #dce1fb);
  border-color: var(--primary, #c0c1ff);
}

.preset-chip--active {
  background: rgba(192, 193, 255, 0.12);
  border-color: var(--primary, #c0c1ff);
  color: var(--primary, #c0c1ff);
  font-weight: 600;
}

/* ── Input Control ────────────────────────────────────────────────── */
.input-wrapper {
  width: 100%;
}

.input-control {
  width: 100%;
  height: 40px;
  padding: 0 14px;
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  background: var(--surface-container-lowest, #070d1f);
  color: var(--on-surface, #dce1fb);
  font-size: 13px;
  outline: none;
  box-sizing: border-box;
  transition: all 0.15s ease;
}

.input-control:focus {
  border-color: var(--primary, #c0c1ff);
  box-shadow: 0 0 0 2px rgba(192, 193, 255, 0.15);
  background: var(--surface-container-lowest, #070d1f);
}

.font-code {
  font-family: var(--font-data, monospace);
}

/* ── Tokens ───────────────────────────────────────────────────────── */
.tokens-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: var(--surface-container-low, #151b2d);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 10px 12px;
}

.tokens-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.tokens-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--outline, #908fa0);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.tokens-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.token-btn {
  display: inline-flex;
  flex-direction: column;
  align-items: flex-start;
  padding: 4px 8px;
  background: var(--surface-container, #191f31);
  border: 1px solid var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  cursor: pointer;
  text-align: left;
  transition: all 0.15s ease;
  user-select: none;
}

.token-btn:hover {
  background: var(--surface-container-high, #23293c);
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.25);
}

.token-text {
  font-size: 11px;
  font-weight: 600;
  line-height: 1.2;
}

.token-sub {
  font-size: 9px;
  color: var(--outline, #908fa0);
  font-family: var(--font-body, sans-serif);
  line-height: 1.2;
  margin-top: 1px;
}

/* Token color variations */
.token-btn--id {
  border-left: 2px solid var(--secondary, #4edea3);
}
.token-btn--id .token-text {
  color: var(--secondary, #4edea3);
}
.token-btn--id:hover {
  border-color: var(--secondary, #4edea3);
}

.token-btn--number {
  border-left: 2px solid var(--tertiary, #ffb95f);
}
.token-btn--number .token-text {
  color: var(--tertiary, #ffb95f);
}
.token-btn--number:hover {
  border-color: var(--tertiary, #ffb95f);
}

.token-btn--spec {
  border-left: 2px solid var(--primary, #c0c1ff);
}
.token-btn--spec .token-text {
  color: var(--primary, #c0c1ff);
}
.token-btn--spec:hover {
  border-color: var(--primary, #c0c1ff);
}

.token-btn--other {
  border-left: 2px solid var(--on-surface-variant, #c7c4d7);
}
.token-btn--other .token-text {
  color: var(--on-surface-variant, #c7c4d7);
}
.token-btn--other:hover {
  border-color: var(--on-surface, #dce1fb);
}

/* ── Live Preview Box ─────────────────────────────────────────────── */
.preview-box {
  background: var(--surface-container-lowest, #070d1f);
  border: 1px dashed var(--outline-variant, #2e3447);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.preview-header {
  display: flex;
  align-items: center;
}

.preview-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--outline, #908fa0);
}

.preview-content {
  display: flex;
  flex-direction: column;
  font-size: 12px;
  gap: 2px;
  line-height: 1.4;
  word-break: break-all;
}

.preview-dir {
  color: var(--tertiary, #ffb95f);
}

.preview-file {
  color: var(--secondary, #4edea3);
  font-weight: 600;
}

/* ── Safety Card ─────────────────────────────────────────────────── */
.safety-card {
  display: flex;
  gap: 12px;
  background: rgba(35, 41, 60, 0.6);
  border: 1px solid var(--outline-variant, #2e3447);
  border-left: 3px solid var(--primary, #c0c1ff);
  border-radius: var(--radius-sm, 0.25rem);
  padding: 14px 16px;
}

.safety-icon-col {
  color: var(--primary, #c0c1ff);
  flex-shrink: 0;
  margin-top: 2px;
}

.safety-text-col {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.safety-title {
  margin: 0;
  font-size: 12px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.safety-list {
  margin: 0;
  padding-left: 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  color: var(--on-surface-variant, #c7c4d7);
  line-height: 1.5;
}

.safety-list strong {
  color: var(--on-surface, #dce1fb);
}

/* ── Form Actions ─────────────────────────────────────────────────── */
.form-actions-row {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  padding-top: 4px;
}

.save-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 38px;
  padding: 0 20px;
  font-weight: 600;
  font-size: 13px;
}

/* ── Feedback Alerts ──────────────────────────────────────────────── */
.feedback-alert {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-sm, 0.25rem);
  font-size: 13px;
  margin-top: 14px;
}

.feedback-alert--error {
  background: rgba(244, 63, 94, 0.12);
  border: 1px solid rgba(244, 63, 94, 0.35);
  color: var(--error, #ffb4ab);
}

.feedback-alert--success {
  background: rgba(78, 222, 163, 0.12);
  border: 1px solid rgba(78, 222, 163, 0.35);
  color: var(--secondary, #4edea3);
}

.spin-slow {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* ── Responsive ───────────────────────────────────────────────────── */
@media (max-width: 760px) {
  .card-body {
    padding: 16px;
  }

  .card-header {
    padding: 16px;
  }

  .section-badge-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
  }

  .pattern-status-badges {
    width: 100%;
    justify-content: space-between;
  }

  .presets-row {
    flex-direction: column;
    align-items: flex-start;
  }

  .tokens-grid {
    gap: 4px;
  }

  .token-btn {
    flex: 1 1 calc(50% - 4px);
    min-width: 120px;
  }

  .safety-card {
    flex-direction: column;
    gap: 8px;
  }
}
</style>

