import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';

class WorkshopPage extends StatefulWidget {
  const WorkshopPage({super.key});

  @override
  State<WorkshopPage> createState() => _WorkshopPageState();
}

class _WorkshopPageState extends State<WorkshopPage> with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  final _badge = TextEditingController();
  final _box = TextEditingController();
  final _weight = TextEditingController();
  final _bag = TextEditingController(text: '0');
  String _scrapType = '';
  String _msg = '';
  Map<String, dynamic>? _last;
  int? _pendingId;
  List<dynamic> _tasks = [];
  List<dynamic> _dispatches = [];
  List<dynamic> _balances = [];
  List<dynamic> _processes = [];
  List<dynamic> _flows = [];

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 6, vsync: this);
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
    _tabs.dispose();
    _badge.dispose();
    _box.dispose();
    _weight.dispose();
    _bag.dispose();
    super.dispose();
  }

  void _onNotify() {
    if (!mounted) return;
    final notify = context.read<NotifyService>();
    for (final raw in notify.inbox) {
      if (raw is! Map) continue;
      if (raw['event_key']?.toString() != 'production.report_confirmed') continue;
      final p = NotifyService.parsePayload(raw['payload'] ?? raw['payload_json']);
      final next = p['next'] is Map ? Map<String, dynamic>.from(p['next'] as Map) : p;
      final code = next['new_box_code'];
      if (code != null && _box.text.trim().isEmpty) {
        setState(() => _box.text = code.toString());
        break;
      }
    }
    _refresh();
  }

  Future<void> _boot() async {
    final auth = context.read<AuthState>();
    if (!canAccessEmployeeModule(EmployeeModule.workshop, auth.permissions, auth.roles)) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('无车间模块权限')));
        Navigator.of(context).pop();
      }
      return;
    }
    await context.read<NotifyService>().start();
    await _refresh();
  }

  Future<void> _refresh() async {
    final api = context.read<AuthState>().api;
    final results = await Future.wait([
      api.get('/production/tasks'),
      api.get('/production/dispatches'),
      api.get('/inventory/balances'),
      api.get('/production/processes'),
      api.get('/production/flow-events'),
    ]);
    if (!mounted) return;
    setState(() {
      _tasks = (results[0].data is Map ? (results[0].data as Map)['list'] as List? : null) ?? [];
      _dispatches = (results[1].data is Map ? (results[1].data as Map)['list'] as List? : null) ?? [];
      _balances = (results[2].data is Map ? (results[2].data as Map)['list'] as List? : null) ?? [];
      _processes = (results[3].data is Map ? (results[3].data as Map)['list'] as List? : null) ?? [];
      _flows = (results[4].data is Map ? (results[4].data as Map)['list'] as List? : null) ?? [];
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
    final net = double.tryParse(_weight.text) ?? 0;
    final r = await api.post(path, {
      'badge_code': _badge.text.trim(),
      'box_code': _box.text.trim(),
      'net_weight': net,
      'output_weight': net,
    });
    setState(() {
      if (!r.ok) {
        _msg = r.msg;
        return;
      }
      final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
      _last = data;
      if (resolveOnly) {
        _msg = '已解析 ${data['worker_name'] ?? ''}';
      } else if (data['needs_confirm'] == true || data['status'] == 'confirm_pending') {
        _pendingId = (data['id'] as num?)?.toInt();
        _msg = '草稿已建，请确认过账';
      } else {
        _pendingId = null;
        _msg = '报工成功';
      }
      final next = data['next'];
      if (next is Map && next['new_box_code'] != null) {
        _box.text = next['new_box_code'].toString();
      }
    });
    if (r.ok && !resolveOnly && _pendingId == null) await _refresh();
  }

  Future<void> _confirm() async {
    final id = _pendingId ?? (_last?['id'] as num?)?.toInt();
    if (id == null) {
      setState(() => _msg = '请先提交报工草稿');
      return;
    }
    final api = context.read<AuthState>().api;
    final net = double.tryParse(_weight.text) ?? 0;
    final body = <String, dynamic>{
      'output_weight': net,
      'process_qc_result': 'pass',
      'bag_qty': double.tryParse(_bag.text) ?? 0,
    };
    if (_scrapType.isNotEmpty) body['scrap_type'] = _scrapType;
    final r = await api.post('/production/report-works/$id/confirm', body);
    setState(() {
      if (r.ok) {
        final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
        _last = data;
        _pendingId = null;
        _msg = '已确认过账';
        final next = data['next'];
        if (next is Map && next['new_box_code'] != null) {
          _box.text = next['new_box_code'].toString();
        }
      } else {
        _msg = r.msg;
      }
    });
    if (r.ok) await _refresh();
  }

  void _useBox(Map m) {
    final code = m['box_code'] ?? m['scan_code'] ?? m['doc_no'];
    if (code != null) {
      setState(() {
        _box.text = code.toString();
        _tabs.index = 0;
      });
    }
  }

  Widget _list(List<dynamic> rows, String Function(Map) title, {void Function(Map)? onTap}) {
    if (rows.isEmpty) return const Center(child: Text('暂无数据'));
    return ListView.builder(
      itemCount: rows.length,
      itemBuilder: (_, i) {
        final m = Map<String, dynamic>.from(rows[i] as Map);
        return ListTile(
          title: Text(title(m)),
          subtitle: Text(m['status']?.toString() ?? ''),
          onTap: onTap == null ? null : () => onTap(m),
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    final mqtt = context.watch<NotifyService>().mqttStatus;
    return Scaffold(
      appBar: AppBar(
        title: Text('车间 · $mqtt'),
        bottom: TabBar(
          controller: _tabs,
          isScrollable: true,
          tabs: const [
            Tab(text: '扫码'),
            Tab(text: '流转'),
            Tab(text: '任务'),
            Tab(text: '派工'),
            Tab(text: '工序'),
            Tab(text: '库存'),
          ],
        ),
      ),
      body: TabBarView(
        controller: _tabs,
        children: [
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              TextField(controller: _badge, decoration: const InputDecoration(labelText: '工牌码')),
              TextField(controller: _box, decoration: const InputDecoration(labelText: '箱码')),
              TextField(controller: _weight, decoration: const InputDecoration(labelText: '净重(kg)'), keyboardType: TextInputType.number),
              TextField(controller: _bag, decoration: const InputDecoration(labelText: '袋数'), keyboardType: TextInputType.number),
              const Text('次品类型', style: TextStyle(fontSize: 12, color: Colors.black54)),
              Wrap(
                spacing: 8,
                children: [
                  for (final e in const [
                    MapEntry('', '无次品'),
                    MapEntry('cut_defect', '切断次品'),
                    MapEntry('core_defect', '去芯次品'),
                    MapEntry('dice_defect', '切块次品'),
                    MapEntry('sieve_bag_defect', '过筛装袋次品'),
                  ])
                    ChoiceChip(
                      label: Text(e.value),
                      selected: _scrapType == e.key,
                      onSelected: (_) => setState(() => _scrapType = e.key),
                    ),
                ],
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(child: OutlinedButton(onPressed: () => _scan(resolveOnly: true), child: const Text('预解析'))),
                  const SizedBox(width: 8),
                  Expanded(child: FilledButton(onPressed: () => _scan(resolveOnly: false), child: const Text('提交草稿'))),
                ],
              ),
              if (_pendingId != null) ...[
                const SizedBox(height: 8),
                FilledButton.tonal(onPressed: _confirm, child: const Text('确认过账')),
              ],
              if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 12), child: Text(_msg)),
            ],
          ),
          _list(_flows, (m) => '#${m['id']} ${m['trigger'] ?? ''}'),
          _list(_tasks, (m) => m['doc_no']?.toString() ?? '#${m['id']}', onTap: _useBox),
          _list(_dispatches, (m) => m['doc_no']?.toString() ?? '#${m['id']}', onTap: _useBox),
          _list(_processes, (m) => m['name']?.toString() ?? m['code']?.toString() ?? ''),
          _list(_balances, (m) => '${m['warehouse_name'] ?? ''} ${m['product_name'] ?? ''} qty=${m['qty']}'),
        ],
      ),
    );
  }
}
