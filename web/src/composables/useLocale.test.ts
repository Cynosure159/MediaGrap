import { beforeEach, describe, expect, it } from 'vitest'
import { useLocale } from './useLocale'

describe('useLocale', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('defaults to en when no saved preference exists', () => {
    const { locale, t } = useLocale()
    expect(locale.value).toBe('en')
    expect(t.value.movies).toBe('Movies')
  })

  it('initializes with zh-CN if saved in localStorage', () => {
    localStorage.setItem('mediagrap.locale', 'zh-CN')
    const { locale, t } = useLocale()
    expect(locale.value).toBe('zh-CN')
    expect(t.value.movies).toBe('电影')
  })

  it('updates locale and persists to localStorage on setLocale', () => {
    const { locale, setLocale, t } = useLocale()
    setLocale('zh-CN')
    expect(locale.value).toBe('zh-CN')
    expect(t.value.movies).toBe('电影')
    expect(localStorage.getItem('mediagrap.locale')).toBe('zh-CN')
  })

  it('toggles locale between en and zh-CN', () => {
    const { locale, toggleLocale, t } = useLocale()
    expect(locale.value).toBe('en')

    toggleLocale()
    expect(locale.value).toBe('zh-CN')
    expect(t.value.movies).toBe('电影')

    toggleLocale()
    expect(locale.value).toBe('en')
    expect(t.value.movies).toBe('Movies')
  })
})
