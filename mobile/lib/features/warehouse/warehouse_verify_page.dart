import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../widgets/form_sticky_actions.dart';

/// 仓管核对页：选产品、分箱复磅后确认入厂/入库，或退回采购。
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
    'WEIGHT_MISMATCH': '分箱合计与票净重偏差过大（允许 ±3% 或 ±5kg）',
    'VERIFY_REQUIRED': '请确认核对相符',
  };

  @override
  void initState() {
    super.initState();
    _ticket = Map<String, dynamic>.from(widget.ticket);
    final pid = (_ticket['product_id'] as num?)?.toInt() ?? 0;
    _productId = pid > 0 ? pid : null;
    _boxCtrls.add(TextEditingController());
    _loadProducts();
  }

  @override
  void dispose() {
    for (final c in _boxCtrls) {
      c.dispose();
    }
    super.dispose();
  }

  String get _kind {
    final k = (_ticket['receive_kind'] ?? '').toString().toLowerCase();
    return k == 'stockin' ? 'stockin' : 'gate';
  }

  String get _kindLabel => _kind == 'stockin' ? '入库' : '入厂';

  String get _confirmLabel => _kind == 'stockin' ? '确认入库' : '确认入厂';

  bool get _ready => _ticket['stockin_ready'] == true;

  Object? get _bizId => _ticket['weigh_ticket_id'] ?? _ticket['biz_id'] ?? _ticket['id'];

  double get _ticketNet => (_ticket['net_weight'] as num?)?.toDouble() ?? 0;

  double get _boxSum {
    var s = 0.0;
    for (final c in _boxCtrls) {
      s += double.tryParse(c.text.trim()) ?? 0;
    }
    return s;
  }

  bool get _weightOk {
    final net = _ticketNet;
    if (net <= 0) return _boxSum > 0;
    final diff = (_boxSum - net).abs();
    var tol = net * 0.03;
    if (tol < 5) tol = 5;
    return diff <= tol && _boxSum > 0;
  }

  bool get _canConfirm {
    if (_busy || !_ready) return false;
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
    if (_productId == null || _productId! <= 0) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请选择入库产品')));
      return;
    }
    final boxes = <Map<String, dynamic>>[];
    for (final c in _boxCtrls) {
      final w = double.tryParse(c.text.trim()) ?? 0;
      if (w <= 0) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('每箱复磅重量须大于 0')));
        return;
      }
      boxes.add({'weight': w});
    }
    if (!_weightOk) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('分箱合计与票净重偏差过大（允许 ±3% 或 ±5kg）')),
      );
      return;
    }
    setState(() => _busy = true);
    final res = await context.read<AuthState>().api.post(
      '/purchase/weigh-tickets/$bizId/warehouse-confirm',
      {
        'verified': true,
        'match_confirmed': true,
        'product_id': _productId,
        'boxes': boxes,
      },
    );
    if (!mounted) return;
    setState(() => _busy = false);
    final okMsg = _kind == 'stockin' ? '入库完成' : '入厂确认完成';
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
    final diffPct = net > 0 ? ((sum - net) / net * 100) : 0.0;
    return Scaffold(
      appBar: AppBar(title: Text('核对$_kindLabel')),
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
                backgroundColor: _kind == 'stockin' ? Colors.teal : Colors.indigo,
                visualDensity: VisualDensity.compact,
                padding: EdgeInsets.zero,
              ),
            ],
          ),
          const SizedBox(height: 12),
          _kv('溯源码', '${_ticket['trace_code'] ?? '-'}'),
          _kv('批号', '${_ticket['batch_no'] ?? '-'}'),
          _kv('票净重', '${_ticket['net_weight'] ?? '-'} kg（参考）'),
          _kv('扣损', '${_ticket['deduct_weight'] ?? '-'}'),
          if (!_ready)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(
                '未就绪：${_ticket['reason'] ?? _ticket['status'] ?? ''}',
                style: TextStyle(color: Colors.orange.shade800),
              ),
            ),
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
          Text('每箱现场复磅后录入重量（kg）', style: TextStyle(fontSize: 12, color: Colors.black54)),
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
              '已录合计 ${sum.toStringAsFixed(2)} kg · 票净重 ${net.toStringAsFixed(2)} kg · 偏差 ${diffPct.toStringAsFixed(1)}%\n'
              '允许偏差：±3% 或 ±5kg（取较宽）',
              style: TextStyle(fontSize: 13, color: _weightOk ? Colors.teal.shade900 : Colors.orange.shade900),
            ),
          ),
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
