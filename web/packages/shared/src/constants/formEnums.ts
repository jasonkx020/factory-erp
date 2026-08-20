export interface FormOption {
  value: string
  label: string
}

export const CURRENCY_OPTIONS: FormOption[] = [
  { value: 'CNY', label: '人民币 CNY' },
  { value: 'USD', label: '美元 USD' },
  { value: 'EUR', label: '欧元 EUR' },
  { value: 'HKD', label: '港币 HKD' },
  { value: 'JPY', label: '日元 JPY' },
]

export const SETTLE_METHOD_OPTIONS: FormOption[] = [
  { value: 'cash', label: '现结' },
  { value: 'monthly', label: '月结' },
  { value: 'cod', label: '货到付款' },
  { value: 'prepaid', label: '预付' },
]

export const CUSTOMER_SOURCE_OPTIONS: FormOption[] = [
  { value: 'phone', label: '电话' },
  { value: 'exhibition', label: '展会' },
  { value: 'referral', label: '转介' },
  { value: 'import', label: '导入' },
  { value: 'online', label: '线上' },
  { value: 'other', label: '其他' },
]

export const PROCESS_TYPE_OPTIONS: FormOption[] = [
  { value: 'gate', label: '入厂' },
  { value: 'wash', label: '清洗' },
  { value: 'cut', label: '切断' },
  { value: 'core', label: '去芯' },
  { value: 'slice', label: '切片' },
  { value: 'peel', label: '去皮' },
  { value: 'dice', label: '切块' },
  { value: 'bag', label: '装袋' },
  { value: 'pack', label: '包装' },
  { value: 'inbound', label: '入库' },
  { value: 'outbound', label: '出库' },
  { value: 'qc', label: '质检' },
  { value: 'other', label: '其他' },
]

/** 工序产量计费：仅 weight|piece 且工人为计件工时才预计算/日结金额 */
export const PROCESS_PAY_MODE_OPTIONS: FormOption[] = [
  { value: 'none', label: '不计费' },
  { value: 'weight', label: '按重量' },
  { value: 'piece', label: '按件' },
]

/** 从枚举选项取中文标签；未知值原样返回 */
export function formOptionLabel(options: FormOption[], value: unknown, fallback = '—'): string {
  const v = String(value ?? '').trim()
  if (!v) return fallback
  const hit = options.find((o) => o.value === v)
  return hit?.label ?? v
}

export const STATION_FLOW_EVENT_OPTIONS: FormOption[] = [
  { value: 'issue', label: '领取' },
  { value: 'return', label: '退库' },
  { value: 'stock_in', label: '入库换码' },
  { value: 'board_close', label: '板结束' },
  { value: 'day_settle', label: '日结' },
  { value: 'process_master', label: '工序主数据' },
]

export const QC_TYPE_OPTIONS: FormOption[] = [
  { value: 'process', label: '工序检' },
  { value: 'final', label: '成品检' },
  { value: 'incoming', label: '来料检' },
]

export const LEAVE_TYPE_OPTIONS: FormOption[] = [
  { value: 'annual', label: '年假' },
  { value: 'personal', label: '事假' },
  { value: 'sick', label: '病假' },
  { value: 'other', label: '其他' },
]

export const PAY_ADJUST_TYPE_OPTIONS: FormOption[] = [
  { value: 'bonus', label: '奖金' },
  { value: 'deduct', label: '扣款' },
  { value: 'manual', label: '手工调整' },
]

/** 月结工资单状态 */
export const PAY_SHEET_STATUS_OPTIONS: FormOption[] = [
  { value: 'draft', label: '草稿' },
  { value: 'confirmed', label: '已确认' },
  { value: 'paid', label: '已发放' },
]

export function paySheetStatusLabel(value: unknown): string {
  return formOptionLabel(PAY_SHEET_STATUS_OPTIONS, value)
}

export const STATUS_ACTIVE_OPTIONS: FormOption[] = [
  { value: 'active', label: '启用' },
  { value: 'inactive', label: '停用' },
]

/** 员工档案人事状态（与启用/停用字典区分） */
export const EMP_STATUS_OPTIONS: FormOption[] = [
  { value: 'active', label: '在职' },
  { value: 'inactive', label: '停用' },
  { value: 'left', label: '离职' },
]

export function empStatusLabel(value: unknown): string {
  return formOptionLabel(EMP_STATUS_OPTIONS, value)
}

/** 工人薪资档案计薪方式 */
export const PAY_TYPE_OPTIONS: FormOption[] = [
  { value: 'piece', label: '计件' },
  { value: 'fixed', label: '固定月薪' },
  { value: 'mixed', label: '混合' },
  { value: 'commission', label: '提成' },
]

export function payTypeLabel(value: unknown): string {
  const v = String(value ?? '').trim().toLowerCase()
  return formOptionLabel(PAY_TYPE_OPTIONS, v || value)
}

export const SUPPLIER_RATING_OPTIONS: FormOption[] = [
  { value: 'A', label: 'A' },
  { value: 'B', label: 'B' },
  { value: 'C', label: 'C' },
  { value: 'D', label: 'D' },
]

export const BASE_UNIT_OPTIONS: FormOption[] = [
  { value: 'kg', label: '千克' },
  { value: 'g', label: '克' },
  { value: 'pcs', label: '件' },
  { value: 'bag', label: '袋' },
  { value: 'box', label: '箱' },
  { value: 'ton', label: '吨' },
]

export const PAY_CHANNEL_OPTIONS: FormOption[] = [
  { value: 'wechat', label: '微信' },
  { value: 'alipay', label: '支付宝' },
  { value: 'bank', label: '银行转账' },
  { value: 'cash', label: '现金' },
  { value: 'other', label: '其他' },
]

export const APPROVAL_DOC_TYPE_OPTIONS: FormOption[] = [
  { value: 'sales_order', label: '销售订单' },
  { value: 'purchase_order', label: '采购单' },
  { value: 'inquiry', label: '询价' },
  { value: 'expense', label: '费用' },
  { value: 'leave', label: '请假' },
]

export const REPAIR_ACTION_OPTIONS: FormOption[] = [
  { value: 'retry_flow', label: '重试流程' },
  { value: 'reopen_box', label: '重开箱码' },
  { value: 'fix_status', label: '修正状态' },
  { value: 'other', label: '其他' },
]

export const REPAIR_TARGET_TYPE_OPTIONS: FormOption[] = [
  { value: 'sales_order', label: '销售订单' },
  { value: 'prod_task', label: '生产任务' },
  { value: 'stock_txn', label: '库存流水' },
  { value: 'box_code', label: '箱码' },
  { value: 'other', label: '其他' },
]

export const CONSIGNMENT_PROGRESS_OPTIONS: FormOption[] = [
  { value: '待投产', label: '待投产' },
  { value: '生产中', label: '生产中' },
  { value: '半成品入库', label: '半成品入库' },
  { value: '已完工', label: '已完工' },
]

export const TIMEZONE_OPTIONS: FormOption[] = [
  { value: 'Asia/Shanghai', label: 'Asia/Shanghai' },
  { value: 'UTC', label: 'UTC' },
]

export const DATE_FORMAT_OPTIONS: FormOption[] = [
  { value: 'YYYY-MM-DD', label: 'YYYY-MM-DD' },
  { value: 'YYYY/MM/DD', label: 'YYYY/MM/DD' },
  { value: 'DD/MM/YYYY', label: 'DD/MM/YYYY' },
]

export const LICENSE_TYPE_OPTIONS: FormOption[] = [
  { value: 'business', label: '营业执照' },
  { value: 'food', label: '食品许可' },
  { value: 'other', label: '其他' },
]

export const OVERTIME_BIZ_OPTIONS: FormOption[] = [
  { value: 'overtime', label: '加班' },
  { value: 'patch', label: '补卡' },
]

/** 工序工价单位（表单存 canonical；库里历史可能是 kg/hour 等别名） */
export const RATE_UNIT_OPTIONS: FormOption[] = [
  { value: 'yuan/kg', label: '元/千克' },
  { value: 'yuan/pcs', label: '元/件' },
  { value: 'yuan/hour', label: '元/小时' },
  { value: 'yuan/day', label: '元/天' },
]

const RATE_UNIT_ALIASES: Record<string, string> = {
  kg: 'yuan/kg',
  'yuan/kg': 'yuan/kg',
  '元/kg': 'yuan/kg',
  '元/千克': 'yuan/kg',
  pcs: 'yuan/pcs',
  piece: 'yuan/pcs',
  'yuan/pcs': 'yuan/pcs',
  '元/件': 'yuan/pcs',
  hour: 'yuan/hour',
  'yuan/hour': 'yuan/hour',
  '元/小时': 'yuan/hour',
  day: 'yuan/day',
  'yuan/day': 'yuan/day',
  '元/天': 'yuan/day',
}

/** 归一化工价单位为表单/接口 canonical 值 */
export function normalizeRateUnit(value: unknown): string {
  const raw = String(value ?? '').trim()
  if (!raw) return 'yuan/kg'
  const lower = raw.toLowerCase()
  return RATE_UNIT_ALIASES[lower] || RATE_UNIT_ALIASES[raw] || 'yuan/kg'
}

export function rateUnitLabel(value: unknown): string {
  return formOptionLabel(RATE_UNIT_OPTIONS, normalizeRateUnit(value))
}

export function statusActiveLabel(value: unknown): string {
  return formOptionLabel(STATUS_ACTIVE_OPTIONS, value)
}
