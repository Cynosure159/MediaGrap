import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { saveSettings } from '@/api/library'
import type { Settings } from '@/api/types'
import SettingsRenameForm from './SettingsRenameForm.vue'
vi.mock('@/api/library', () => ({ saveSettings: vi.fn() }))
beforeEach(() => vi.clearAllMocks())
describe('rename defaults settings', () => {
  it('saves movie and TV patterns together using session CSRF', async () => {
    const settings = { movieRenamePattern: '${title}', tvRenamePattern: '${showTitle}' } as Settings
    vi.mocked(saveSettings).mockResolvedValue(settings)
    const wrapper = mount(SettingsRenameForm, { props: { settings, csrfToken: 'csrf', locale: 'en' } })
    const inputs = wrapper.findAll('input')
    expect(inputs[0]!.element.value).toBe('${title}')
    expect(inputs[1]!.element.value).toBe('${showTitle}')
    await inputs[0]!.setValue('Movies/${title}')
    await inputs[1]!.setValue('TV/${showTitle}')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(saveSettings).toHaveBeenCalledWith('csrf', { movieRenamePattern: 'Movies/${title}', tvRenamePattern: 'TV/${showTitle}' })
    expect(wrapper.emitted('saved')?.[0]).toEqual([settings])
    wrapper.unmount()
  })
  it('shows server validation failures without losing input', async () => {
    vi.mocked(saveSettings).mockRejectedValue(new Error('unsupported naming token'))
    const wrapper = mount(SettingsRenameForm, { props: { settings: {} as Settings, csrfToken: 'csrf', locale: 'en' } })
    await wrapper.findAll('input')[0]!.setValue('${unknown}')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.get('[role=alert]').text()).toContain('unsupported naming token')
    expect(wrapper.findAll('input')[0]!.element.value).toBe('${unknown}')
    wrapper.unmount()
  })

  it('updates input and live preview when clicking a preset chip', async () => {
    const settings = { movieRenamePattern: '${title}', tvRenamePattern: '${showTitle}' } as Settings
    const wrapper = mount(SettingsRenameForm, { props: { settings, csrfToken: 'csrf', locale: 'zh-CN' } })
    
    // Find preset buttons for movie
    const presetBtns = wrapper.findAll('.preset-chip')
    expect(presetBtns.length).toBeGreaterThan(0)
    // Click second preset (with specs)
    await presetBtns[1]!.trigger('click')
    
    const inputs = wrapper.findAll('input')
    expect(inputs[0]!.element.value).toContain('${resolution}')
    
    // Check live preview updates
    const previewContent = wrapper.find('.preview-box').text()
    expect(previewContent).toContain('4K')
    wrapper.unmount()
  })

  it('resets to default patterns when clicking reset buttons', async () => {
    const settings = { movieRenamePattern: 'custom/movie', tvRenamePattern: 'custom/tv' } as Settings
    const wrapper = mount(SettingsRenameForm, { props: { settings, csrfToken: 'csrf', locale: 'en' } })
    
    const resetBtns = wrapper.findAll('.btn-reset-link')
    expect(resetBtns.length).toBe(2)
    
    await resetBtns[0]!.trigger('click')
    expect(wrapper.findAll('input')[0]!.element.value).toBe('${title} (${year})/${title} (${year})')
    
    await resetBtns[1]!.trigger('click')
    expect(wrapper.findAll('input')[1]!.element.value).toContain('${showTitle}')
    wrapper.unmount()
  })

  it('inserts token when clicking a token chip button', async () => {
    const settings = { movieRenamePattern: 'Movies/', tvRenamePattern: 'TV/' } as Settings
    const wrapper = mount(SettingsRenameForm, { props: { settings, csrfToken: 'csrf', locale: 'zh-CN' } })
    
    const tokenBtn = wrapper.find('.token-btn')
    expect(tokenBtn.exists()).toBe(true)
    await tokenBtn.trigger('click')
    
    expect(wrapper.findAll('input')[0]!.element.value).toContain('${title}')
    wrapper.unmount()
  })
})
