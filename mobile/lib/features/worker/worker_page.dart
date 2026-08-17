import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';

const _scrapOptions = <MapEntry<String, String>>[
  MapEntry('', '无次品'),
  MapEntry('cut_defect', '切断次品'),
  MapEntry('core_defect', '去芯次品'),
  MapEntry('dice_defect', '切块次品'),
  MapEntry('sieve_bag_defect', '过筛装袋次品'),
];

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
  final _bag = TextEditingController(text: '0');
  String _scrapType = '';
  String _msg = '';
  Map<String, dynamic>? _daily;
  List<dynamic> _wages = [];
  Map<String, dynamic>? _last;
  int? _pendingReportId;
  List<Map<String, dynamic>> _notices = [];
  List<Map<String, dynamic>> _issueLines = [];
  List<Map<String, dynamic>> _tools = [];

  static const _titles = ['双扫报工', '今日核对', '领料/工具', '联动领料', '提醒'];
  final _reqQty = TextEditingController(text: '50');
  List<dynamic> _reqs = [];

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
    _bag.dispose();
    _reqQty.dispose();
    super.dispose();
  }

  void _onNotify() => _applyNotices();

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

  Future<void> _loadLedger() async {
    final api = context.read<AuthState>().api;
    final auth = context.read<AuthState>();
    final me = auth.name ?? '';
    final today = DateTime.now().toIso8601String().substring(0, 10);
    final sheetsRes = await api.get('/production/piece-issue-sheets');
    final toolsRes = await api.get('/hr/tool-issues');
    final sheetList = (sheetsRes.data is Map ? (sheetsRes.data as Map)['list'] as List? : null) ?? [];
    final lines = <Map<String, dynamic>>[];
    for (final raw in sheetList.take(5)) {
      final sh = Map<String, dynamic>.from(raw as Map);
      final biz = sh['biz_date']?.toString() ?? '';
      if (biz.isNotEmpty && !biz.startsWith(today)) continue;
      final id = (sh['id'] as num?)?.toInt();
      if (id == null) continue;
      final det = await api.get('/production/piece-issue-sheets/$id');
      final arr = (det.data is Map ? (det.data as Map)['lines'] as List? : null) ?? [];
      for (final lnRaw in arr) {
        final ln = Map<String, dynamic>.from(lnRaw as Map);
        final name = ln['employee_name']?.toString() ?? '';
        if (me.isEmpty || name.isEmpty || name.contains(me)) {
          lines.add({...ln, 'sheet_doc': sh['doc_no']});
        }
      }
    }
    final tools = ((toolsRes.data is Map ? (toolsRes.data as Map)['list'] as List? : null) ?? [])
        .map((e) => Map<String, dynamic>.from(e as Map))
        .where((t) {
          final name = t['employee_name']?.toString() ?? '';
          return me.isEmpty || name.contains(me);
        })
        .toList();
    if (!mounted) return;
    setState(() {
      _issueLines = lines.take(30).toList();
      _tools = tools;
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

  Future<void> _loadReqs() async {
    final r = await context.read<AuthState>().api.get('/production/requisitions');
    if (!mounted) return;
    setState(() => _reqs = ApiClient.listOf(r.data));
  }

  Future<void> _createReq() async {
    final api = context.read<AuthState>().api;
    final qty = double.tryParse(_reqQty.text) ?? 50;
    final r = await api.post('/production/requisitions', {
      'product_id': 1,
      'qty': qty,
      'warehouse_id': 1,
      'weight': qty,
      'remark': '工人联动领料',
      'lines': [
        {'product_id': 1, 'qty': qty},
      ],
    });
    if (!mounted) return;
    setState(() => _msg = r.ok ? '领料单已建' : r.msg);
    if (r.ok) await _loadReqs();
  }

  Future<void> _postReq(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) {
      setState(() => _msg = '单据无 id，无法过账');
      return;
    }
    final r = await context.read<AuthState>().api.post('/production/requisitions/$id/post', {});
    if (!mounted) return;
    setState(() => _msg = r.ok ? '领料已过账出库' : r.msg);
    if (r.ok) await _loadReqs();
  }

  Future<void> _confirm() async {
    final id = _pendingReportId ?? (_last?['id'] as num?)?.toInt();
    if (id == null) {
      setState(() => _msg = '请先提交报工草稿');
      return;
    }
    final api = context.read<AuthState>().api;
    final outW = double.tryParse(_out.text) ?? 0;
    final body = <String, dynamic>{
      'input_weight': double.tryParse(_in.text) ?? 0,
      'output_weight': outW,
      'process_qc_result': 'pass',
      'bag_qty': double.tryParse(_bag.text) ?? 0,
    };
    if (_scrapType.isNotEmpty) body['scrap_type'] = _scrapType;
    final r = await api.post('/production/report-works/$id/confirm', body);
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
      appBar: AppBar(title: Text(_titles[_tab])),
      body: IndexedStack(
        index: _tab,
        children: [
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              TextField(controller: _badge, decoration: const InputDecoration(labelText: '工牌码')),
              TextField(controller: _box, decoration: const InputDecoration(labelText: '板码')),
              TextField(controller: _in, decoration: const InputDecoration(labelText: '投料重(kg)'), keyboardType: TextInputType.number),
              TextField(controller: _out, decoration: const InputDecoration(labelText: '完工重(kg)'), keyboardType: TextInputType.number),
              TextField(controller: _bag, decoration: const InputDecoration(labelText: '袋数'), keyboardType: TextInputType.number),
              const Text('次品类型', style: TextStyle(fontSize: 12, color: Colors.black54)),
              Wrap(
                spacing: 8,
                children: _scrapOptions
                    .map((e) => ChoiceChip(
                          label: Text(e.value),
                          selected: _scrapType == e.key,
                          onSelected: (_) => setState(() => _scrapType = e.key),
                        ))
                    .toList(),
              ),
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
              Card(
                color: Colors.teal.shade50,
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('今日产量与工钱', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                      const SizedBox(height: 8),
                      Text('预计工钱  ¥${_daily?['total_amount'] ?? 0}', style: const TextStyle(fontSize: 28, fontWeight: FontWeight.bold)),
                      Text('完工重 ${_daily?['total_output_weight'] ?? _daily?['total_qty'] ?? 0} kg'),
                      Text('报工次数 ${_daily?['report_count'] ?? _daily?['count'] ?? '-'}'),
                      const SizedBox(height: 8),
                      OutlinedButton(onPressed: _loadWage, child: const Text('刷新核对')),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 12),
              const Text('工序工价参考', style: TextStyle(fontWeight: FontWeight.bold)),
              ..._wages.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['process_name'] ?? '工序 ${m['process_id']}'}'),
                  trailing: Text('¥${m['rate']}/kg'),
                );
              }),
            ],
          ),
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              const Text('计件领料（只读）', style: TextStyle(fontWeight: FontWeight.bold)),
              if (_issueLines.isEmpty) const Padding(padding: EdgeInsets.symmetric(vertical: 8), child: Text('暂无')),
              ..._issueLines.map((ln) => ListTile(
                    title: Text('${ln['process_name'] ?? ln['process_kind'] ?? ''}'),
                    subtitle: Text('${ln['employee_name'] ?? ''} · 数量${ln['qty_total'] ?? ln['qty']} · ¥${ln['amount'] ?? 0}'),
                    trailing: Text('${ln['sheet_doc'] ?? ''}'),
                  )),
              const Divider(),
              ListTile(
                title: const Text('物料工具申请/归还', style: TextStyle(fontWeight: FontWeight.bold)),
                subtitle: const Text('提交申请并指定下一手处理人'),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => Navigator.of(context).pushNamed('/tools'),
              ),
              ListTile(
                title: const Text('我的工单'),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => Navigator.of(context).pushNamed('/tickets'),
              ),
              if (_tools.isEmpty) const Padding(padding: EdgeInsets.symmetric(vertical: 8), child: Text('暂无工具单据')),
              ..._tools.map((t) => ListTile(
                    title: Text('${t['tool_name'] ?? ''}'),
                    subtitle: Text('领${t['issue_qty']} / 还${t['return_qty']} · 合计${t['total_qty']}'),
                    trailing: Text('${t['status'] ?? ''}'),
                  )),
            ],
          ),
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              const Text('联动领料（提交并过账出库）', style: TextStyle(fontWeight: FontWeight.bold)),
              TextField(controller: _reqQty, decoration: const InputDecoration(labelText: '领料数量/重量'), keyboardType: TextInputType.number),
              FilledButton(onPressed: _createReq, child: const Text('新建领料单')),
              if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_msg)),
              const Divider(),
              ..._reqs.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['doc_no'] ?? m['id'] ?? '领料'}'),
                  subtitle: Text('${m['status'] ?? m['doc_type'] ?? ''} · qty ${m['qty'] ?? ''}'),
                  trailing: TextButton(onPressed: () => _postReq(m), child: const Text('过账')),
                );
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
          if (i == 2) _loadLedger();
          if (i == 3) _loadReqs();
          if (i == 4) _applyNotices();
        },
        destinations: const [
          NavigationDestination(icon: Icon(Icons.qr_code), label: '报工'),
          NavigationDestination(icon: Icon(Icons.payments), label: '核对'),
          NavigationDestination(icon: Icon(Icons.inventory_2), label: '领料'),
          NavigationDestination(icon: Icon(Icons.outbox), label: '联动'),
          NavigationDestination(icon: Icon(Icons.notifications), label: '提醒'),
        ],
      ),
    );
  }
}
