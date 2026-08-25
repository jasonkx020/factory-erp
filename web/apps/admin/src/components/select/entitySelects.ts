import {
  crmApi,
  productApi,
  purchaseApi,
  hrApi,
  iamApi,
  productionApi,
  inventoryApi,
  salesApi,
} from '@erp/shared'

type Row = Record<string, unknown>
type Pageish = { list?: Row[] }

async function asList(res: { code: number; data?: unknown }): Promise<Row[]> {
  if (res.code !== 1) return []
  const data = res.data as Pageish | Row[] | undefined
  if (Array.isArray(data)) return data
  return data?.list || []
}

export async function loadCustomers(): Promise<Row[]> {
  return asList(await crmApi.customers())
}

export async function loadProducts(): Promise<Row[]> {
  return asList(await productApi.list())
}

export async function loadSuppliers(): Promise<Row[]> {
  return asList(await purchaseApi.suppliers())
}

export async function loadEmployees(): Promise<Row[]> {
  return asList(await hrApi.employees())
}

export async function loadUsers(): Promise<Row[]> {
  return asList(await iamApi.users())
}

export async function loadRoles(): Promise<Row[]> {
  return asList(await iamApi.roles())
}

export async function loadWorkshops(): Promise<Row[]> {
  return asList(await hrApi.departments('dept_type=workshop'))
}

export async function loadDepartments(): Promise<Row[]> {
  const res = await hrApi.departments()
  const list = await asList(res)
  return list.map((d) => ({
    ...d,
    name: String(d.path || d.name || ''),
  }))
}

export async function loadWorkTeams(deptId?: number): Promise<Row[]> {
  return asList(await hrApi.workTeams(deptId))
}

export async function loadJobTitles(empType?: string): Promise<Row[]> {
  return asList(await hrApi.jobTitles(empType))
}

export async function ensureJobTitle(name: string, empType?: string): Promise<Row | null> {
  const res = await hrApi.ensureJobTitle({ name, emp_type: empType || '' })
  if (res.code !== 1) return null
  return (res.data as Row) || null
}

export async function loadProcesses(): Promise<Row[]> {
  return asList(await productionApi.processes())
}

export async function loadRoutings(): Promise<Row[]> {
  return asList(await productionApi.listRoutings())
}

export async function loadWarehouses(): Promise<Row[]> {
  return asList(await inventoryApi.warehouses())
}

export async function loadSalesOrders(): Promise<Row[]> {
  return asList(await salesApi.orders())
}

export async function loadProdTasks(): Promise<Row[]> {
  return asList(await productionApi.listTasks())
}

export async function loadDispatches(): Promise<Row[]> {
  return asList(await productionApi.listDispatches())
}

export async function loadStockTxns(): Promise<Row[]> {
  return asList(await inventoryApi.listTxns())
}

export async function loadPurchaseInbounds(): Promise<Row[]> {
  return asList(await purchaseApi.inbounds())
}

export function orderLabel(row: Row): string {
  const no = row.doc_no || row.order_no
  const name = row.customer_name
  if (no && name) return `${no} · ${name}`
  return String(no || row.id || '')
}

export function taskLabel(row: Row): string {
  const no = row.doc_no
  const st = row.status
  if (no && st) return `${no} (${st})`
  return String(no || row.id || '')
}

export function userLabel(row: Row): string {
  const login = row.login_name || row.username
  const name = row.name || row.employee_name
  if (name && login) return `${name} (${login})`
  return String(name || login || row.id || '')
}

export function txnLabel(row: Row): string {
  const no = row.doc_no
  const dir = row.direction || row.biz_type
  if (no && dir) return `${no} · ${dir}`
  return String(no || row.id || '')
}
