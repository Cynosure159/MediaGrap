import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import InspectorToolbar from './InspectorToolbar.vue'

const labels = {
  overview: 'Overview',
  artworkTab: 'Artwork',
  castTab: 'Cast',
  nfoRaw: 'NFO raw',
  fileAudit: 'File audit',
  edit: 'Edit',
  saveAndWriteNfo: 'Save & write NFO',
  scrape: 'Scrape',
  lock: 'Lock',
  locked: 'Locked',
  backToList: 'Back to list',
}

describe('InspectorToolbar', () => {
  it('renders all 5 tabs and indicates the active tab', () => {
    const wrapper = mount(InspectorToolbar, {
      props: {
        activeTab: 'artwork',
        hasDetail: true,
        isWritable: true,
        isEditing: false,
        isSaving: false,
        isScraping: false,
        isLocked: false,
        labels,
      },
    })

    const tabs = wrapper.findAll('.tab-btn')
    expect(tabs).toHaveLength(5)
    expect(wrapper.find('.tab-btn--active').text()).toContain('Artwork')
  })

  it('emits selectTab when a tab button is clicked', async () => {
    const wrapper = mount(InspectorToolbar, {
      props: {
        activeTab: 'overview',
        hasDetail: true,
        isWritable: true,
        isEditing: false,
        isSaving: false,
        isScraping: false,
        isLocked: false,
        labels,
      },
    })

    const castTab = wrapper.findAll('.tab-btn')[2]
    await castTab.trigger('click')

    expect(wrapper.emitted('selectTab')).toHaveLength(1)
    expect(wrapper.emitted('selectTab')?.[0]).toEqual(['cast'])
  })

  it('emits scrape and toggleEdit actions in view mode', async () => {
    const wrapper = mount(InspectorToolbar, {
      props: {
        activeTab: 'overview',
        hasDetail: true,
        isWritable: true,
        isEditing: false,
        isSaving: false,
        isScraping: false,
        isLocked: false,
        labels,
      },
    })

    const editBtn = wrapper.find('.btn.btn-outline')
    await editBtn.trigger('click')
    expect(wrapper.emitted('toggleEdit')).toHaveLength(1)

    const scrapeBtn = wrapper.find('.btn.btn-scrape')
    await scrapeBtn.trigger('click')
    expect(wrapper.emitted('scrape')).toHaveLength(1)
  })

  it('emits close when back button is clicked', async () => {
    const wrapper = mount(InspectorToolbar, {
      props: {
        activeTab: 'overview',
        hasDetail: false,
        isWritable: false,
        isEditing: false,
        isSaving: false,
        isScraping: false,
        isLocked: false,
        labels,
      },
    })

    await wrapper.find('.mobile-back-btn').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
