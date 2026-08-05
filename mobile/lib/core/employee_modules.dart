/// Mirrors web/packages/shared employeeModules.ts
enum EmployeeModule { workshop, worker, receiving, warehouse, sales, assets, collab, knowledge, mine }

class EmployeeModuleInfo {
  const EmployeeModuleInfo(this.key, this.title, this.desc, this.route, this.icon);

  final EmployeeModule key;
  final String title;
  final String desc;
  final String route;
  final String icon;
}

const employeeModules = [
  EmployeeModuleInfo(EmployeeModule.workshop, '车间工作台', '扫码、派工、灵活派发、质检废料', '/workshop', 'precision_manufacturing'),
  EmployeeModuleInfo(EmployeeModule.worker, '工人报工', '双扫报工、今日核对与联动领料', '/worker', 'badge'),
  EmployeeModuleInfo(EmployeeModule.receiving, '过磅收货', '农户过磅、质检、出码推仓', '/receiving', 'scale'),
  EmployeeModuleInfo(EmployeeModule.warehouse, '仓管作业', '待入库、库存、箱码、盘点', '/warehouse', 'warehouse'),
  EmployeeModuleInfo(EmployeeModule.sales, '销售外勤', '订单、发货进度、报价与跟进', '/sales', 'storefront'),
  EmployeeModuleInfo(EmployeeModule.assets, '固定资产', '资产查询与内部转移申请', '/assets', 'precision_manufacturing'),
  EmployeeModuleInfo(EmployeeModule.collab, '收款协同', '收款预警处理与销售认款', '/collab', 'payments'),
  EmployeeModuleInfo(EmployeeModule.knowledge, '资料中心', '知识库、图纸、公告、学堂', '/knowledge', 'menu_book'),
  EmployeeModuleInfo(EmployeeModule.mine, '我的', '打卡、假勤、审批、工资与消息', '/mine', 'person'),
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
      return _matchAny(permissions, ['生产管理', '派工', '车间', '库存管理', '扫码报工', 'production', 'inventory']) ||
          roles.contains('foreman') ||
          roles.contains('车间主任');
    case EmployeeModule.worker:
      return _matchAny(permissions, ['生产管理', '扫码报工', '计件', '工资管理', 'payroll', '报工']) ||
          roles.contains('piece') ||
          roles.contains('fixed') ||
          roles.contains('计件工') ||
          roles.contains('固定工');
    case EmployeeModule.receiving:
      return roles.contains('purchase') ||
          roles.contains('采购员') ||
          roles.contains('qc') ||
          roles.contains('质检') ||
          _matchAny(permissions, ['采购管理', '过磅', '农户', 'purchase', 'weigh']);
    case EmployeeModule.warehouse:
      return roles.contains('warehouse') ||
          roles.contains('仓管员') ||
          _matchAny(permissions, ['库存管理', '仓管', 'warehouse', '入库']);
    case EmployeeModule.sales:
      return roles.contains('sales') ||
          roles.contains('销售员') ||
          _matchAny(permissions, ['销售管理', '客户', '询价', '订单', 'sales', 'crm', 'CRM']);
    case EmployeeModule.assets:
      return _matchAny(permissions, ['固定资产', 'asset', 'fixed']) ||
          roles.contains('warehouse') ||
          roles.contains('仓管员') ||
          roles.contains('foreman');
    case EmployeeModule.collab:
      return roles.contains('sales') ||
          roles.contains('销售员') ||
          roles.contains('finance') ||
          roles.contains('财务') ||
          _matchAny(permissions, ['财务', '认款', '收款', 'finance', 'sales']);
    case EmployeeModule.knowledge:
      return true;
    case EmployeeModule.mine:
      return true;
  }
}

List<EmployeeModuleInfo> visibleEmployeeModules(List<String> permissions, List<String> roles) =>
    employeeModules.where((m) => canAccessEmployeeModule(m.key, permissions, roles)).toList();
