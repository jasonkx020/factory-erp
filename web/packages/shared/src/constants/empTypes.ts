export interface EmpTypeOption {
  value: string
  label: string
}

// 员工类型：各类型采集的档案字段一致，仅类别不同（影响入职默认角色与工资口径）
export const EMP_TYPE_OPTIONS: EmpTypeOption[] = [
  { value: 'piece', label: '计件工' },
  { value: 'temp', label: '临时工' },
  { value: 'fixed', label: '固定工' },
  { value: 'office', label: '职能/内勤' },
]

export const DEFAULT_EMP_TYPE: string = 'piece'

export const EMP_TYPE_LABEL: Record<string, string> = {
  piece: '计件工',
  temp: '临时工',
  fixed: '固定工',
  office: '职能/内勤',
  admin: '系统管理',
  // 演示/角色编码写入 emp_type 时的兼容显示
  warehouse: '仓管',
  sales: '销售',
  purchase: '采购',
  qc: '质检',
  foreman: '车间主任',
  finance: '财务',
  sys_admin: '系统管理员',
  hr: '人事',
}

export function empTypeLabel(value: unknown): string {
  const key = String(value ?? '').trim()
  if (!key) return '—'
  return EMP_TYPE_LABEL[key] || EMP_TYPE_LABEL[key.toLowerCase()] || key
}
