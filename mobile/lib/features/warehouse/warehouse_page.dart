import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';

class WarehousePage extends StatefulWidget {
  const WarehousePage({super.key});

  @override
  State<WarehousePage> createState() => _WarehousePageState();
}

class _WarehousePageState extends State<WarehousePage> {
  List<dynamic> _tasks = [];
  String? _error;
  bool _loading = false;
  Map<String, dynamic>? _active;
  final _verify = TextEditingController();
  NotifyService? _notify;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _notify = context.read<NotifyService>();
      _notify!.addListener(_onNotify);
      _load();
    });
  }

  @override
  void dispose() {
    _notify?.removeListener(_onNotify);
    _verify.dispose();
    super.dispose();
  }

  void _onNotify() {
    _load();
  }

  Future<void> _load() async {
    final auth = context.read<AuthState>();
    await auth.fetchMe();
    if (!canAccessEmployeeModule(EmployeeModule.warehouse, auth.permissions, auth.roles)) {
      setState(() => _error = '无仓管模块权限');
      return;
    }
    setState(() {
      _loading = true;
      _error = null;
    });
    final api = auth.api;
    final res = await api.get('/workflow/tasks?status=pending');
    if (!mounted) return;
    setState(() {
      _loading = false;
      if (!res.ok) {
        _error = res.msg;
        return;
      }
      final list = (res.data is Map ? res.data['list'] : null) as List? ?? [];
      _tasks = list
          .where((t) => t is Map && (t['event_key'] == 'purchase.weigh_confirmed' || t['to_role'] == 'warehouse'))
          .toList();
    });
  }

  Future<void> _claim(Map row) async {
    final id = row['id'];
    final res = await context.read<AuthState>().api.post('/workflow/tasks/$id/claim', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(res.ok ? '已认领' : res.msg)));
    if (res.ok) await _load();
  }

  Future<void> _confirm() async {
    final row = _active;
    if (row == null) return;
    final expect = (row['trace_code'] ?? NotifyService.parsePayload(row['payload'] ?? row['payload_json'])['trace_code'] ?? '')
        .toString()
        .trim();
    final got = _verify.text.trim();
    if (got.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请输入溯源码核对')));
      return;
    }
    if (expect.isNotEmpty && got != expect) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('溯源码不一致')));
      return;
    }
    final bizId = row['biz_id'];
    final res = await context.read<AuthState>().api.post('/purchase/weigh-tickets/$bizId/warehouse-confirm', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(res.ok ? '入库完成，已推送财务' : res.msg)));
    if (res.ok) {
      setState(() {
        _active = null;
        _verify.clear();
      });
      await _load();
    }
  }

  @override
  Widget build(BuildContext context) {
    final notify = context.watch<NotifyService>();
    return Scaffold(
      appBar: AppBar(
        title: Text('仓管入库 · ${notify.mqttStatus}'),
        actions: [
          IconButton(
            onPressed: () async {
              await notify.refresh();
              await _load();
            },
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(child: Text(_error!))
              : ListView(
                  padding: const EdgeInsets.all(12),
                  children: [
                    Text('待办 ${_tasks.length} · 未读通知 ${notify.unread}', style: const TextStyle(color: Colors.black54)),
                    const SizedBox(height: 8),
                    ..._tasks.map((e) {
                      final t = Map<String, dynamic>.from(e as Map);
                      final p = NotifyService.parsePayload(t['payload'] ?? t['payload_json']);
                      return Card(
                        child: ListTile(
                          title: Text('${t['doc_no'] ?? ''}'),
                          subtitle: Text(
                            '溯源 ${t['trace_code'] ?? ''}\n'
                            '${p['farmer_name'] ?? ''} ${p['net_weight'] != null ? '· ${p['net_weight']}kg' : ''}',
                          ),
                          isThreeLine: true,
                          trailing: Wrap(
                            spacing: 4,
                            children: [
                              TextButton(onPressed: () => _claim(t), child: const Text('认领')),
                              FilledButton(
                                onPressed: () => setState(() {
                                  _active = t;
                                  _verify.clear();
                                }),
                                child: const Text('核对'),
                              ),
                            ],
                          ),
                        ),
                      );
                    }),
                    if (_active != null) ...[
                      const Divider(),
                      Text('核对入库 · ${_active!['doc_no']}', style: const TextStyle(fontWeight: FontWeight.bold)),
                      Text('推送溯源码：${_active!['trace_code']}'),
                      TextField(controller: _verify, decoration: const InputDecoration(labelText: '输入/扫描溯源码')),
                      const SizedBox(height: 8),
                      FilledButton(onPressed: _confirm, child: const Text('确认入库')),
                    ],
                  ],
                ),
    );
  }
}
