import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/notify_service.dart';
import 'ticket_widgets.dart';

/// 首页三 Tab：
/// - 待处理：当前指派给我且未结案（可操作）
/// - 处理中：我发起、已转出、未结案（只读跟踪）
/// - 我发起的：我发起且已结案（历史）
class TicketHomePage extends StatefulWidget {
  const TicketHomePage({super.key, this.embedded = true});

  final bool embedded;

  @override
  State<TicketHomePage> createState() => TicketHomePageState();
}

enum _HomeTab { open, progress, mine }

class TicketHomePageState extends State<TicketHomePage> {
  List<Map<String, dynamic>> _open = [];
  List<Map<String, dynamic>> _progress = [];
  List<Map<String, dynamic>> _mine = [];
  _HomeTab _tab = _HomeTab.open;
  String _msg = '';
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      final auth = context.read<AuthState>();
      final notify = context.read<NotifyService>();
      await auth.fetchMe();
      await notify.start();
      if (!mounted) return;
      await reload();
    });
  }

  int _assigneeId(Map<String, dynamic> m) => (m['current_assignee_user_id'] as num?)?.toInt() ?? 0;

  Future<void> reload() async {
    setState(() => _loading = true);
    final auth = context.read<AuthState>();
    final myId = auth.userId;
    final api = auth.api;
    final results = await Future.wait([
      api.get('/workflow/tickets?scope=mine_assignee'),
      api.get('/workflow/tickets?scope=mine_applicant'),
    ]);
    if (!mounted) return;
    final assignee = ApiClient.listOf(results[0].data)
        .map((e) => Map<String, dynamic>.from(e as Map))
        .toList();
    final applicant = ApiClient.listOf(results[1].data)
        .map((e) => Map<String, dynamic>.from(e as Map))
        .toList();

    // 待处理：指派给我的未结案（API 已限 open|in_progress）
    final open = List<Map<String, dynamic>>.from(assignee);

    // 处理中：我发起、已转给他人、未结案（仍指派给我的在「待处理」）
    final progress = applicant.where((m) {
      if (ticketIsClosed('${m['status']}')) return false;
      return _assigneeId(m) != myId;
    }).toList();

    // 我发起的：已办结历史
    final mine = applicant.where((m) => ticketIsClosed('${m['status']}')).toList();

    setState(() {
      _open = open;
      _progress = progress;
      _mine = mine;
      _msg = !results[0].ok
          ? results[0].msg
          : (!results[1].ok ? results[1].msg : '');
      _loading = false;
    });
  }

  List<Map<String, dynamic>> get _currentRows {
    switch (_tab) {
      case _HomeTab.open:
        return _open;
      case _HomeTab.progress:
        return _progress;
      case _HomeTab.mine:
        return _mine;
    }
  }

  /// 仅「待处理」可操作；处理中 / 我发起的只读。
  bool get _currentShowActions => _tab == _HomeTab.open;

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthState>();
    final notify = context.watch<NotifyService>();
    final rows = _currentRows;
    final body = RefreshIndicator(
      onRefresh: reload,
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: [
          if (_loading) const LinearProgressIndicator(minHeight: 2),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
            child: Row(
              children: [
                Expanded(
                  child: _TabChip(
                    label: '待处理',
                    count: _open.length,
                    selected: _tab == _HomeTab.open,
                    onTap: () => setState(() => _tab = _HomeTab.open),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: _TabChip(
                    label: '处理中',
                    count: _progress.length,
                    selected: _tab == _HomeTab.progress,
                    onTap: () => setState(() => _tab = _HomeTab.progress),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: _TabChip(
                    label: '我发起的',
                    count: _mine.length,
                    selected: _tab == _HomeTab.mine,
                    onTap: () => setState(() => _tab = _HomeTab.mine),
                  ),
                ),
              ],
            ),
          ),
          if (_msg.isNotEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Text(_msg, style: const TextStyle(color: Colors.red)),
            ),
          if (_tab == _HomeTab.open)
            const Padding(
              padding: EdgeInsets.fromLTRB(16, 0, 16, 8),
              child: Text('指派给我、尚未办结的工单', style: TextStyle(color: Colors.black54, fontSize: 12)),
            ),
          if (_tab == _HomeTab.progress)
            const Padding(
              padding: EdgeInsets.fromLTRB(16, 0, 16, 8),
              child: Text('我发起且已转出、尚未办结（只读）', style: TextStyle(color: Colors.black54, fontSize: 12)),
            ),
          if (_tab == _HomeTab.mine)
            const Padding(
              padding: EdgeInsets.fromLTRB(16, 0, 16, 8),
              child: Text('我发起且已办结的历史', style: TextStyle(color: Colors.black54, fontSize: 12)),
            ),
          if (rows.isEmpty)
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 16, vertical: 32),
              child: Center(child: Text('暂无', style: TextStyle(color: Colors.black54))),
            )
          else
            ...rows.map(
              (m) => TicketListCard(
                row: m,
                showActions: _currentShowActions,
                emphasizeAssignee: _tab == _HomeTab.progress,
                onTap: () => openTicketDetail(
                  context,
                  m,
                  allowActions: _currentShowActions,
                  onActed: reload,
                ),
                onAction: (a) => ticketAct(context, m, a, onDone: reload),
              ),
            ),
          const SizedBox(height: 24),
        ],
      ),
    );

    if (!widget.embedded) return body;

    return Scaffold(
      appBar: AppBar(
        title: const Text('首页'),
        actions: [
          ...ticketShellMessageActions(context, notify.unread),
          TextButton(
            onPressed: () async {
              await context.read<NotifyService>().stop();
              await auth.logout();
            },
            child: Text(auth.name?.isNotEmpty == true ? auth.name! : (auth.loginName ?? '退出')),
          ),
        ],
      ),
      body: body,
    );
  }
}

class _TabChip extends StatelessWidget {
  const _TabChip({
    required this.label,
    required this.count,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final int count;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Material(
      color: selected ? scheme.primaryContainer : scheme.surfaceContainerHighest,
      borderRadius: BorderRadius.circular(10),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(10),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 10, horizontal: 6),
          child: Column(
            children: [
              Text(
                '$count',
                style: TextStyle(
                  fontWeight: FontWeight.w700,
                  fontSize: 16,
                  color: selected ? scheme.onPrimaryContainer : scheme.onSurface,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: selected ? FontWeight.w600 : FontWeight.w400,
                  color: selected ? scheme.onPrimaryContainer : scheme.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
