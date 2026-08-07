import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../core/auth_state.dart';
import '../core/employee_modules.dart';
import '../features/receiving/receiving_page.dart';
import '../features/ticket/ticket_widgets.dart';

/// App「+」可创建的工单种类（按权限过滤）
/// 表单与过磅收货页同源：确认后进入 ReceivingPage，不走独立 TicketCreate 表单
class CreatableTicketKind {
  const CreatableTicketKind({
    required this.code,
    required this.title,
    required this.subtitle,
    required this.receiveKind,
  });

  final String code;
  final String title;
  final String subtitle;
  /// 过磅收货 `gate` | `stockin`
  final String receiveKind;
}

const creatableTicketKinds = [
  CreatableTicketKind(
    code: 'farm_inbound',
    title: '过磅入厂',
    subtitle: '与「过磅收货·入厂」同一套表单',
    receiveKind: 'gate',
  ),
  CreatableTicketKind(
    code: 'stock_inbound',
    title: '过磅入库',
    subtitle: '与「过磅收货·入库」同一套表单',
    receiveKind: 'stockin',
  ),
];

bool canCreateTicketKind(String code, List<String> permissions, List<String> roles) {
  if (permissions.contains('*:*:*') ||
      roles.contains('sys_admin') ||
      roles.contains('系统管理员')) {
    return true;
  }
  switch (code) {
    case 'farm_inbound':
      return canAccessEmployeeModule(EmployeeModule.receiving, permissions, roles);
    case 'stock_inbound':
      return canAccessEmployeeModule(EmployeeModule.warehouse, permissions, roles) ||
          canAccessEmployeeModule(EmployeeModule.receiving, permissions, roles);
    default:
      return false;
  }
}

List<CreatableTicketKind> visibleCreatableTicketKinds(
  List<String> permissions,
  List<String> roles,
) =>
    creatableTicketKinds.where((k) => canCreateTicketKind(k.code, permissions, roles)).toList();

/// 弹出可选种类 → 确认后进入过磅收货表单。返回是否创建成功。
Future<bool> pickAndCreateTicket(BuildContext context) async {
  final auth = context.read<AuthState>();
  final kinds = visibleCreatableTicketKinds(auth.permissions, auth.roles);
  if (kinds.isEmpty) {
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('当前账号无权限创建过磅工单')),
    );
    return false;
  }

  CreatableTicketKind? selected = kinds.length == 1 ? kinds.first : null;
  final confirmed = await showModalBottomSheet<CreatableTicketKind>(
    context: context,
    isScrollControlled: true,
    builder: (ctx) {
      return StatefulBuilder(
        builder: (ctx, setLocal) {
          return Padding(
            padding: EdgeInsets.only(
              left: 16,
              right: 16,
              top: 16,
              bottom: MediaQuery.of(ctx).viewInsets.bottom + 16,
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Text('选择工单种类', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                const SizedBox(height: 4),
                const Text('将打开与过磅收货相同的填报表单', style: TextStyle(color: Colors.black54, fontSize: 13)),
                const SizedBox(height: 12),
                for (final k in kinds)
                  ListTile(
                    leading: Icon(
                      selected?.code == k.code ? Icons.radio_button_checked : Icons.radio_button_off,
                      color: Theme.of(ctx).colorScheme.primary,
                    ),
                    title: Text(k.title),
                    subtitle: Text(k.subtitle),
                    selected: selected?.code == k.code,
                    onTap: () => setLocal(() => selected = k),
                  ),
                const SizedBox(height: 8),
                FilledButton(
                  onPressed: selected == null ? null : () => Navigator.pop(ctx, selected),
                  child: const Text('确认创建'),
                ),
                TextButton(
                  onPressed: () => Navigator.pop(ctx),
                  child: const Text('取消'),
                ),
              ],
            ),
          );
        },
      );
    },
  );

  if (confirmed == null || !context.mounted) return false;

  final ok = await Navigator.of(context).push<bool>(
    MaterialPageRoute(
      builder: (_) => ReceivingPage(
        initialReceiveKind: confirmed.receiveKind,
        lockKind: true,
        popOnCreated: true,
      ),
    ),
  );
  if (ok == true && context.mounted) {
    try {
      context.read<TicketRefreshBus>().bump();
    } catch (_) {}
  }
  return ok == true;
}
