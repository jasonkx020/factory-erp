import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import 'ticket_create_page.dart';

/// 我的申请 / 待我处理工单
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
  String? _deepEventKey;
  bool _deepHandled = false;

  static const _statusLabel = {
    'open': '待处理',
    'in_progress': '处理中',
    'done': '已办结',
    'rejected': '已驳回',
    'cancelled': '已取消',
  };

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
    _deepEventKey = m['event_key']?.toString();
    // assigned → 待我处理; done/rejected → 我发起的
    final ek = _deepEventKey ?? '';
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
    final id = _deepTicketId!;
    // Prefer list row when present so action menu still works; else fetch by id.
    Map<String, dynamic>? row;
    for (final e in _list) {
      final m = Map<String, dynamic>.from(e as Map);
      if ((m['id'] as num?)?.toInt() == id) {
        row = m;
        break;
      }
    }
    if (row == null) {
      // Try the other tab once if not found.
      final otherScope = _tab == 0 ? 'mine_applicant' : 'mine_assignee';
      final r = await context.read<AuthState>().api.get('/workflow/tickets?scope=$otherScope');
      if (!mounted) return;
      if (r.ok) {
        final other = ApiClient.listOf(r.data);
        for (final e in other) {
          final m = Map<String, dynamic>.from(e as Map);
          if ((m['id'] as num?)?.toInt() == id) {
            setState(() {
              _tab = _tab == 0 ? 1 : 0;
              _list = other;
            });
            row = m;
            break;
          }
        }
      }
    }
    row ??= {'id': id};
    if (!mounted) return;
    await _openDetail(row);
  }

  Future<void> _act(Map<String, dynamic> row, String action) async {
    final id = (row['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/workflow/tickets/$id/action', {
      'action': action,
    });
    if (!mounted) return;
    setState(() => _msg = r.ok ? '已$action' : r.msg);
    if (r.ok) await _load();
  }

  Future<void> _openCreate() async {
    final ok = await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => const TicketCreatePage()),
    );
    if (ok == true) await _load();
  }

  Future<void> _openDetail(Map<String, dynamic> row) async {
    final id = (row['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.get('/workflow/tickets/$id');
    if (!mounted) return;
    if (!r.ok || r.data is! Map) {
      setState(() => _msg = r.msg);
      return;
    }
    final d = Map<String, dynamic>.from(r.data as Map);
    final schema = (d['form_schema'] as List?) ?? [];
    final payload = d['payload'] is Map ? Map<String, dynamic>.from(d['payload'] as Map) : <String, dynamic>{};
    final st = '${d['status'] ?? ''}';
    final canAct = st == 'open' || st == 'in_progress';
    if (!mounted) return;
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
            Text('${d['doc_no']} · ${d['category_name']} · ${_statusLabel['${d['status']}'] ?? d['status']}'),
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
                  await _act(d, 'approve');
                },
                child: const Text('通过/办结'),
              ),
              const SizedBox(height: 8),
              OutlinedButton(
                onPressed: () async {
                  Navigator.pop(ctx);
                  await _act(d, 'return_confirm');
                },
                child: const Text('确认归还'),
              ),
              const SizedBox(height: 8),
              TextButton(
                onPressed: () async {
                  Navigator.pop(ctx);
                  await _act(d, 'reject');
                },
                child: const Text('驳回'),
              ),
            ],
          ],
        ),
      ),
    );
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
      floatingActionButton: FloatingActionButton(
        onPressed: _openCreate,
        child: const Icon(Icons.add),
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
                  final st = '${m['status'] ?? ''}';
                  final canAct = _tab == 0 && (st == 'open' || st == 'in_progress');
                  return Card(
                    margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                    child: ListTile(
                      title: Text('${m['title'] ?? ''}'),
                      subtitle: Text(
                        '${m['doc_no']} · ${m['category_name']}\n'
                        '${_statusLabel[st] ?? st} · 处理人 ${m['assignee_name'] ?? '-'}',
                      ),
                      isThreeLine: true,
                      onTap: () => _openDetail(m),
                      trailing: canAct
                          ? PopupMenuButton<String>(
                              onSelected: (a) => _act(m, a),
                              itemBuilder: (_) => const [
                                PopupMenuItem(value: 'approve', child: Text('通过/办结')),
                                PopupMenuItem(value: 'return_confirm', child: Text('确认归还')),
                                PopupMenuItem(value: 'reject', child: Text('驳回')),
                              ],
                            )
                          : null,
                    ),
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
