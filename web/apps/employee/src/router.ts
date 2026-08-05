import { createRouter, createWebHistory } from 'vue-router'
import { getAccessToken } from '@erp/shared'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/login', name: 'login', component: () => import('./views/LoginView.vue'), meta: { public: true } },
    { path: '/', name: 'home', component: () => import('./views/HomeView.vue') },
    { path: '/m/workshop', name: 'workshop', component: () => import('./views/modules/WorkshopModule.vue') },
    { path: '/m/worker', name: 'worker', component: () => import('./views/modules/WorkerModule.vue') },
    { path: '/m/warehouse', name: 'warehouse', component: () => import('./views/modules/WarehouseModule.vue') },
    { path: '/m/sales', name: 'sales', component: () => import('./views/modules/SalesModule.vue') },
  ],
})

router.beforeEach((to) => {
  if (to.meta.public) return true
  if (!getAccessToken()) return { path: '/login', query: { redirect: to.fullPath } }
  return true
})

export default router
