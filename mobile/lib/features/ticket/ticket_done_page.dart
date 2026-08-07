import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/notify_service.dart';
import 'ticket_widgets.dart';

/// 已办：我发起且已结案（申请人侧历史）
class TicketDonePage extends StatefulWidget {
  const TicketDonePage({super.key});

  @override
  State<TicketDonePage> createState() => TicketDonePageState();
}

class TicketDonePageState extends State<TicketDonePage> {
  List<Map<String, dynamic>> _list = [];
  String _msg = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => reload());
  }

  Future<void> reload() async {
    final api = context.read<AuthState>().api;
    // 结案后 assignee 清空，以「我发起的」为主；再合并默认可见列表中的结案单
    final results = await Future.wait([
      api.get('/workflow/tickets?scope=mine_applicant'),
      api.get('/workflow/tickets'),
    ]);
    if (!mounted) return;
    final map = <int, Map<String, dynamic>>{};
    for (final res in results) {
      for (final e in ApiClient.listOf(res.data)) {
        final m = Map<String, dynamic>.from(e as Map);
        if (!ticketIsClosed('${m['status']}')) continue;
        final id = (m['id'] as num?)?.toInt();
        if (id == null) continue;
        map[id] = m;
      }
    }
    final list = map.values.toList()
      ..sort((a, b) => ((b['id'] as num?)?.toInt() ?? 0).compareTo((a['id'] as num?)?.toInt() ?? 0));
    setState(() {
      _list = list;
      _msg = results.any((r) => !r.ok) ? results.firstWhere((r) => !r.ok).msg : '';
    });
  }

  @override
  Widget build(BuildContext context) {
    final notify = context.watch<NotifyService>();
    return Scaffold(
      appBar: AppBar(
        title: const Text('已办'),
        actions: ticketShellMessageActions(context, notify.unread),
      ),
      body: RefreshIndicator(
        onRefresh: reload,
        child: _list.isEmpty
            ? ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                children: [
                  if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.all(12), child: Text(_msg)),
                  const SizedBox(height: 120),
                  const Center(child: Text('暂无已办历史')),
                ],
              )
            : ListView.builder(
                itemCount: _list.length,
                itemBuilder: (_, i) {
                  final m = _list[i];
                  return TicketListCard(
                    row: m,
                    onTap: () => openTicketDetail(context, m, allowActions: false),
                  );
                },
              ),
      ),
    );
  }
}
