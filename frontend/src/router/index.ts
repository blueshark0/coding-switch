import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  { path: '/', component: () => import('../components/Main/Index.vue') },
  { path: '/logs', component: () => import('../components/Logs/Index.vue') },
  { path: '/settings', component: () => import('../components/General/Index.vue') },
  { path: '/sessions', component: () => import('../components/Session/Index.vue') },
]

export default createRouter({
  history: createWebHashHistory(), // Use createWebHashHistory for hash-based routing
  routes
})
