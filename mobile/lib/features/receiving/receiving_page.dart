import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';

/// 现场过磅收货：建单 → 质检 → 确认出码 → 推仓管
class ReceivingPage extends StatefulWidget {
  const ReceivingPage({super.key});

  @override
  State<ReceivingPage> createState() => _ReceivingPageState();
}

class _ReceivingPageState extends State<ReceivingPage> with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  List<dynamic> _tickets = [];
  List<dynamic> _farmers = [];
  List<dynamic> _purTasks = [];
  int? _farmerId;
  String _channel = 'internal';
  String _grade = 'A';
  final _gross = TextEditingController();
  final _deductRate = TextEditingController(text: '5');
  final _variety = TextEditingController(text: '鲜木薯');
  final _origin = TextEditingController();
  final _plate = TextEditingController();
  final _image = TextEditingController();
  final _remark = TextEditingController();
  String _msg = '';
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 3, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) => _boot());
  }

  @override
  void dispose() {
    _tabs.dispose();
    _gross.dispose();
    _deductRate.dispose();
    _variety.dispose();
    _origin.dispose();
    _plate.dispose();
    _image.dispose();
    _remark.dispose();
    super.dispose();
  }

  Future<void> _boot() async {
    final auth = context.read<AuthState>();
    if (!canAccessEmployeeModule(EmployeeModule.receiving, auth.permissions, auth.roles)) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('无过磅收货权限')));
        Navigator.of(context).pop();
      }
      return;
    }
    await context.read<NotifyService>().start();
    await _refresh();
  }

  Future<void> _refresh() async {
    setState(() => _loading = true);
    final api = context.read<AuthState>().api;
    final results = await Future.wait([
      api.get('/purchase/weigh-tickets?page_size=50'),
      api.get('/purchase/farmers?page_size=100'),
      api.get('/purchase/tasks?page_size=50'),
    ]);
    if (!mounted) return;
    setState(() {
      _loading = false;
      _tickets = ApiClient.listOf(results[0].data);
      _farmers = ApiClient.listOf(results[1].data);
      _purTasks = ApiClient.listOf(results[2].data);
      if (_farmerId == null && _farmers.isNotEmpty) {
        final first = Map<String, dynamic>.from(_farmers.first as Map);
        _farmerId = (first['id'] as num?)?.toInt();
        if (_origin.text.isEmpty) _origin.text = first['origin']?.toString() ?? '';
      }
    });
  }

  Future<void> _completeTask(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/purchase/tasks/$id/complete', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '任务已完成' : r.msg)));
    if (r.ok) await _refresh();
  }

  Future<void> _claimTask(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/purchase/tasks/$id/assign', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已认领任务' : r.msg)));
    if (r.ok) await _refresh();
  }

  void _fillEvidence() {
    final ts = DateTime.now().toIso8601String().replaceAll(':', '');
    _image.text = 'mobile://weigh_photo/$ts';
    setState(() {});
  }

  Future<void> _create() async {
    if (_farmerId == null || _farmerId! <= 0) {
      setState(() => _msg = '请选择农户');
      return;
    }
    if (_image.text.trim().isEmpty) {
      setState(() => _msg = '过磅图必填（可点「生成现场证据」）');
      return;
    }
    final gross = double.tryParse(_gross.text) ?? 0;
    if (gross <= 0) {
      setState(() => _msg = '请输入毛重');
      return;
    }
    final rate = double.tryParse(_deductRate.text) ?? 0;
    final api = context.read<AuthState>().api;
    final r = await api.post('/purchase/weigh-tickets', {
      'farmer_id': _farmerId,
      'channel': _channel,
      'product_id': 1,
      'variety': _variety.text.trim(),
      'origin': _origin.text.trim(),
      'gross_weight': gross,
      'deduct_rate': rate > 1 ? rate : rate, // 5 或 0.05 后端都处理
      'grade': _grade,
      'biz_date': DateTime.now().toIso8601String().substring(0, 10),
      'source_type': 'self',
      'image_url': _image.text.trim(),
      'plate_no': _plate.text.trim(),
      'remark': _remark.text.trim(),
    });
    setState(() => _msg = r.ok ? '草稿已创建 #${(r.data is Map) ? (r.data as Map)['doc_no'] : ''}' : r.msg);
    if (r.ok) {
      _gross.clear();
      await _refresh();
      _tabs.animateTo(1);
    }
  }

  Future<void> _qc(Map row, {required bool pass}) async {
    final id = (row['id'] as num?)?.toInt();
    if (id == null) return;
    final api = context.read<AuthState>().api;
    final body = <String, dynamic>{
      'qc_result': pass ? 'pass' : 'fail',
      if (pass) 'grade': _grade,
    };
    final r = await api.post('/purchase/weigh-tickets/$id/qc', body);
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? (pass ? '质检合格' : '质检不合格') : r.msg)));
    if (r.ok) await _refresh();
  }

  Future<void> _confirm(Map row) async {
    final id = (row['id'] as num?)?.toInt();
    if (id == null) return;
    final api = context.read<AuthState>().api;
    final r = await api.post('/purchase/weigh-tickets/$id/confirm', {
      'confirmed': true,
      'gross_weight': row['gross_weight'],
      'deduct_rate': row['deduct_rate'],
      'deduct_weight': row['deduct_weight'],
      'net_weight': row['net_weight'],
      'grade': row['grade'] ?? _grade,
    });
    if (!mounted) return;
    final trace = r.ok && r.data is Map ? (r.data as Map)['trace_code'] : '';
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(r.ok ? '已出码并推仓管 · $trace' : r.msg)),
    );
    if (r.ok) await _refresh();
  }

  Color _statusColor(String st) {
    switch (st) {
      case 'draft':
        return Colors.orange;
      case 'qc_pass':
      case 'pending_confirm':
        return Colors.blue;
      case 'weighed':
        return Colors.teal;
      case 'stocked':
        return Colors.green;
      case 'qc_fail':
        return Colors.red;
      default:
        return Colors.grey;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('过磅收货'),
        bottom: TabBar(controller: _tabs, tabs: const [
          Tab(text: '新建过磅'),
          Tab(text: '单据列表'),
          Tab(text: '采购任务'),
        ]),
        actions: [
          IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh)),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tabs,
              children: [
                ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    DropdownButtonFormField<int>(
                      value: _farmerId,
                      decoration: const InputDecoration(labelText: '农户'),
                      items: _farmers.map((e) {
                        final m = Map<String, dynamic>.from(e as Map);
                        return DropdownMenuItem(
                          value: (m['id'] as num?)?.toInt(),
                          child: Text('${m['name'] ?? m['code'] ?? m['id']}'),
                        );
                      }).toList(),
                      onChanged: (v) {
                        setState(() => _farmerId = v);
                        final f = _farmers.cast<dynamic>().map((e) => Map<String, dynamic>.from(e as Map)).where((m) => (m['id'] as num?)?.toInt() == v);
                        if (f.isNotEmpty && _origin.text.isEmpty) {
                          _origin.text = f.first['origin']?.toString() ?? '';
                        }
                      },
                    ),
                    const SizedBox(height: 8),
                    SegmentedButton<String>(
                      segments: const [
                        ButtonSegment(value: 'internal', label: Text('厂内秤')),
                        ButtonSegment(value: 'external', label: Text('外磅单')),
                      ],
                      selected: {_channel},
                      onSelectionChanged: (s) => setState(() => _channel = s.first),
                    ),
                    TextField(controller: _gross, decoration: const InputDecoration(labelText: '毛重(kg)'), keyboardType: TextInputType.number),
                    TextField(controller: _deductRate, decoration: const InputDecoration(labelText: '扣损率(% 或小数)'), keyboardType: TextInputType.number),
                    TextField(controller: _variety, decoration: const InputDecoration(labelText: '品种')),
                    TextField(controller: _origin, decoration: const InputDecoration(labelText: '产地')),
                    TextField(controller: _plate, decoration: const InputDecoration(labelText: '车牌')),
                    DropdownButtonFormField<String>(
                      value: _grade,
                      decoration: const InputDecoration(labelText: '等级(质检用)'),
                      items: const [
                        DropdownMenuItem(value: 'A', child: Text('A')),
                        DropdownMenuItem(value: 'B', child: Text('B')),
                        DropdownMenuItem(value: 'C', child: Text('C')),
                      ],
                      onChanged: (v) => setState(() => _grade = v ?? 'A'),
                    ),
                    TextField(controller: _image, decoration: const InputDecoration(labelText: '过磅图 URL / 证据')),
                    Row(
                      children: [
                        OutlinedButton(onPressed: _fillEvidence, child: const Text('生成现场证据')),
                        const SizedBox(width: 8),
                        Expanded(child: Text('无相机时可用占位证据过流程', style: TextStyle(fontSize: 11, color: Colors.black54))),
                      ],
                    ),
                    TextField(controller: _remark, decoration: const InputDecoration(labelText: '备注')),
                    const SizedBox(height: 12),
                    FilledButton(onPressed: _create, child: const Text('创建过磅草稿')),
                    if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 12), child: Text(_msg)),
                  ],
                ),
                RefreshIndicator(
                  onRefresh: _refresh,
                  child: ListView.builder(
                    padding: const EdgeInsets.all(12),
                    itemCount: _tickets.length,
                    itemBuilder: (context, i) {
                      final m = Map<String, dynamic>.from(_tickets[i] as Map);
                      final st = m['status']?.toString() ?? '';
                      return Card(
                        child: Padding(
                          padding: const EdgeInsets.all(12),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Row(
                                children: [
                                  Expanded(child: Text('${m['doc_no']}', style: const TextStyle(fontWeight: FontWeight.bold))),
                                  Chip(
                                    label: Text(st, style: const TextStyle(fontSize: 11, color: Colors.white)),
                                    backgroundColor: _statusColor(st),
                                    visualDensity: VisualDensity.compact,
                                  ),
                                ],
                              ),
                              Text('${m['farmer_name'] ?? ''} · ${m['channel'] == 'external' ? '外磅' : '厂内'}'),
                              Text('毛${m['gross_weight']} / 扣${m['deduct_weight']} / 净${m['net_weight']} kg'),
                              if ((m['trace_code']?.toString() ?? '').isNotEmpty) Text('溯源 ${m['trace_code']}', style: const TextStyle(color: Colors.teal)),
                              const SizedBox(height: 8),
                              Wrap(
                                spacing: 8,
                                children: [
                                  if (st == 'draft' || st == 'qc_pending') ...[
                                    FilledButton.tonal(onPressed: () => _qc(m, pass: true), child: const Text('质检合格')),
                                    OutlinedButton(onPressed: () => _qc(m, pass: false), child: const Text('不合格')),
                                  ],
                                  if (st == 'qc_pass' || st == 'pending_confirm')
                                    FilledButton(onPressed: () => _confirm(m), child: const Text('确认出码推仓')),
                                ],
                              ),
                            ],
                          ),
                        ),
                      );
                    },
                  ),
                ),
                RefreshIndicator(
                  onRefresh: _refresh,
                  child: ListView(
                    padding: const EdgeInsets.all(12),
                    children: [
                      const Text('现场采购任务（认领 / 完成）', style: TextStyle(fontWeight: FontWeight.bold)),
                      const SizedBox(height: 8),
                      if (_purTasks.isEmpty) const Padding(padding: EdgeInsets.all(24), child: Center(child: Text('暂无采购任务'))),
                      ..._purTasks.map((e) {
                        final m = Map<String, dynamic>.from(e as Map);
                        final st = m['status']?.toString() ?? '';
                        return Card(
                          child: ListTile(
                            title: Text('${m['title'] ?? m['doc_no'] ?? m['id']}'),
                            subtitle: Text('$st · 数量 ${m['qty'] ?? '-'} · 到期 ${m['due_date'] ?? '-'}'),
                            trailing: Wrap(
                              children: [
                                if (st == 'open') TextButton(onPressed: () => _claimTask(m), child: const Text('认领')),
                                if (st != 'done') FilledButton.tonal(onPressed: () => _completeTask(m), child: const Text('完成')),
                              ],
                            ),
                          ),
                        );
                      }),
                    ],
                  ),
                ),
              ],
            ),
    );
  }
}
