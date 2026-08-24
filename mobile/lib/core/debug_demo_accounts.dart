import 'package:flutter/foundation.dart';

/// Debug-only demo accounts (password all admin123). Synced with EnsureDemoRoleUsers / seed.sql.
class DebugDemoAccount {
  const DebugDemoAccount({
    required this.login,
    required this.label,
    required this.roleCode,
  });

  final String login;
  final String label;
  final String roleCode;

  static const password = 'admin123';

  String get menuLabel => '$label · $login';
}

const kDebugDemoAccounts = <DebugDemoAccount>[
  DebugDemoAccount(login: 'admin', label: '系统管理员', roleCode: 'sys_admin'),
  DebugDemoAccount(login: 'u_purchase', label: '采购/过磅', roleCode: 'purchase'),
  DebugDemoAccount(login: 'u_qc', label: '质检', roleCode: 'qc'),
  DebugDemoAccount(login: 'u_warehouse', label: '仓管', roleCode: 'warehouse'),
  DebugDemoAccount(login: 'u_foreman', label: '车间主任', roleCode: 'foreman'),
  DebugDemoAccount(login: 'u_piece', label: '计件工', roleCode: 'piece'),
  DebugDemoAccount(login: 'u_fixed', label: '固定工', roleCode: 'fixed'),
  DebugDemoAccount(login: 'u_planner', label: '生产计划', roleCode: 'planner'),
  DebugDemoAccount(login: 'u_payroll', label: '薪资', roleCode: 'payroll'),
  DebugDemoAccount(login: 'u_finance', label: '财务(成本)', roleCode: 'finance'),
  DebugDemoAccount(login: 'u_boss', label: '老板', roleCode: 'boss'),
];

/// Debug 构建默认开启；Profile/Release 可用：flutter run --dart-define=DEMO_LOGIN=true
bool get showDebugDemoAccounts =>
    kDebugMode || const bool.fromEnvironment('DEMO_LOGIN', defaultValue: false);
