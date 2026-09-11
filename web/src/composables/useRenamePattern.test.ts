import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { settings } from '@/api/library'
import type { Settings } from '@/api/types'
import { useRenamePattern } from './useRenamePattern'
vi.mock('@/api/library', () => ({ settings: vi.fn() }))
beforeEach(() => vi.clearAllMocks())
const host = (kind: 'movie' | 'tv') => defineComponent({ setup: () => useRenamePattern(kind), template: '<input v-model="pattern" />' })
describe('saved rename defaults', () => {
  it.each(['movie', 'tv'] as const)('loads the %s default', async kind => {
    vi.mocked(settings).mockResolvedValue({ movieRenamePattern: 'Movie/${title}', tvRenamePattern: 'TV/${showTitle}' } as Settings)
    const wrapper = mount(host(kind))
    await flushPromises()
    expect(wrapper.get('input').element.value).toBe(kind === 'movie' ? 'Movie/${title}' : 'TV/${showTitle}')
    wrapper.unmount()
  })
  it('does not overwrite edits when settings arrive late', async () => {
    let resolve!: (value: Settings) => void
    vi.mocked(settings).mockReturnValue(new Promise(done => { resolve = done }))
    const wrapper = mount(host('movie'))
    await wrapper.get('input').setValue('My/${title}')
    resolve({ movieRenamePattern: 'Saved/${title}' } as Settings)
    await flushPromises()
    expect(wrapper.get('input').element.value).toBe('My/${title}')
    wrapper.unmount()
  })
})
