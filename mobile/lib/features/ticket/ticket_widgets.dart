import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/auth_state.dart';

const ticketStatusLabel = {
  'open': '待处理',
  'in_progress': '处理中',
  'done': '已办结',
  'rejected': '已驳回',
  'cancelled': '已取消',
};

bool ticketIsClosed(String status) =>
    status == 'done' || status == 'rejected' || status == 'cancelled';

bool ticketCanAct(String status) => status == 'open' || status == 'in_progress';

class TicketRefreshBus extends ChangeNotifier {
  void bump() => notifyListeners();
}

Future<void> ticketAct(
  BuildContext context,
  Map<String, dynamic> row,
  String action, {
  VoidCallback? onDone,
}) async {
  final id = (row['id'] as num?)?.toInt();
  if (id == null) return;
  final r = await context.read<AuthState>().api.post('/workflow/tickets/$id/action', {
    'action': action,
  });
  if (!context.mounted) return;
  ScaffoldMessenger.of(context).showSnackBar(
    SnackBar(content: Text(r.ok ? '已$action' : r.msg)),
  );
  if (r.ok) {
    onDone?.call();
    try {
      context.read<TicketRefreshBus>().bump();
    } catch (_) {}
  }
}

Future<void> openTicketDetail(
  BuildContext context,
  Map<String, dynamic> row, {
  bool allowActions = true,
  VoidCallback? onActed,
}) async {
  final id = (row['id'] as num?)?.toInt();
  if (id == null) return;
  final r = await context.read<AuthState>().api.get('/workflow/tickets/$id');
  if (!context.mounted) return;
  if (!r.ok || r.data is! Map) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.msg)));
    return;
  }
  final d = Map<String, dynamic>.from(r.data as Map);
  final schema = (d['form_schema'] as List?) ?? [];
  final payload =
      d['payload'] is Map ? Map<String, dynamic>.from(d['payload'] as Map) : <String, dynamic>{};
  final st = '${d['status'] ?? ''}';
  final canAct = allowActions && ticketCanAct(st);

  await showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    builder: (ctx) => Padding(
      padding: EdgeInsets.only(
        left: 16,
        right: 16,
        top: 16,
        bottom: MediaQuery.of(ctx).viewInsets.bottom + 16,
      ),
      child: ListView(
        shrinkWrap: true,
        children: [
          Text('${d['title']}', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
          Text(
            '${d['doc_no']} · ${d['category_name']} · ${ticketStatusLabel[st] ?? st}',
          ),
          const Divider(),
          ...schema.map((raw) {
            final f = Map<String, dynamic>.from(raw as Map);
            final key = '${f['key']}';
            return ListTile(
              dense: true,
              title: Text('${f['label']}'),
              trailing: Text('${payload[key] ?? '-'}'),
            );
          }),
          if (canAct) ...[
            const SizedBox(height: 8),
            FilledButton(
              onPressed: () async {
                Navigator.pop(ctx);
                await ticketAct(context, d, 'approve', onDone: onActed);
              },
              child: const Text('通过/办结'),
            ),
            const SizedBox(height: 8),
            OutlinedButton(
              onPressed: () async {
                Navigator.pop(ctx);
                await ticketAct(context, d, 'return_confirm', onDone: onActed);
              },
              child: const Text('确认归还'),
            ),
            const SizedBox(height: 8),
            TextButton(
              onPressed: () async {
                Navigator.pop(ctx);
                await ticketAct(context, d, 'reject', onDone: onActed);
              },
              child: const Text('驳回'),
            ),
          ],
        ],
      ),
    ),
  );
}

class TicketListCard extends StatelessWidget {
  const TicketListCard({
    super.key,
    required this.row,
    this.showActions = false,
    this.onTap,
    this.onAction,
  });

  final Map<String, dynamic> row;
  final bool showActions;
  final VoidCallback? onTap;
  final void Function(String action)? onAction;

  @override
  Widget build(BuildContext context) {
    final st = '${row['status'] ?? ''}';
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      child: ListTile(
        title: Text('${row['title'] ?? ''}'),
        subtitle: Text(
          '${row['doc_no']} · ${row['category_name']}\n'
          '${ticketStatusLabel[st] ?? st} · 处理人 ${row['assignee_name'] ?? '-'}',
        ),
        isThreeLine: true,
        onTap: onTap,
        trailing: showActions && ticketCanAct(st)
            ? PopupMenuButton<String>(
                onSelected: onAction,
                itemBuilder: (_) => const [
                  PopupMenuItem(value: 'approve', child: Text('通过/办结')),
                  PopupMenuItem(value: 'return_confirm', child: Text('确认归还')),
                  PopupMenuItem(value: 'reject', child: Text('驳回')),
                ],
              )
            : null,
      ),
    );
  }
}

/// AppBar 消息入口（与现网位置一致）
List<Widget> ticketShellMessageActions(BuildContext context, int unread) {
  return [
    IconButton(
      onPressed: () => Navigator.of(context).pushNamed('/inbox'),
      icon: Badge(
        isLabelVisible: unread > 0,
        label: Text('${unread > 99 ? '99+' : unread}'),
        child: const Icon(Icons.notifications_outlined),
      ),
    ),
  ];
}
