// Role codes used by employee app workbench (align with IAM seed + employeeModules).

bool roleCodeIsQc(String code) {
  final c = code.trim().toLowerCase();
  return c == 'qc' || c == '质检' || c == '质检员' || c.contains('质检');
}

bool roleCodeIsPurchase(String code) {
  final c = code.trim().toLowerCase();
  return c == 'purchase' || c == '采购员' || c == '采购';
}

bool rolesHasQc(List<String> roles) => roles.any(roleCodeIsQc);

bool rolesHasPurchase(List<String> roles) => roles.any(roleCodeIsPurchase);

/// 纯质检（无采购角色）：应进入质检壳，禁止采购建单页。
bool rolesPreferQcShell(List<String> roles) => rolesHasQc(roles) && !rolesHasPurchase(roles);

/// Priority: first match in [roles] wins as primary business role.
const List<String> employeeRolePriority = [
  'purchase',
  '采购员',
  'qc',
  '质检',
  '质检员',
  'warehouse',
  '仓管员',
  'foreman',
  '车间主任',
  'piece',
  '计件工',
  'fixed',
  '固定工',
  'sales',
  '销售员',
  'finance',
  '财务',
  'sys_admin',
  '系统管理员',
];

/// Canonical workbench keys derived from role codes.
enum WorkbenchRole {
  receiving, // purchase
  qc, // 质检：工单判定，不进采购建单
  warehouse,
  workshop, // foreman
  worker, // piece / fixed
  sales,
  collab, // finance (收款协同为主)
  admin, // sys_admin overview
  none,
}

String workbenchRoleLabel(WorkbenchRole r) {
  switch (r) {
    case WorkbenchRole.receiving:
      return '采购';
    case WorkbenchRole.qc:
      return '质检';
    case WorkbenchRole.warehouse:
      return '仓管作业';
    case WorkbenchRole.worker:
      return '生产';
    case WorkbenchRole.workshop:
      return '班组管理';
    case WorkbenchRole.sales:
      return '销售外勤';
    case WorkbenchRole.collab:
      return '结算财务';
    case WorkbenchRole.admin:
      return '全部模块';
    case WorkbenchRole.none:
      return '未分配';
  }
}

WorkbenchRole workbenchRoleFromCode(String code) {
  final raw = code.trim();
  if (roleCodeIsPurchase(raw)) return WorkbenchRole.receiving;
  if (roleCodeIsQc(raw)) return WorkbenchRole.qc;
  switch (raw) {
    case 'warehouse':
    case '仓管员':
      return WorkbenchRole.warehouse;
    case 'foreman':
    case '车间主任':
      return WorkbenchRole.workshop;
    case 'piece':
    case '计件工':
    case 'fixed':
    case '固定工':
      return WorkbenchRole.worker;
    case 'sales':
    case '销售员':
      return WorkbenchRole.sales;
    case 'finance':
    case '财务':
      return WorkbenchRole.collab;
    case 'sys_admin':
    case '系统管理员':
      return WorkbenchRole.admin;
    default:
      return WorkbenchRole.none;
  }
}

bool _rolesContainCode(List<String> roles, String code) {
  final want = code.trim().toLowerCase();
  for (final r in roles) {
    if (r.trim().toLowerCase() == want) return true;
  }
  return false;
}

/// Resolve primary workbench role from IAM role codes (priority order).
WorkbenchRole resolvePrimaryWorkbenchRole(List<String> roles) {
  for (final code in employeeRolePriority) {
    if (_rolesContainCode(roles, code)) {
      final wr = workbenchRoleFromCode(code);
      if (wr != WorkbenchRole.none) return wr;
    }
  }
  for (final r in roles) {
    final wr = workbenchRoleFromCode(r);
    if (wr != WorkbenchRole.none) return wr;
  }
  return WorkbenchRole.none;
}

/// Distinct business workbench roles the user can switch among (excludes admin/none until needed).
List<WorkbenchRole> availableWorkbenchRoles(List<String> roles) {
  final seen = <WorkbenchRole>{};
  final out = <WorkbenchRole>[];
  for (final code in roles) {
    final wr = workbenchRoleFromCode(code);
    if (wr == WorkbenchRole.none || wr == WorkbenchRole.admin) continue;
    if (seen.add(wr)) out.add(wr);
  }
  if (_rolesContainCode(roles, 'sys_admin') || _rolesContainCode(roles, '系统管理员')) {
    if (seen.add(WorkbenchRole.admin)) out.add(WorkbenchRole.admin);
  }
  return out;
}
