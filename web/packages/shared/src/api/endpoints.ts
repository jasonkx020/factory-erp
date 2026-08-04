import { api } from './client'
import type { LoginData, MeData, PageData } from '../types'

export const authApi = {
  login: (login_name: string, password: string, client_type = 'web') =>
    api.post<LoginData>('/auth/login', { login_name, password, client_type }),
  refresh: (refresh_token: string) =>
    api.post<LoginData>('/auth/refresh', { refresh_token }),
  me: () => api.get<MeData>('/auth/me'),
}

export const productApi = {
  list: () => api.get<PageData>('/product/products'),
  create: (body: Record<string, unknown>) => api.post('/product/products', body),
  get: (id: number) => api.get(`/product/products/${id}`),
  update: (id: number, body: Record<string, unknown>) => api.put(`/product/products/${id}`, body),
  remove: (id: number) => api.del(`/product/products/${id}`),
  activate: (id: number) => api.post(`/product/products/${id}/activate`),
  deactivate: (id: number) => api.post(`/product/products/${id}/deactivate`),
}

export const inventoryApi = {
  balances: () => api.get<PageData>('/inventory/balances'),
  availability: () => api.get<{ list: Record<string, unknown>[] }>('/inventory/availability'),
  listTxns: () => api.get<PageData>('/inventory/stock-txns'),
  createTxn: (body: Record<string, unknown>) => api.post('/inventory/stock-txns', body),
  getTxn: (id: number) => api.get(`/inventory/stock-txns/${id}`),
  postTxn: (id: number) => api.post(`/inventory/stock-txns/${id}/post`, {}),
  cancelTxn: (id: number) => api.post(`/inventory/stock-txns/${id}/cancel`, {}),
  boxTrace: (code: string) => api.get(`/inventory/box-codes/trace/${encodeURIComponent(code)}`),
  listBoxes: () => api.get<PageData>('/inventory/box-codes'),
  createBox: (body: Record<string, unknown>) => api.post('/inventory/box-codes', body),
}

export const productionApi = {
  processes: () => api.get<PageData>('/production/processes'),
  listTasks: () => api.get<PageData>('/production/tasks'),
  createTask: (body: Record<string, unknown>) => api.post('/production/tasks', body),
  getTask: (id: number) => api.get(`/production/tasks/${id}`),
  listDispatches: () => api.get<PageData>('/production/dispatches'),
  createDispatch: (body: Record<string, unknown>) => api.post('/production/dispatches', body),
  listReportWorks: () => api.get<PageData>('/production/report-works'),
  createReportWork: (body: Record<string, unknown>) => api.post('/production/report-works', body),
  listRequisitions: () => api.get<PageData>('/production/requisitions'),
  createRequisition: (body: Record<string, unknown>) => api.post('/production/requisitions', body),
  postRequisition: (id: number) => api.post(`/production/requisitions/${id}/post`, {}),
  scan: (body: { badge_code: string; box_code: string; net_weight: number }) =>
    api.post('/production/scan', body),
  scanResolve: (body: { badge_code: string; box_code: string; net_weight?: number }) =>
    api.post('/production/scan/resolve', body),
  flowEvents: () => api.get<PageData>('/production/flow-events'),
  retryFlow: (id: number) => api.post(`/production/flow-events/${id}/retry`, {}),
  flowRules: () => api.get('/production/flow-rules'),
  saveFlowRules: (body: Record<string, unknown>) => api.put('/production/flow-rules', body),
  listRoutings: () => api.get<PageData>('/production/routings'),
  getRouting: (id: number) => api.get(`/production/routings/${id}`),
}

export const systemApi = {
  settings: () => api.get('/system/settings'),
  logs: () => api.get<PageData>('/system/operation-logs'),
  logTrace: (traceId: string) => api.get(`/system/operation-logs/trace/${encodeURIComponent(traceId)}`),
  listRepairs: () => api.get<PageData>('/system/data-repairs'),
  createRepair: (body: Record<string, unknown>) => api.post('/system/data-repairs', body),
  applyRepair: (id: number) => api.post(`/system/data-repairs/${id}/apply`, {}),
}

export const hrApi = {
  employees: () => api.get<PageData>('/hr/employees'),
  getEmployee: (id: number) => api.get(`/hr/employees/${id}`),
  createEmployee: (body: Record<string, unknown>) => api.post('/hr/employees', body),
  updateEmployee: (id: number, body: Record<string, unknown>) => api.put(`/hr/employees/${id}`, body),
  setBadge: (id: number, badge_code: string) => api.put(`/hr/employees/${id}/badge`, { badge_code }),
  openAccount: (id: number, body?: Record<string, unknown>) =>
    api.post(`/hr/employees/${id}/open-account`, body || {}),
  onboards: (params?: string) => api.get<PageData>(`/hr/onboards${params ? `?${params}` : ''}`),
  getOnboard: (id: number) => api.get(`/hr/onboards/${id}`),
  createOnboard: (body: Record<string, unknown>) => api.post('/hr/onboards', body),
  updateOnboard: (id: number, body: Record<string, unknown>) => api.put(`/hr/onboards/${id}`, body),
  confirmOnboard: (id: number, body?: Record<string, unknown>) =>
    api.post(`/hr/onboards/${id}/confirm`, body || {}),
  cancelOnboard: (id: number) => api.post(`/hr/onboards/${id}/cancel`, {}),
  offboards: (params?: string) => api.get<PageData>(`/hr/offboards${params ? `?${params}` : ''}`),
  getOffboard: (id: number) => api.get(`/hr/offboards/${id}`),
  createOffboard: (body: Record<string, unknown>) => api.post('/hr/offboards', body),
  updateOffboard: (id: number, body: Record<string, unknown>) => api.put(`/hr/offboards/${id}`, body),
  confirmOffboard: (id: number) => api.post(`/hr/offboards/${id}/confirm`, {}),
  shifts: () => api.get<PageData>('/hr/shifts'),
  createShift: (body: Record<string, unknown>) => api.post('/hr/shifts', body),
  updateShift: (id: number, body: Record<string, unknown>) => api.put(`/hr/shifts/${id}`, body),
  removeShift: (id: number) => api.del(`/hr/shifts/${id}`),
  attendanceRules: () => api.get<PageData>('/hr/attendance/rules'),
  createAttendanceRule: (body: Record<string, unknown>) => api.post('/hr/attendance/rules', body),
  updateAttendanceRule: (id: number, body: Record<string, unknown>) => api.put(`/hr/attendance/rules/${id}`, body),
  attendanceRecords: (params?: string) =>
    api.get<PageData>(`/hr/attendance/records${params ? `?${params}` : ''}`),
  createAttendanceRecord: (body: Record<string, unknown>) => api.post('/hr/attendance/records', body),
  punch: (body: Record<string, unknown>) => api.post('/hr/attendance/records/punch', body),
  leaveRequests: () => api.get<PageData>('/hr/leave-requests'),
  createLeave: (body: Record<string, unknown>) => api.post('/hr/leave-requests', body),
  cancelLeave: (id: number) => api.post(`/hr/leave-requests/${id}/cancel`, {}),
  overtimePatches: () => api.get<PageData>('/hr/overtime-patches'),
  createOvertimePatch: (body: Record<string, unknown>) => api.post('/hr/overtime-patches', body),
  overtimeStats: (params?: string) =>
    api.get(`/hr/overtime-patches/stats${params ? `?${params}` : ''}`),
  monthStats: (params?: string) =>
    api.get<PageData>(`/hr/attendance/month-stats${params ? `?${params}` : ''}`),
  recalcMonthStats: (body: Record<string, unknown>) => api.post('/hr/attendance/month-stats/recalc', body),
  perfSchemes: () => api.get<PageData>('/hr/performance/schemes'),
  createPerfScheme: (body: Record<string, unknown>) => api.post('/hr/performance/schemes', body),
  perfResults: (params?: string) =>
    api.get<PageData>(`/hr/performance/results${params ? `?${params}` : ''}`),
  createPerfResult: (body: Record<string, unknown>) => api.post('/hr/performance/results', body),
  attPerfSummaries: (params?: string) =>
    api.get<PageData>(`/hr/attendance-perf-summaries${params ? `?${params}` : ''}`),
  visits: () => api.get<PageData>('/hr/visits'),
  createVisit: (body: Record<string, unknown>) => api.post('/hr/visits', body),
  memos: () => api.get<PageData>('/hr/memos'),
  createMemo: (body: Record<string, unknown>) => api.post('/hr/memos', body),
  updateMemo: (id: number, body: Record<string, unknown>) => api.put(`/hr/memos/${id}`, body),
  removeMemo: (id: number) => api.del(`/hr/memos/${id}`),
  journals: (params?: string) =>
    api.get<PageData>(`/hr/employee-journals${params ? `?${params}` : ''}`),
  createJournal: (body: Record<string, unknown>) => api.post('/hr/employee-journals', body),
}

export const payrollApi = {
  calcSheet: (body?: Record<string, unknown>) => api.post('/payroll/sheets/batch-generate', body || {}),
  createSheet: (body: Record<string, unknown>) => api.post('/payroll/sheets', body),
  listSheets: () => api.get<PageData>('/payroll/sheets'),
  wageRates: () => api.get<PageData>('/payroll/wage-rates'),
  createWageRate: (body: Record<string, unknown>) => api.post('/payroll/wage-rates', body),
}

export const approvalApi = {
  tasks: () => api.get<PageData>('/approval/tasks'),
  approve: (id: number, comment?: string) => api.post(`/approval/tasks/${id}/approve`, { comment }),
  reject: (id: number, comment?: string) => api.post(`/approval/tasks/${id}/reject`, { comment }),
}

export const iamApi = {
  users: () => api.get<PageData>('/iam/users'),
  getUser: (id: number) => api.get(`/iam/users/${id}`),
  createUser: (body: Record<string, unknown>) => api.post('/iam/users', body),
  updateUser: (id: number, body: Record<string, unknown>) => api.put(`/iam/users/${id}`, body),
  setRoles: (id: number, role_ids: number[]) => api.put(`/iam/users/${id}/roles`, { role_ids }),
  freeze: (id: number, body?: Record<string, unknown>) => api.post(`/iam/users/${id}/freeze`, body || {}),
  unfreeze: (id: number) => api.post(`/iam/users/${id}/unfreeze`, {}),
  bindEmployee: (id: number, employee_id: number) =>
    api.put(`/iam/users/${id}/bind-employee`, { employee_id }),
  unbindEmployee: (id: number) => api.del(`/iam/users/${id}/bind-employee`),
  getDataScope: (id: number) => api.get(`/iam/users/${id}/data-scope`),
  setDataScope: (id: number, body: Record<string, unknown>) =>
    api.put(`/iam/users/${id}/data-scope`, body),
  hrPermOverview: () => api.get('/iam/hr-perm-overview'),
  roles: () => api.get<{ list: Record<string, unknown>[] }>('/iam/roles'),
  getRole: (id: number) => api.get(`/iam/roles/${id}`),
  createRole: (body: Record<string, unknown>) => api.post('/iam/roles', body),
  updateRole: (id: number, body: Record<string, unknown>) => api.put(`/iam/roles/${id}`, body),
  setPermissions: (id: number, body: Record<string, unknown>) => api.put(`/iam/roles/${id}/permissions`, body),
  setWarehouseScope: (id: number, warehouse_ids: number[]) =>
    api.put(`/iam/roles/${id}/warehouse-scope`, { warehouse_ids }),
  setProcessScope: (id: number, items: Record<string, unknown>[]) =>
    api.put(`/iam/roles/${id}/process-scope`, { items }),
  permissions: () => api.get<{ list: Record<string, unknown>[] }>('/iam/permissions'),
  menus: (role_id?: number | string) => api.get<{ list: Record<string, unknown>[] }>(`/iam/menus${role_id ? `?role_id=${role_id}` : ''}`),
  saveMenus: (items: Record<string, unknown>[]) => api.put('/iam/menus', { items }),
  fieldPolicies: () => api.get<{ list: Record<string, unknown>[] }>('/iam/field-policies'),
  saveFieldPolicies: (items: Record<string, unknown>[]) => api.put('/iam/field-policies', { items }),
  loginPolicy: () => api.get('/iam/login-policy'),
  saveLoginPolicy: (body: Record<string, unknown>) => api.put('/iam/login-policy', body),
  groups: () => api.get<{ list: Record<string, unknown>[] }>('/iam/admin-groups'),
  createGroup: (body: Record<string, unknown>) => api.post('/iam/admin-groups', body),
  sessions: () => api.get<PageData>('/iam/sessions'),
  revokeSession: (id: number) => api.post(`/iam/sessions/${id}/revoke`, {}),
}

export const salesApi = {
  orders: () => api.get<PageData>('/sales/orders'),
  createOrder: (body: Record<string, unknown>) => api.post('/sales/orders', body),
  inquiries: () => api.get<PageData>('/sales/inquiries'),
  createInquiry: (body: Record<string, unknown>) => api.post('/sales/inquiries', body),
  myOrders: () => api.get<PageData>('/sales/my-orders'),
  confirmPreShip: (id: number, body?: Record<string, unknown>) =>
    api.post(`/sales/pre-shipments/${id}/confirm`, body || {}),
  createPreShip: (body: Record<string, unknown>) => api.post('/sales/pre-shipments', body),
}

export const purchaseApi = {
  suppliers: (params?: string) => api.get<PageData>(`/purchase/suppliers${params ? `?${params}` : ''}`),
  getSupplier: (id: number) => api.get(`/purchase/suppliers/${id}`),
  createSupplier: (body: Record<string, unknown>) => api.post('/purchase/suppliers', body),
  updateSupplier: (id: number, body: Record<string, unknown>) => api.put(`/purchase/suppliers/${id}`, body),
  removeSupplier: (id: number) => api.del(`/purchase/suppliers/${id}`),
  qualify: (id: number) => api.post(`/purchase/suppliers/${id}/qualify`, {}),
  freeze: (id: number) => api.post(`/purchase/suppliers/${id}/freeze`, {}),
  blacklist: (id: number) => api.post(`/purchase/suppliers/${id}/blacklist`, {}),
  activate: (id: number) => api.post(`/purchase/suppliers/${id}/activate`, {}),
  licenses: (id: number) => api.get(`/purchase/suppliers/${id}/licenses`),
  saveLicenses: (id: number, items: Record<string, unknown>[]) =>
    api.put(`/purchase/suppliers/${id}/licenses`, { items }),
  supplyItems: (id: number) => api.get(`/purchase/suppliers/${id}/supply-items`),
  saveSupplyItems: (id: number, items: Record<string, unknown>[]) =>
    api.put(`/purchase/suppliers/${id}/supply-items`, { items }),
  performance: (id: number) => api.get(`/purchase/suppliers/${id}/performance`),
  certificateAlerts: (days = 60) => api.get(`/purchase/certificate-alerts?days=${days}`),
  priceHistories: (params?: string) =>
    api.get<PageData>(`/purchase/price-histories${params ? `?${params}` : ''}`),
  volumePrice: () => api.get('/purchase/analytics/volume-price'),
  supplierPerformance: () => api.get('/purchase/analytics/supplier-performance'),
  requests: () => api.get<PageData>('/purchase/requests'),
  createRequest: (body: Record<string, unknown>) => api.post('/purchase/requests', body),
  inbounds: () => api.get<PageData>('/purchase/inbounds'),
  createInbound: (body: Record<string, unknown>) => api.post('/purchase/inbounds', body),
  postInbound: (id: number) => api.post(`/purchase/inbounds/${id}/post`, {}),
  qcs: () => api.get<PageData>('/purchase/incoming-qcs'),
  createQc: (body: Record<string, unknown>) => api.post('/purchase/incoming-qcs', body),
  passQc: (id: number) => api.post(`/purchase/incoming-qcs/${id}/pass`, {}),
  failQc: (id: number) => api.post(`/purchase/incoming-qcs/${id}/fail`, {}),
  createReturn: (body: Record<string, unknown>) => api.post('/purchase/returns', body),
  postReturn: (id: number) => api.post(`/purchase/returns/${id}/post`, {}),
}

export const financeApi = {
  vouchers: () => api.get<PageData>('/finance/vouchers'),
  writeoffs: () => api.get<PageData>('/finance/receipt-writeoffs'),
  createWriteoff: (body: Record<string, unknown>) => api.post('/finance/receipt-writeoffs', body),
}

export const reportApi = {
  boss: () => api.get('/report/dashboards/boss'),
  production: () => api.get('/report/dashboards/production'),
  live: () => api.get('/report/dashboards/live'),
}

export const crmApi = {
  customers: () => api.get<PageData>('/crm/customers'),
  createCustomer: (body: Record<string, unknown>) => api.post('/crm/customers', body),
}

/** Generic module CRUD by meta paths */
export function moduleList(path: string, page = 1, size = 20) {
  const sep = path.includes('?') ? '&' : '?'
  return api.get<PageData>(`${path}${sep}page_num=${page}&page_size=${size}`)
}
export function moduleCreate(path: string, body: Record<string, unknown>) {
  return api.post(path, body)
}
export function moduleGet(path: string, id: number) {
  return api.get(path.replace('{id}', String(id)))
}
export function moduleUpdate(path: string, id: number, body: Record<string, unknown>) {
  return api.put(path.replace('{id}', String(id)), body)
}
export function moduleDelete(path: string, id: number) {
  return api.del(path.replace('{id}', String(id)))
}
export function moduleAction(path: string, id: number, action: string, body?: Record<string, unknown>) {
  const clean = path.replace(/\/\{id\}.*/, '').replace(/\/$/, '')
  return api.post(`${clean}/${id}/${action}`, body || {})
}
