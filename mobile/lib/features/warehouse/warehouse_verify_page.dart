import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../widgets/form_row.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_sticky_actions.dart';

/// 仓管核对页：入厂接收（信息页直接拒收/接收）；已入厂则扫溯源分箱入库（填表→预览→提交）。
class WarehouseVerifyPage extends StatefulWidget {
  const WarehouseVerifyPage({super.key, required this.ticket});

  final Map<String, dynamic> ticket;

  @override
  State<WarehouseVerifyPage> createState() => _WarehouseVerifyPageState();
}

class _WarehouseVerifyPageState extends State<WarehouseVerifyPage> {
  late Map<String, dynamic> _ticket;
  bool _busy = false;
  /// 0 填表 · 1 预览
  int _step = 0;
  String _msg = '';
  bool _msgIsError = false;
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

  bool get _isBoxMode =>
      _ticket['box_stockin_ready'] == true || _status == 'gate_accepted';

  String get _kindLabel => _isBoxMode ? '分箱入库' : '入厂接收';

  String get _confirmLabel => _isBoxMode ? '确认分箱入库' : '接收入厂';

  bool get _ready => _isBoxMode || _ticket['stockin_ready'] == true;

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

  String get _productLabel {
    if (_productId == null) return '-';
    for (final p in _products) {
      if ((p['id'] as num?)?.toInt() == _productId) {
        return '${p['name'] ?? p['code'] ?? p['id']}';
      }
    }
    return '$_productId';
  }

  String? _validateForm() {
    if (!_ready) return '单据未就绪：${_ticket['reason'] ?? _ticket['status'] ?? ''}';
    if (!_isBoxMode) return null;
    if (_productId == null || _productId! <= 0) return '请选择入库产品';
    if (_boxCtrls.isEmpty) return '请至少录入一箱复磅重量';
    for (final c in _boxCtrls) {
      final w = double.tryParse(c.text.trim()) ?? 0;
      if (w <= 0) return '每箱复磅重量须大于 0';
    }
    if (!_weightOk) return '分箱合计超过票净重过多（允许超重 ±3% 或 5kg）';
    return null;
  }

  void _prompt(String msg) {
    setState(() {
      _msg = msg;
      _msgIsError = true;
    });
    final messenger = ScaffoldMessenger.of(context);
    messenger.clearSnackBars();
    final scheme = Theme.of(context).colorScheme;
    messenger.showSnackBar(
      SnackBar(
        content: Text(msg, style: TextStyle(color: scheme.onError)),
        backgroundColor: scheme.error,
        behavior: SnackBarBehavior.floating,
      ),
    );
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

  void _goPreview() {
    final err = _validateForm();
    if (err != null) {
      _prompt(err);
      return;
    }
    setState(() {
      _step = 1;
      _msg = '';
      _msgIsError = false;
    });
  }

  Future<void> _confirm() async {
    final err = _validateForm();
    if (err != null) {
      _prompt(err);
      return;
    }
    final bizId = _bizId;
    if (bizId == null) {
      _prompt('未定位到过磅单');
      return;
    }

    setState(() => _busy = true);
    late final dynamic res;
    if (_isBoxMode) {
      final boxes = <Map<String, dynamic>>[];
      for (final c in _boxCtrls) {
        boxes.add({'weight': double.tryParse(c.text.trim()) ?? 0});
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
      _prompt('未定位到过磅单');
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
          title: const Text('拒绝接受'),
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
            FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('确认拒绝')),
          ],
        ),
      ),
    );
    final reason = reasonCtrl.text.trim();
    reasonCtrl.dispose();
    if (ok != true || !mounted) return;
    if (reason.isEmpty) {
      _prompt('请填写拒绝原因');
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
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(res.ok ? '已拒绝并退回采购' : res.msg)));
    if (res.ok) Navigator.of(context).pop(true);
  }

  Widget _kv(String k, String v) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(width: 108, child: Text(k, style: TextStyle(fontSize: 13, color: Colors.black.withValues(alpha: 0.6)))),
          Expanded(child: Text(v, textAlign: TextAlign.right, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w500))),
        ],
      ),
    );
  }

  Future<void> _showPhotoPreview(String url) async {
    await showDialog<void>(
      context: context,
      barrierColor: Colors.black87,
      builder: (ctx) => Dialog(
        backgroundColor: Colors.transparent,
        insetPadding: const EdgeInsets.all(12),
        child: Stack(
          children: [
            Center(
              child: InteractiveViewer(
                minScale: 0.8,
                maxScale: 4,
                child: Image.network(
                  url,
                  fit: BoxFit.contain,
                  errorBuilder: (_, __, ___) => const Padding(
                    padding: EdgeInsets.all(24),
                    child: Text('图片加载失败', style: TextStyle(color: Colors.white)),
                  ),
                ),
              ),
            ),
            Positioned(
              top: 0,
              right: 0,
              child: IconButton(
                onPressed: () => Navigator.of(ctx).pop(),
                icon: const Icon(Icons.close, color: Colors.white),
              ),
            ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final imgs = _photoUrls();
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;
    final net = _ticketNet;
    final sum = _boxSum;
    final loss = _inboundLoss;
    final overPct = net > 0 && sum > net ? ((sum - net) / net * 100) : 0.0;
    final title = _isBoxMode && _step == 1 ? '$_kindLabel · 预览' : _kindLabel;

    return Scaffold(
      appBar: AppBar(
        title: Text(title),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () {
            if (_isBoxMode && _step == 1) {
              setState(() {
                _step = 0;
                _msg = '';
                _msgIsError = false;
              });
            } else {
              Navigator.of(context).maybePop();
            }
          },
        ),
      ),
      body: Column(
        children: [
          Expanded(
            child: _isBoxMode && _step == 1
                ? ListView(
                    padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + bottomInset),
                    children: [
                      const Text('核对预览', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
                      const Text('请核对以下信息，有误请返回修改', style: TextStyle(fontSize: 12, color: Colors.black54)),
                      const SizedBox(height: 8),
                      _kv('类型', _kindLabel),
                      _kv('单号', '${_ticket['doc_no'] ?? '-'}'),
                      _kv('溯源码', '${_ticket['trace_code'] ?? '-'}'),
                      _kv('票净重', '${_ticket['net_weight'] ?? '-'} kg'),
                      _kv('入库产品', _productLabel),
                      _kv('箱数', '${_boxCtrls.length}'),
                      _kv('分箱合计', '${sum.toStringAsFixed(2)} kg'),
                      _kv('仓前损耗', loss > 0 ? '${loss.toStringAsFixed(2)} kg（${_lossRate.toStringAsFixed(1)}%）' : '0 kg'),
                      for (var i = 0; i < _boxCtrls.length; i++)
                        _kv('箱 ${i + 1}', '${_boxCtrls[i].text.trim()} kg'),
                      const SizedBox(height: 8),
                      const Text(
                        '确认后直接分箱入库并记仓前损耗。',
                        style: TextStyle(fontSize: 12, color: Colors.black54),
                      ),
                    ],
                  )
                : ListView(
                    keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
                    padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + bottomInset),
                    children: [
                      Chip(
                        label: Text(_kindLabel, style: const TextStyle(fontSize: 12, color: Colors.white)),
                        backgroundColor: _isBoxMode ? Colors.teal : Colors.indigo,
                        visualDensity: VisualDensity.compact,
                      ),
                      const FormSectionHeader('单据信息'),
                      _kv('单号', '${_ticket['doc_no'] ?? '-'}'),
                      if ((_ticket['party_name'] ?? _ticket['farmer_name'] ?? '').toString().isNotEmpty)
                        _kv('农户', '${_ticket['party_name'] ?? _ticket['farmer_name']}'),
                      if ((_ticket['product_name'] ?? _ticket['variety'] ?? '').toString().isNotEmpty)
                        _kv('品种', '${_ticket['product_name'] ?? _ticket['variety']}'),
                      _kv('溯源码', '${_ticket['trace_code'] ?? '-'}'),
                      _kv('批号', '${_ticket['batch_no'] ?? '-'}'),
                      if (_ticket['gross_weight'] != null)
                        _kv('毛重', '${_ticket['gross_weight']} kg'),
                      _kv(
                        '扣损',
                        _ticket['deduct_weight'] == null || '${_ticket['deduct_weight']}' == ''
                            ? '-'
                            : '${_ticket['deduct_weight']} kg'
                                '${_ticket['deduct_rate'] != null && '${_ticket['deduct_rate']}' != '' ? '（${_ticket['deduct_rate']}%）' : ''}',
                      ),
                      _kv('票净重', '${_ticket['net_weight'] ?? '-'} kg'),
                      if ((_ticket['plate_no'] ?? '').toString().isNotEmpty)
                        _kv('车牌', '${_ticket['plate_no']}'),
                      if ((_ticket['biz_date'] ?? '').toString().isNotEmpty)
                        _kv('业务日', '${_ticket['biz_date']}'),
                      if (!_ready)
                        Padding(
                          padding: const EdgeInsets.only(top: 8),
                          child: Text(
                            '未就绪：${_ticket['reason'] ?? _ticket['status'] ?? ''}',
                            style: TextStyle(color: Colors.orange.shade800),
                          ),
                        ),
                      if (_isBoxMode) ...[
                        const FormSectionHeader('分箱复磅'),
                        FormRow(
                          label: '入库产品',
                          requiredMark: true,
                          child: DropdownButtonHideUnderline(
                            child: DropdownButton<int>(
                              isExpanded: true,
                              value: _productId,
                              alignment: Alignment.centerRight,
                              hint: const Text('必选', textAlign: TextAlign.right),
                              items: [
                                for (final p in _products)
                                  DropdownMenuItem(
                                    value: (p['id'] as num?)?.toInt(),
                                    child: Text('${p['name'] ?? p['code'] ?? p['id']}', textAlign: TextAlign.right),
                                  ),
                              ],
                              onChanged: (v) => setState(() => _productId = v),
                            ),
                          ),
                        ),
                        Align(
                          alignment: Alignment.centerRight,
                          child: TextButton.icon(
                            onPressed: _addBox,
                            icon: const Icon(Icons.add, size: 18),
                            label: const Text('加箱'),
                          ),
                        ),
                        for (var i = 0; i < _boxCtrls.length; i++)
                          FormRow(
                            label: '箱 ${i + 1}(kg)',
                            requiredMark: true,
                            child: Row(
                              children: [
                                Expanded(
                                  child: TextField(
                                    controller: _boxCtrls[i],
                                    textAlign: TextAlign.right,
                                    keyboardType: const TextInputType.numberWithOptions(decimal: true),
                                    style: const TextStyle(fontSize: 15),
                                    decoration: FormRow.fieldDecoration(hint: '复磅重量'),
                                    onChanged: (_) => setState(() {}),
                                  ),
                                ),
                                if (_boxCtrls.length > 1)
                                  IconButton(
                                    onPressed: () => _removeBox(i),
                                    icon: const Icon(Icons.delete_outline, size: 20),
                                  ),
                              ],
                            ),
                          ),
                        Container(
                          width: double.infinity,
                          margin: const EdgeInsets.only(top: 8),
                          padding: const EdgeInsets.all(10),
                          decoration: BoxDecoration(
                            color: _weightOk ? const Color(0xFFEEF8F4) : const Color(0xFFFFF4E5),
                            borderRadius: BorderRadius.circular(8),
                            border: Border.all(
                              color: _weightOk ? const Color(0xFFB7E0CD) : const Color(0xFFFFD8A8),
                            ),
                          ),
                          child: Text(
                            loss > 0
                                ? '已录合计 ${sum.toStringAsFixed(2)} kg · 票净重 ${net.toStringAsFixed(2)} kg\n'
                                    '将记仓前损耗 ${loss.toStringAsFixed(2)} kg，扣损率 ${_lossRate.toStringAsFixed(1)}%'
                                : '已录合计 ${sum.toStringAsFixed(2)} kg · 票净重 ${net.toStringAsFixed(2)} kg'
                                    '${sum > net ? ' · 超重 ${overPct.toStringAsFixed(1)}%' : ''}\n'
                                    '欠重自动记仓前损耗；超重允许 ±3% 或 5kg',
                            style: TextStyle(
                              fontSize: 13,
                              color: _weightOk ? Colors.teal.shade900 : Colors.orange.shade900,
                            ),
                          ),
                        ),
                      ] else
                        const Padding(
                          padding: EdgeInsets.only(top: 8),
                          child: Text(
                            '请核对以上信息。无误请接收入厂；有误请拒绝接受并退回采购。本环节不分箱。',
                            style: TextStyle(fontSize: 13, color: Colors.black54),
                          ),
                        ),
                      if (imgs.isNotEmpty) ...[
                        const FormSectionHeader('现场照片'),
                        SizedBox(
                          height: 96,
                          child: ListView.separated(
                            scrollDirection: Axis.horizontal,
                            itemCount: imgs.length,
                            separatorBuilder: (_, __) => const SizedBox(width: 8),
                            itemBuilder: (_, i) {
                              final url = imgs[i];
                              return Material(
                                color: Colors.transparent,
                                child: InkWell(
                                  onTap: () => _showPhotoPreview(url),
                                  borderRadius: BorderRadius.circular(8),
                                  child: ClipRRect(
                                    borderRadius: BorderRadius.circular(8),
                                    child: Image.network(
                                      url,
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
                              );
                            },
                          ),
                        ),
                      ],
                    ],
                  ),
          ),
          if (_msg.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 4),
              child: Text(
                _msg,
                style: TextStyle(
                  fontSize: 13,
                  color: _msgIsError ? Theme.of(context).colorScheme.error : Colors.teal,
                  fontWeight: _msgIsError ? FontWeight.w600 : FontWeight.normal,
                ),
              ),
            ),
          if (!_isBoxMode)
            FormStickyActions(
              secondaryLabel: '拒绝接受',
              onSecondary: _busy ? null : _returnToPurchase,
              primaryLabel: '接收入厂',
              onPrimary: _busy || !_ready ? null : _confirm,
              primaryBusy: _busy,
              busyLabel: '处理中…',
            )
          else if (_step == 0)
            FormStickyActions(
              primaryLabel: '下一步',
              onPrimary: _busy ? null : _goPreview,
            )
          else
            FormStickyActions(
              secondaryLabel: '修改',
              onSecondary: _busy
                  ? null
                  : () => setState(() {
                        _step = 0;
                        _msg = '';
                        _msgIsError = false;
                      }),
              primaryLabel: _confirmLabel,
              onPrimary: _busy ? null : _confirm,
              primaryBusy: _busy,
              busyLabel: '处理中…',
            ),
        ],
      ),
    );
  }
}
