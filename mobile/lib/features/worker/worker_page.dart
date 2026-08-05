import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';

class WorkerPage extends StatefulWidget {
  const WorkerPage({super.key});

  @override
  State<WorkerPage> createState() => _WorkerPageState();
}

class _WorkerPageState extends State<WorkerPage> {
  int _tab = 0;
  final _badge = TextEditingController();
  final _box = TextEditingController();
  final _in = TextEditingController();
  final _out = TextEditingController();
  String _msg = '';
  Map<String, dynamic>? _daily;
  List<dynamic> _wages = [];
  Map<String, dynamic>? _last;
  int? _pendingReportId;
  List<Map<String, dynamic>> _notices = [];

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      final prefs = await SharedPreferences.getInstance();
      if (!mounted) return;
      _badge.text = prefs.getString('erp.worker.badge') ?? '';
      context.read<NotifyService>().addListener(_onNotify);
      await _boot();
    });
  }

  @override
  void dispose() {
    try {
      context.read<NotifyService>().removeListener(_onNotify);
    } catch (_) {}
    _badge.dispose();
    _box.dispose();
    _in.dispose();
    _out.dispose();
    super.dispose();
  }

  void _onNotify() {
    _applyNotices();
  }

  void _applyNotices() {
    if (!mounted) return;
    final notify = context.read<NotifyService>();
    final rows = notify.inbox
        .whereType<Map>()
        .map((e) => Map<String, dynamic>.from(e))
        .where((r) {
          final k = r['event_key']?.toString() ?? '';
          return k == 'production.report_confirmed' || k == 'payroll.labor_paid';
        })
        .toList();
    setState(() => _notices = rows);
    for (final row in rows) {
      final p = NotifyService.parsePayload(row['payload'] ?? row['payload_json']);
      final next = p['next'] is Map ? Map<String, dynamic>.from(p['next'] as Map) : p;
      final code = next['new_box_code'] ?? p['new_box_code'] ?? p['scan_code'];
      if (code != null && _box.text.trim().isEmpty) {
        _box.text = code.toString();
        break;
      }
    }
  }

  Future<void> _boot() async {
    final auth = context.read<AuthState>();
    if (!canAccessEmployeeModule(EmployeeModule.worker, auth.permissions, auth.roles)) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('无工人模块权限')));
        Navigator.of(context).pop();
      }
      return;
    }
    await context.read<NotifyService>().start();
    await _loadWage();
    _applyNotices();
  }

  Future<void> _loadWage() async {
    final api = context.read<AuthState>().api;
    final badge = _badge.text.trim();
    final w = await api.get('/payroll/wage-rates');
    final d = await api.get(
      badge.isEmpty
          ? '/production/piecework-summaries/mine'
          : '/production/piecework-summaries/mine?badge_code=${Uri.encodeQueryComponent(badge)}',
    );
    if (!mounted) return;
    setState(() {
      _wages = (w.data is Map ? (w.data as Map)['list'] as List? : null) ?? [];
      _daily = d.data is Map ? Map<String, dynamic>.from(d.data as Map) : null;
    });
  }

  Future<void> _scan({required bool resolveOnly}) async {
    final prefs = await SharedPreferences.getInstance();
    if (_badge.text.trim().isNotEmpty) {
      await prefs.setString('erp.worker.badge', _badge.text.trim());
    }
    if (!mounted) return;
    final api = context.read<AuthState>().api;
    final path = resolveOnly ? '/production/scan/resolve' : '/production/scan';
    final outW = double.tryParse(_out.text) ?? 0;
    final r = await api.post(path, {
      'badge_code': _badge.text.trim(),
      'box_code': _box.text.trim(),
      'input_weight': double.tryParse(_in.text) ?? 0,
      'output_weight': outW,
      'net_weight': outW,
    });
    setState(() {
      if (r.ok) {
        final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
        _last = data;
        if (resolveOnly) {
          _msg = '已解析 ${data['worker_name'] ?? ''}';
        } else if (data['needs_confirm'] == true || data['status'] == 'confirm_pending') {
          _pendingReportId = (data['id'] as num?)?.toInt();
          _msg = '草稿已建，请确认过账';
        } else {
          _msg = '报工成功 工钱¥${data['wage_amount'] ?? 0}';
          _pendingReportId = null;
        }
        final next = data['next'];
        if (next is Map && next['new_box_code'] != null) {
          _box.text = next['new_box_code'].toString();
        }
      } else {
        _msg = r.msg;
      }
    });
    if (r.ok && !resolveOnly && _pendingReportId == null) await _loadWage();
  }

  Future<void> _confirm() async {
    final id = _pendingReportId ?? (_last?['id'] as num?)?.toInt();
    if (id == null) {
      setState(() => _msg = '请先提交报工草稿');
      return;
    }
    final api = context.read<AuthState>().api;
    final outW = double.tryParse(_out.text) ?? 0;
    final r = await api.post('/production/report-works/$id/confirm', {
      'input_weight': double.tryParse(_in.text) ?? 0,
      'output_weight': outW,
      'process_qc_result': 'pass',
    });
    setState(() {
      if (r.ok) {
        final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
        _last = data;
        _pendingReportId = null;
        _msg = '已确认过账 工钱¥${data['wage_amount'] ?? 0}';
        final next = data['next'];
        if (next is Map && next['new_box_code'] != null) {
          _box.text = next['new_box_code'].toString();
        }
      } else {
        _msg = r.msg;
      }
    });
    if (r.ok) await _loadWage();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(['双扫报工', '今日核对', '提醒'][_tab])),
      body: IndexedStack(
        index: _tab,
        children: [
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              TextField(controller: _badge, decoration: const InputDecoration(labelText: '工牌码')),
              TextField(controller: _box, decoration: const InputDecoration(labelText: '箱码')),
              TextField(controller: _in, decoration: const InputDecoration(labelText: '投料重(kg)'), keyboardType: TextInputType.number),
              TextField(controller: _out, decoration: const InputDecoration(labelText: '完工重(kg)'), keyboardType: TextInputType.number),
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(child: OutlinedButton(onPressed: () => _scan(resolveOnly: true), child: const Text('预览'))),
                  const SizedBox(width: 8),
                  Expanded(child: FilledButton(onPressed: () => _scan(resolveOnly: false), child: const Text('提交草稿'))),
                ],
              ),
              if (_pendingReportId != null) ...[
                const SizedBox(height: 8),
                FilledButton.tonal(onPressed: _confirm, child: const Text('确认过账（定损/QC）')),
              ],
              if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 12), child: Text(_msg)),
            ],
          ),
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              ListTile(title: const Text('预计工钱'), subtitle: Text('¥${_daily?['total_amount'] ?? 0}')),
              ListTile(title: const Text('总完工重'), subtitle: Text('${_daily?['total_output_weight'] ?? _daily?['total_qty'] ?? 0}')),
              const Divider(),
              const Text('工序工价参考', style: TextStyle(fontWeight: FontWeight.bold)),
              ..._wages.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(title: Text('工序 ${m['process_id']}'), trailing: Text('¥${m['rate']}'));
              }),
            ],
          ),
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Text('MQTT ${context.watch<NotifyService>().mqttStatus}', style: const TextStyle(color: Colors.black54, fontSize: 12)),
              const SizedBox(height: 8),
              if (_notices.isEmpty) const Text('暂无提醒（报工确认 / 劳动支付）'),
              ..._notices.map((n) => ListTile(
                    title: Text(n['title']?.toString() ?? n['event_key']?.toString() ?? ''),
                    subtitle: Text(n['body']?.toString() ?? ''),
                    onTap: () {
                      final id = (n['id'] as num?)?.toInt();
                      if (id != null) context.read<NotifyService>().markRead(id);
                    },
                  )),
            ],
          ),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _tab,
        onDestinationSelected: (i) {
          setState(() => _tab = i);
          if (i == 1) _loadWage();
          if (i == 2) _applyNotices();
        },
        destinations: const [
          NavigationDestination(icon: Icon(Icons.qr_code), label: '报工'),
          NavigationDestination(icon: Icon(Icons.payments), label: '核对'),
          NavigationDestination(icon: Icon(Icons.notifications), label: '提醒'),
        ],
      ),
    );
  }
}
