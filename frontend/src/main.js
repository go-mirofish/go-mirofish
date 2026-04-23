import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import i18n from './i18n'
import { initTheme } from './composables/theme.js'
import './styles/doc-layout.css'

initTheme()

const app = createApp(App)

app.use(router)
app.use(i18n)

app.mount('#app')
