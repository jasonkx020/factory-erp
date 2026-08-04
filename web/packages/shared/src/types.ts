export type ApiCode = 1 | 0

export interface ApiEnvelope<T = unknown> {
  code: ApiCode
  msg: string
  data?: T
}

export interface PageData<T = Record<string, unknown>> {
  list: T[]
  total: number
  page_num?: number
  page_size?: number
}

export interface LoginUser {
  id: number
  login_name: string
  user_type: string
  employee_id?: number
  name?: string
  status?: string
}

export interface LoginData {
  access_token: string
  refresh_token: string
  expires_in: number
  user?: LoginUser
  roles: string[]
  permissions: string[]
}

export interface MeData {
  user: { id: number; login_name: string; user_type: string }
  roles: string[]
  permissions: string[]
  menus: Array<{
    domain: string
    module: string
    menu_key: string
    visible: boolean
    sort_no: number
  }>
  field_policies: Array<{
    role_id: number
    field_key: string
    field_name: string
    visible: boolean
    editable: boolean
  }>
}

export interface ModuleMeta {
  domain: string
  module: string
  phase: number
  list: string
  create?: string
  detail?: string
  update?: string
  remove?: string
  actions: string[]
  readOnly?: boolean
  /** True when module has write/action endpoints but no listable GET collection */
  actionOnly?: boolean
}
