import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../widgets/form_sticky_actions.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_row.dart';
import '../../core/notify_service.dart';

/// 车间主任工作台：报工 + 任务派工 + 灵活派发 + 质检/返修/废料
class WorkshopPage extends StatefulWidget {
  const WorkshopPage({super.key, this.asTab = false});

  /// 作为产线壳 Tab 时隐藏标题，仅保留 TabBar / 操作。
  final bool asTab;

  @override
  State<WorkshopPage> createState() => _WorkshopPageState();
}

class _WorkshopPageState extends State<WorkshopPage> with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  final _badge = TextEditingController();
  final _box = TextEditingController();
  final _weight = TextEditingController();
  final _bag = TextEditingController(text: '0');
  final _flexQty = TextEditingController(text: '100');
  final _flexWorker = TextEditingController(text: '2');
  final _qcQty = TextEditingController(text: '10');
  final _scrapQty = TextEditingController(text: '5');
  final _reworkQty = TextEditingController(text: '5');
  String _scrapType = '';
  String _msg = '';
  Map<String, dynamic>? _last;
  int? _pendingId;
  int? _taskId;
  int? _processId;
  List<dynamic> _tasks = [];
  List<dynamic> _dispatches = [];
  List<dynamic> _flex = [];
  List<dynamic> _balances = [];
  List<dynamic> _processes = [];
  List<dynamic> _flows = [];
  List<dynamic> _qcs = [];
  List<dynamic> _scraps = [];
  List<dynamic> _reworks = [];
  List<dynamic> _drawings = [];
  Map<String, dynamic>? _overview;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 8, vsync: this);
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
    for (final c in [_badge, _box, _weight, _bag, _flexQty, _flexWorker, _qcQty, _scrapQty, _reworkQty]) {
      c.dispose();
    }
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
      api.get('/production/flex-dispatches'),
      api.get('/production/qc-orders'),
      api.get('/production/scraps'),
      api.get('/production/reworks'),
      api.get('/production/workshop-workbench/overview'),
      api.get('/production/drawing-links'),
    ]);
    if (!mounted) return;
    setState(() {
      _tasks = ApiClient.listOf(results[0].data);
      _dispatches = ApiClient.listOf(results[1].data);
      _balances = ApiClient.listOf(results[2].data);
      _processes = ApiClient.listOf(results[3].data);
      _flows = ApiClient.listOf(results[4].data);
      _flex = ApiClient.listOf(results[5].data);
      _qcs = ApiClient.listOf(results[6].data);
      _scraps = ApiClient.listOf(results[7].data);
      _reworks = ApiClient.listOf(results[8].data);
      _overview = results[9].data is Map ? Map<String, dynamic>.from(results[9].data as Map) : null;
      _drawings = ApiClient.listOf(results[10].data);
      if (_taskId == null && _tasks.isNotEmpty) {
        _taskId = (_tasks.first as Map)['id'] is num ? ((_tasks.first as Map)['id'] as num).toInt() : null;
      }
      if (_processId == null && _processes.isNotEmpty) {
        _processId = (_processes.first as Map)['id'] is num ? ((_processes.first as Map)['id'] as num).toInt() : null;
      }
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

  Future<void> _receiveDispatch(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/production/dispatches/$id/receive', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已接收派工' : r.msg)));
    if (r.ok) await _refresh();
  }

  Future<void> _createFlex() async {
    final r = await context.read<AuthState>().api.post('/production/flex-dispatches', {
      'task_id': _taskId,
      'process_id': _processId,
      'worker_id': int.tryParse(_flexWorker.text) ?? 2,
      'qty': double.tryParse(_flexQty.text) ?? 100,
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '灵活派发成功' : r.msg)));
    if (r.ok) await _refresh();
  }

  Future<void> _reassign(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final wid = int.tryParse(_flexWorker.text) ?? 2;
    final r = await context.read<AuthState>().api.post('/production/flex-dispatches/$id/reassign', {'worker_id': wid});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已改派工人 $wid' : r.msg)));
    if (r.ok) await _refresh();
  }

  Future<void> _createQc() async {
    final r = await context.read<AuthState>().api.post('/production/qc-orders', {
      'qc_type': 'process',
      'product_id': 3,
      'process_id': _processId,
      'qty': double.tryParse(_qcQty.text) ?? 10,
      'remark': '车间质检',
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '质检单已建' : r.msg)));
    if (r.ok) await _refresh();
  }

  Future<void> _completeQc(Map m, String result) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/production/qc-orders/$id/complete', {'result': result});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '质检$result' : r.msg)));
    if (r.ok) await _refresh();
  }

  Future<void> _createScrap() async {
    final r = await context.read<AuthState>().api.post('/production/scraps', {
      'task_id': _taskId,
      'process_id': _processId,
      'product_id': 3,
      'qty': double.tryParse(_scrapQty.text) ?? 5,
      'weight': double.tryParse(_scrapQty.text) ?? 5,
      'disposition': 'waste',
      'scrap_type': _scrapType.isEmpty ? 'cut_defect' : _scrapType,
      'remark': '车间废料登记',
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '废料已登记' : r.msg)));
    if (r.ok) await _refresh();
  }

  Future<void> _createRework() async {
    final r = await context.read<AuthState>().api.post('/production/reworks', {
      'task_id': _taskId,
      'process_id': _processId,
      'qty': double.tryParse(_reworkQty.text) ?? 5,
      'remark': '车间返修',
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '返修单已建' : r.msg)));
    if (r.ok) await _refresh();
  }

  Future<void> _closeRework(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/production/reworks/$id/close', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '返修已关闭' : r.msg)));
    if (r.ok) await _refresh();
  }

  Widget _list(List<dynamic> rows, String Function(Map) title, {Widget Function(Map)? trailing, void Function(Map)? onTap}) {
    if (rows.isEmpty) return const Padding(padding: EdgeInsets.all(24), child: Center(child: Text('暂无数据')));
    return ListView.builder(
      itemCount: rows.length,
      itemBuilder: (_, i) {
        final m = Map<String, dynamic>.from(rows[i] as Map);
        return ListTile(
          title: Text(title(m)),
          subtitle: Text(m['status']?.toString() ?? ''),
          trailing: trailing?.call(m),
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
        title: widget.asTab ? null : Text('车间 · $mqtt'),
        toolbarHeight: widget.asTab ? 0 : kToolbarHeight,
        bottom: TabBar(
          controller: _tabs,
          isScrollable: true,
          tabs: const [
            Tab(text: '扫码'),
            Tab(text: '概览'),
            Tab(text: '任务'),
            Tab(text: '派工'),
            Tab(text: '灵活派发'),
            Tab(text: '质检废料'),
            Tab(text: '工序'),
            Tab(text: '库存'),
          ],
        ),
        actions: [IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh))],
      ),
      body: TabBarView(
        controller: _tabs,
        children: [
          Column(
            children: [
              Expanded(
                child: ListView(
                  keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
                  padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
                  children: [
                    const FormSectionHeader('扫码过站'),
                    FormRow.text(label: '工牌码', controller: _badge, requiredMark: true),
                    FormRow.text(label: '箱码', controller: _box, requiredMark: true),
                    FormRow.text(label: '净重(kg)', controller: _weight, keyboardType: TextInputType.number, requiredMark: true),
                    FormRow.text(label: '袋数', controller: _bag, keyboardType: TextInputType.number),
                    const FormSectionHeader('次品类型'),
                    FormRow(
                      label: '次品',
                      child: Wrap(
                        alignment: WrapAlignment.end,
                        spacing: 6,
                        runSpacing: 4,
                        children: [
                          for (final e in const [
                            MapEntry('', '无次品'),
                            MapEntry('cut_defect', '切断次品'),
                            MapEntry('core_defect', '去芯次品'),
                            MapEntry('dice_defect', '切块次品'),
                            MapEntry('sieve_bag_defect', '过筛装袋次品'),
                          ])
                            ChoiceChip(
                              label: Text(e.value, style: const TextStyle(fontSize: 12)),
                              selected: _scrapType == e.key,
                              visualDensity: VisualDensity.compact,
                              onSelected: (_) => setState(() => _scrapType = e.key),
                            ),
                        ],
                      ),
                    ),
                    if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 12), child: Text(_msg)),
                  ],
                ),
              ),
              FormStickyButtonBar(
                children: [
                  OutlinedButton(onPressed: () => _scan(resolveOnly: true), child: const Text('预解析')),
                  FilledButton(onPressed: () => _scan(resolveOnly: false), child: const Text('提交草稿')),
                ],
              ),
              if (_pendingId != null)
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
                  child: FilledButton.tonal(onPressed: _confirm, child: const Text('确认过账')),
                ),
            ],
          ),
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              if (_overview == null) const Text('加载中…'),
              if (_overview != null) ...[
                ListTile(title: const Text('进行中任务'), trailing: Text('${_overview!['open_tasks'] ?? 0}')),
                ListTile(title: const Text('未完派工'), trailing: Text('${_overview!['open_dispatches'] ?? 0}')),
                ListTile(title: const Text('今日报工'), trailing: Text('${_overview!['today_reports'] ?? 0}')),
                ListTile(title: const Text('流转失败'), trailing: Text('${_overview!['failed_flow_events'] ?? 0}')),
                Text('${_overview!['hint'] ?? ''}', style: const TextStyle(color: Colors.black54, fontSize: 12)),
              ],
              const Divider(),
              const Text('图纸分发', style: TextStyle(fontWeight: FontWeight.bold)),
              if (_drawings.isEmpty) const Text('暂无图纸'),
              ..._drawings.take(15).map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  dense: true,
                  title: Text('${m['drawing_name'] ?? m['drawing_code'] ?? m['id']}'),
                  subtitle: Text('${m['file_url'] ?? m['status'] ?? ''} · 任务${m['task_id'] ?? '-'}'),
                );
              }),
              const Divider(),
              const Text('最近流转', style: TextStyle(fontWeight: FontWeight.bold)),
              ..._flows.take(15).map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  dense: true,
                  title: Text('${m['event_type'] ?? m['status'] ?? m['id']}'),
                  subtitle: Text('${m['box_code'] ?? m['doc_no'] ?? ''}'),
                );
              }),
            ],
          ),
          _list(_tasks, (m) => '${m['doc_no'] ?? m['id']} · ${m['status']}', onTap: (m) {
            setState(() => _taskId = (m['id'] as num?)?.toInt());
            ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('已选任务 $_taskId')));
          }),
          _list(
            _dispatches,
            (m) => '${m['doc_no'] ?? m['id']} · 工人${m['worker_id'] ?? '-'}',
            trailing: (m) => TextButton(onPressed: () => _receiveDispatch(m), child: const Text('接收')),
          ),
          ListView(
            keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
            padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
            children: [
              const FormSectionHeader('灵活派发'),
              FormRow(
                label: '任务',
                child: DropdownButtonHideUnderline(
                  child: DropdownButton<int>(
                    isExpanded: true,
                    value: _taskId,
                    alignment: Alignment.centerRight,
                    hint: const Text('请选择', textAlign: TextAlign.right),
                    items: _tasks.map((e) {
                      final m = Map<String, dynamic>.from(e as Map);
                      return DropdownMenuItem(value: (m['id'] as num?)?.toInt(), child: Text('${m['doc_no'] ?? m['id']}', textAlign: TextAlign.right));
                    }).toList(),
                    onChanged: (v) => setState(() => _taskId = v),
                  ),
                ),
              ),
              FormRow(
                label: '工序',
                child: DropdownButtonHideUnderline(
                  child: DropdownButton<int>(
                    isExpanded: true,
                    value: _processId,
                    alignment: Alignment.centerRight,
                    hint: const Text('请选择', textAlign: TextAlign.right),
                    items: _processes.map((e) {
                      final m = Map<String, dynamic>.from(e as Map);
                      return DropdownMenuItem(value: (m['id'] as num?)?.toInt(), child: Text('${m['name'] ?? m['id']}', textAlign: TextAlign.right));
                    }).toList(),
                    onChanged: (v) => setState(() => _processId = v),
                  ),
                ),
              ),
              FormRow.text(label: '工人员工ID', controller: _flexWorker, keyboardType: TextInputType.number, requiredMark: true),
              FormRow.text(label: '计划数量', controller: _flexQty, keyboardType: TextInputType.number, requiredMark: true),
              FormStickyActions(primaryLabel: '灵活派发', onPrimary: _createFlex),
              const Divider(),
              ..._flex.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['doc_no'] ?? m['id']}'),
                  subtitle: Text('工人 ${m['worker_id']} · ${m['status']}'),
                  trailing: TextButton(onPressed: () => _reassign(m), child: const Text('改派')),
                );
              }),
            ],
          ),
          ListView(
            keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
            padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
            children: [
              const FormSectionHeader('质检'),
              FormRow.text(label: '质检数量', controller: _qcQty, keyboardType: TextInputType.number, requiredMark: true),
              FormStickyActions(primaryLabel: '新建质检单', onPrimary: _createQc),
              ..._qcs.take(8).map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['doc_no']} · ${m['status']}'),
                  trailing: (m['status']?.toString() == 'draft' || m['status']?.toString() == 'open')
                      ? Wrap(children: [
                          TextButton(onPressed: () => _completeQc(m, 'pass'), child: const Text('合格')),
                          TextButton(onPressed: () => _completeQc(m, 'fail'), child: const Text('不合格')),
                        ])
                      : Text('${m['result'] ?? ''}'),
                );
              }),
              const Divider(),
              const FormSectionHeader('废料'),
              FormRow.text(label: '废料重量/数量', controller: _scrapQty, keyboardType: TextInputType.number, requiredMark: true),
              FormStickyActions(primaryLabel: '登记废料', onPrimary: _createScrap),
              ..._scraps.take(5).map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(title: Text('${m['doc_no']}'), subtitle: Text('qty ${m['qty']} · ${m['status']}'));
              }),
              const Divider(),
              const FormSectionHeader('返修'),
              FormRow.text(label: '返修数量', controller: _reworkQty, keyboardType: TextInputType.number, requiredMark: true),
              FormStickyActions(primaryLabel: '新建返修', onPrimary: _createRework),
              ..._reworks.take(5).map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['doc_no']} · ${m['status']}'),
                  trailing: m['status']?.toString() != 'closed'
                      ? TextButton(onPressed: () => _closeRework(m), child: const Text('关闭'))
                      : null,
                );
              }),
            ],
          ),
          _list(_processes, (m) => '${m['name'] ?? m['code'] ?? m['id']}'),
          _list(_balances, (m) => '${m['product_name'] ?? m['product_id']} · ${m['qty']}'),
        ],
      ),
    );
  }
}
