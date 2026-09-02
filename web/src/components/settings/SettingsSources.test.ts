import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import SettingsSources from './SettingsSources.vue'

describe('SettingsSources', () => {
  it('shows source creation feedback', () => {
    const wrapper = mount(SettingsSources, {
      props: {
        sources: [],
        mediaRoots: ['/media'],
        labels: {
          libraryEyebrow: 'Library', directorySettings: 'Media directories', directoryHelp: 'Help',
          source: 'Media source', sourceType: 'Media type', selectSourceType: 'Select type', containerPath: 'Container path', addSource: 'Add',
          noSources: 'No sources', sourceNamePlaceholder: 'Media library', sourcePathPlaceholder: '/media/movies',
          movies: 'Movies', tvShows: 'Shows',
        },
        feedback: { kind: 'error', message: 'This media source has already been added.' },
        sourceName: 'Movies',
        sourcePath: '/media/movies',
        'onUpdate:sourceName': () => {},
        'onUpdate:sourcePath': () => {},
      },
    })

    expect(wrapper.get('.source-feedback').text()).toContain('already been added')
    expect(wrapper.find('select#new-source-name').exists()).toBe(true)
  })
})
