import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../widgets/form_sticky_actions.dart';

/// 仓管核对页：入厂接收；已入厂则扫溯源分箱入库。
class WarehouseVerifyPage extends StatefulWidget {
  const WarehouseVerifyPage({super.key, required this.ticket});

  final Map<String, dynamic> ticket;

  @override
  State<WarehouseVerifyPage> createState() => _WarehouseVerifyPageState();
}

class _WarehouseVerifyPageState extends State<WarehouseVerifyPage> {
  late Map<String, dynamic> _ticket;
  bool _busy = false;
  List<dynamic> _purchaseUsers = [];
  List<Map<String, dynamic>> _products = [];
  int? _productId;
  final List<TextEditingController> _boxCtrls = [];

  static const _errLabel = {
    'BOXES_REQUIRED': '请至少录入一箱复磅重量',
    'BOX_WEIGHT_REQUIRED': '每箱复磅重量须大于 0',
    'PRODUCT_REQUIRED': '请选择入库产品',
    'ROUTING_REQUIRED': '该产品未配置入厂工艺，请先在工艺流程绑定',
    'WEIGHT_MISMATCH': '分箱合计超过票净重过多（允许超重 ±3% 或 5kg）',
    'TRACE_CODE_REQUIRED': '缺少溯源码，无法建箱',
    'VERIFY_REQUIRED': '请确认核对相符',
    'GATE_ACCEPT_REQUIRED': '请先完成入厂接收后再分箱',
    'USE_WAREHOUSE_BOX_STOCKIN': '请仓管扫溯源分箱入库',
  };

  @override
  void initState() {
    super.initState();
    _ticket = Map<String, dynamic>.from(widget.ticket);
    final pid = (_ticket['product_id'] as num?)?.toInt() ?? 0;
    _productId = pid > 0 ? pid : null;
    _boxCtrls.add(TextEditingController());
    if (_isBoxMode) {
      _loadProducts();
    }
  }

  @override
  void dispose() {
    for (final c in _boxCtrls) {
      c.dispose();
    }
    super.dispose();
  }

  String get _status => (_ticket['status'] ?? '').toString().toLowerCase();

  /// 分箱入库：已入厂待分箱
  bool get _isBoxMode =>
      _ticket['box_stockin_ready'] == true || _status == 'gate_accepted';

  String get _kindLabel => _isBoxMode ? '分箱入库' : '入厂接收';

  String get _confirmLabel => _isBoxMode ? '确认分箱入库' : '确认入厂接收';

  bool get _ready =>
      _isBoxMode || _ticket['stockin_ready'] == true;

  Object? get _bizId => _ticket['weigh_ticket_id'] ?? _ticket['biz_id'] ?? _ticket['id'];

  double get _ticketNet => (_ticket['net_weight'] as num?)?.toDouble() ?? 0;

  double get _boxSum {
    var s = 0.0;
    for (final c in _boxCtrls) {
      s += double.tryParse(c.text.trim()) ?? 0;
    }
    return s;
  }

  double get _inboundLoss {
    final loss = _ticketNet - _boxSum;
    return loss > 0 ? loss : 0;
  }

  double get _lossRate {
    if (_ticketNet <= 0 || _inboundLoss <= 0) return 0;
    return _inboundLoss / _ticketNet * 100;
  }

  bool get _weightOk {
    if (!_isBoxMode) return true;
    final net = _ticketNet;
    if (_boxSum <= 0) return false;
    if (net <= 0) return true;
    if (_boxSum <= net) return true;
    final diff = _boxSum - net;
    var tol = net * 0.03;
    if (tol < 5) tol = 5;
    return diff <= tol;
  }

  bool get _canConfirm {
    if (_busy || !_ready) return false;
    if (!_isBoxMode) return true;
    if (_productId == null || _productId! <= 0) return false;
    if (_boxCtrls.isEmpty) return false;
    for (final c in _boxCtrls) {
      final w = double.tryParse(c.text.trim()) ?? 0;
      if (w <= 0) return false;
    }
    return _weightOk;
  }

  Future<void> _loadProducts() async {
    final res = await context.read<AuthState>().api.get('/product/products?page_size=200');
    if (!mounted || !res.ok) return;
    final list = ApiClient.listOf(res.data);
    final maps = <Map<String, dynamic>>[];
    for (final e in list) {
      if (e is Map) maps.add(Map<String, dynamic>.from(e));
    }
    setState(() {
      _products = maps;
      if (_productId == null && maps.isNotEmpty) {
        _productId = (maps.first['id'] as num?)?.toInt();
      }
    });
  }

  List<String> _photoUrls() {
    final api = context.read<AuthState>().api;
    final imgs = <String>[];
    void add(dynamic v) {
      final s = api.resolveMediaUrl(v?.toString() ?? '');
      if (s.isNotEmpty && !imgs.contains(s)) imgs.add(s);
    }

    add(_ticket['image_url']);
    for (final k in ['verify_images', 'site_photos', 'image_urls']) {
      final raw = _ticket[k];
      if (raw is List) {
        for (final e in raw) {
          add(e);
        }
      }
    }
    final evidences = _ticket['evidences'];
    if (evidences is List) {
      for (final e in evidences) {
        if (e is Map) add(e['file_url'] ?? e['url']);
      }
    }
    return imgs;
  }

  void _addBox() {
    setState(() => _boxCtrls.add(TextEditingController()));
  }

  void _removeBox(int i) {
    if (_boxCtrls.length <= 1) return;
    setState(() {
      _boxCtrls[i].dispose();
      _boxCtrls.removeAt(i);
    });
  }

  Future<void> _confirm() async {
    final bizId = _bizId;
    if (bizId == null) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('未定位到过磅单')));
      return;
    }
    if (!_ready) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('单据未就绪：${_ticket['reason'] ?? _ticket['status'] ?? ''}')),
      );
      return;
    }

    setState(() => _busy = true);
    late final dynamic res;
    if (_isBoxMode) {
      if (_productId == null || _productId! <= 0) {
        setState(() => _busy = false);
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请选择入库产品')));
        return;
      }
      final boxes = <Map<String, dynamic>>[];
      for (final c in _boxCtrls) {
        final w = double.tryParse(c.text.trim()) ?? 0;
        if (w <= 0) {
          setState(() => _busy = false);
          ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('每箱复磅重量须大于 0')));
          return;
        }
        boxes.add({'weight': w});
      }
      if (!_weightOk) {
        setState(() => _busy = false);
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('分箱合计超过票净重过多（允许超重 ±3% 或 5kg）')),
        );
        return;
      }
      res = await context.read<AuthState>().api.post(
        '/purchase/weigh-tickets/$bizId/box-stock-in',
        {'product_id': _productId, 'boxes': boxes},
      );
    } else {
      res = await context.read<AuthState>().api.post(
        '/purchase/weigh-tickets/$bizId/warehouse-confirm',
        {'verified': true, 'match_confirmed': true},
      );
    }
    if (!mounted) return;
    setState(() => _busy = false);

    var okMsg = _isBoxMode ? '分箱入库完成' : '入厂接收完成';
    if (res.ok && res.data is Map) {
      final data = res.data as Map;
      final loss = data['inbound_loss_kg'];
      final rate = data['inbound_loss_rate'];
      if (loss is num && loss > 0) {
        final pct = rate is num ? (rate * 100) : (loss / (_ticketNet > 0 ? _ticketNet : 1) * 100);
        okMsg = '分箱入库完成（仓前损耗 ${loss.toStringAsFixed(2)} kg，扣损率 ${pct.toStringAsFixed(1)}%）';
      }
      if (data['settlement_id'] != null) {
        okMsg = '$okMsg；已生成结算';
      }
    }
    final msg = res.ok ? okMsg : (_errLabel[res.msg] ?? res.msg);
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
    if (res.ok) Navigator.of(context).pop(true);
  }

  Future<void> _loadPurchaseUsers() async {
    final res = await context.read<AuthState>().api.get('/purchase/role-users?role=purchase');
    if (!mounted || !res.ok) return;
    final list = res.data is Map ? (res.data as Map)['list'] : null;
    setState(() {
      _purchaseUsers = list is List ? list : ApiClient.listOf(res.data);
    });
  }

  Future<void> _returnToPurchase() async {
    final bizId = _bizId;
    if (bizId == null) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('未定位到过磅单')));
      return;
    }
    await _loadPurchaseUsers();
    if (!mounted) return;
    final reasonCtrl = TextEditingController();
    int? selectedUserId = (_ticket['applicant_user_id'] as num?)?.toInt() ??
        (_ticket['confirmed_by'] as num?)?.toInt() ??
        (_ticket['from_user_id'] as num?)?.toInt();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setLocal) => AlertDialog(
          title: const Text('退回采购'),
          content: SizedBox(
            width: 360,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: reasonCtrl,
                  decoration: const InputDecoration(labelText: '原因', border: OutlineInputBorder()),
                  maxLines: 2,
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<int>(
                  value: selectedUserId,
                  decoration: const InputDecoration(labelText: '退回给', border: OutlineInputBorder()),
                  items: [
                    for (final u in _purchaseUsers)
                      if (u is Map)
                        DropdownMenuItem(
                          value: (u['id'] as num?)?.toInt(),
                          child: Text('${u['name'] ?? u['login_name'] ?? u['id']}'),
                        ),
                  ],
                  onChanged: (v) => setLocal(() => selectedUserId = v),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
            FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('确认退回')),
          ],
        ),
      ),
    );
    final reason = reasonCtrl.text.trim();
    reasonCtrl.dispose();
    if (ok != true || !mounted) return;
    if (reason.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请填写退回原因')));
      return;
    }
    final body = <String, dynamic>{'reason': reason};
    if (selectedUserId != null && selectedUserId! > 0) {
      body['to_user_id'] = selectedUserId;
    }
    final api = context.read<AuthState>().api;
    setState(() => _busy = true);
    final res = await api.post(
      '/purchase/weigh-tickets/$bizId/warehouse-return',
      body,
    );
    if (!mounted) return;
    setState(() => _busy = false);
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(res.ok ? '已退回采购' : res.msg)));
    if (res.ok) Navigator.of(context).pop(true);
  }

  @override
  Widget build(BuildContext context) {
    final imgs = _photoUrls();
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;
    final net = _ticketNet;
    final sum = _boxSum;
    final loss = _inboundLoss;
    final overPct = net > 0 && sum > net ? ((sum - net) / net * 100) : 0.0;
    return Scaffold(
      appBar: AppBar(title: Text(_isBoxMode ? '扫溯源分箱入库' : '入厂接收核对')),
      body: ListView(
        padding: EdgeInsets.fromLTRB(16, 12, 16, 24 + bottomInset),
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  '${_ticket['doc_no'] ?? ''}',
                  style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
                ),
              ),
              Chip(
                label: Text(_kindLabel, style: const TextStyle(fontSize: 12, color: Colors.white)),
                backgroundColor: _isBoxMode ? Colors.teal : Colors.indigo,
                visualDensity: VisualDensity.compact,
                padding: EdgeInsets.zero,
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            _isBoxMode
                ? '对已入厂溯源批次分箱复磅；完成后得到真实仓前扣损率'
                : '核对后确认接收进场；本环节不分箱。入厂后可再扫溯源分箱。',
            style: TextStyle(fontSize: 13, color: Colors.blueGrey.shade700),
          ),
          const SizedBox(height: 12),
          _kv('溯源码', '${_ticket['trace_code'] ?? '-'}'),
          _kv('批号', '${_ticket['batch_no'] ?? '-'}'),
          _kv('票净重', '${_ticket['net_weight'] ?? '-'} kg'),
          _kv('扣损', '${_ticket['deduct_weight'] ?? '-'}'),
          if (!_ready)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(
                '未就绪：${_ticket['reason'] ?? _ticket['status'] ?? ''}',
                style: TextStyle(color: Colors.orange.shade800),
              ),
            ),
          if (_isBoxMode) ...[
            const SizedBox(height: 16),
            const Text('入库产品', style: TextStyle(fontWeight: FontWeight.w600)),
            const SizedBox(height: 8),
            DropdownButtonFormField<int>(
              value: _productId,
              decoration: const InputDecoration(
                border: OutlineInputBorder(),
                hintText: '必选：原料或半成品',
              ),
              items: [
                for (final p in _products)
                  DropdownMenuItem(
                    value: (p['id'] as num?)?.toInt(),
                    child: Text('${p['name'] ?? p['code'] ?? p['id']}'),
                  ),
              ],
              onChanged: (v) => setState(() => _productId = v),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                const Expanded(child: Text('分箱复磅', style: TextStyle(fontWeight: FontWeight.w600))),
                TextButton.icon(onPressed: _addBox, icon: const Icon(Icons.add, size: 18), label: const Text('加箱')),
              ],
            ),
            const SizedBox(height: 4),
            Text('每箱现场复磅后录入；箱码自动绑定本溯源', style: TextStyle(fontSize: 12, color: Colors.black54)),
            const SizedBox(height: 8),
            for (var i = 0; i < _boxCtrls.length; i++)
              Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: Row(
                  children: [
                    SizedBox(width: 56, child: Text('箱 ${i + 1}', style: const TextStyle(fontWeight: FontWeight.w500))),
                    Expanded(
                      child: TextField(
                        controller: _boxCtrls[i],
                        keyboardType: const TextInputType.numberWithOptions(decimal: true),
                        decoration: const InputDecoration(
                          border: OutlineInputBorder(),
                          isDense: true,
                          suffixText: 'kg',
                        ),
                        onChanged: (_) => setState(() {}),
                      ),
                    ),
                    IconButton(
                      onPressed: _boxCtrls.length > 1 ? () => _removeBox(i) : null,
                      icon: const Icon(Icons.delete_outline),
                    ),
                  ],
                ),
              ),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: _weightOk ? const Color(0xFFEEF8F4) : const Color(0xFFFFF4E5),
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: _weightOk ? const Color(0xFFB7E0CD) : const Color(0xFFFFD8A8)),
              ),
              child: Text(
                loss > 0
                    ? '已录合计 ${sum.toStringAsFixed(2)} kg · 票净重 ${net.toStringAsFixed(2)} kg\n'
                        '将记仓前损耗 ${loss.toStringAsFixed(2)} kg，扣损率 ${_lossRate.toStringAsFixed(1)}%'
                    : '已录合计 ${sum.toStringAsFixed(2)} kg · 票净重 ${net.toStringAsFixed(2)} kg'
                        '${sum > net ? ' · 超重 ${overPct.toStringAsFixed(1)}%' : ''}\n'
                        '欠重自动记仓前损耗；超重允许 ±3% 或 5kg',
                style: TextStyle(fontSize: 13, color: _weightOk ? Colors.teal.shade900 : Colors.orange.shade900),
              ),
            ),
          ],
          if (imgs.isNotEmpty) ...[
            const SizedBox(height: 16),
            const Text('现场照片', style: TextStyle(fontWeight: FontWeight.w600)),
            const SizedBox(height: 8),
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
        ],
      ),
      bottomNavigationBar: FormStickyButtonBar(
        children: [
          if (!_isBoxMode)
            OutlinedButton(
              onPressed: _busy ? null : _returnToPurchase,
              child: const Text('退回采购'),
            ),
          FilledButton(
            onPressed: _canConfirm ? _confirm : null,
            child: Text(_busy ? '处理中…' : _confirmLabel),
          ),
        ],
      ),
    );
  }

  Widget _kv(String k, String v) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(width: 72, child: Text(k, style: const TextStyle(color: Colors.black54))),
          Expanded(child: Text(v, style: const TextStyle(fontWeight: FontWeight.w500))),
        ],
      ),
    );
  }
}
