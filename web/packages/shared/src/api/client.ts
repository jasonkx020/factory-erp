import type { ApiEnvelope, LoginData } from '../types'

const TOKEN_KEY = 'erp_access_token'
const REFRESH_KEY = 'erp_refresh_token'

export function getApiBase(): string {
  if (typeof localStorage !== 'undefined') {
    return localStorage.getItem('erp_api_base') || '/api/v1'
  }
  return '/api/v1'
}

export function getAccessToken(): string {
  if (typeof localStorage === 'undefined') return ''
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function getRefreshToken(): string {
  if (typeof localStorage === 'undefined') return ''
  return localStorage.getItem(REFRESH_KEY) || ''
}

export function setTokens(access: string, refresh?: string) {
  if (typeof localStorage === 'undefined') return
  if (access) localStorage.setItem(TOKEN_KEY, access)
  else localStorage.removeItem(TOKEN_KEY)
  if (refresh !== undefined) {
    if (refresh) localStorage.setItem(REFRESH_KEY, refresh)
    else localStorage.removeItem(REFRESH_KEY)
  }
}

export function clearTokens() {
  setTokens('', '')
}

type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

let refreshing: Promise<boolean> | null = null

async function tryRefresh(): Promise<boolean> {
  const rt = getRefreshToken()
  if (!rt) return false
  try {
    const res = await fetch(`${getApiBase()}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: rt }),
    })
    const data = (await res.json()) as ApiEnvelope<LoginData>
    if (data.code === 1 && data.data?.access_token) {
      setTokens(data.data.access_token, data.data.refresh_token || rt)
      return true
    }
  } catch {
    /* ignore */
  }
  return false
}

export async function apiRequest<T = unknown>(
  method: Method,
  path: string,
  body?: unknown,
  opts?: { skipAuth?: boolean; _retried?: boolean },
): Promise<ApiEnvelope<T>> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (!opts?.skipAuth) {
    const t = getAccessToken()
    if (t) headers.Authorization = `Bearer ${t}`
  }
  const url = path.startsWith('http') ? path : `${getApiBase()}${path.startsWith('/') ? path : '/' + path}`
  let res: Response
  try {
    res = await fetch(url, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch (e) {
    return { code: 0, msg: 'NETWORK_ERROR' }
  }

  if (res.status === 401 && !opts?.skipAuth && !opts?._retried) {
    if (!refreshing) refreshing = tryRefresh().finally(() => { refreshing = null })
    const ok = await refreshing
    if (ok) return apiRequest<T>(method, path, body, { ...opts, _retried: true })
    clearTokens()
    return { code: 0, msg: 'UNAUTHORIZED' }
  }

  let data: ApiEnvelope<T>
  try {
    data = (await res.json()) as ApiEnvelope<T>
  } catch {
    return { code: 0, msg: res.status === 401 ? 'UNAUTHORIZED' : 'BAD_RESPONSE' }
  }
  if (data.code === 0 && data.msg === 'UNAUTHORIZED') clearTokens()
  return data
}

export const api = {
  get: <T = unknown>(path: string) => apiRequest<T>('GET', path),
  post: <T = unknown>(path: string, body?: unknown) => apiRequest<T>('POST', path, body),
  put: <T = unknown>(path: string, body?: unknown) => apiRequest<T>('PUT', path, body),
  patch: <T = unknown>(path: string, body?: unknown) => apiRequest<T>('PATCH', path, body),
  del: <T = unknown>(path: string) => apiRequest<T>('DELETE', path),
}
