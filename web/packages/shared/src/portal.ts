/** Dev ports and production public paths — aligned with demo layout */
export const TERMINAL_DEV_PORTS = {
  portal: 5170,
  admin: 5173,
  workshop: 5174,
  worker: 5175,
  sales: 5176,
  boss: 5177,
  customer: 5178,
} as const

export const TERMINAL_PROD_PATHS = {
  portal: '/',
  admin: '/admin/',
  workshop: '/front/workshop/',
  worker: '/front/worker/',
  sales: '/front/sales/',
  boss: '/front/boss/',
  customer: '/user/',
} as const

export type TerminalKey = keyof typeof TERMINAL_PROD_PATHS

export function isDevRuntime(): boolean {
  try {
    return Boolean(import.meta.env?.DEV)
  } catch {
    return false
  }
}

/** URL of the portal home (for 「返回入口」) */
export function portalHomeUrl(): string {
  if (isDevRuntime()) return `http://127.0.0.1:${TERMINAL_DEV_PORTS.portal}/`
  return TERMINAL_PROD_PATHS.portal
}

/** URL of a terminal entry (for portal cards) */
export function terminalUrl(key: Exclude<TerminalKey, 'portal'>): string {
  if (isDevRuntime()) return `http://127.0.0.1:${TERMINAL_DEV_PORTS[key]}/`
  return TERMINAL_PROD_PATHS[key]
}
