import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/notify_service.dart';
import 'ticket_widgets.dart';

/// 首页：待我处理 / 我发起的 / 处理中
class TicketHomePage extends StatefulWidget {
  const TicketHomePage({super.key, this.embedded = true});

  final bool embedded;

  @override
  State<TicketHomePage> createState() => TicketHomePageState();
}

class TicketHomePageState extends State<TicketHomePage> {
  List<Map<String, dynamic>> _open = [];
  List<Map<String, dynamic>> _mine = [];
  List<Map<String, dynamic>> _progress = [];
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

  Future<void> reload() async {
    setState(() => _loading = true);
    final api = context.read<AuthState>().api;
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
    setState(() {
      _open = assignee.where((m) => '${m['status']}' == 'open').toList();
      _progress = assignee.where((m) => '${m['status']}' == 'in_progress').toList();
      _mine = applicant.where((m) => !ticketIsClosed('${m['status']}')).toList();
      _msg = !results[0].ok
          ? results[0].msg
          : (!results[1].ok ? results[1].msg : '');
      _loading = false;
    });
  }

  Widget _section(String title, List<Map<String, dynamic>> rows, {bool actions = false}) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
          child: Row(
            children: [
              Text(title, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
              const SizedBox(width: 8),
              CircleAvatar(
                radius: 11,
                child: Text('${rows.length}', style: const TextStyle(fontSize: 11)),
              ),
            ],
          ),
        ),
        if (rows.isEmpty)
          const Padding(
            padding: EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Text('暂无', style: TextStyle(color: Colors.black54)),
          )
        else
          ...rows.map(
            (m) => TicketListCard(
              row: m,
              showActions: actions,
              onTap: () => openTicketDetail(context, m, onActed: reload),
              onAction: (a) => ticketAct(context, m, a, onDone: reload),
            ),
          ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthState>();
    final notify = context.watch<NotifyService>();
    final body = RefreshIndicator(
      onRefresh: reload,
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: [
          if (_loading) const LinearProgressIndicator(minHeight: 2),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
            child: Wrap(
              spacing: 12,
              runSpacing: 8,
              children: [
                _StatChip(label: '待我处理', count: _open.length),
                _StatChip(label: '处理中', count: _progress.length),
                _StatChip(label: '我发起', count: _mine.length),
              ],
            ),
          ),
          if (_msg.isNotEmpty)
            Padding(padding: const EdgeInsets.all(12), child: Text(_msg, style: const TextStyle(color: Colors.red))),
          _section('待我处理', _open, actions: true),
          _section('处理中', _progress, actions: true),
          _section('我发起的', _mine),
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

class _StatChip extends StatelessWidget {
  const _StatChip({required this.label, required this.count});
  final String label;
  final int count;

  @override
  Widget build(BuildContext context) {
    return Chip(
      avatar: CircleAvatar(child: Text('$count', style: const TextStyle(fontSize: 11))),
      label: Text(label),
    );
  }
}
