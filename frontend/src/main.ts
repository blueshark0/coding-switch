import { createApp } from 'vue'
import App from './App.vue'
import './style.css'
import './styles/theme.css'
import './styles/common-ui.css'
import './styles/shell.css'
import './styles/heatmap.css'
import './styles/logs.css'
import './styles/provider-cards.css'
import './styles/modal.css'
import './styles/settings.css'
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
