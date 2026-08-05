import { createRouter, createWebHistory } from 'vue-router'
import { getAccessToken } from '@erp/shared'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/login', name: 'login', component: () => import('./views/LoginView.vue'), meta: { public: true } },
    {
      path: '/',
      component: () => import('./layouts/AdminLayout.vue'),
      children: [
        { path: '', name: 'home', component: () => import('./views/HomeView.vue') },
        { path: 'loops', name: 'loops', component: () => import('./views/LoopsView.vue') },
        { path: 'automation/routing', name: 'routing', component: () => import('./views/automation/RoutingView.vue') },
        { path: 'automation/trace', name: 'trace', component: () => import('./views/automation/TraceView.vue') },
        { path: 'automation/logs', name: 'logs', component: () => import('./views/automation/LogsView.vue') },
        { path: 'automation/repairs', name: 'repairs', component: () => import('./views/automation/RepairView.vue') },
        { path: 'warehouse/inbound', name: 'warehouse-inbound', component: () => import('./views/warehouse/WarehouseInboundView.vue') },
        { path: 'sales/outbound-settle', name: 'outbound-settle', component: () => import('./views/sales/OutboundSettleView.vue') },
        { path: 'production/process-reports', name: 'process-reports', component: () => import('./views/production/ProcessReportView.vue') },
        { path: 'production/piece-issue', name: 'piece-issue', component: () => import('./views/production/PieceIssueView.vue') },
        { path: 'hr/tool-issues', name: 'tool-issues', component: () => import('./views/hr/ToolIssueView.vue') },
        { path: 'inventory/ledger', name: 'stock-ledger', component: () => import('./views/inventory/StockLedgerView.vue') },
        {
          path: 'm/:domain/:module',
          name: 'module',
          component: () => import('./views/ModuleCrudView.vue'),
        },
      ],
    },
  ],
})

router.beforeEach((to) => {
  if (to.meta.public) return true
  if (!getAccessToken()) return { path: '/login', query: { redirect: to.fullPath } }
  return true
})

export default router
