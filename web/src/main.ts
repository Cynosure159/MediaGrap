import { createApp } from 'vue'
import { registerSW } from 'virtual:pwa-register'
import App from './App.vue'
import './assets/main.css'
import { router } from './router'

registerSW({ immediate: true })

createApp(App).use(router).mount('#app')
