import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../core/api_client.dart';
import '../core/auth_state.dart';
import '../core/employee_modules.dart';
import '../features/hr/ticket_create_page.dart';
import '../features/receiving/batch_code_scanner_page.dart';
import '../features/receiving/receiving_page.dart';
import '../features/ticket/ticket_widgets.dart';
import '../widgets/trace_code_field.dart';

/// 这些类型有独立业务向导，不走通用动态表单
const dedicatedReceiveKinds = <String, String>{
  'farm_inbound': 'gate',
};

/// App「+」可创建的工单种类
class CreatableTicketKind {
  const CreatableTicketKind({
    required this.code,
    required this.title,
    required this.subtitle,
    this.receiveKind,
  });

  final String code;
  final String title;
  final String subtitle;
  /// 非空则打开过磅收货向导（`gate` | `stockin`）；空则按管理端 form_schema 动态填表
  final String? receiveKind;

  bool get isWeighFlow => receiveKind != null && receiveKind!.isNotEmpty;
}

const weighTicketKinds = [
  CreatableTicketKind(
    code: 'farm_inbound',
    title: '过磅入厂',
    subtitle: '与「采购·入厂」同一套向导',
    receiveKind: 'gate',
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
    default:
      return true;
  }
}

/// 过磅走专用向导；其余启用中的类型拉管理端配置（名称 + 填报字段）
Future<List<CreatableTicketKind>> loadCreatableTicketKinds(AuthState auth) async {
  final perms = auth.permissions;
  final roles = auth.roles;
  final out = <CreatableTicketKind>[];
  final seen = <String>{};

  for (final k in weighTicketKinds) {
    if (canCreateTicketKind(k.code, perms, roles)) {
      out.add(k);
      seen.add(k.code);
    }
  }

  final r = await auth.api.get('/workflow/ticket-categories');
  for (final raw in ApiClient.listOf(r.data)) {
    if (raw is! Map) continue;
    final m = Map<String, dynamic>.from(raw);
    if (m['enabled'] == false) continue;
    final code = '${m['code'] ?? ''}'.trim();
    if (code.isEmpty || seen.contains(code)) continue;
    if (dedicatedReceiveKinds.containsKey(code)) continue;
    final name = '${m['name'] ?? code}'.trim();
    final remark = '${m['remark'] ?? ''}'.trim();
    out.add(CreatableTicketKind(
      code: code,
      title: name.isEmpty ? code : name,
      subtitle: remark.isEmpty ? '按管理端配置的字段填报' : remark,
    ));
    seen.add(code);
  }
  return out;
}

bool canScanTraceOpenTicket(List<String> permissions, List<String> roles) {
  if (permissions.contains('*:*:*') ||
      roles.contains('sys_admin') ||
      roles.contains('系统管理员')) {
    return true;
  }
  return canAccessEmployeeModule(EmployeeModule.warehouse, permissions, roles) ||
      canAccessEmployeeModule(EmployeeModule.receiving, permissions, roles);
}

/// 弹出可选种类 / 快捷扫码 → 创建或打开关联工单。返回是否有业务成功（创建或打开）。
Future<bool> pickAndCreateTicket(BuildContext context) async {
  final auth = context.read<AuthState>();
  final canScan = canScanTraceOpenTicket(auth.permissions, auth.roles);
  final kinds = await loadCreatableTicketKinds(auth);
  if (!context.mounted) return false;
  if (kinds.isEmpty && !canScan) {
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('没有可创建的工单。过磅请确认有收货权限；自定义类型请到管理端「工单中心」新建并启用')),
    );
    return false;
  }

  CreatableTicketKind? selected = kinds.length == 1 ? kinds.first : null;
  final action = await showModalBottomSheet<_PlusSheetAction>(
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
                const Text('快捷操作', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                if (canScan)
                  ListTile(
                    leading: Icon(Icons.qr_code_scanner, color: Theme.of(ctx).colorScheme.primary),
                    title: const Text('扫溯源码查工单'),
                    subtitle: const Text('定位过磅单并打开关联协作工单'),
                    onTap: () => Navigator.pop(ctx, _PlusSheetAction.scanTrace),
                  ),
                if (kinds.isNotEmpty) ...[
                  if (canScan) const Divider(),
                  const Text('创建工单', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                  const SizedBox(height: 4),
                  const Text(
                    '过磅入厂走收货向导；其它类型按管理端配置的字段填报',
                    style: TextStyle(color: Colors.black54, fontSize: 13),
                  ),
                  const SizedBox(height: 8),
                  ConstrainedBox(
                    constraints: BoxConstraints(
                      maxHeight: MediaQuery.of(ctx).size.height * 0.42,
                    ),
                    child: ListView(
                      shrinkWrap: true,
                      children: [
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
                      ],
                    ),
                  ),
                  const SizedBox(height: 8),
                  FilledButton(
                    onPressed: selected == null
                        ? null
                        : () => Navigator.pop(ctx, _PlusSheetAction.create(selected!)),
                    child: const Text('确认创建'),
                  ),
                ],
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

  if (action == null || !context.mounted) return false;

  if (action.isScan) {
    return scanTraceOpenRelatedTicket(context);
  }

  final confirmed = action.kind!;
  final bool? ok;
  if (confirmed.isWeighFlow) {
    ok = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) => ReceivingPage(
          initialReceiveKind: confirmed.receiveKind!,
          lockKind: true,
          popOnCreated: true,
        ),
      ),
    );
  } else {
    ok = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) => TicketCreatePage(
          lockedCategoryCode: confirmed.code,
          lockedCategoryTitle: confirmed.title,
        ),
      ),
    );
  }
  if (ok == true && context.mounted) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('${confirmed.title}已提交成功')),
    );
    try {
      context.read<TicketRefreshBus>().bump();
    } catch (_) {}
  }
  return ok == true;
}

/// 扫/输入溯源码 → by-trace → 打开关联 wf_ticket
Future<bool> scanTraceOpenRelatedTicket(BuildContext context) async {
  final scanned = await Navigator.of(context).push<String>(
    MaterialPageRoute(builder: (_) => const BatchCodeScannerPage(title: '扫描溯源码查工单')),
  );
  if (!context.mounted) return false;

  String code = (scanned ?? '').trim();
  if (code.isEmpty) {
    // 扫码取消时允许手动输入
    code = await _promptTraceCode(context) ?? '';
  }
  if (code.isEmpty || !context.mounted) return false;

  final r = await context.read<AuthState>().api.get(
        '/purchase/weigh-tickets/by-trace?code=${Uri.encodeComponent(code)}',
      );
  if (!context.mounted) return false;
  if (!r.ok || r.data is! Map) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.msg)));
    return false;
  }
  final m = Map<String, dynamic>.from(r.data as Map);
  final ticketId = (m['ticket_id'] as num?)?.toInt() ?? 0;
  if (ticketId <= 0) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(
          '已定位过磅单 ${m['doc_no'] ?? ''}，暂无进行中的关联工单'
          '${m['stockin_ready'] == true ? '' : '（${m['reason'] ?? m['status'] ?? '未就绪'}）'}',
        ),
      ),
    );
    return false;
  }

  await openTicketDetail(
    context,
    {'id': ticketId},
    onActed: () {
      try {
        context.read<TicketRefreshBus>().bump();
      } catch (_) {}
    },
  );
  return true;
}

Future<String?> _promptTraceCode(BuildContext context) async {
  final ctrl = TextEditingController();
  return showDialog<String>(
    context: context,
    builder: (ctx) => AlertDialog(
      title: const Text('输入溯源码/批号'),
      content: TraceCodeField(
        controller: ctrl,
        label: '溯源码',
        hint: '可继续点右侧扫码',
        scannerTitle: '扫描溯源码',
        textCapitalization: TextCapitalization.none,
      ),
      actions: [
        TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
        FilledButton(
          onPressed: () => Navigator.pop(ctx, ctrl.text.trim()),
          child: const Text('查询'),
        ),
      ],
    ),
  );
}

class _PlusSheetAction {
  const _PlusSheetAction._({this.kind, this.isScan = false});
  factory _PlusSheetAction.create(CreatableTicketKind k) => _PlusSheetAction._(kind: k);
  static const scanTrace = _PlusSheetAction._(isScan: true);

  final CreatableTicketKind? kind;
  final bool isScan;
}
