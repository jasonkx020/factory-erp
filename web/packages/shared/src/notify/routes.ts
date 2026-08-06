/** Map workflow notify event_key → employee app path (relative). */
export function employeeRouteForEvent(eventKey: string | undefined | null): string | null {
  const k = String(eventKey || '')
  if (k === 'purchase.weigh_confirmed') return '/m/warehouse'
  if (k === 'production.report_confirmed') return '/m/workshop'
  if (k === 'payroll.labor_paid') return '/m/worker'
  if (k === 'purchase.stocked' || k === 'purchase.settle_paid') return '/'
  if (k.startsWith('workflow.ticket')) return '/tickets'
  return null
}

/** Admin app path for inbox deep-link. */
export function adminRouteForEvent(eventKey: string | undefined | null): string | null {
  const k = String(eventKey || '')
  if (k === 'purchase.weigh_confirmed') return '/warehouse/inbound'
  if (k.startsWith('workflow.ticket')) return '/workflow/tickets'
  return null
}

export function parsePayload(raw: unknown): Record<string, unknown> {
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) return raw as Record<string, unknown>
  if (typeof raw === 'string' && raw.trim()) {
    try {
      const v = JSON.parse(raw)
      if (v && typeof v === 'object') return v as Record<string, unknown>
    } catch {
      /* ignore */
    }
  }
  return {}
}
