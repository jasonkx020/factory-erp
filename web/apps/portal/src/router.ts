import { createRouter, createWebHistory } from 'vue-router'
import { getAccessToken } from '@erp/shared'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', name: 'home', component: () => import('./views/HomeView.vue'), meta: { public: true } },
    { path: '/shop/login', name: 'shop-login', component: () => import('./views/LoginView.vue'), meta: { public: true } },
    { path: '/shop/:section?', name: 'shop', component: () => import('./views/ShopHubView.vue') },
  ],
})

router.beforeEach((to) => {
  if (to.meta.public) return true
  if (!getAccessToken()) return { path: '/shop/login', query: { redirect: to.fullPath } }
  return true
})

export default router
