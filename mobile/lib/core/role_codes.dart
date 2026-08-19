/// Role codes used by employee app workbench (align with IAM seed + employeeModules).

/// Priority: first match in [roles] wins as primary business role.
const List<String> employeeRolePriority = [
  'purchase',
  '采购员',
  'qc',
  '质检',
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
  receiving, // purchase / qc
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
    case WorkbenchRole.warehouse:
      return '仓管作业';
    case WorkbenchRole.worker:
      return '生产';
    case WorkbenchRole.workshop:
      return '班组管理';
    case WorkbenchRole.sales:
      return '销售外勤';
    case WorkbenchRole.collab:
      return '收款协同';
    case WorkbenchRole.admin:
      return '全部模块';
    case WorkbenchRole.none:
      return '未分配';
  }
}

WorkbenchRole workbenchRoleFromCode(String code) {
  switch (code) {
    case 'purchase':
    case '采购员':
    case 'qc':
    case '质检':
      return WorkbenchRole.receiving;
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

/// Resolve primary workbench role from IAM role codes (priority order).
WorkbenchRole resolvePrimaryWorkbenchRole(List<String> roles) {
  for (final code in employeeRolePriority) {
    if (roles.contains(code)) {
      final wr = workbenchRoleFromCode(code);
      if (wr != WorkbenchRole.none) return wr;
    }
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
  if (roles.contains('sys_admin') || roles.contains('系统管理员')) {
    if (seen.add(WorkbenchRole.admin)) out.add(WorkbenchRole.admin);
  }
  return out;
}
