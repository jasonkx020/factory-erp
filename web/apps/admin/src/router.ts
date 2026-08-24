import { createRouter, createWebHistory } from 'vue-router'
import {
  getAccessToken,
  adminSpecialPath,
  adminModuleForPath,
  systemPermCode,
  useAuthStore,
} from '@erp/shared'
import { ElMessage } from 'element-plus'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/login', name: 'login', component: () => import('./views/LoginView.vue'), meta: { public: true } },
    {
      path: '/',
      component: () => import('./layouts/AdminLayout.vue'),
      children: [
        { path: '', name: 'home', component: () => import('./views/HomeView.vue') },
        { path: 'account', name: 'account', component: () => import('./views/account/AccountView.vue') },
        { path: 'loops', name: 'loops', component: () => import('./views/LoopsView.vue') },
        { path: 'automation/trace', name: 'trace', component: () => import('./views/automation/TraceView.vue') },
        { path: 'automation/logs', name: 'logs', component: () => import('./views/automation/LogsView.vue') },
        { path: 'automation/repairs', name: 'repairs', component: () => import('./views/automation/RepairView.vue') },
        { path: 'sales/outbound-settle', name: 'outbound-settle', component: () => import('./views/sales/OutboundSettleView.vue') },
        {
          path: 'sales/hub/:section?',
          name: 'sales-hub',
          component: () => import('./views/sales/SalesHubView.vue'),
          props: (r) => ({ module: undefined, section: r.params.section }),
        },
        {
          path: 'crm/hub/:section?',
          name: 'crm-hub',
          component: () => import('./views/crm/CrmHubView.vue'),
          props: (r) => ({ section: r.params.section }),
        },
        {
          path: 'purchase/hub/:section?',
          name: 'purchase-hub',
          component: () => import('./views/purchase/PurchaseHubView.vue'),
          props: (r) => ({ section: r.params.section }),
        },
        {
          path: 'production/hub/:section?',
          name: 'production-hub',
          component: () => import('./views/production/ProductionHubView.vue'),
          props: (r) => ({ section: r.params.section }),
        },
        {
          path: 'inventory/hub/:section?',
          name: 'inventory-hub',
          component: () => import('./views/inventory/InventoryHubView.vue'),
          props: (r) => ({ section: r.params.section }),
        },
        {
          path: 'product/hub/:section?',
          name: 'product-hub',
          component: () => import('./views/product/ProductHubView.vue'),
          props: (r) => ({ section: r.params.section }),
        },
        {
          path: 'asset/hub/:section?',
          name: 'asset-hub',
          component: () => import('./views/asset/FixedAssetHubView.vue'),
          props: (r) => ({ section: r.params.section }),
        },
        {
          path: 'finance/hub/:section?',
          name: 'finance-hub',
          component: () => import('./views/finance/FinanceHubView.vue'),
          props: (r) => ({ section: r.params.section }),
        },
        {
          path: 'report/hub/:section?',
          name: 'report-hub',
          component: () => import('./views/report/ReportHubView.vue'),
          props: (r) => ({ section: r.params.section }),
        },
        {
          path: 'approval/hub/:section?',
          name: 'approval-hub',
          component: () => import('./views/approval/ApprovalHubView.vue'),
          props: (r) => ({ section: r.params.section }),
        },
        { path: 'product/products', redirect: '/product/hub/products' },
        { path: 'asset/fixed-assets', redirect: '/asset/hub/fixed-assets' },
        { path: 'finance/vouchers', redirect: '/finance/hub/vouchers' },
        { path: 'approval/tasks', redirect: '/approval/hub/tasks' },
        { path: 'report/dashboards/boss', redirect: '/report/hub/boss' },
        { path: 'report/dashboards/production', redirect: '/report/hub/production-board' },
        { path: 'report/dashboards/live', redirect: '/report/hub/live' },
        { path: 'production/process-reports', redirect: '/production/hub/reports' },
        { path: 'production/process-mgmt', redirect: '/production/hub/processes' },
        { path: 'production/piece-issue', redirect: '/production/hub/piece-issue' },
        { path: 'automation/routing', redirect: '/production/hub/routings' },
        { path: 'hr/tool-issues', name: 'tool-issues', component: () => import('./views/hr/ToolIssueView.vue') },
        { path: 'workflow/tickets', name: 'workflow-tickets', component: () => import('./views/workflow/TicketCenterView.vue') },
        { path: 'system/personnel-transfers', redirect: '/hr/personnel-transfers' },
        { path: 'hr/permissions', redirect: '/hr/roles' },
        {
          path: 'hr/:section?',
          name: 'hr-hub',
          component: () => import('./views/ModuleCrudView.vue'),
        },
        {
          path: 'payroll/:section?',
          name: 'payroll-hub',
          component: () => import('./views/ModuleCrudView.vue'),
        },
        { path: 'warehouse/inbound', redirect: '/inventory/hub/inbound' },
        { path: 'inventory/ledger', redirect: '/inventory/hub/ledger' },
        { path: 'inventory/balances', redirect: '/inventory/hub/balances' },
        {
          path: 'system/:pathMatch(.*)*',
          name: 'system-admin',
          component: () => import('./views/ModuleCrudView.vue'),
        },
        {
          path: 'iam/:section',
          name: 'iam-admin',
          component: () => import('./views/ModuleCrudView.vue'),
        },
        {
          path: 'm/:domain/:module',
          name: 'module',
          component: () => import('./views/ModuleCrudView.vue'),
          beforeEnter(to) {
            const domain = decodeURIComponent(String(to.params.domain || ''))
            const module = decodeURIComponent(String(to.params.module || ''))
            const special = adminSpecialPath(domain, module)
            if (special) return special
            return true
          },
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  if (!getAccessToken()) return { path: '/login', query: { redirect: to.fullPath } }

  const auth = useAuthStore()
  if (auth.accessToken && !auth.roles.length && !auth.permissions.length) {
    await auth.fetchMe()
  }

  const path = to.path
  if (
    path.startsWith('/system/') ||
    path.startsWith('/iam/') ||
    path.startsWith('/workflow/') ||
    path === '/automation/logs' ||
    path.startsWith('/automation/logs/') ||
    path === '/automation/repairs' ||
    path.startsWith('/automation/repairs/') ||
    path === '/loops'
  ) {
    const hit = adminModuleForPath(path)
    if (hit?.domain === '系统管理') {
      const ok =
        auth.hasPerm(systemPermCode(hit.module, '查看')) ||
        auth.hasPerm(systemPermCode(hit.module, '编辑'))
      if (!ok) {
        ElMessage.warning('无权限访问该系统管理功能')
        return { path: '/' }
      }
    }
  }
  return true
})

export default router
