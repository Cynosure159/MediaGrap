import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AuthPanel from './AuthPanel.vue'

const labels = {
  language: '中文',
  authEyebrow: 'MediaGrap · private library',
  setup: 'Create administrator',
  welcome: 'Welcome back',
  setupIntro: 'This account controls media sources.',
  signInIntro: 'Sign in to review your library.',
  username: 'Username',
  password: 'Password',
  create: 'Create',
  signIn: 'Sign in',
}

describe('AuthPanel', () => {
  it('renders in setup mode with create button', () => {
    const wrapper = mount(AuthPanel, {
      props: {
        setup: true,
        error: null,
        labels,
      },
    })

    expect(wrapper.text()).toContain('Create administrator')
    expect(wrapper.find('button[type="submit"]').text()).toContain('Create')
  })

  it('renders in login mode with sign in button', () => {
    const wrapper = mount(AuthPanel, {
      props: {
        setup: false,
        error: null,
        labels,
      },
    })

    expect(wrapper.text()).toContain('Welcome back')
    expect(wrapper.find('button[type="submit"]').text()).toContain('Sign in')
  })

  it('displays error message when provided', () => {
    const wrapper = mount(AuthPanel, {
      props: {
        setup: false,
        error: 'Invalid username or password',
        labels,
      },
    })

    expect(wrapper.text()).toContain('Invalid username or password')
  })

  it('emits submit event with credentials upon form submission', async () => {
    const wrapper = mount(AuthPanel, {
      props: {
        setup: false,
        error: null,
        labels,
      },
    })

    await wrapper.find('#auth-username').setValue('myuser')
    await wrapper.find('#auth-password').setValue('mypassword123')
    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.emitted('submit')).toHaveLength(1)
    expect(wrapper.emitted('submit')?.[0]).toEqual(['myuser', 'mypassword123'])
  })

  it('emits toggleLocale when language button is clicked', async () => {
    const wrapper = mount(AuthPanel, {
      props: {
        setup: false,
        error: null,
        labels,
      },
    })

    await wrapper.find('.locale-btn').trigger('click')
    expect(wrapper.emitted('toggleLocale')).toHaveLength(1)
  })
})
