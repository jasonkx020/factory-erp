/// Mirrors web/packages/shared employeeModules.ts
/// 木薯产线试点：默认入口仅保留现场工序相关模块；其余成员供路由页权限校验使用。
enum EmployeeModule {
  station,
  receiving,
  warehouse,
  workshop,
  worker,
  sales,
  assets,
  collab,
  knowledge,
  mine,
}

class EmployeeModuleInfo {
  const EmployeeModuleInfo(this.key, this.title, this.desc, this.route, this.icon);

  final EmployeeModule key;
  final String title;
  final String desc;
  final String route;
  final String icon;
}

const employeeModules = [
  EmployeeModuleInfo(EmployeeModule.station, '工序过站', '扫工牌+板码，按 kg 领取/退库/进下道', '/station', 'qr_code_scanner'),
  EmployeeModuleInfo(EmployeeModule.receiving, '过磅收货', '农户过磅、质检、出码推仓', '/receiving', 'scale'),
  EmployeeModuleInfo(EmployeeModule.warehouse, '仓管作业', '待入库、库存、板码、盘点', '/warehouse', 'warehouse'),
  EmployeeModuleInfo(EmployeeModule.workshop, '班组管理', '班次、异常、返工派岗', '/workshop', 'groups'),
  EmployeeModuleInfo(EmployeeModule.mine, '我的', '今日核对、假勤、工具与消息', '/mine', 'person'),
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
    case EmployeeModule.station:
      return _matchAny(permissions, ['生产管理', '过站记录', '扫码报工', 'production', '报工']) ||
          roles.contains('piece') ||
          roles.contains('fixed') ||
          roles.contains('计件工') ||
          roles.contains('固定工') ||
          roles.contains('foreman') ||
          roles.contains('车间主任');
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
    case EmployeeModule.workshop:
      return _matchAny(permissions, ['生产管理', '派工', '车间', 'production']) ||
          roles.contains('foreman') ||
          roles.contains('车间主任');
    case EmployeeModule.worker:
      return _matchAny(permissions, ['生产管理', '扫码报工', '计件', '工资管理', 'payroll', '报工']) ||
          roles.contains('piece') ||
          roles.contains('fixed') ||
          roles.contains('计件工') ||
          roles.contains('固定工');
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
