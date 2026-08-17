import { computed, ref } from 'vue'
import { systemApi } from '@erp/shared'

export type CarrierCodeUnit = 'board' | 'box'

export type CarrierCodeLabels = {
  unit: CarrierCodeUnit
  codeLabel: string
  short: string
  manageTitle: string
  splitVerb: string
}

const unitRef = ref<CarrierCodeUnit>('board')
let loadPromise: Promise<void> | null = null

export function resolveCarrierLabels(unit: string | null | undefined): CarrierCodeLabels {
  if (String(unit || '').toLowerCase() === 'box') {
    return {
      unit: 'box',
      codeLabel: '箱码',
      short: '箱',
      manageTitle: '箱码管理',
      splitVerb: '分箱',
    }
  }
  return {
    unit: 'board',
    codeLabel: '板码',
    short: '板',
    manageTitle: '板码管理',
    splitVerb: '分板',
  }
}

export function setCarrierCodeUnit(unit: string | null | undefined) {
  unitRef.value = resolveCarrierLabels(unit).unit
}

export async function loadCarrierCodeUnit(force = false): Promise<CarrierCodeUnit> {
  if (!force && loadPromise) {
    await loadPromise
    return unitRef.value
  }
  loadPromise = (async () => {
    try {
      const r = await systemApi.settings()
      const data = (r.data || {}) as { list?: Record<string, unknown>[]; carrier_code_unit?: string }
      const row = Array.isArray(data.list) && data.list.length ? data.list[0] : (data as Record<string, unknown>)
      setCarrierCodeUnit(String(row?.carrier_code_unit || 'board'))
    } catch {
      /* keep default board */
    }
  })()
  await loadPromise
  return unitRef.value
}

/** Permission / menu module key stays fixed; only display title changes. */
export const CARRIER_MODULE_KEY = '箱码管理'

export function displayModuleTitle(moduleName: string): string {
  if (moduleName === CARRIER_MODULE_KEY) {
    return resolveCarrierLabels(unitRef.value).manageTitle
  }
  return moduleName
}

export function useCarrierCodeLabel() {
  const labels = computed(() => resolveCarrierLabels(unitRef.value))
  const codeLabel = computed(() => labels.value.codeLabel)
  const short = computed(() => labels.value.short)
  const manageTitle = computed(() => labels.value.manageTitle)
  const splitVerb = computed(() => labels.value.splitVerb)
  const unit = computed(() => labels.value.unit)

  return {
    unit,
    codeLabel,
    short,
    manageTitle,
    splitVerb,
    labels,
    ensureLoaded: () => loadCarrierCodeUnit(false),
    refresh: () => loadCarrierCodeUnit(true),
    setUnit: setCarrierCodeUnit,
    displayModuleTitle,
    CARRIER_MODULE_KEY,
  }
}
