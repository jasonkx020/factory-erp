import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';
import '../../widgets/trace_code_field.dart';

class WarehousePage extends StatefulWidget {
  const WarehousePage({super.key});

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
  Map<String, dynamic>? _active;
  Map<String, dynamic>? _scanHit;
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

  Future<void> _claim(Map row) async {
    final id = row['id'];
    final res = await context.read<AuthState>().api.post('/workflow/tasks/$id/claim', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(res.ok ? '已认领' : res.msg)));
    if (res.ok) await _loadTasks();
  }

  Future<void> _confirm() async {
    final row = _active;
    if (row == null) return;
    if (_verify.text.trim().isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请先扫码/输入溯源码定位单据')));
      return;
    }
    final bizId = row['weigh_ticket_id'] ?? row['biz_id'] ?? row['id'];
    if (bizId == null) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('未定位到过磅单')));
      return;
    }
    final res = await context.read<AuthState>().api.post(
      '/purchase/weigh-tickets/$bizId/warehouse-confirm',
      {'verified': true, 'match_confirmed': true},
    );
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(res.ok ? '入库完成' : res.msg)));
    if (res.ok) {
      setState(() {
        _active = null;
        _verify.clear();
        _scanHit = null;
      });
      await _loadTasks();
      await _loadBalances();
    }
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
    setState(() {
      _scanHit = m;
      _active = m;
    });
    if (m['stockin_ready'] != true) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('单据未就绪：${m['reason'] ?? m['status'] ?? ''}')),
      );
    }
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
    return Scaffold(
      appBar: AppBar(
        title: Text('仓管作业 · ${notify.mqttStatus}'),
        actions: [
          IconButton(
            onPressed: () async {
              await notify.refresh();
              if (_tab == 0) await _loadTasks();
              if (_tab == 1) await _loadBalances();
              if (_tab == 2) await _loadBoxes();
              if (_tab == 3) await _loadStocktakes();
              if (_tab == 4) await _loadTxns();
            },
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: _error != null
          ? Center(child: Text(_error!))
          :             IndexedStack(
              index: _tab,
              children: [
                _loading
                    ? const Center(child: CircularProgressIndicator())
                    : ListView(
                        padding: const EdgeInsets.all(12),
                        children: [
                          Text('扫码入库 · 待办 ${_tasks.length}', style: const TextStyle(color: Colors.black54)),
                          const SizedBox(height: 8),
                          TraceCodeField(
                            controller: _verify,
                            label: '扫溯源码定位',
                            hint: '扫码或输入溯源码/批号',
                            scannerTitle: '扫描溯源码',
                            textCapitalization: TextCapitalization.none,
                          ),
                          const SizedBox(height: 8),
                          Row(
                            children: [
                              FilledButton.tonal(onPressed: _scanLocate, child: const Text('定位单据')),
                              const SizedBox(width: 8),
                              FilledButton(
                                onPressed: (_active != null && _active!['stockin_ready'] == true) ? _confirm : null,
                                child: const Text('核对相符并入库'),
                              ),
                            ],
                          ),
                          if (_scanHit != null) ...[
                            const Divider(),
                            Text('单据 ${_scanHit!['doc_no'] ?? ''}', style: const TextStyle(fontWeight: FontWeight.bold)),
                            Text('批号 ${_scanHit!['batch_no'] ?? '-'} · 溯源 ${_scanHit!['trace_code'] ?? '-'}'),
                            Text(
                              '品种 ${_scanHit!['variety'] ?? _scanHit!['product_name'] ?? '-'} · '
                              '净重 ${_scanHit!['net_weight'] ?? '-'}kg · '
                              '扣损 ${_scanHit!['deduct_weight'] ?? '-'}',
                            ),
                            if (_scanHit!['stockin_ready'] != true)
                              Text('未就绪：${_scanHit!['reason'] ?? _scanHit!['status']}', style: const TextStyle(color: Colors.orange)),
                            Builder(builder: (_) {
                              final api = context.read<AuthState>().api;
                              final imgs = <String>[];
                              void add(dynamic v) {
                                final s = api.resolveMediaUrl(v?.toString() ?? '');
                                if (s.isNotEmpty && !imgs.contains(s)) imgs.add(s);
                              }
                              add(_scanHit!['image_url']);
                              for (final k in ['verify_images', 'site_photos', 'image_urls']) {
                                final raw = _scanHit![k];
                                if (raw is List) {
                                  for (final e in raw) {
                                    add(e);
                                  }
                                }
                              }
                              final evidences = _scanHit!['evidences'];
                              if (evidences is List) {
                                for (final e in evidences) {
                                  if (e is Map) add(e['file_url'] ?? e['url']);
                                }
                              }
                              if (imgs.isEmpty) return const SizedBox.shrink();
                              return Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  const SizedBox(height: 8),
                                  const Text('现场照片', style: TextStyle(fontWeight: FontWeight.w600)),
                                  const SizedBox(height: 6),
                                  SizedBox(
                                    height: 96,
                                    child: ListView.separated(
                                      scrollDirection: Axis.horizontal,
                                      itemCount: imgs.length,
                                      separatorBuilder: (_, __) => const SizedBox(width: 8),
                                      itemBuilder: (_, i) => ClipRRect(
                                        borderRadius: BorderRadius.circular(8),
                                        child: Image.network(
                                          imgs[i],
                                          width: 96,
                                          height: 96,
                                          fit: BoxFit.cover,
                                          errorBuilder: (_, __, ___) => Container(
                                            width: 96,
                                            height: 96,
                                            color: Colors.black12,
                                            child: const Icon(Icons.broken_image),
                                          ),
                                        ),
                                      ),
                                    ),
                                  ),
                                ],
                              );
                            }),
                          ],
                          const Divider(),
                          Text('推送待办', style: const TextStyle(fontWeight: FontWeight.w600)),
                          if (_tasks.isEmpty) const Padding(padding: EdgeInsets.all(16), child: Center(child: Text('暂无过磅推送待办'))),
                          ..._tasks.map((e) {
                            final t = Map<String, dynamic>.from(e as Map);
                            final p = NotifyService.parsePayload(t['payload'] ?? t['payload_json']);
                            return Card(
                              child: ListTile(
                                title: Text('${t['doc_no'] ?? ''}'),
                                subtitle: Text(
                                  '溯源 ${t['trace_code'] ?? p['trace_code'] ?? ''}\n'
                                  '净重 ${p['net_weight'] ?? '-'}kg · ${p['variety'] ?? p['product_name'] ?? ''}',
                                ),
                                isThreeLine: true,
                                trailing: Wrap(
                                  spacing: 4,
                                  children: [
                                    TextButton(onPressed: () => _claim(t), child: const Text('认领')),
                                    FilledButton(
                                      onPressed: () {
                                        final code = (t['trace_code'] ?? p['trace_code'] ?? '').toString();
                                        setState(() {
                                          _verify.text = code;
                                          _active = {
                                            ...t,
                                            ...p,
                                            'weigh_ticket_id': t['biz_id'],
                                            'stockin_ready': true,
                                          };
                                          _scanHit = _active;
                                        });
                                      },
                                      child: const Text('核对'),
                                    ),
                                  ],
                                ),
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
                  padding: const EdgeInsets.all(12),
                  children: [
                    TraceCodeField(
                      controller: _boxQuery,
                      label: '箱码查询/追溯',
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
                  padding: const EdgeInsets.all(16),
                  children: [
                    const Text('仓库盘点（扫码/录入实盘）', style: TextStyle(fontWeight: FontWeight.bold)),
                    DropdownButtonFormField<int>(
                      value: _stkWarehouseId,
                      decoration: const InputDecoration(labelText: '仓库'),
                      items: const [
                        DropdownMenuItem(value: 1, child: Text('原料仓(1)')),
                        DropdownMenuItem(value: 2, child: Text('半成品仓(2)')),
                        DropdownMenuItem(value: 3, child: Text('成品仓(3)')),
                      ],
                      onChanged: (v) => setState(() => _stkWarehouseId = v ?? 1),
                    ),
                    DropdownButtonFormField<int>(
                      value: _stkProductId,
                      decoration: const InputDecoration(labelText: '产品'),
                      items: _products.map((e) {
                        final m = Map<String, dynamic>.from(e as Map);
                        return DropdownMenuItem(value: (m['id'] as num?)?.toInt(), child: Text('${m['name'] ?? m['id']}'));
                      }).toList(),
                      onChanged: (v) => setState(() => _stkProductId = v),
                    ),
                    TextField(controller: _countQty, decoration: const InputDecoration(labelText: '实盘数量'), keyboardType: TextInputType.number),
                    FilledButton(onPressed: _createStocktake, child: const Text('新建盘点草稿')),
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
                  padding: const EdgeInsets.all(16),
                  children: [
                    const Text('扫码出入库', style: TextStyle(fontWeight: FontWeight.bold)),
                    const SizedBox(height: 8),
                    TraceCodeField(
                      controller: _txnScan,
                      label: '箱码/条码',
                      hint: '点击输入，或点右侧图标扫码',
                      scannerTitle: '扫描箱码',
                      textCapitalization: TextCapitalization.none,
                    ),
                    SegmentedButton<String>(
                      segments: const [
                        ButtonSegment(value: 'in', label: Text('入库')),
                        ButtonSegment(value: 'out', label: Text('出库')),
                      ],
                      selected: {_txnDirection},
                      onSelectionChanged: (s) => setState(() => _txnDirection = s.first),
                    ),
                    DropdownButtonFormField<int>(
                      value: _txnWarehouseId,
                      decoration: const InputDecoration(labelText: '仓库'),
                      items: const [
                        DropdownMenuItem(value: 1, child: Text('原料仓(1)')),
                        DropdownMenuItem(value: 2, child: Text('半成品仓(2)')),
                        DropdownMenuItem(value: 3, child: Text('成品仓(3)')),
                      ],
                      onChanged: (v) => setState(() => _txnWarehouseId = v ?? 1),
                    ),
                    DropdownButtonFormField<int>(
                      value: _txnProductId,
                      decoration: const InputDecoration(labelText: '产品'),
                      items: _products.map((e) {
                        final m = Map<String, dynamic>.from(e as Map);
                        return DropdownMenuItem(value: (m['id'] as num?)?.toInt(), child: Text('${m['name'] ?? m['id']}'));
                      }).toList(),
                      onChanged: (v) => setState(() => _txnProductId = v),
                    ),
                    TextField(controller: _txnQty, decoration: const InputDecoration(labelText: '数量'), keyboardType: TextInputType.number),
                    FilledButton(onPressed: _createTxn, child: const Text('建单')),
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
        destinations: const [
          NavigationDestination(icon: Icon(Icons.inbox), label: '待入库'),
          NavigationDestination(icon: Icon(Icons.inventory), label: '库存'),
          NavigationDestination(icon: Icon(Icons.qr_code_2), label: '箱码'),
          NavigationDestination(icon: Icon(Icons.checklist), label: '盘点'),
          NavigationDestination(icon: Icon(Icons.swap_horiz), label: '出入库'),
        ],
      ),
    );
  }
}
