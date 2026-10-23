import { createApp } from 'vue'
import App from './App.vue'
import './style.css'
import './styles/theme.css'
import './styles/cs.css'
import { getStoredLocale, i18n, setupI18n } from './utils/i18n'
import { initTheme } from './utils/ThemeManager'
import router from './router/index'

initTheme()
const isMac = navigator.userAgent.includes('Mac')
if (isMac) {
  document.documentElement.classList.add('mac')
}

async function bootstrap() {
  await setupI18n(getStoredLocale())
  createApp(App).use(router).use(i18n).mount('#app')
}
bootstrap()
