/// Mirrors web/packages/shared employeeModules.ts
enum EmployeeModule { workshop, worker, sales, warehouse }

class EmployeeModuleInfo {
  const EmployeeModuleInfo(this.key, this.title, this.desc, this.route);

  final EmployeeModule key;
  final String title;
  final String desc;
  final String route;
}

const employeeModules = [
  EmployeeModuleInfo(EmployeeModule.workshop, '车间工作台', '扫码、流转、任务、派工、工序与库存', '/workshop'),
  EmployeeModuleInfo(EmployeeModule.worker, '工人报工', '双扫报工、今日核对与考勤提醒', '/worker'),
  EmployeeModuleInfo(EmployeeModule.warehouse, '仓管入库', '采购推送待办、溯源核对、确认入库', '/warehouse'),
  EmployeeModuleInfo(EmployeeModule.sales, '销售外勤', '订单、询价、客户跟进与任务提醒', '/sales'),
];

bool _isAdmin(List<String> perms, List<String> roles) =>
    perms.contains('*:*:*') || roles.contains('sys_admin') || roles.contains('系统管理员');

bool _matchAny(List<String> perms, List<String> needles) {
  final joined = perms.join('\n');
  return needles.any(joined.contains);
}

bool canAccessEmployeeModule(EmployeeModule module, List<String> permissions, List<String> roles) {
  if (_isAdmin(permissions, roles)) return true;
  switch (module) {
    case EmployeeModule.workshop:
      return _matchAny(permissions, ['生产管理', '派工', '车间', '库存管理', '扫码报工', 'production', 'inventory']);
    case EmployeeModule.worker:
      return _matchAny(permissions, ['生产管理', '扫码报工', '计件', '工资管理', 'payroll', '报工']);
    case EmployeeModule.warehouse:
      return roles.contains('warehouse') ||
          roles.contains('仓管员') ||
          _matchAny(permissions, ['库存管理', '仓管', 'warehouse', '入库']);
    case EmployeeModule.sales:
      return _matchAny(permissions, ['销售管理', '客户', '询价', '订单', 'sales', 'crm', 'CRM']);
  }
}

List<EmployeeModuleInfo> visibleEmployeeModules(List<String> permissions, List<String> roles) =>
    employeeModules.where((m) => canAccessEmployeeModule(m.key, permissions, roles)).toList();
