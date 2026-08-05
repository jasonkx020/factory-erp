/** Dev ports and production public paths */
export const TERMINAL_DEV_PORTS = {
  portal: 5170,
  admin: 5173,
  employee: 5174,
  boss: 5177,
} as const

export const TERMINAL_PROD_PATHS = {
  portal: '/',
  admin: '/admin/',
  employee: '/front/',
  boss: '/front/boss/',
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
