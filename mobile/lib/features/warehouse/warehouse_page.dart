import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';
import '../../widgets/form_row.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_sticky_actions.dart';
import '../../widgets/hub_entry_tile.dart';
import '../../widgets/trace_code_field.dart';
import '../workshop/process_return_page.dart';
import 'warehouse_verify_page.dart';

/// 仓管作业：Hub 入口 + Navigator.push 全屏子页（对齐收货/过站）。
enum WarehouseSection { home, scan, todos, boxes, stocktake, txn, stock }

class WarehousePage extends StatefulWidget {
  const WarehousePage({
    super.key,
    this.asTab = false,
    this.initialSection = WarehouseSection.home,
  });

  /// 作为产线壳 Tab 时仅展示 Hub 首页。
  final bool asTab;

  /// 非 home 时作为独立子页（Navigator.push），返回即销毁。
  final WarehouseSection initialSection;

  @override
  State<WarehousePage> createState() => _WarehousePageState();
}

class _WarehousePageState extends State<WarehousePage> {
  late WarehouseSection _section;
  List<dynamic> _tasks = [];
  List<dynamic> _balances = [];
  List<dynamic> _boxes = [];
  List<dynamic> _stocktakes = [];
  List<dynamic> _products = [];
  List<dynamic> _alertHits = [];
  List<dynamic> _txns = [];
  String? _error;
  bool _loading = false;
  bool _busy = false;
  String _msg = '';
  bool _msgIsError = false;

  /// 盘点/出入库：0 填表 · 1 预览
  int _formStep = 0;

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

  bool get _isSubPage => widget.initialSection != WarehouseSection.home;

  String get _pageTitle {
    switch (_section) {
      case WarehouseSection.home:
        return '仓管作业';
      case WarehouseSection.scan:
        return '扫溯源接收/分板';
      case WarehouseSection.todos:
        return '待办';
      case WarehouseSection.boxes:
        return '板码';
      case WarehouseSection.stocktake:
        return _formStep == 1 ? '盘点 · 预览' : '盘点';
      case WarehouseSection.txn:
        return _formStep == 1 ? '出入库 · 预览' : '出入库';
      case WarehouseSection.stock:
        return '库存查询';
    }
  }

  String _taskReceiveKind(Map t) {
    final p = NotifyService.parsePayload(t['payload'] ?? t['payload_json']);
    final k = (t['receive_kind'] ?? p['receive_kind'] ?? 'gate').toString().toLowerCase();
    return k == 'stockin' ? 'stockin' : 'gate';
  }

  String _kindChipLabel(String kind) => kind == 'stockin' ? '入库' : '入厂';

  String _productName(int? id) {
    if (id == null) return '-';
    for (final e in _products) {
      if (e is Map && (e['id'] as num?)?.toInt() == id) {
        return '${e['name'] ?? e['id']}';
      }
    }
    return '$id';
  }

  String _warehouseLabel(int id) {
    switch (id) {
      case 2:
        return '半成品仓(2)';
      case 3:
        return '成品仓(3)';
      default:
        return '原料仓(1)';
    }
  }

  @override
  void initState() {
    super.initState();
    _section = widget.initialSection;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _notify = context.read<NotifyService>();
      _notify!.addListener(_onNotify);
      _bootstrap();
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
    if (_section == WarehouseSection.todos || _section == WarehouseSection.home) {
      _loadTasks();
    }
  }

  Future<void> _bootstrap() async {
    final auth = context.read<AuthState>();
    await auth.fetchMe();
    if (!mounted) return;
    if (!canAccessEmployeeModule(EmployeeModule.warehouse, auth.permissions, auth.roles)) {
      setState(() => _error = '无仓管模块权限');
      return;
    }
    await _refreshSection();
  }

  Future<void> _refreshSection() async {
    switch (_section) {
      case WarehouseSection.home:
        await Future.wait([_loadTasks(), _loadBalances()]);
      case WarehouseSection.scan:
      case WarehouseSection.todos:
        await _loadTasks();
      case WarehouseSection.boxes:
        await _loadBoxes();
      case WarehouseSection.stocktake:
        await _loadStocktakes();
      case WarehouseSection.txn:
        await _loadTxns();
      case WarehouseSection.stock:
        await _loadBalances();
    }
  }

  void _prompt(String msg, {bool error = true}) {
    if (!mounted) return;
    setState(() {
      _msg = msg;
      _msgIsError = error;
    });
    final messenger = ScaffoldMessenger.of(context);
    messenger.clearSnackBars();
    if (error) {
      final scheme = Theme.of(context).colorScheme;
      messenger.showSnackBar(
        SnackBar(
          content: Text(msg, style: TextStyle(color: scheme.onError)),
          backgroundColor: scheme.error,
          behavior: SnackBarBehavior.floating,
        ),
      );
    } else {
      messenger.showSnackBar(SnackBar(content: Text(msg)));
    }
  }

  Future<void> _openSection(WarehouseSection section) async {
    if (section == WarehouseSection.home) return;
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => WarehousePage(initialSection: section),
      ),
    );
    if (!mounted) return;
    await _loadTasks();
  }

  Future<void> _openProcessReturn() async {
    await Navigator.of(context).push(
      MaterialPageRoute(builder: (_) => const ProcessReturnPage(warehouseMode: true)),
    );
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

  Future<void> _openVerifyPage(Map<String, dynamic> ticket) async {
    final ok = await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => WarehouseVerifyPage(ticket: ticket)),
    );
    if (!mounted) return;
    if (ok == true) {
      _verify.clear();
      setState(() {
        _msg = '已处理完成';
        _msgIsError = false;
      });
      await _loadTasks();
      await _loadBalances();
    }
  }

  Future<void> _openVerifyFromTask(Map t) async {
    final p = NotifyService.parsePayload(t['payload'] ?? t['payload_json']);
    final code = (t['trace_code'] ?? p['trace_code'] ?? p['batch_no'] ?? '').toString().trim();
    if (code.isEmpty) {
      _prompt('待办无溯源码，请到「扫溯源」手动定位');
      return;
    }
    _verify.text = code;
    await _scanLocate();
  }

  Future<void> _scanLocate() async {
    final code = _verify.text.trim();
    if (code.isEmpty) {
      _prompt('请扫码或输入溯源码');
      return;
    }
    setState(() => _busy = true);
    final res = await context.read<AuthState>().api.get(
          '/purchase/weigh-tickets/by-trace?code=${Uri.encodeComponent(code)}',
        );
    if (!mounted) return;
    setState(() => _busy = false);
    if (!res.ok || res.data is! Map) {
      _prompt(res.msg);
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

  Future<void> _resolveTxnScan() async {
    final code = _txnScan.text.trim();
    if (code.isEmpty) return;
    final r = await context.read<AuthState>().api.get(
          '/inventory/box-codes/trace/${Uri.encodeComponent(code)}',
        );
    if (!mounted || !r.ok || r.data is! Map) return;
    final m = Map<String, dynamic>.from(r.data as Map);
    final pid = (m['product_id'] as num?)?.toInt();
    if (pid != null && pid > 0) {
      setState(() => _txnProductId = pid);
      final w = m['weight'] ?? m['qty'];
      if (w != null && (double.tryParse(_txnQty.text) ?? 0) <= 0) {
        _txnQty.text = '$w';
      }
      _prompt('已带出产品 ${_productName(pid)}', error: false);
    }
  }

  Future<void> _destroyBox(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null || id <= 0) return;
    final st = (m['status'] ?? '').toString().toLowerCase();
    if (st == 'destroyed' || st == 'finished') {
      _prompt('该板不可销毁');
      return;
    }
    final reasonCtrl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('销毁板码 ${m['code'] ?? ''}'),
        content: TextField(
          controller: reasonCtrl,
          decoration: const InputDecoration(
            labelText: '销毁原因',
            hintText: '如：仓前损耗用不了',
            border: OutlineInputBorder(),
          ),
          maxLines: 2,
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('确认销毁')),
        ],
      ),
    );
    final reason = reasonCtrl.text.trim();
    reasonCtrl.dispose();
    if (ok != true || !mounted) return;
    if (reason.isEmpty) {
      _prompt('请填写销毁原因');
      return;
    }
    final r = await context.read<AuthState>().api.post('/inventory/box-codes/$id/destroy', {'reason': reason});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已销毁' : r.msg)));
    if (r.ok) {
      await _loadBoxes();
      if (_boxTrace != null &&
          (_boxTrace!['code']?.toString() == m['code']?.toString() || _boxTrace!['id'] == id)) {
        setState(() => _boxTrace = null);
      }
    }
  }

  String? _validateStocktake() {
    if (_stkProductId == null) return '请选择产品';
    final q = double.tryParse(_countQty.text) ?? -1;
    if (q < 0) return '请填写实盘数量';
    return null;
  }

  String? _validateTxn() {
    if (_txnProductId == null) return '请选择产品';
    final q = double.tryParse(_txnQty.text) ?? 0;
    if (q <= 0) return '请填写数量';
    return null;
  }

  Future<void> _submitStocktakeDirect() async {
    final err = _validateStocktake();
    if (err != null) {
      _prompt(err);
      return;
    }
    setState(() => _busy = true);
    final api = context.read<AuthState>().api;
    final create = await api.post('/inventory/stocktakes', {
      'warehouse_id': _stkWarehouseId,
      'biz_date': DateTime.now().toIso8601String().substring(0, 10),
      'product_id': _stkProductId,
      'count_qty': double.tryParse(_countQty.text) ?? 0,
      'remark': '手机盘点',
    });
    if (!mounted) return;
    if (!create.ok) {
      setState(() => _busy = false);
      _prompt(create.msg);
      return;
    }
    final id = create.data is Map ? (create.data as Map)['id'] as num? : null;
    if (id == null) {
      setState(() => _busy = false);
      _prompt('建单成功但未返回单据号');
      await _loadStocktakes();
      return;
    }
    final submit = await api.post('/inventory/stocktakes/${id.toInt()}/submit', {});
    if (!mounted) return;
    if (!submit.ok) {
      setState(() => _busy = false);
      _prompt('已建草稿，提交失败：${submit.msg}');
      await _loadStocktakes();
      return;
    }
    final post = await api.post('/inventory/stocktakes/${id.toInt()}/post', {});
    if (!mounted) return;
    setState(() => _busy = false);
    if (!post.ok) {
      _prompt('已提交，过账失败：${post.msg}');
      await _loadStocktakes();
      return;
    }
    _countQty.text = '0';
    setState(() {
      _formStep = 0;
      _msg = '盘点已过账';
      _msgIsError = false;
    });
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('盘点已过账')));
    await _loadStocktakes();
    await _loadBalances();
  }

  Future<void> _submitTxnDirect() async {
    final err = _validateTxn();
    if (err != null) {
      _prompt(err);
      return;
    }
    setState(() => _busy = true);
    final api = context.read<AuthState>().api;
    final scan = _txnScan.text.trim();
    final create = await api.post('/inventory/stock-txns', {
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
    if (!create.ok) {
      setState(() => _busy = false);
      _prompt(create.msg);
      return;
    }
    final id = create.data is Map ? (create.data as Map)['id'] as num? : null;
    if (id == null) {
      setState(() => _busy = false);
      _prompt('建单成功但未返回单据号');
      await _loadTxns();
      return;
    }
    final post = await api.post('/inventory/stock-txns/${id.toInt()}/post', {});
    if (!mounted) return;
    setState(() => _busy = false);
    if (!post.ok) {
      _prompt('已建草稿，过账失败：${post.msg}');
      await _loadTxns();
      return;
    }
    _txnScan.clear();
    _txnQty.text = '1';
    setState(() {
      _formStep = 0;
      _msg = '出入库已过账';
      _msgIsError = false;
    });
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('出入库已过账')));
    await _loadTxns();
    await _loadBalances();
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

  Future<void> _postStocktake(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final api = context.read<AuthState>().api;
    final st = (m['status'] ?? '').toString();
    if (st == 'draft') {
      final submit = await api.post('/inventory/stocktakes/$id/submit', {});
      if (!mounted) return;
      if (!submit.ok) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(submit.msg)));
        return;
      }
    }
    final r = await api.post('/inventory/stocktakes/$id/post', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '盘点已过账' : r.msg)));
    if (r.ok) {
      await _loadStocktakes();
      await _loadBalances();
    }
  }

  Widget _previewRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          SizedBox(
            width: 108,
            child: Text(label, style: TextStyle(fontSize: 13, color: Colors.black.withValues(alpha: 0.6))),
          ),
          Expanded(child: Text(value, textAlign: TextAlign.right, style: const TextStyle(fontSize: 14))),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final showAppBar = _isSubPage || !widget.asTab;
    return Scaffold(
      appBar: showAppBar
          ? AppBar(
              title: Text(_pageTitle),
              actions: [
                IconButton(
                  tooltip: '刷新',
                  onPressed: () async {
                    await context.read<NotifyService>().refresh();
                    await _refreshSection();
                  },
                  icon: const Icon(Icons.refresh),
                ),
              ],
            )
          : AppBar(toolbarHeight: 0),
      body: _error != null
          ? Center(child: Text(_error!))
          : switch (_section) {
              WarehouseSection.home => _buildHome(),
              WarehouseSection.scan => _buildScan(),
              WarehouseSection.todos => _buildTodos(),
              WarehouseSection.boxes => _buildBoxes(),
              WarehouseSection.stocktake => _buildStocktake(),
              WarehouseSection.txn => _buildTxn(),
              WarehouseSection.stock => _buildStock(),
            },
    );
  }

  Widget _buildHome() {
    return ListView(
      padding: EdgeInsets.fromLTRB(16, widget.asTab ? 40 : 16, 16, 16),
      children: [
        const Text('仓管作业', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
        const SizedBox(height: 4),
        const Text('选择业务入口；完成后回到本页', style: TextStyle(fontSize: 13, color: Colors.black54)),
        if (_msg.isNotEmpty) ...[
          const SizedBox(height: 12),
          Text(
            _msg,
            style: TextStyle(
              fontSize: 13,
              color: _msgIsError ? Theme.of(context).colorScheme.error : Colors.teal,
              fontWeight: _msgIsError ? FontWeight.w600 : FontWeight.normal,
            ),
          ),
        ],
        const SizedBox(height: 16),
        const Text('常用', style: TextStyle(fontWeight: FontWeight.bold)),
        const SizedBox(height: 8),
        HubEntryTile(
          icon: Icons.qr_code_scanner,
          title: '扫溯源接收/分板',
          subtitle: '扫或手输溯源码，入厂接收或分板入库',
          onTap: () => _openSection(WarehouseSection.scan),
        ),
        HubEntryTile(
          icon: Icons.inbox_outlined,
          title: '待办',
          subtitle: _tasks.isEmpty ? '查看仓管待办任务' : '待处理 ${_tasks.length} 条',
          onTap: () => _openSection(WarehouseSection.todos),
        ),
        HubEntryTile(
          icon: Icons.qr_code_2,
          title: '板码',
          subtitle: '查询板码、销毁未用板',
          onTap: () => _openSection(WarehouseSection.boxes),
        ),
        HubEntryTile(
          icon: Icons.checklist,
          title: '盘点',
          subtitle: '填表预览后一次过账',
          onTap: () => _openSection(WarehouseSection.stocktake),
        ),
        HubEntryTile(
          icon: Icons.swap_horiz,
          title: '出入库',
          subtitle: '扫板码或手选产品，预览后过账',
          onTap: () => _openSection(WarehouseSection.txn),
        ),
        HubEntryTile(
          icon: Icons.undo_outlined,
          title: '工序退库',
          subtitle: '未用完领料还仓 · 不回冲计件',
          onTap: _openProcessReturn,
        ),
        HubEntryTile(
          icon: Icons.inventory_2_outlined,
          title: '库存查询',
          subtitle: '余额与亏料/过量预警',
          onTap: () => _openSection(WarehouseSection.stock),
        ),
      ],
    );
  }

  Widget _msgBar() {
    if (_msg.isEmpty) return const SizedBox.shrink();
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 4),
      child: Text(
        _msg,
        style: TextStyle(
          fontSize: 13,
          color: _msgIsError ? Theme.of(context).colorScheme.error : Colors.teal,
          fontWeight: _msgIsError ? FontWeight.w600 : FontWeight.normal,
        ),
      ),
    );
  }

  Widget _buildScan() {
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;
    return Column(
      children: [
        Expanded(
          child: ListView(
            keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
            padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + bottomInset),
            children: [
              const FormSectionHeader('溯源码'),
              TraceCodeField(
                controller: _verify,
                label: '溯源码',
                hint: '手输或扫码',
                scannerTitle: '扫描溯源码',
                textCapitalization: TextCapitalization.none,
                onEditingComplete: _scanLocate,
                onScanned: (_) => _scanLocate(),
              ),
              const SizedBox(height: 8),
              const Text(
                '定位到过磅单后进入核对页：待入厂则接收；已入厂则分板入库。',
                style: TextStyle(fontSize: 12, color: Colors.black54),
              ),
            ],
          ),
        ),
        _msgBar(),
        FormStickyActions(
          primaryLabel: _busy ? '定位中…' : '定位并核对',
          onPrimary: _busy ? null : _scanLocate,
          primaryBusy: _busy,
        ),
      ],
    );
  }

  Widget _buildTodos() {
    if (_loading) return const Center(child: CircularProgressIndicator());
    return RefreshIndicator(
      onRefresh: _loadTasks,
      child: ListView(
        padding: const EdgeInsets.all(12),
        children: [
          if (_tasks.isEmpty)
            const Padding(
              padding: EdgeInsets.all(24),
              child: Center(child: Text('暂无待办', style: TextStyle(color: Colors.black54))),
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
    );
  }

  Widget _buildBoxes() {
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;
    return ListView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + bottomInset),
      children: [
        const FormSectionHeader('板码查询'),
        TraceCodeField(
          controller: _boxQuery,
          label: '板码',
          hint: '手输或扫码',
          scannerTitle: '扫描板码',
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
              child: Text(
                _boxTrace!['error'] != null
                    ? '${_boxTrace!['error']}'
                    : '码 ${_boxTrace!['code'] ?? _boxQuery.text}\n'
                        '品 ${_boxTrace!['product_name'] ?? _boxTrace!['product_id'] ?? ''}\n'
                        '状态 ${_boxTrace!['status'] ?? ''}\n'
                        '重量 ${_boxTrace!['weight'] ?? _boxTrace!['qty'] ?? ''}',
              ),
            ),
          ),
        ],
        const Divider(),
        const Text('最近板码', style: TextStyle(fontWeight: FontWeight.bold)),
        ..._boxes.map((e) {
          final m = Map<String, dynamic>.from(e as Map);
          final st = (m['status'] ?? '').toString().toLowerCase();
          final canDestroy = st != 'destroyed' && st != 'finished';
          return ListTile(
            title: Text('${m['code']}'),
            subtitle: Text(
              '${m['status'] ?? ''} · ${m['trace_code'] ?? '-'} · ${m['product_name'] ?? m['product_id'] ?? ''}',
            ),
            trailing: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('${m['weight'] ?? m['qty'] ?? ''}'),
                if (canDestroy)
                  IconButton(
                    tooltip: '销毁',
                    icon: const Icon(Icons.delete_forever_outlined, color: Colors.redAccent),
                    onPressed: () => _destroyBox(m),
                  ),
              ],
            ),
            onTap: () {
              _boxQuery.text = m['code']?.toString() ?? '';
              _traceBox();
            },
          );
        }),
      ],
    );
  }

  Widget _buildStock() {
    return RefreshIndicator(
      onRefresh: _loadBalances,
      child: ListView(
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
    );
  }

  Widget _warehouseDropdown({required int value, required ValueChanged<int> onChanged}) {
    return FormRow(
      label: '仓库',
      requiredMark: true,
      child: DropdownButtonHideUnderline(
        child: DropdownButton<int>(
          isExpanded: true,
          value: value,
          alignment: Alignment.centerRight,
          items: const [
            DropdownMenuItem(value: 1, child: Text('原料仓(1)', textAlign: TextAlign.right)),
            DropdownMenuItem(value: 2, child: Text('半成品仓(2)', textAlign: TextAlign.right)),
            DropdownMenuItem(value: 3, child: Text('成品仓(3)', textAlign: TextAlign.right)),
          ],
          onChanged: (v) => onChanged(v ?? 1),
        ),
      ),
    );
  }

  Widget _productDropdown({required int? value, required ValueChanged<int?> onChanged}) {
    return FormRow(
      label: '产品',
      requiredMark: true,
      child: DropdownButtonHideUnderline(
        child: DropdownButton<int>(
          isExpanded: true,
          value: value,
          alignment: Alignment.centerRight,
          hint: const Text('请选择', textAlign: TextAlign.right),
          items: _products.map((e) {
            final m = Map<String, dynamic>.from(e as Map);
            return DropdownMenuItem(
              value: (m['id'] as num?)?.toInt(),
              child: Text('${m['name'] ?? m['id']}', textAlign: TextAlign.right),
            );
          }).toList(),
          onChanged: onChanged,
        ),
      ),
    );
  }

  Widget _buildStocktake() {
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;
    return Column(
      children: [
        Expanded(
          child: _formStep == 0
              ? ListView(
                  keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
                  padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + bottomInset),
                  children: [
                    const FormSectionHeader('新建盘点'),
                    _warehouseDropdown(
                      value: _stkWarehouseId,
                      onChanged: (v) => setState(() => _stkWarehouseId = v),
                    ),
                    _productDropdown(
                      value: _stkProductId,
                      onChanged: (v) => setState(() => _stkProductId = v),
                    ),
                    FormRow.text(
                      label: '实盘数量',
                      controller: _countQty,
                      keyboardType: TextInputType.number,
                      requiredMark: true,
                    ),
                    const Divider(height: 24),
                    const Text('近期盘点（异常可补过账）', style: TextStyle(fontWeight: FontWeight.w600)),
                    const SizedBox(height: 8),
                    ..._stocktakes.map((e) {
                      final m = Map<String, dynamic>.from(e as Map);
                      final st = m['status']?.toString() ?? '';
                      return Card(
                        child: ListTile(
                          title: Text('${m['doc_no'] ?? m['id']}'),
                          subtitle: Text('$st · 仓${m['warehouse_id']}'),
                          trailing: (st == 'draft' || st == 'submitted')
                              ? FilledButton.tonal(onPressed: () => _postStocktake(m), child: const Text('过账'))
                              : Text(st),
                        ),
                      );
                    }),
                  ],
                )
              : ListView(
                  padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + bottomInset),
                  children: [
                    const Text('盘点预览', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
                    const Text('请核对以下信息，确认后一次过账', style: TextStyle(fontSize: 12, color: Colors.black54)),
                    const SizedBox(height: 8),
                    _previewRow('仓库', _warehouseLabel(_stkWarehouseId)),
                    _previewRow('产品', _productName(_stkProductId)),
                    _previewRow('实盘数量', _countQty.text.trim()),
                  ],
                ),
        ),
        _msgBar(),
        if (_formStep == 0)
          FormStickyActions(
            primaryLabel: '下一步',
            onPrimary: () {
              final err = _validateStocktake();
              if (err != null) {
                _prompt(err);
                return;
              }
              setState(() {
                _formStep = 1;
                _msg = '';
                _msgIsError = false;
              });
            },
          )
        else
          FormStickyActions(
            secondaryLabel: '修改',
            onSecondary: _busy
                ? null
                : () => setState(() {
                      _formStep = 0;
                      _msg = '';
                    }),
            primaryLabel: '确认盘点并过账',
            onPrimary: _busy ? null : _submitStocktakeDirect,
            primaryBusy: _busy,
            busyLabel: '提交中…',
          ),
      ],
    );
  }

  Widget _buildTxn() {
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;
    return Column(
      children: [
        Expanded(
          child: _formStep == 0
              ? ListView(
                  keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
                  padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + bottomInset),
                  children: [
                    const FormSectionHeader('新建出入库'),
                    TraceCodeField(
                      controller: _txnScan,
                      label: '板码',
                      hint: '手输或扫码，可带出产品',
                      scannerTitle: '扫描板码',
                      textCapitalization: TextCapitalization.none,
                      onEditingComplete: _resolveTxnScan,
                      onScanned: (_) => _resolveTxnScan(),
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
                    _warehouseDropdown(
                      value: _txnWarehouseId,
                      onChanged: (v) => setState(() => _txnWarehouseId = v),
                    ),
                    _productDropdown(
                      value: _txnProductId,
                      onChanged: (v) => setState(() => _txnProductId = v),
                    ),
                    FormRow.text(
                      label: '数量',
                      controller: _txnQty,
                      keyboardType: TextInputType.number,
                      requiredMark: true,
                    ),
                    const Divider(height: 24),
                    const Text('近期单据（草稿可补过账）', style: TextStyle(fontWeight: FontWeight.w600)),
                    const SizedBox(height: 8),
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
                )
              : ListView(
                  padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + bottomInset),
                  children: [
                    const Text('出入库预览', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
                    const Text('请核对以下信息，确认后一次过账', style: TextStyle(fontSize: 12, color: Colors.black54)),
                    const SizedBox(height: 8),
                    _previewRow('方向', _txnDirection == 'in' ? '入库' : '出库'),
                    _previewRow('板码', _txnScan.text.trim().isEmpty ? '-' : _txnScan.text.trim()),
                    _previewRow('仓库', _warehouseLabel(_txnWarehouseId)),
                    _previewRow('产品', _productName(_txnProductId)),
                    _previewRow('数量', _txnQty.text.trim()),
                  ],
                ),
        ),
        _msgBar(),
        if (_formStep == 0)
          FormStickyActions(
            primaryLabel: '下一步',
            onPrimary: () {
              final err = _validateTxn();
              if (err != null) {
                _prompt(err);
                return;
              }
              setState(() {
                _formStep = 1;
                _msg = '';
                _msgIsError = false;
              });
            },
          )
        else
          FormStickyActions(
            secondaryLabel: '修改',
            onSecondary: _busy
                ? null
                : () => setState(() {
                      _formStep = 0;
                      _msg = '';
                    }),
            primaryLabel: '确认并过账',
            onPrimary: _busy ? null : _submitTxnDirect,
            primaryBusy: _busy,
            busyLabel: '提交中…',
          ),
      ],
    );
  }
}
