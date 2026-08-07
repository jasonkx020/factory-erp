import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../ticket/ticket_widgets.dart';
import 'ticket_create_page.dart';

/// 路由兼容页：推送深链 /tickets 仍可用
class TicketsPage extends StatefulWidget {
  const TicketsPage({super.key});

  @override
  State<TicketsPage> createState() => _TicketsPageState();
}

class _TicketsPageState extends State<TicketsPage> {
  int _tab = 0;
  List<dynamic> _list = [];
  String _msg = '';
  int? _deepTicketId;
  bool _deepHandled = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _readDeepLinkArgs();
      _load();
    });
  }

  void _readDeepLinkArgs() {
    final args = ModalRoute.of(context)?.settings.arguments;
    if (args is! Map) return;
    final m = Map<String, dynamic>.from(args);
    final tid = m['ticket_id'];
    if (tid is num) {
      _deepTicketId = tid.toInt();
    } else if (tid != null) {
      _deepTicketId = int.tryParse(tid.toString());
    }
    final ek = m['event_key']?.toString() ?? '';
    if (ek.contains('done') || ek.contains('rejected')) {
      _tab = 1;
    } else {
      _tab = 0;
    }
  }

  Future<void> _load() async {
    final scope = _tab == 0 ? 'mine_assignee' : 'mine_applicant';
    final r = await context.read<AuthState>().api.get('/workflow/tickets?scope=$scope');
    if (!mounted) return;
    setState(() {
      _list = ApiClient.listOf(r.data);
      if (!r.ok) _msg = r.msg;
    });
    await _maybeOpenDeepLink();
  }

  Future<void> _maybeOpenDeepLink() async {
    if (_deepHandled || _deepTicketId == null) return;
    _deepHandled = true;
    if (!mounted) return;
    final id = _deepTicketId!;
    Map<String, dynamic>? row;
    for (final e in _list) {
      final m = Map<String, dynamic>.from(e as Map);
      if ((m['id'] as num?)?.toInt() == id) {
        row = m;
        break;
      }
    }
    row ??= {'id': id};
    await openTicketDetail(context, row, onActed: _load);
  }

  Future<void> _openCreate() async {
    final ok = await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => const TicketCreatePage()),
    );
    if (!mounted) return;
    if (ok == true) {
      try {
        context.read<TicketRefreshBus>().bump();
      } catch (_) {}
      await _load();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('工单'),
        actions: [
          IconButton(tooltip: '新建', onPressed: _openCreate, icon: const Icon(Icons.add)),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(8),
            child: SegmentedButton<int>(
              segments: const [
                ButtonSegment(value: 0, label: Text('待我处理')),
                ButtonSegment(value: 1, label: Text('我发起的')),
              ],
              selected: {_tab},
              onSelectionChanged: (s) {
                setState(() => _tab = s.first);
                _load();
              },
            ),
          ),
          if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.all(8), child: Text(_msg)),
          Expanded(
            child: RefreshIndicator(
              onRefresh: _load,
              child: ListView.builder(
                itemCount: _list.length,
                itemBuilder: (_, i) {
                  final m = Map<String, dynamic>.from(_list[i] as Map);
                  return TicketListCard(
                    row: m,
                    showActions: _tab == 0,
                    onTap: () => openTicketDetail(context, m, onActed: _load),
                    onAction: (a) => ticketAct(context, m, a, onDone: _load),
                  );
                },
              ),
            ),
          ),
        ],
      ),
    );
  }
}
