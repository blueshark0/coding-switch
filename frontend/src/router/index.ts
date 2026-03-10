import { createRouter, createWebHashHistory } from 'vue-router'
import MainPage from '../components/Main/Index.vue'
import LogsPage from '../components/Logs/Index.vue'
import GeneralPage from '../components/General/Index.vue'
import SessionPage from '../components/Session/Index.vue'

const routes = [
  { path: '/', component: MainPage },
  { path: '/logs', component: LogsPage },
  { path: '/settings', component: GeneralPage },
  { path: '/sessions', component: SessionPage },
]

export default createRouter({
  history: createWebHashHistory(), // Use createWebHashHistory for hash-based routing
  routes
})
