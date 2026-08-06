import { api } from './client'
import type { LoginData, MeData, PageData } from '../types'

export const authApi = {
  login: (login_name: string, password: string, client_type = 'admin') =>
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
  units: (productId: number) => api.get<PageData>(`/product/products/${productId}/units`),
  replaceUnits: (productId: number, body: Record<string, unknown>) =>
    api.put(`/product/products/${productId}/units`, body),
  appSorts: (channel = 'app') => api.get<PageData>(`/product/app-sorts?channel=${encodeURIComponent(channel)}`),
  saveAppSorts: (body: Record<string, unknown>) => api.put('/product/app-sorts', body),
  specs: (productId?: number) =>
    api.get<PageData>(productId ? `/product/specs?product_id=${productId}` : '/product/specs'),
  createSpec: (body: Record<string, unknown>) => api.post('/product/specs', body),
  getSpec: (id: number) => api.get(`/product/specs/${id}`),
  updateSpec: (id: number, body: Record<string, unknown>) => api.put(`/product/specs/${id}`, body),
  removeSpec: (id: number) => api.del(`/product/specs/${id}`),
}

export const assetApi = {
  categories: () => api.get<PageData>('/asset/categories'),
  createCategory: (body: Record<string, unknown>) => api.post('/asset/categories', body),
  updateCategory: (id: number, body: Record<string, unknown>) => api.put(`/asset/categories/${id}`, body),
  removeCategory: (id: number) => api.del(`/asset/categories/${id}`),
  list: (params?: string) => api.get<PageData>(params ? `/asset/fixed-assets?${params}` : '/asset/fixed-assets'),
  create: (body: Record<string, unknown>) => api.post('/asset/fixed-assets', body),
  get: (id: number) => api.get(`/asset/fixed-assets/${id}`),
  update: (id: number, body: Record<string, unknown>) => api.put(`/asset/fixed-assets/${id}`, body),
  remove: (id: number) => api.del(`/asset/fixed-assets/${id}`),
  transfers: () => api.get<PageData>('/asset/transfers'),
  createTransfer: (body: Record<string, unknown>) => api.post('/asset/transfers', body),
  updateTransfer: (id: number, body: Record<string, unknown>) => api.put(`/asset/transfers/${id}`, body),
  confirmTransfer: (id: number) => api.post(`/asset/transfers/${id}/confirm`, {}),
  stats: () => api.get('/asset/stats'),
}

export const inventoryApi = {
  warehouses: () => api.get<PageData>('/inventory/warehouses'),
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
  bindBox: (id: number, body?: Record<string, unknown>) =>
    api.post(`/inventory/box-codes/${id}/bind`, body || {}),
  // 盘点
  stocktakes: (workshop = false) =>
    api.get<PageData>(workshop ? '/inventory/workshop-stocktakes' : '/inventory/stocktakes'),
  createStocktake: (body: Record<string, unknown>, workshop = false) =>
    api.post(workshop ? '/inventory/workshop-stocktakes' : '/inventory/stocktakes', body),
  getStocktake: (id: number, workshop = false) =>
    api.get(workshop ? `/inventory/workshop-stocktakes/${id}` : `/inventory/stocktakes/${id}`),
  submitStocktake: (id: number, workshop = false) =>
    api.post(workshop ? `/inventory/workshop-stocktakes/${id}/submit` : `/inventory/stocktakes/${id}/submit`, {}),
  postStocktake: (id: number, workshop = false) =>
    api.post(workshop ? `/inventory/workshop-stocktakes/${id}/post` : `/inventory/stocktakes/${id}/post`, {}),
  stocktakeRecords: () => api.get<PageData>('/inventory/stocktake-records'),
  // 调拨/耗用
  transfers: () => api.get<PageData>('/inventory/transfers'),
  createTransfer: (body: Record<string, unknown>) => api.post('/inventory/transfers', body),
  postTransfer: (id: number) => api.post(`/inventory/transfers/${id}/post`, {}),
  consumes: () => api.get<PageData>('/inventory/consumes'),
  createConsume: (body: Record<string, unknown>) => api.post('/inventory/consumes', body),
  postConsume: (id: number) => api.post(`/inventory/consumes/${id}/post`, {}),
  // 预警 / 质检
  alertShortage: () => api.get('/inventory/alert-rules/shortage'),
  alertExcess: () => api.get('/inventory/alert-rules/excess'),
  upsertAlertShortage: (body: Record<string, unknown>) => api.put('/inventory/alert-rules/shortage', body),
  upsertAlertExcess: (body: Record<string, unknown>) => api.put('/inventory/alert-rules/excess', body),
  inboundQcs: () => api.get<PageData>('/inventory/inbound-qcs'),
  createInboundQc: (body: Record<string, unknown>) => api.post('/inventory/inbound-qcs', body),
  passInboundQc: (id: number, body?: Record<string, unknown>) =>
    api.post(`/inventory/inbound-qcs/${id}/pass`, body || {}),
  failInboundQc: (id: number, body?: Record<string, unknown>) =>
    api.post(`/inventory/inbound-qcs/${id}/fail`, body || {}),
  // 在途/占用/可用
  inTransits: () => api.get<PageData>('/inventory/in-transits'),
  reservations: () => api.get<PageData>('/inventory/reservations'),
  releaseReservation: (id: number) => api.post(`/inventory/reservations/${id}/release`, {}),
  // 期初
  openings: () => api.get<PageData>('/inventory/openings'),
  createOpening: (body: Record<string, unknown>) => api.post('/inventory/openings', body),
  postOpening: (id: number) => api.post(`/inventory/openings/${id}/post`, {}),
  // 组装拆分 / 调价
  assembleSplits: () => api.get<PageData>('/inventory/assemble-splits'),
  createAssembleSplit: (body: Record<string, unknown>) => api.post('/inventory/assemble-splits', body),
  postAssembleSplit: (id: number) => api.post(`/inventory/assemble-splits/${id}/post`, {}),
  priceAdjusts: () => api.get<PageData>('/inventory/price-adjusts'),
  createPriceAdjust: (body: Record<string, unknown>) => api.post('/inventory/price-adjusts', body),
  // 退皮 / 转应付 / 采购退货视图
  peelReturns: () => api.get<PageData>('/inventory/sales-peel-returns'),
  createPeelReturn: (body: Record<string, unknown>) => api.post('/inventory/sales-peel-returns', body),
  postPeelReturn: (id: number) => api.post(`/inventory/sales-peel-returns/${id}/post`, {}),
  materialToPayables: () => api.get<PageData>('/inventory/material-to-payables'),
  createMaterialToPayable: (body: Record<string, unknown>) => api.post('/inventory/material-to-payables', body),
  submitMaterialToPayable: (id: number) => api.post(`/inventory/material-to-payables/${id}/submit`, {}),
  purchaseReturns: () => api.get<PageData>('/inventory/purchase-returns'),
}

export const productionApi = {
  processes: () => api.get<PageData>('/production/processes'),
  createProcess: (body: Record<string, unknown>) => api.post('/production/processes', body),
  updateProcess: (id: number, body: Record<string, unknown>) => api.put(`/production/processes/${id}`, body),
  listTasks: () => api.get<PageData>('/production/tasks'),
  createTask: (body: Record<string, unknown>) => api.post('/production/tasks', body),
  getTask: (id: number) => api.get(`/production/tasks/${id}`),
  updateTask: (id: number, body: Record<string, unknown>) => api.put(`/production/tasks/${id}`, body),
  closeTask: (id: number) => api.post(`/production/tasks/${id}/close`, {}),
  taskItems: (id: number) => api.get(`/production/tasks/${id}/items`),
  addTaskItem: (id: number, body: Record<string, unknown>) => api.post(`/production/tasks/${id}/items`, body),
  listDispatches: () => api.get<PageData>('/production/dispatches'),
  createDispatch: (body: Record<string, unknown>) => api.post('/production/dispatches', body),
  receiveDispatch: (id: number) => api.post(`/production/dispatches/${id}/receive`, {}),
  listFlexDispatches: () => api.get<PageData>('/production/flex-dispatches'),
  createFlexDispatch: (body: Record<string, unknown>) => api.post('/production/flex-dispatches', body),
  reassignFlex: (id: number, body: Record<string, unknown>) =>
    api.post(`/production/flex-dispatches/${id}/reassign`, body),
  listReportWorks: () => api.get<PageData>('/production/report-works'),
  createReportWork: (body: Record<string, unknown>) => api.post('/production/report-works', body),
  confirmReportWork: (id: number, body?: Record<string, unknown>) =>
    api.post(`/production/report-works/${id}/confirm`, body || {}),
  correctReportWork: (id: number, body: Record<string, unknown>) =>
    api.post(`/production/report-works/${id}/correct`, body),
  listRequisitions: () => api.get<PageData>('/production/requisitions'),
  createRequisition: (body: Record<string, unknown>) => api.post('/production/requisitions', body),
  postRequisition: (id: number) => api.post(`/production/requisitions/${id}/post`, {}),
  scan: (body: {
    badge_code: string
    box_code: string
    net_weight?: number
    input_weight?: number
    output_weight?: number
    loss?: number
  }) => api.post('/production/scan', body),
  scanResolve: (body: {
    badge_code: string
    box_code: string
    net_weight?: number
    input_weight?: number
    output_weight?: number
  }) => api.post('/production/scan/resolve', body),
  pieceworkMine: (params?: string) =>
    api.get(`/production/piecework-summaries/mine${params ? `?${params}` : ''}`),
  pieceworkSummaries: (params?: string) =>
    api.get(`/production/piecework-summaries${params ? `?${params}` : ''}`),
  payPiecework: (id: number, body: Record<string, unknown>) =>
    api.post(`/production/piecework-summaries/${id}/pay`, body),
  recalcPiecework: (body?: Record<string, unknown>) =>
    api.post('/production/piecework-summaries/recalc', body || {}),
  flowEvents: () => api.get<PageData>('/production/flow-events'),
  retryFlow: (id: number) => api.post(`/production/flow-events/${id}/retry`, {}),
  flowRules: () => api.get('/production/flow-rules'),
  saveFlowRules: (body: Record<string, unknown>) => api.put('/production/flow-rules', body),
  listRoutings: () => api.get<PageData>('/production/routings'),
  getRouting: (id: number) => api.get(`/production/routings/${id}`),
  workshops: () => api.get<PageData>('/production/workshops'),
  createWorkshop: (body: Record<string, unknown>) => api.post('/production/workshops', body),
  updateWorkshop: (id: number, body: Record<string, unknown>) => api.put(`/production/workshops/${id}`, body),
  workbenchOverview: () => api.get('/production/workshop-workbench/overview'),
  workbenchToday: () => api.get('/production/workshop-workbench/today-tasks'),
  progress: () => api.get<PageData>('/production/progress'),
  taskMerges: () => api.get<PageData>('/production/task-merges'),
  createTaskMerge: (body: Record<string, unknown>) => api.post('/production/task-merges', body),
  confirmTaskMerge: (id: number) => api.post(`/production/task-merges/${id}/confirm`, {}),
  boms: () => api.get<PageData>('/production/boms'),
  createBom: (body: Record<string, unknown>) => api.post('/production/boms', body),
  getBom: (id: number) => api.get(`/production/boms/${id}`),
  generateBom: (body: Record<string, unknown>) => api.post('/production/boms/generate', body),
  mrpRuns: () => api.get<PageData>('/production/mrp-runs'),
  createMrpRun: (body?: Record<string, unknown>) => api.post('/production/mrp-runs', body || {}),
  mrpResults: (id: number) => api.get(`/production/mrp-runs/${id}/results`),
  qcOrders: () => api.get<PageData>('/production/qc-orders'),
  createQcOrder: (body: Record<string, unknown>) => api.post('/production/qc-orders', body),
  completeQcOrder: (id: number, body?: Record<string, unknown>) =>
    api.post(`/production/qc-orders/${id}/complete`, body || {}),
  reworks: () => api.get<PageData>('/production/reworks'),
  createRework: (body: Record<string, unknown>) => api.post('/production/reworks', body),
  closeRework: (id: number) => api.post(`/production/reworks/${id}/close`, {}),
  scraps: () => api.get<PageData>('/production/scraps'),
  createScrap: (body: Record<string, unknown>) => api.post('/production/scraps', body),
  drawingLinks: () => api.get<PageData>('/production/drawing-links'),
  createDrawingLink: (body: Record<string, unknown>) => api.post('/production/drawing-links', body),
  costHidePolicies: () => api.get<PageData>('/production/cost-hide-policies'),
  createCostHidePolicy: (body: Record<string, unknown>) => api.post('/production/cost-hide-policies', body),
  outsources: () => api.get<PageData>('/production/outsources'),
  createOutsource: (body: Record<string, unknown>) => api.post('/production/outsources', body),
  receiveOutsource: (id: number, body?: Record<string, unknown>) =>
    api.post(`/production/outsources/${id}/receive`, body || {}),
  consignments: () => api.get<PageData>('/production/consignments'),
  createConsignment: (body: Record<string, unknown>) => api.post('/production/consignments', body),
  consignmentProgress: (id: number, body: Record<string, unknown>) =>
    api.post(`/production/consignments/${id}/progress`, body),
}

export const bizApi = {
  listEvidence: (bizType: string, bizId: number) =>
    api.get(`/biz/evidences?biz_type=${encodeURIComponent(bizType)}&biz_id=${bizId}`),
  addEvidence: (body: Record<string, unknown>) => api.post('/biz/evidences', body),
  correct: (body: Record<string, unknown>) => api.post('/biz/corrections', body),
}

export const notifyApi = {
  mqttConnect: () => api.get<{ mqtt?: Record<string, unknown> }>('/notify/mqtt-connect'),
  inbox: (params?: string) => api.get(`/notify/inbox${params ? `?${params}` : ''}`),
  readInbox: (id: number) => api.post(`/notify/inbox/${id}/read`, {}),
  tasks: (params?: string) => api.get(`/workflow/tasks${params ? `?${params}` : ''}`),
  claimTask: (id: number) => api.post(`/workflow/tasks/${id}/claim`, {}),
}

export const ticketApi = {
  categories: () => api.get('/workflow/ticket-categories'),
  getCategory: (id: number) => api.get(`/workflow/ticket-categories/${id}`),
  createCategory: (body: Record<string, unknown>) => api.post('/workflow/ticket-categories', body),
  updateCategory: (id: number, body: Record<string, unknown>) =>
    api.put(`/workflow/ticket-categories/${id}`, body),
  getHandlers: (id: number) => api.get(`/workflow/ticket-categories/${id}/handlers`),
  putHandlers: (id: number, handlers: unknown[]) =>
    api.put(`/workflow/ticket-categories/${id}/handlers`, { handlers }),
  handlerPool: (params?: string) =>
    api.get(`/workflow/ticket-handler-pool${params ? `?${params}` : ''}`),
  tickets: (params?: string) => api.get<PageData>(`/workflow/tickets${params ? `?${params}` : ''}`),
  getTicket: (id: number) => api.get(`/workflow/tickets/${id}`),
  createTicket: (body: Record<string, unknown>) => api.post('/workflow/tickets', body),
  assignTicket: (id: number, body: Record<string, unknown>) =>
    api.post(`/workflow/tickets/${id}/assign`, body),
  actionTicket: (id: number, body: Record<string, unknown>) =>
    api.post(`/workflow/tickets/${id}/action`, body),
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
  batchImportEmployees: (body: Record<string, unknown>) => api.post('/hr/employee-imports', body),
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
  confirmOffboard: (id: number, body?: Record<string, unknown>) =>
    api.post(`/hr/offboards/${id}/confirm`, body || {}),
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
  approveLeave: (id: number) => api.post(`/hr/leave-requests/${id}/approve`, {}),
  rejectLeave: (id: number) => api.post(`/hr/leave-requests/${id}/reject`, {}),
  overtimePatches: () => api.get<PageData>('/hr/overtime-patches'),
  createOvertimePatch: (body: Record<string, unknown>) => api.post('/hr/overtime-patches', body),
  approveOvertime: (id: number) => api.post(`/hr/overtime-patches/${id}/approve`, {}),
  rejectOvertime: (id: number) => api.post(`/hr/overtime-patches/${id}/reject`, {}),
  cancelOvertime: (id: number) => api.post(`/hr/overtime-patches/${id}/cancel`, {}),
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
  recalcAttPerf: (body: Record<string, unknown>) => api.post('/hr/attendance-perf-summaries/recalc', body),
  visits: () => api.get<PageData>('/hr/visits'),
  createVisit: (body: Record<string, unknown>) => api.post('/hr/visits', body),
  updateVisit: (id: number, body: Record<string, unknown>) => api.put(`/hr/visits/${id}`, body),
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
  getSheet: (id: number) => api.get(`/payroll/sheets/${id}`),
  adjustSheet: (id: number, body: Record<string, unknown>) => api.post(`/payroll/sheets/${id}/adjust`, body),
  confirmSheet: (id: number) => api.post(`/payroll/sheets/${id}/confirm`, {}),
  paySheet: (id: number) => api.post(`/payroll/sheets/${id}/pay`, {}),
  wageRates: () => api.get<PageData>('/payroll/wage-rates'),
  createWageRate: (body: Record<string, unknown>) => api.post('/payroll/wage-rates', body),
  updateWageRate: (id: number, body: Record<string, unknown>) => api.put(`/payroll/wage-rates/${id}`, body),
  removeWageRate: (id: number) => api.del(`/payroll/wage-rates/${id}`),
  workerProfiles: () => api.get<PageData>('/payroll/worker-profiles'),
  saveWorkerProfile: (body: Record<string, unknown>) => {
    const id = Number(body.id || 0)
    if (id > 0) return api.put(`/payroll/worker-profiles/${id}`, body)
    return api.post('/payroll/worker-profiles', body)
  },
  calculations: () => api.get<PageData>('/payroll/calculations'),
  createCalculation: (body: Record<string, unknown>) => api.post('/payroll/calculations', body),
  commissionRules: () => api.get<PageData>('/payroll/commission-rules'),
  createCommissionRule: (body: Record<string, unknown>) => api.post('/payroll/commission-rules', body),
  updateCommissionRule: (id: number, body: Record<string, unknown>) =>
    api.put(`/payroll/commission-rules/${id}`, body),
  commissionCalcs: () => api.get<PageData>('/payroll/commission-calcs'),
  createCommissionCalc: (body: Record<string, unknown>) => api.post('/payroll/commission-calcs', body),
  runCommission: (body?: Record<string, unknown>) => api.post('/payroll/commission-calcs/run', body || {}),
}

export const approvalApi = {
  tasks: (qs?: string) => api.get<PageData>('/approval/tasks' + (qs ? `?${qs}` : '')),
  getTask: (id: number) => api.get(`/approval/tasks/${id}`),
  createTask: (body: Record<string, unknown>) => api.post('/approval/tasks', body),
  approveTask: (id: number, comment?: string) => api.post(`/approval/tasks/${id}/approve`, { comment }),
  rejectTask: (id: number, comment?: string) => api.post(`/approval/tasks/${id}/reject`, { comment }),
  /** @deprecated use approveTask */
  approve: (id: number, comment?: string) => api.post(`/approval/tasks/${id}/approve`, { comment }),
  /** @deprecated use rejectTask */
  reject: (id: number, comment?: string) => api.post(`/approval/tasks/${id}/reject`, { comment }),

  docReviews: (qs?: string) => api.get<PageData>('/approval/doc-reviews' + (qs ? `?${qs}` : '')),
  createDocReview: (body: Record<string, unknown>) => api.post('/approval/doc-reviews', body),
  approveDocReview: (id: number, comment?: string) =>
    api.post(`/approval/doc-reviews/${id}/approve`, { comment }),
  rejectDocReview: (id: number, comment?: string) =>
    api.post(`/approval/doc-reviews/${id}/reject`, { comment }),

  expenseFinance: (qs?: string) =>
    api.get<PageData>('/approval/expense-finance' + (qs ? `?${qs}` : '')),
  createExpenseFinance: (body: Record<string, unknown>) => api.post('/approval/expense-finance', body),
  approveExpenseFinance: (id: number, comment?: string) =>
    api.post(`/approval/expense-finance/${id}/approve`, { comment }),
  rejectExpenseFinance: (id: number, comment?: string) =>
    api.post(`/approval/expense-finance/${id}/reject`, { comment }),

  inquiryFinance: (qs?: string) =>
    api.get<PageData>('/approval/inquiry-finance' + (qs ? `?${qs}` : '')),
  createInquiryFinance: (body: Record<string, unknown>) => api.post('/approval/inquiry-finance', body),
  approveInquiryFinance: (id: number, comment?: string) =>
    api.post(`/approval/inquiry-finance/${id}/approve`, { comment }),
  rejectInquiryFinance: (id: number, comment?: string) =>
    api.post(`/approval/inquiry-finance/${id}/reject`, { comment }),

  inquiryLines: (qs?: string) =>
    api.get<PageData>('/approval/inquiry-lines' + (qs ? `?${qs}` : '')),
  createInquiryLine: (body: Record<string, unknown>) => api.post('/approval/inquiry-lines', body),
  approveInquiryLine: (id: number, comment?: string) =>
    api.post(`/approval/inquiry-lines/${id}/approve`, { comment }),
  rejectInquiryLine: (id: number, comment?: string) =>
    api.post(`/approval/inquiry-lines/${id}/reject`, { comment }),

  purchases: (qs?: string) => api.get<PageData>('/approval/purchases' + (qs ? `?${qs}` : '')),
  createPurchase: (body: Record<string, unknown>) => api.post('/approval/purchases', body),
  approvePurchase: (id: number, comment?: string) =>
    api.post(`/approval/purchases/${id}/approve`, { comment }),
  rejectPurchase: (id: number, comment?: string) =>
    api.post(`/approval/purchases/${id}/reject`, { comment }),

  purchasePlans: (qs?: string) =>
    api.get<PageData>('/approval/purchase-plans' + (qs ? `?${qs}` : '')),
  createPurchasePlan: (body: Record<string, unknown>) => api.post('/approval/purchase-plans', body),
  approvePurchasePlan: (id: number, comment?: string) =>
    api.post(`/approval/purchase-plans/${id}/approve`, { comment }),
  rejectPurchasePlan: (id: number, comment?: string) =>
    api.post(`/approval/purchase-plans/${id}/reject`, { comment }),

  affairs: () => api.get<PageData>('/approval/affairs'),
  createAffair: (body: Record<string, unknown>) => api.post('/approval/affairs', body),
  approveAffair: (id: number, comment?: string) =>
    api.post(`/approval/affairs/${id}/approve`, { comment }),
  rejectAffair: (id: number, comment?: string) =>
    api.post(`/approval/affairs/${id}/reject`, { comment }),

  expenseRequests: () => api.get<PageData>('/approval/expense-requests'),
  getExpenseRequest: (id: number) => api.get(`/approval/expense-requests/${id}`),
  createExpenseRequest: (body: Record<string, unknown>) => api.post('/approval/expense-requests', body),
  updateExpenseRequest: (id: number, body: Record<string, unknown>) =>
    api.put(`/approval/expense-requests/${id}`, body),
  submitExpenseRequest: (id: number) => api.post(`/approval/expense-requests/${id}/submit`, {}),

  attendance: () => api.get<PageData>('/approval/attendance'),
  createAttendance: (body: Record<string, unknown>) => api.post('/approval/attendance', body),
  approveAttendance: (id: number, body?: Record<string, unknown>) =>
    api.post(`/approval/attendance/${id}/approve`, body || {}),
  rejectAttendance: (id: number, body?: Record<string, unknown>) =>
    api.post(`/approval/attendance/${id}/reject`, body || {}),
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
  orders: (params?: string) => api.get<PageData>(`/sales/orders${params ? `?${params}` : ''}`),
  getOrder: (id: number) => api.get(`/sales/orders/${id}`),
  createOrder: (body: Record<string, unknown>) => api.post('/sales/orders', body),
  updateOrder: (id: number, body: Record<string, unknown>) => api.put(`/sales/orders/${id}`, body),
  submitOrder: (id: number) => api.post(`/sales/orders/${id}/submit`, {}),
  cancelOrder: (id: number) => api.post(`/sales/orders/${id}/cancel`, {}),
  rebuyOrder: (id: number) => api.post(`/sales/orders/${id}/rebuy`, {}),
  orderChanges: (id: number) => api.get(`/sales/orders/${id}/changes`),
  inquiries: () => api.get<PageData>('/sales/inquiries'),
  createInquiry: (body: Record<string, unknown>) => api.post('/sales/inquiries', body),
  getInquiry: (id: number) => api.get(`/sales/inquiries/${id}`),
  approveInquiry: (id: number) => api.post(`/sales/inquiries/${id}/approve`, {}),
  inquiryToOrder: (id: number) => api.post(`/sales/inquiries/${id}/to-order`, {}),
  myOrders: () => api.get<PageData>('/sales/my-orders'),
  preShipments: () => api.get<PageData>('/sales/pre-shipments'),
  createPreShip: (body: Record<string, unknown>) => api.post('/sales/pre-shipments', body),
  reservePreShip: (id: number) => api.post(`/sales/pre-shipments/${id}/reserve`, {}),
  releasePreShip: (id: number) => api.post(`/sales/pre-shipments/${id}/release`, {}),
  confirmPreShip: (id: number, body?: Record<string, unknown>) =>
    api.post(`/sales/pre-shipments/${id}/confirm`, body || {}),
  deliveries: () => api.get<PageData>('/sales/deliveries'),
  createDelivery: (body: Record<string, unknown>) => api.post('/sales/deliveries', body),
  approveDelivery: (id: number) => api.post(`/sales/deliveries/${id}/approve`, {}),
  rejectDelivery: (id: number) => api.post(`/sales/deliveries/${id}/reject`, {}),
  shipDelivery: (id: number, body?: Record<string, unknown>) =>
    api.post(`/sales/deliveries/${id}/ship`, body || {}),
  priceLocks: () => api.get<PageData>('/sales/price-locks'),
  createPriceLock: (body: Record<string, unknown>) => api.post('/sales/price-locks', body),
  contracts: () => api.get<PageData>('/sales/contracts'),
  createContract: (body: Record<string, unknown>) => api.post('/sales/contracts', body),
  quoteHistories: (params?: string) =>
    api.get<PageData>(`/sales/quote-histories${params ? `?${params}` : ''}`),
  rankings: () => api.get('/sales/rankings'),
  salesBoms: () => api.get<PageData>('/sales/sales-boms'),
  createSalesBom: (body: Record<string, unknown>) => api.post('/sales/sales-boms', body),
  saveSalesBomLines: (id: number, body: Record<string, unknown>) =>
    api.put(`/sales/sales-boms/${id}/lines`, body),
  costBudgets: () => api.get<PageData>('/sales/cost-budgets'),
  createCostBudget: (body: Record<string, unknown>) => api.post('/sales/cost-budgets', body),
  quoteCalculator: () => api.get('/sales/quote-calculator'),
  calcQuote: (body: Record<string, unknown>) => api.post('/sales/quote-calculator/calc', body),
  applyQuote: (body: Record<string, unknown>) => api.post('/sales/quote-calculator/apply', body),
  prints: () => api.get<PageData>('/sales/prints'),
  createPrint: (body: Record<string, unknown>) => api.post('/sales/prints', body),
  selfOrders: () => api.get('/sales/self-orders'),
  submitSelfOrder: (body: Record<string, unknown>) => api.post('/sales/self-orders', body),
  outboundSettles: () => api.get<PageData>('/sales/outbound-settles'),
  createOutboundSettle: (body: Record<string, unknown>) => api.post('/sales/outbound-settles', body),
  updateOutboundSettle: (id: number, body: Record<string, unknown>) =>
    api.put(`/sales/outbound-settles/${id}`, body),
  closeOutboundSettle: (id: number) => api.post(`/sales/outbound-settles/${id}/close`, {}),
}

export const crmApi = {
  customers: (params?: string) => api.get<PageData>(`/crm/customers${params ? `?${params}` : ''}`),
  createCustomer: (body: Record<string, unknown>) => api.post('/crm/customers', body),
  getCustomer: (id: number) => api.get(`/crm/customers/${id}`),
  updateCustomer: (id: number, body: Record<string, unknown>) => api.put(`/crm/customers/${id}`, body),
  deleteCustomer: (id: number) => api.del(`/crm/customers/${id}`),
  hideCustomer: (id: number) => api.post(`/crm/customers/${id}/hide`, {}),
  unhideCustomer: (id: number) => api.del(`/crm/customers/${id}/hide`),
  lockCustomer: (id: number) => api.post(`/crm/customers/${id}/lock`, {}),
  unlockCustomer: (id: number) => api.del(`/crm/customers/${id}/lock`),
  getProfile: (id: number) => api.get(`/crm/customers/${id}/profile`),
  updateProfile: (id: number, body: Record<string, unknown>) => api.put(`/crm/customers/${id}/profile`, body),
  opportunities: (params?: string) =>
    api.get<PageData>(`/crm/opportunities${params ? `?${params}` : ''}`),
  createOpportunity: (body: Record<string, unknown>) => api.post('/crm/opportunities', body),
  getOpportunity: (id: number) => api.get(`/crm/opportunities/${id}`),
  updateOpportunity: (id: number, body: Record<string, unknown>) =>
    api.put(`/crm/opportunities/${id}`, body),
  convertOpportunity: (id: number, body?: Record<string, unknown>) =>
    api.post(`/crm/opportunities/${id}/convert`, body || {}),
  followUps: (params?: string) => api.get<PageData>(`/crm/follow-ups${params ? `?${params}` : ''}`),
  createFollowUp: (body: Record<string, unknown>) => api.post('/crm/follow-ups', body),
  getFollowUp: (id: number) => api.get(`/crm/follow-ups/${id}`),
  updateFollowUp: (id: number, body: Record<string, unknown>) => api.put(`/crm/follow-ups/${id}`, body),
  leadAssigns: () => api.get<PageData>('/crm/lead-assigns'),
  assignLead: (body: Record<string, unknown>) => api.post('/crm/lead-assigns', body),
  protectRules: () => api.get<PageData>('/crm/protect-rules'),
  createProtectRule: (body: Record<string, unknown>) => api.post('/crm/protect-rules', body),
  updateProtectRule: (id: number, body: Record<string, unknown>) =>
    api.put(`/crm/protect-rules/${id}`, body),
  deleteProtectRule: (id: number) => api.del(`/crm/protect-rules/${id}`),
  releases: () => api.get<PageData>('/crm/releases'),
  releaseLead: (body: Record<string, unknown>) => api.post('/crm/releases', body),
  imports: () => api.get<PageData>('/crm/imports'),
  importCustomers: (body: Record<string, unknown>) => api.post('/crm/imports', body),
  getImport: (id: number) => api.get(`/crm/imports/${id}`),
  inquiries: (params?: string) => api.get<PageData>(`/crm/inquiries${params ? `?${params}` : ''}`),
  taskReminders: (params?: string) =>
    api.get<PageData>(`/crm/task-reminders${params ? `?${params}` : ''}`),
  createTaskReminder: (body: Record<string, unknown>) => api.post('/crm/task-reminders', body),
  updateTaskReminder: (id: number, body: Record<string, unknown>) =>
    api.put(`/crm/task-reminders/${id}`, body),
  deleteTaskReminder: (id: number) => api.del(`/crm/task-reminders/${id}`),
}

export const fieldLedgerApi = {
  pieceIssueSheets: () => api.get<PageData>('/production/piece-issue-sheets'),
  createPieceIssueSheet: (body: Record<string, unknown>) => api.post('/production/piece-issue-sheets', body),
  getPieceIssueSheet: (id: number) => api.get(`/production/piece-issue-sheets/${id}`),
  generatePieceIssue: (body: Record<string, unknown>) =>
    api.post('/production/piece-issue-sheets/generate', body),
  processReports: (params?: string) =>
    api.get<PageData>(`/production/process-reports${params ? `?${params}` : ''}`),
  toolItems: () => api.get<PageData>('/hr/tool-items'),
  toolIssues: (params?: string) => api.get<PageData>(`/hr/tool-issues${params ? `?${params}` : ''}`),
  createToolIssue: (body: Record<string, unknown>) => api.post('/hr/tool-issues', body),
  returnToolIssue: (id: number, body: Record<string, unknown>) =>
    api.post(`/hr/tool-issues/${id}/return`, body),
  approveToolIssue: (id: number, body?: Record<string, unknown>) =>
    api.post(`/hr/tool-issues/${id}/approve`, body || {}),
  rejectToolIssue: (id: number, body?: Record<string, unknown>) =>
    api.post(`/hr/tool-issues/${id}/reject`, body || {}),
  returnRequestToolIssue: (id: number, body: Record<string, unknown>) =>
    api.post(`/hr/tool-issues/${id}/return-request`, body),
  returnConfirmToolIssue: (id: number, body?: Record<string, unknown>) =>
    api.post(`/hr/tool-issues/${id}/return-confirm`, body || {}),
  weighbridges: () => api.get<PageData>('/inventory/weighbridges'),
  createWeighbridge: (body: Record<string, unknown>) => api.post('/inventory/weighbridges', body),
  updateWeighbridge: (id: number, body: Record<string, unknown>) =>
    api.put(`/inventory/weighbridges/${id}`, body),
}

export const purchaseApi = {
  farmers: (params?: string) => api.get<PageData>(`/purchase/farmers${params ? `?${params}` : ''}`),
  createFarmer: (body: Record<string, unknown>) => api.post('/purchase/farmers', body),
  getFarmer: (id: number) => api.get(`/purchase/farmers/${id}`),
  updateFarmer: (id: number, body: Record<string, unknown>) => api.put(`/purchase/farmers/${id}`, body),
  arrivals: (params?: string) =>
    api.get<PageData>(`/purchase/inbound-arrivals${params ? `?${params}` : ''}`),
  createArrival: (body: Record<string, unknown>) => api.post('/purchase/inbound-arrivals', body),
  getArrival: (id: number) => api.get(`/purchase/inbound-arrivals/${id}`),
  qcArrival: (id: number, body: Record<string, unknown>) =>
    api.post(`/purchase/inbound-arrivals/${id}/qc`, body),
  weighTickets: () => api.get<PageData>('/purchase/weigh-tickets'),
  createWeighTicket: (body: Record<string, unknown>) => api.post('/purchase/weigh-tickets', body),
  getWeighTicket: (id: number) => api.get(`/purchase/weigh-tickets/${id}`),
  qcWeighTicket: (id: number, body: Record<string, unknown>) =>
    api.post(`/purchase/weigh-tickets/${id}/qc`, body),
  confirmWeighTicket: (id: number, body: Record<string, unknown>) =>
    api.post(`/purchase/weigh-tickets/${id}/confirm`, body),
  labelWeighTicket: (id: number) => api.get(`/purchase/weigh-tickets/${id}/label`),
  warehouseConfirmWeigh: (id: number, body?: Record<string, unknown>) =>
    api.post(`/purchase/weigh-tickets/${id}/warehouse-confirm`, body || {}),
  stockInWeighTicket: (id: number) => api.post(`/purchase/weigh-tickets/${id}/stock-in`, {}),
  farmerSettlements: () => api.get<PageData>('/purchase/farmer-settlements'),
  farmerSettlementSummary: () => api.get('/purchase/farmer-settlements/summary'),
  settleFarmer: (body: Record<string, unknown>) => api.post('/purchase/farmer-settlements', body),
  payFarmerSettlement: (id: number, body: Record<string, unknown>) =>
    api.post(`/purchase/farmer-settlements/${id}/pay`, body),
  traceLot: (code: string) => api.get(`/purchase/trace-lots/${encodeURIComponent(code)}`),
  verifyTraceLot: (body: Record<string, unknown>) => api.post('/purchase/trace-lots/verify', body),
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
  getRequest: (id: number) => api.get(`/purchase/requests/${id}`),
  updateRequest: (id: number, body: Record<string, unknown>) => api.put(`/purchase/requests/${id}`, body),
  submitRequest: (id: number) => api.post(`/purchase/requests/${id}/submit`, {}),
  approveRequest: (id: number) => api.post(`/purchase/requests/${id}/approve`, {}),
  rejectRequest: (id: number, reason?: string) =>
    api.post(`/purchase/requests/${id}/reject`, { reason }),
  requestToPlan: (id: number) => api.post(`/purchase/requests/${id}/to-plan`, {}),
  plans: () => api.get<PageData>('/purchase/plans'),
  createPlan: (body: Record<string, unknown>) => api.post('/purchase/plans', body),
  getPlan: (id: number) => api.get(`/purchase/plans/${id}`),
  updatePlan: (id: number, body: Record<string, unknown>) => api.put(`/purchase/plans/${id}`, body),
  submitPlan: (id: number) => api.post(`/purchase/plans/${id}/submit`, {}),
  approvePlan: (id: number) => api.post(`/purchase/plans/${id}/approve`, {}),
  planToInbound: (id: number, body?: Record<string, unknown>) =>
    api.post(`/purchase/plans/${id}/to-inbound`, body || {}),
  inbounds: () => api.get<PageData>('/purchase/inbounds'),
  createInbound: (body: Record<string, unknown>) => api.post('/purchase/inbounds', body),
  getInbound: (id: number) => api.get(`/purchase/inbounds/${id}`),
  postInbound: (id: number) => api.post(`/purchase/inbounds/${id}/post`, {}),
  qcs: () => api.get<PageData>('/purchase/incoming-qcs'),
  createQc: (body: Record<string, unknown>) => api.post('/purchase/incoming-qcs', body),
  passQc: (id: number) => api.post(`/purchase/incoming-qcs/${id}/pass`, {}),
  failQc: (id: number) => api.post(`/purchase/incoming-qcs/${id}/fail`, {}),
  returns: () => api.get<PageData>('/purchase/returns'),
  createReturn: (body: Record<string, unknown>) => api.post('/purchase/returns', body),
  getReturn: (id: number) => api.get(`/purchase/returns/${id}`),
  postReturn: (id: number) => api.post(`/purchase/returns/${id}/post`, {}),
  tasks: () => api.get<PageData>('/purchase/tasks'),
  createTask: (body: Record<string, unknown>) => api.post('/purchase/tasks', body),
  assignTask: (id: number, body?: Record<string, unknown>) =>
    api.post(`/purchase/tasks/${id}/assign`, body || {}),
  completeTask: (id: number) => api.post(`/purchase/tasks/${id}/complete`, {}),
}

export const financeApi = {
  // 账目
  subjects: () => api.get<PageData>('/finance/account-subjects'),
  createSubject: (body: Record<string, unknown>) => api.post('/finance/account-subjects', body),
  updateSubject: (id: number, body: Record<string, unknown>) => api.put(`/finance/account-subjects/${id}`, body),
  removeSubject: (id: number) => api.del(`/finance/account-subjects/${id}`),
  // 流水 / 收支
  ledger: () => api.get<PageData>('/finance/ledger-entries'),
  createLedger: (body: Record<string, unknown>) => api.post('/finance/ledger-entries', body),
  incomeExpenses: () => api.get<PageData>('/finance/income-expenses'),
  // 订单 / 小程序
  orders: () => api.get<PageData>('/finance/orders'),
  miniprogramBills: () => api.get<PageData>('/finance/miniprogram-bills'),
  createMiniprogramBill: (body: Record<string, unknown>) => api.post('/finance/miniprogram-bills', body),
  reconcileMiniprogram: (body?: Record<string, unknown>) =>
    api.post('/finance/miniprogram-bills/reconcile', body || {}),
  // 凭证 / 发票
  vouchers: () => api.get<PageData>('/finance/vouchers'),
  createVoucher: (body: Record<string, unknown>) => api.post('/finance/vouchers', body),
  getVoucher: (id: number) => api.get(`/finance/vouchers/${id}`),
  approveVoucher: (id: number) => api.post(`/finance/vouchers/${id}/approve`, {}),
  postVoucher: (id: number) => api.post(`/finance/vouchers/${id}/post`, {}),
  invoices: () => api.get<PageData>('/finance/invoices'),
  createInvoice: (body: Record<string, unknown>) => api.post('/finance/invoices', body),
  removeInvoice: (id: number) => api.del(`/finance/invoices/${id}`),
  // 收款核单 / 认款
  writeoffs: () => api.get<PageData>('/finance/receipt-writeoffs'),
  createWriteoff: (body: Record<string, unknown>) => api.post('/finance/receipt-writeoffs', body),
  confirmWriteoff: (id: number) => api.post(`/finance/receipt-writeoffs/${id}/confirm`, {}),
  recognitions: () => api.get<PageData>('/finance/payment-recognitions'),
  createRecognition: (body: Record<string, unknown>) => api.post('/finance/payment-recognitions', body),
  confirmRecognition: (id: number) => api.post(`/finance/payment-recognitions/${id}/confirm`, {}),
  // 结汇
  fxSettlements: () => api.get<PageData>('/finance/fx-settlements'),
  fxQuery: () => api.get<PageData>('/finance/fx-settlements/query'),
  createFx: (body: Record<string, unknown>) => api.post('/finance/fx-settlements', body),
  confirmFx: (id: number) => api.post(`/finance/fx-settlements/${id}/confirm`, {}),
  // 分摊 / 预警 / 对账
  allocations: () => api.get<PageData>('/finance/cost-allocations'),
  createAllocation: (body: Record<string, unknown>) => api.post('/finance/cost-allocations', body),
  revokeAllocation: (id: number) => api.post(`/finance/cost-allocations/${id}/revoke`, {}),
  alerts: () => api.get<PageData>('/finance/receipt-alerts'),
  createAlert: (body: Record<string, unknown>) => api.post('/finance/receipt-alerts', body),
  handleAlert: (id: number, body?: Record<string, unknown>) =>
    api.post(`/finance/receipt-alerts/${id}/handle`, body || {}),
  reconciles: () => api.get<PageData>('/finance/cashier-reconciles'),
  createReconcile: (body: Record<string, unknown>) => api.post('/finance/cashier-reconciles', body),
  confirmReconcile: (id: number) => api.post(`/finance/cashier-reconciles/${id}/confirm`, {}),
  // 预收预付 / 往来
  prepays: () => api.get<PageData>('/finance/prepay-prepaids'),
  createPrepay: (body: Record<string, unknown>) => api.post('/finance/prepay-prepaids', body),
  arapAdjusts: () => api.get<PageData>('/finance/arap-adjusts'),
  createArap: (body: Record<string, unknown>) => api.post('/finance/arap-adjusts', body),
  postArap: (id: number) => api.post(`/finance/arap-adjusts/${id}/post`, {}),
  // 成本 / 合同利润 / 退货财务
  costAccountings: () => api.get<PageData>('/finance/cost-accountings'),
  createCostAccounting: (body: Record<string, unknown>) => api.post('/finance/cost-accountings', body),
  calcCost: (id: number) => api.post(`/finance/cost-accountings/${id}/calc`, {}),
  costTraces: () => api.get<PageData>('/finance/cost-traces'),
  contractProfits: () => api.get<PageData>('/finance/contract-profits'),
  recalcContractProfit: (body?: Record<string, unknown>) =>
    api.post('/finance/contract-profits/recalc', body || {}),
  returnFinances: () => api.get<PageData>('/finance/sales-return-finances'),
  createReturnFinance: (body: Record<string, unknown>) => api.post('/finance/sales-return-finances', body),
  confirmReturnFinance: (id: number) => api.post(`/finance/sales-return-finances/${id}/confirm`, {}),
  // 审批 / 资金 / 报表 / 月结
  approvals: () => api.get<PageData>('/finance/approvals'),
  approveFinance: (id: number) => api.post(`/finance/approvals/${id}/approve`, {}),
  rejectFinance: (id: number) => api.post(`/finance/approvals/${id}/reject`, {}),
  fundAccounts: () => api.get<PageData>('/finance/fund-accounts'),
  createFundAccount: (body: Record<string, unknown>) => api.post('/finance/fund-accounts', body),
  fundTransfers: () => api.get<PageData>('/finance/fund-transfers'),
  createFundTransfer: (body: Record<string, unknown>) => api.post('/finance/fund-transfers', body),
  postFundTransfer: (id: number) => api.post(`/finance/fund-transfers/${id}/post`, {}),
  statements: () => api.get<PageData>('/finance/statements'),
  generateStatements: (body?: Record<string, unknown>) =>
    api.post('/finance/statements/generate', body || {}),
  exportStatement: (code: string) => api.get(`/finance/statements/${code}/export`),
  monthCloses: () => api.get<PageData>('/finance/month-closes'),
  closeMonth: (body: Record<string, unknown>) => api.post('/finance/month-closes', body),
  reopenMonth: (id: number) => api.post(`/finance/month-closes/${id}/reopen`, {}),
}

export const reportApi = {
  boss: () => api.get('/report/dashboards/boss'),
  bossWidgets: () => api.get('/report/dashboards/boss/widgets'),
  replaceBossWidgets: (body: Record<string, unknown>) =>
    api.put('/report/dashboards/boss/widgets', body),
  production: () => api.get('/report/dashboards/production'),
  live: () => api.get('/report/dashboards/live'),
  enterprise: () => api.get<PageData>('/report/enterprise'),
  enterpriseByCode: (code: string) => api.get(`/report/enterprise/${code}`),
  daily: (bizDate?: string) =>
    api.get('/report/daily' + (bizDate ? `?biz_date=${encodeURIComponent(bizDate)}` : '')),
  crmStats: () => api.get('/report/crm-stats'),
  inquiries: () => api.get<PageData>('/report/inquiry-queries'),
  followUps: () => api.get<PageData>('/report/follow-ups'),
  grossProfit: () => api.get('/report/gross-profit'),
  qc: () => api.get<PageData>('/report/qc'),
  accounts: () => api.get('/report/accounts'),
  stockTxns: () => api.get<PageData>('/report/stock-txns'),
  stockLedger: () => api.get<PageData>('/report/stock-ledger'),
  salesWeight: () => api.get<PageData>('/report/sales-weight'),
  productSales: () => api.get<PageData>('/report/product-sales'),
  logistics: () => api.get<PageData>('/report/logistics'),
  costProfit: () => api.get('/report/cost-profit'),
  balanceSheet: () => api.get('/report/balance-sheet'),
  cashFlow: () => api.get('/report/cash-flow'),
  incomeStatement: () => api.get('/report/income-statement'),
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
export function moduleReplace(path: string, body: Record<string, unknown>) {
  return api.put(path, body)
}
