import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';
import '../../widgets/trace_code_field.dart';
import '../../widgets/form_sticky_actions.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_row.dart';
import 'warehouse_verify_page.dart';
import '../workshop/process_return_page.dart';

class WarehousePage extends StatefulWidget {
  const WarehousePage({super.key, this.asTab = false});

  /// 作为产线壳 Tab 时隐藏标题栏。
  final bool asTab;

  @override
  State<WarehousePage> createState() => _WarehousePageState();
}

class _WarehousePageState extends State<WarehousePage> {
  int _tab = 0;
  List<dynamic> _tasks = [];
  List<dynamic> _balances = [];
  List<dynamic> _boxes = [];
  List<dynamic> _stocktakes = [];
  List<dynamic> _products = [];
  List<dynamic> _alertHits = [];
  List<dynamic> _txns = [];
  String? _error;
  bool _loading = false;
  final _verify = TextEditingController();
  final _boxQuery = TextEditingController();
  final _countQty = TextEditingController(text: '0');
  final _txnQty = TextEditingController(text: '1');
  final _txnScan = TextEditingController();
  int? _stkProductId;
  int _stkWarehouseId = 1;
  int? _txnProductId;
  int _txnWarehouseId = 1;
  String _txnDirection = 'in';
  Map<String, dynamic>? _boxTrace;
  NotifyService? _notify;

  String _taskReceiveKind(Map t) {
    final p = NotifyService.parsePayload(t['payload'] ?? t['payload_json']);
    final k = (t['receive_kind'] ?? p['receive_kind'] ?? 'gate').toString().toLowerCase();
    return k == 'stockin' ? 'stockin' : 'gate';
  }

  String _kindChipLabel(String kind) => kind == 'stockin' ? '入库' : '入厂';

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
    _boxQuery.dispose();
    _countQty.dispose();
    _txnQty.dispose();
    _txnScan.dispose();
    super.dispose();
  }

  void _onNotify() {
    if (_tab == 0) _loadTasks();
  }

  Future<void> _load() async {
    final auth = context.read<AuthState>();
    await auth.fetchMe();
    if (!canAccessEmployeeModule(EmployeeModule.warehouse, auth.permissions, auth.roles)) {
      setState(() => _error = '无仓管模块权限');
      return;
    }
    await Future.wait([_loadTasks(), _loadBalances()]);
  }

  Future<void> _loadTasks() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    final res = await context.read<AuthState>().api.get('/workflow/tasks?status=pending');
    if (!mounted) return;
    setState(() {
      _loading = false;
      if (!res.ok) {
        _error = res.msg;
        return;
      }
      final list = ApiClient.listOf(res.data);
      _tasks = list
          .where((t) => t is Map && (t['event_key'] == 'purchase.weigh_confirmed' || t['to_role'] == 'warehouse'))
          .toList();
    });
  }

  Future<void> _loadBalances() async {
    final res = await context.read<AuthState>().api.get('/inventory/balances?page_size=100');
    if (!mounted) return;
    if (res.ok) setState(() => _balances = ApiClient.listOf(res.data));
    await _loadAlerts();
  }

  Future<void> _loadAlerts() async {
    final api = context.read<AuthState>().api;
    final results = await Future.wait([
      api.get('/inventory/alert-rules/shortage'),
      api.get('/inventory/alert-rules/excess'),
    ]);
    if (!mounted) return;
    final hits = <dynamic>[];
    for (final r in results) {
      if (r.ok && r.data is Map) {
        final h = (r.data as Map)['hits'];
        if (h is List) hits.addAll(h);
      }
    }
    setState(() => _alertHits = hits);
  }

  Future<void> _loadTxns() async {
    final api = context.read<AuthState>().api;
    final results = await Future.wait([
      api.get('/inventory/stock-txns'),
      api.get('/product/products?page_size=100'),
    ]);
    if (!mounted) return;
    setState(() {
      _txns = ApiClient.listOf(results[0].data);
      _products = ApiClient.listOf(results[1].data);
      if (_txnProductId == null && _products.isNotEmpty) {
        _txnProductId = (_products.first as Map)['id'] is num ? ((_products.first as Map)['id'] as num).toInt() : null;
      }
    });
  }

  Future<void> _createTxn() async {
    if (_txnProductId == null) return;
    final api = context.read<AuthState>().api;
    final scan = _txnScan.text.trim();
    final r = await api.post('/inventory/stock-txns', {
      'doc_type': _txnDirection == 'in' ? 'inbound' : 'outbound',
      'warehouse_id': _txnWarehouseId,
      'remark': scan.isEmpty ? '手机扫码出入库' : '扫码:$scan',
      'lines': [
        {
          'product_id': _txnProductId,
          'qty': double.tryParse(_txnQty.text) ?? 1,
          'direction': _txnDirection,
        }
      ],
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '出入库草稿已建' : r.msg)));
    if (r.ok) {
      _txnScan.clear();
      await _loadTxns();
    }
  }

  Future<void> _postTxn(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/inventory/stock-txns/$id/post', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已过账' : r.msg)));
    if (r.ok) {
      await _loadTxns();
      await _loadBalances();
    }
  }

  Future<void> _loadBoxes() async {
    final res = await context.read<AuthState>().api.get('/inventory/box-codes?page_size=50');
    if (!mounted) return;
    if (res.ok) setState(() => _boxes = ApiClient.listOf(res.data));
  }

  Future<void> _loadStocktakes() async {
    final api = context.read<AuthState>().api;
    final results = await Future.wait([
      api.get('/inventory/stocktakes?page_size=30'),
      api.get('/product/products?page_size=100'),
    ]);
    if (!mounted) return;
    setState(() {
      _stocktakes = ApiClient.listOf(results[0].data);
      _products = ApiClient.listOf(results[1].data);
      if (_stkProductId == null && _products.isNotEmpty) {
        _stkProductId = (_products.first as Map)['id'] is num ? ((_products.first as Map)['id'] as num).toInt() : null;
      }
    });
  }

  Future<void> _createStocktake() async {
    if (_stkProductId == null) return;
    final api = context.read<AuthState>().api;
    final r = await api.post('/inventory/stocktakes', {
      'warehouse_id': _stkWarehouseId,
      'biz_date': DateTime.now().toIso8601String().substring(0, 10),
      'product_id': _stkProductId,
      'count_qty': double.tryParse(_countQty.text) ?? 0,
      'remark': '手机盘点',
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '盘点草稿已建' : r.msg)));
    if (r.ok) await _loadStocktakes();
  }

  Future<void> _submitStocktake(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final api = context.read<AuthState>().api;
    final r = await api.post('/inventory/stocktakes/$id/submit', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已提交' : r.msg)));
    if (r.ok) await _loadStocktakes();
  }

  Future<void> _postStocktake(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final api = context.read<AuthState>().api;
    final r = await api.post('/inventory/stocktakes/$id/post', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '盘点已过账' : r.msg)));
    if (r.ok) {
      await _loadStocktakes();
      await _loadBalances();
    }
  }

  Future<void> _openVerifyPage(Map<String, dynamic> ticket) async {
    final ok = await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => WarehouseVerifyPage(ticket: ticket)),
    );
    if (!mounted) return;
    if (ok == true) {
      _verify.clear();
      await _loadTasks();
      await _loadBalances();
    }
  }

  Future<void> _openVerifyFromTask(Map t) async {
    final p = NotifyService.parsePayload(t['payload'] ?? t['payload_json']);
    final code = (t['trace_code'] ?? p['trace_code'] ?? p['batch_no'] ?? '').toString().trim();
    if (code.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('待办无溯源码，请手动扫码定位')));
      return;
    }
    _verify.text = code;
    await _scanLocate();
  }

  Future<void> _scanLocate() async {
    final code = _verify.text.trim();
    if (code.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请扫码或输入溯源码')));
      return;
    }
    final res = await context.read<AuthState>().api.get(
      '/purchase/weigh-tickets/by-trace?code=${Uri.encodeComponent(code)}',
    );
    if (!mounted) return;
    if (!res.ok || res.data is! Map) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(res.msg)));
      return;
    }
    final m = Map<String, dynamic>.from(res.data as Map);
    await _openVerifyPage(m);
  }

  Future<void> _traceBox() async {
    final code = _boxQuery.text.trim();
    if (code.isEmpty) return;
    final api = context.read<AuthState>().api;
    final r = await api.get('/inventory/box-codes/trace/${Uri.encodeComponent(code)}');
    if (!mounted) return;
    setState(() {
      _boxTrace = r.ok && r.data is Map ? Map<String, dynamic>.from(r.data as Map) : {'error': r.msg};
    });
  }

  @override
  Widget build(BuildContext context) {
    final notify = context.watch<NotifyService>();
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;
    Future<void> refresh() async {
      await notify.refresh();
      if (_tab == 0) await _loadTasks();
      if (_tab == 1) await _loadBalances();
      if (_tab == 2) await _loadBoxes();
      if (_tab == 3) await _loadStocktakes();
      if (_tab == 4) await _loadTxns();
    }

    return Scaffold(
      appBar: widget.asTab
          ? null
          : AppBar(
              title: Text('仓管作业 · '),
              actions: [
                IconButton(onPressed: refresh, icon: const Icon(Icons.refresh)),
              ],
            ),
      body: Column(
        children: [
          if (widget.asTab)
            SafeArea(
              bottom: false,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(8, 4, 48, 0),
                child: Row(
                  children: [
                    Text('仓管 · ', style: const TextStyle(fontSize: 13, color: Colors.black54)),
                    const Spacer(),
                    IconButton(
                      tooltip: '刷新',
                      visualDensity: VisualDensity.compact,
                      onPressed: refresh,
                      icon: const Icon(Icons.refresh, size: 22),
                    ),
                  ],
                ),
              ),
            ),
          Expanded(
            child: _error != null
                ? Center(child: Text(_error!))
                : IndexedStack(
                    index: _tab,
                    children: [
                      _loading
                          ? const Center(child: CircularProgressIndicator())
                          : ListView(
                              keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
                              padding: EdgeInsets.fromLTRB(12, 12, 12, 12 + bottomInset),
                              children: [
                                const FormSectionHeader('溯源码搜索'),
                                TraceCodeField(
                                  controller: _verify,
                                  label: '溯源码',
                                  hint: '扫码或输入溯源码/批号后进入核对',
                                  scannerTitle: '扫描溯源码',
                                  compact: true,
                                  textCapitalization: TextCapitalization.none,
                                  onEditingComplete: _scanLocate,
                                  onScanned: (_) => _scanLocate(),
                                ),
                                const SizedBox(height: 8),
                                ListTile(
                                  contentPadding: EdgeInsets.zero,
                                  leading: const Icon(Icons.undo_outlined),
                                  title: const Text('工序退库确认'),
                                  subtitle: const Text('未用完领料还仓 · 不回冲计件'),
                                  trailing: const Icon(Icons.chevron_right),
                                  onTap: () {
                                    Navigator.of(context).push(
                                      MaterialPageRoute(
                                        builder: (_) => const ProcessReturnPage(warehouseMode: true),
                                      ),
                                    );
                                  },
                                ),
                                const SizedBox(height: 12),
                                const Text('待办列表', style: TextStyle(fontWeight: FontWeight.w600)),
                                if (_tasks.isEmpty)
                                  const Padding(
                                    padding: EdgeInsets.all(16),
                                    child: Center(child: Text('暂无待办')),
                                  ),
                                ..._tasks.map((e) {
                                  final t = Map<String, dynamic>.from(e as Map);
                                  final p = NotifyService.parsePayload(t['payload'] ?? t['payload_json']);
                                  final kind = _taskReceiveKind(t);
                                  final trace = (t['trace_code'] ?? p['trace_code'] ?? p['batch_no'] ?? '').toString();
                                  return Card(
                                    child: ListTile(
                                      title: Row(
                                        children: [
                                          Expanded(child: Text('${t['doc_no'] ?? ''}')),
                                          Chip(
                                            label: Text(
                                              _kindChipLabel(kind),
                                              style: const TextStyle(fontSize: 12, color: Colors.white),
                                            ),
                                            backgroundColor: kind == 'stockin' ? Colors.teal : Colors.indigo,
                                            visualDensity: VisualDensity.compact,
                                            materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                                            padding: EdgeInsets.zero,
                                            labelPadding: const EdgeInsets.symmetric(horizontal: 8),
                                          ),
                                        ],
                                      ),
                                      subtitle: Text(
                                        '溯源 $trace · 净重 ${p['net_weight'] ?? '-'}kg'
                                        '${(p['variety'] ?? p['product_name']) != null ? ' · ${p['variety'] ?? p['product_name']}' : ''}',
                                      ),
                                      trailing: const Icon(Icons.chevron_right),
                                      onTap: () => _openVerifyFromTask(t),
                                    ),
                                  );
                                }),
                              ],
                            ),
                ListView(
                  padding: const EdgeInsets.all(12),
                  children: [
                    const Text('亏料 / 过量预警', style: TextStyle(fontWeight: FontWeight.bold)),
                    const SizedBox(height: 8),
                    if (_alertHits.isEmpty) const Text('暂无命中预警', style: TextStyle(color: Colors.black54)),
                    ..._alertHits.map((e) {
                      final m = Map<String, dynamic>.from(e as Map);
                      final typ = m['alert_type']?.toString() ?? '';
                      return Card(
                        color: typ == 'shortage' ? Colors.orange.shade50 : Colors.red.shade50,
                        child: ListTile(
                          title: Text(typ == 'shortage' ? '亏料预警' : '过量预警'),
                          subtitle: Text('品 ${m['product_id']} · 仓 ${m['warehouse_id']} · 现存量 ${m['qty']}'),
                          trailing: Text(
                            typ == 'shortage' ? '下限 ${m['min_qty']}' : '上限 ${m['max_qty']}',
                            style: const TextStyle(fontSize: 12),
                          ),
                        ),
                      );
                    }),
                    const Divider(height: 24),
                    const Text('库存余额（授权仓）', style: TextStyle(fontWeight: FontWeight.bold)),
                    const SizedBox(height: 8),
                    if (_balances.isEmpty) const Text('暂无库存'),
                    ..._balances.map((e) {
                      final m = Map<String, dynamic>.from(e as Map);
                      return Card(
                        child: ListTile(
                          title: Text('${m['product_name'] ?? m['product_id'] ?? ''}'),
                          subtitle: Text('仓 ${m['warehouse_name'] ?? m['warehouse_id'] ?? ''} · 批 ${m['batch_no'] ?? '-'}'),
                          trailing: Text('${m['qty'] ?? 0}', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
                        ),
                      );
                    }),
                  ],
                ),
                ListView(
                  keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
                  padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
                  children: [
                    const FormSectionHeader('箱码查询'),
                    TraceCodeField(
                      controller: _boxQuery,
                      label: '箱码',
                      hint: '点击输入，或点右侧图标扫码',
                      scannerTitle: '扫描箱码',
                      textCapitalization: TextCapitalization.none,
                      onEditingComplete: _traceBox,
                      onScanned: (_) => _traceBox(),
                    ),
                    if (_boxTrace != null) ...[
                      const SizedBox(height: 8),
                      Card(
                        color: Colors.teal.shade50,
                        child: Padding(
                          padding: const EdgeInsets.all(12),
                          child: Text(_boxTrace!['error'] != null
                              ? '${_boxTrace!['error']}'
                              : '码 ${_boxTrace!['code'] ?? _boxQuery.text}\n'
                                  '品 ${_boxTrace!['product_name'] ?? _boxTrace!['product_id'] ?? ''}\n'
                                  '状态 ${_boxTrace!['status'] ?? ''}\n'
                                  '重量 ${_boxTrace!['weight'] ?? _boxTrace!['qty'] ?? ''}'),
                        ),
                      ),
                    ],
                    const Divider(),
                    const Text('最近箱码', style: TextStyle(fontWeight: FontWeight.bold)),
                    ..._boxes.map((e) {
                      final m = Map<String, dynamic>.from(e as Map);
                      return ListTile(
                        title: Text('${m['code']}'),
                        subtitle: Text('${m['status'] ?? ''} · ${m['product_name'] ?? m['product_id'] ?? ''}'),
                        trailing: Text('${m['weight'] ?? m['qty'] ?? ''}'),
                        onTap: () {
                          _boxQuery.text = m['code']?.toString() ?? '';
                          _traceBox();
                        },
                      );
                    }),
                  ],
                ),
                ListView(
                  keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
                  padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
                  children: [
                    const FormSectionHeader('仓库盘点'),
                    FormRow(
                      label: '仓库',
                      requiredMark: true,
                      child: DropdownButtonHideUnderline(
                        child: DropdownButton<int>(
                          isExpanded: true,
                          value: _stkWarehouseId,
                          alignment: Alignment.centerRight,
                          items: const [
                            DropdownMenuItem(value: 1, child: Text('原料仓(1)', textAlign: TextAlign.right)),
                            DropdownMenuItem(value: 2, child: Text('半成品仓(2)', textAlign: TextAlign.right)),
                            DropdownMenuItem(value: 3, child: Text('成品仓(3)', textAlign: TextAlign.right)),
                          ],
                          onChanged: (v) => setState(() => _stkWarehouseId = v ?? 1),
                        ),
                      ),
                    ),
                    FormRow(
                      label: '产品',
                      requiredMark: true,
                      child: DropdownButtonHideUnderline(
                        child: DropdownButton<int>(
                          isExpanded: true,
                          value: _stkProductId,
                          alignment: Alignment.centerRight,
                          hint: const Text('请选择', textAlign: TextAlign.right),
                          items: _products.map((e) {
                            final m = Map<String, dynamic>.from(e as Map);
                            return DropdownMenuItem(value: (m['id'] as num?)?.toInt(), child: Text('${m['name'] ?? m['id']}', textAlign: TextAlign.right));
                          }).toList(),
                          onChanged: (v) => setState(() => _stkProductId = v),
                        ),
                      ),
                    ),
                    FormRow.text(label: '实盘数量', controller: _countQty, keyboardType: TextInputType.number, requiredMark: true),
                    FormStickyActions(primaryLabel: '新建盘点草稿', onPrimary: _createStocktake),
                    const Divider(),
                    ..._stocktakes.map((e) {
                      final m = Map<String, dynamic>.from(e as Map);
                      final st = m['status']?.toString() ?? '';
                      return Card(
                        child: ListTile(
                          title: Text('${m['doc_no'] ?? m['id']}'),
                          subtitle: Text('$st · 仓${m['warehouse_id']}'),
                          trailing: Wrap(
                            children: [
                              if (st == 'draft') TextButton(onPressed: () => _submitStocktake(m), child: const Text('提交')),
                              if (st == 'submitted' || st == 'draft')
                                FilledButton.tonal(onPressed: () => _postStocktake(m), child: const Text('过账')),
                            ],
                          ),
                        ),
                      );
                    }),
                  ],
                ),
                ListView(
                  keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
                  padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
                  children: [
                    const FormSectionHeader('扫码出入库'),
                    TraceCodeField(
                      controller: _txnScan,
                      label: '箱码/条码',
                      hint: '点击输入，或点右侧图标扫码',
                      scannerTitle: '扫描箱码',
                      compact: true,
                      textCapitalization: TextCapitalization.none,
                    ),
                    FormRow(
                      label: '方向',
                      child: Align(
                        alignment: Alignment.centerRight,
                        child: SegmentedButton<String>(
                          segments: const [
                            ButtonSegment(value: 'in', label: Text('入库')),
                            ButtonSegment(value: 'out', label: Text('出库')),
                          ],
                          selected: {_txnDirection},
                          onSelectionChanged: (s) => setState(() => _txnDirection = s.first),
                        ),
                      ),
                    ),
                    FormRow(
                      label: '仓库',
                      requiredMark: true,
                      child: DropdownButtonHideUnderline(
                        child: DropdownButton<int>(
                          isExpanded: true,
                          value: _txnWarehouseId,
                          alignment: Alignment.centerRight,
                          items: const [
                            DropdownMenuItem(value: 1, child: Text('原料仓(1)', textAlign: TextAlign.right)),
                            DropdownMenuItem(value: 2, child: Text('半成品仓(2)', textAlign: TextAlign.right)),
                            DropdownMenuItem(value: 3, child: Text('成品仓(3)', textAlign: TextAlign.right)),
                          ],
                          onChanged: (v) => setState(() => _txnWarehouseId = v ?? 1),
                        ),
                      ),
                    ),
                    FormRow(
                      label: '产品',
                      requiredMark: true,
                      child: DropdownButtonHideUnderline(
                        child: DropdownButton<int>(
                          isExpanded: true,
                          value: _txnProductId,
                          alignment: Alignment.centerRight,
                          hint: const Text('请选择', textAlign: TextAlign.right),
                          items: _products.map((e) {
                            final m = Map<String, dynamic>.from(e as Map);
                            return DropdownMenuItem(value: (m['id'] as num?)?.toInt(), child: Text('${m['name'] ?? m['id']}', textAlign: TextAlign.right));
                          }).toList(),
                          onChanged: (v) => setState(() => _txnProductId = v),
                        ),
                      ),
                    ),
                    FormRow.text(label: '数量', controller: _txnQty, keyboardType: TextInputType.number, requiredMark: true),
                    FormStickyActions(primaryLabel: '建单', onPrimary: _createTxn),
                    const Divider(),
                    ..._txns.take(30).map((e) {
                      final m = Map<String, dynamic>.from(e as Map);
                      final st = m['status']?.toString() ?? '';
                      return Card(
                        child: ListTile(
                          title: Text('${m['doc_no'] ?? m['id']} · ${m['doc_type'] ?? m['txn_type'] ?? ''}'),
                          subtitle: Text('$st · 仓${m['warehouse_id']}'),
                          trailing: st == 'draft'
                              ? FilledButton.tonal(onPressed: () => _postTxn(m), child: const Text('过账'))
                              : Text(st),
                        ),
                      );
                    }),
                  ],
                ),
                    ],
                  ),
          ),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _tab,
        onDestinationSelected: (i) {
          setState(() => _tab = i);
          if (i == 0) _loadTasks();
          if (i == 1) _loadBalances();
          if (i == 2) _loadBoxes();
          if (i == 3) _loadStocktakes();
          if (i == 4) _loadTxns();
        },
        destinations: [
          NavigationDestination(
            icon: Badge(
              isLabelVisible: _tasks.isNotEmpty,
              label: Text('${_tasks.length}'),
              child: const Icon(Icons.inbox),
            ),
            label: '待办',
          ),
          const NavigationDestination(icon: Icon(Icons.inventory), label: '库存'),
          const NavigationDestination(icon: Icon(Icons.qr_code_2), label: '箱码'),
          const NavigationDestination(icon: Icon(Icons.checklist), label: '盘点'),
          const NavigationDestination(icon: Icon(Icons.swap_horiz), label: '出入库'),
        ],
      ),
    );
  }
}
