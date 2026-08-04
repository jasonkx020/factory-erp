import { createRouter, createWebHistory } from 'vue-router'
import { getAccessToken } from '@erp/shared'
const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/login', component: () => import('./views/LoginView.vue'), meta: { public: true } },
    { path: '/', component: () => import('./views/CustomerHome.vue') },
  ],
})
router.beforeEach((to) => {
  if (to.meta.public) return true
  if (!getAccessToken()) return '/login'
  return true
})
export default router
