/// Primary role codes for employee app workbench (mirrors web employeeModules).
const rolePriority = [
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

/// Normalize role aliases to a canonical workbench key.
String? canonicalRole(String role) {
  switch (role) {
    case 'purchase':
    case '采购员':
    case 'qc':
    case '质检':
      return 'purchase';
    case 'warehouse':
    case '仓管员':
      return 'warehouse';
    case 'foreman':
    case '车间主任':
      return 'foreman';
    case 'piece':
    case '计件工':
    case 'fixed':
    case '固定工':
      return 'piece';
    case 'sales':
    case '销售员':
      return 'sales';
    case 'finance':
    case '财务':
      return 'finance';
    case 'sys_admin':
    case '系统管理员':
      return 'sys_admin';
    default:
      return null;
  }
}

String roleDisplayName(String canonical) {
  switch (canonical) {
    case 'purchase':
      return '过磅收货';
    case 'warehouse':
      return '仓管作业';
    case 'foreman':
      return '车间主任';
    case 'piece':
      return '工人报工';
    case 'sales':
      return '销售外勤';
    case 'finance':
      return '收款协同';
    case 'sys_admin':
      return '管理员';
    default:
      return canonical;
  }
}

/// Pick primary role from user's roles by fixed priority.
String? resolvePrimaryRole(List<String> roles, {String? preferred}) {
  if (preferred != null) {
    final c = canonicalRole(preferred);
    if (c != null && roles.any((r) => canonicalRole(r) == c)) return c;
  }
  final seen = <String>{};
  for (final p in rolePriority) {
    final c = canonicalRole(p);
    if (c == null || seen.contains(c)) continue;
    seen.add(c);
    if (roles.any((r) => canonicalRole(r) == c)) return c;
  }
  return null;
}

/// Distinct business roles the user can switch among (excludes pure admin-only if other roles exist).
List<String> selectableBusinessRoles(List<String> roles) {
  final out = <String>[];
  final seen = <String>{};
  for (final r in roles) {
    final c = canonicalRole(r);
    if (c == null || c == 'sys_admin') continue;
    if (seen.add(c)) out.add(c);
  }
  if (out.isEmpty && roles.any((r) => canonicalRole(r) == 'sys_admin')) {
    return ['sys_admin'];
  }
  return out;
}
