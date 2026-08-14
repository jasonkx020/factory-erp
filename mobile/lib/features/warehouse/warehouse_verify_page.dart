import 'dart:async';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../widgets/form_row.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_sticky_actions.dart';
import 'box_stockin_draft.dart';

class _BoxLine {
  _BoxLine() : weight = TextEditingController();

  final TextEditingController weight;
  final List<String> imageUrls = [];

  void dispose() => weight.dispose();
}

/// 仓管核对页：入厂接收（信息页直接拒收/接收）；已入厂则扫溯源分箱入库（可多日分批+草稿）。
class WarehouseVerifyPage extends StatefulWidget {
  const WarehouseVerifyPage({super.key, required this.ticket});

  final Map<String, dynamic> ticket;

  @override
  State<WarehouseVerifyPage> createState() => _WarehouseVerifyPageState();
}

class _WarehouseVerifyPageState extends State<WarehouseVerifyPage> {
  late Map<String, dynamic> _ticket;
  bool _busy = false;
  /// 0 填表 · 1 预览（分箱）
  int _step = 0;
  String _msg = '';
  bool _msgIsError = false;
  List<dynamic> _purchaseUsers = [];
  List<Map<String, dynamic>> _products = [];
  int? _productId;
  final List<_BoxLine> _boxes = [];
  Timer? _draftTimer;
  List<Map<String, dynamic>> _doneBoxes = [];

  static const _errLabel = {
    'BOXES_REQUIRED': '请至少录入一箱复磅重量',
    'BOX_WEIGHT_REQUIRED': '每箱复磅重量须大于 0',
    'BOX_PHOTO_REQUIRED': '每箱须拍摄至少一张复磅照片',
    'PRODUCT_REQUIRED': '请选择入库产品',
    'ROUTING_REQUIRED': '该产品未配置入厂工艺，请先在工艺流程绑定',
    'WEIGHT_MISMATCH': '分箱合计超过票净重过多（允许超重 ±3% 或 5kg）',
    'TRACE_CODE_REQUIRED': '缺少溯源码，无法建箱',
    'VERIFY_REQUIRED': '请确认核对相符',
    'GATE_ACCEPT_REQUIRED': '请先完成入厂接收后再分箱',
    'USE_WAREHOUSE_BOX_STOCKIN': '请仓管扫溯源分箱入库',
    'ALREADY_STOCKED': '本批已完成分箱入库',
    'APP_ONLY': '请在 App 仓管端操作',
  };

  @override
  void initState() {
    super.initState();
    _ticket = Map<String, dynamic>.from(widget.ticket);
    final pid = (_ticket['product_id'] as num?)?.toInt() ?? 0;
    _productId = pid > 0 ? pid : null;
    _syncDoneBoxesFromTicket();
    if (_isBoxMode) {
      _boxes.add(_BoxLine());
      _loadProducts();
      _restoreDraft();
    }
  }

  @override
  void dispose() {
    _draftTimer?.cancel();
    for (final b in _boxes) {
      b.dispose();
    }
    super.dispose();
  }

  String get _status => (_ticket['status'] ?? '').toString().toLowerCase();

  bool get _isBoxMode =>
      _ticket['box_stockin_ready'] == true || _status == 'gate_accepted';

  String get _kindLabel => _isBoxMode ? '分箱入库' : '入厂接收';

  bool get _ready => _isBoxMode || _ticket['stockin_ready'] == true;

  Object? get _bizId => _ticket['weigh_ticket_id'] ?? _ticket['biz_id'] ?? _ticket['id'];

  String get _draftKey {
    final trace = (_ticket['trace_code'] ?? '').toString().trim();
    if (trace.isNotEmpty) return trace;
    return '${_bizId ?? ''}';
  }

  double get _ticketNet => (_ticket['net_weight'] as num?)?.toDouble() ?? 0;

  double get _boxedWeight => (_ticket['boxed_weight'] as num?)?.toDouble() ??
      _doneBoxes.fold<double>(0, (s, e) => s + ((e['weight'] as num?)?.toDouble() ?? 0));

  int get _boxedQty => (_ticket['boxed_qty'] as num?)?.toInt() ?? _doneBoxes.length;

  double get _draftSum {
    var s = 0.0;
    for (final b in _boxes) {
      s += double.tryParse(b.weight.text.trim()) ?? 0;
    }
    return s;
  }

  double get _combinedSum => _boxedWeight + _draftSum;

  double get _remaining {
    final r = _ticketNet - _boxedWeight;
    return r > 0 ? r : 0;
  }

  double get _projectedLoss {
    final loss = _ticketNet - _combinedSum;
    return loss > 0 ? loss : 0;
  }

  bool get _weightOk {
    if (!_isBoxMode) return true;
    if (_draftSum <= 0 && _boxedWeight <= 0) return false;
    final net = _ticketNet;
    if (net <= 0) return true;
    if (_combinedSum <= net) return true;
    final diff = _combinedSum - net;
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

  void _syncDoneBoxesFromTicket() {
    final raw = _ticket['boxes'];
    final list = <Map<String, dynamic>>[];
    if (raw is List) {
      for (final e in raw) {
        if (e is Map) list.add(Map<String, dynamic>.from(e));
      }
    }
    _doneBoxes = list;
  }

  String? _validateBoxDraft({required bool requireLines}) {
    if (!_ready) return '单据未就绪：${_ticket['reason'] ?? _ticket['status'] ?? ''}';
    if (_productId == null || _productId! <= 0) return '请选择入库产品';
    if (requireLines) {
      if (_boxes.isEmpty) return '请至少录入一箱复磅重量';
      for (var i = 0; i < _boxes.length; i++) {
        final w = double.tryParse(_boxes[i].weight.text.trim()) ?? 0;
        if (w <= 0) return '第 ${i + 1} 箱复磅重量须大于 0';
        if (_boxes[i].imageUrls.isEmpty) return '第 ${i + 1} 箱须拍摄至少一张复磅照片';
      }
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

  Future<void> _restoreDraft() async {
    final draft = await BoxStockinDraftStore.load(_draftKey);
    if (!mounted || draft == null) return;
    final pid = (draft['product_id'] as num?)?.toInt();
    final rawBoxes = draft['boxes'];
    if (rawBoxes is! List || rawBoxes.isEmpty) return;
    for (final b in _boxes) {
      b.dispose();
    }
    _boxes.clear();
    for (final e in rawBoxes) {
      if (e is! Map) continue;
      final line = _BoxLine();
      line.weight.text = '${e['weight'] ?? ''}';
      final urls = e['image_urls'] ?? e['uploadedUrls'];
      if (urls is List) {
        for (final u in urls) {
          final s = u?.toString() ?? '';
          if (s.isNotEmpty) line.imageUrls.add(s);
        }
      }
      _boxes.add(line);
    }
    if (_boxes.isEmpty) _boxes.add(_BoxLine());
    setState(() {
      if (pid != null && pid > 0) _productId = pid;
    });
  }

  void _scheduleSaveDraft() {
    if (!_isBoxMode) return;
    _draftTimer?.cancel();
    _draftTimer = Timer(const Duration(milliseconds: 400), _saveDraft);
  }

  Future<void> _saveDraft() async {
    if (!_isBoxMode) return;
    final boxes = <Map<String, dynamic>>[];
    for (final b in _boxes) {
      boxes.add({
        'weight': b.weight.text.trim(),
        'image_urls': List<String>.from(b.imageUrls),
      });
    }
    await BoxStockinDraftStore.save(_draftKey, {
      'product_id': _productId,
      'boxes': boxes,
      'updated_at': DateTime.now().toIso8601String(),
    });
  }

  List<String> _sitePhotos() {
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
    setState(() => _boxes.add(_BoxLine()));
    _scheduleSaveDraft();
  }

  Future<void> _removeBox(int i) async {
    if (_boxes.length <= 1) {
      _prompt('至少保留一箱');
      return;
    }
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('确认删除'),
        content: Text('确认删除第 ${i + 1} 箱？未提交的重量与照片将丢失。'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('删除')),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    setState(() {
      _boxes[i].dispose();
      _boxes.removeAt(i);
    });
    _scheduleSaveDraft();
  }

  Future<void> _takeBoxPhoto(int i) async {
    if (_boxes[i].imageUrls.length >= 3) {
      _prompt('每箱最多 3 张复磅照片');
      return;
    }
    try {
      final picker = ImagePicker();
      final file = await picker.pickImage(source: ImageSource.camera, imageQuality: 85);
      if (file == null) return;
      final bytes = await file.readAsBytes();
      setState(() {
        _msg = '上传中…';
        _msgIsError = false;
      });
      final r = await context.read<AuthState>().api.postMultipart(
            '/biz/uploads',
            bytes,
            filename: file.name.isEmpty ? 'reweigh.jpg' : file.name,
          );
      if (!mounted) return;
      if (!r.ok || r.data is! Map) {
        _prompt('上传失败：${r.msg}');
        return;
      }
      final url = (r.data as Map)['url']?.toString() ?? (r.data as Map)['file_url']?.toString() ?? '';
      if (url.isEmpty) {
        _prompt('上传无返回 URL');
        return;
      }
      setState(() {
        _boxes[i].imageUrls.add(url);
        _msg = '第 ${i + 1} 箱已上传 ${_boxes[i].imageUrls.length} 张';
        _msgIsError = false;
      });
      _scheduleSaveDraft();
    } catch (e) {
      _prompt('拍照失败：$e');
    }
  }

  void _goPreview() {
    final err = _validateBoxDraft(requireLines: true);
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

  List<Map<String, dynamic>> _boxesPayload() {
    return [
      for (final b in _boxes)
        {
          'weight': double.tryParse(b.weight.text.trim()) ?? 0,
          'image_urls': List<String>.from(b.imageUrls),
        },
    ];
  }

  Future<void> _mergeTicketFromResponse(Map data) async {
    _ticket = {..._ticket, ...Map<String, dynamic>.from(data)};
    _syncDoneBoxesFromTicket();
  }

  Future<void> _submitBatch() async {
    final err = _validateBoxDraft(requireLines: true);
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
    final res = await context.read<AuthState>().api.post(
      '/purchase/weigh-tickets/$bizId/box-stock-in',
      {'product_id': _productId, 'boxes': _boxesPayload()},
    );
    if (!mounted) return;
    setState(() => _busy = false);
    if (!res.ok) {
      _prompt(_errLabel[res.msg] ?? res.msg);
      return;
    }
    if (res.data is Map) await _mergeTicketFromResponse(res.data as Map);
    for (final b in _boxes) {
      b.dispose();
    }
    _boxes
      ..clear()
      ..add(_BoxLine());
    await BoxStockinDraftStore.clear(_draftKey);
    setState(() {
      _step = 0;
      _msg = '本批已入库 ${_ticket['batch_box_codes'] is List ? (res.data as Map)['batch_box_codes'] : ''}；可继续加箱或完成本批';
      _msgIsError = false;
    });
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('已提交本批 ${_boxedQty > 0 ? "累计 $_boxedQty 箱" : "入库"}')),
    );
  }

  Future<void> _completeBatch() async {
    final hasDraft = _boxes.any((b) =>
        (double.tryParse(b.weight.text.trim()) ?? 0) > 0 || b.imageUrls.isNotEmpty);
    if (hasDraft) {
      final choice = await showDialog<String>(
        context: context,
        builder: (ctx) => AlertDialog(
          title: const Text('还有未提交草稿'),
          content: const Text('完成本批前请先提交本页箱子，或放弃草稿仅完成已入库部分。'),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx, 'cancel'), child: const Text('取消')),
            TextButton(onPressed: () => Navigator.pop(ctx, 'discard'), child: const Text('放弃草稿并完成')),
            FilledButton(onPressed: () => Navigator.pop(ctx, 'submit'), child: const Text('先提交再完成')),
          ],
        ),
      );
      if (choice == null || choice == 'cancel' || !mounted) return;
      if (choice == 'submit') {
        final err = _validateBoxDraft(requireLines: true);
        if (err != null) {
          _prompt(err);
          return;
        }
      }
      if (choice == 'discard') {
        for (final b in _boxes) {
          b.dispose();
        }
        _boxes
          ..clear()
          ..add(_BoxLine());
        await BoxStockinDraftStore.clear(_draftKey);
      }
    }

    if (_boxedQty <= 0 && !_boxes.any((b) => (double.tryParse(b.weight.text.trim()) ?? 0) > 0)) {
      _prompt('请先至少入库一箱再完成本批');
      return;
    }

    final bizId = _bizId;
    if (bizId == null) {
      _prompt('未定位到过磅单');
      return;
    }

    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('完成本批分箱'),
        content: Text(
          '票净重 ${_ticketNet.toStringAsFixed(2)} kg，已入库 ${_boxedWeight.toStringAsFixed(2)} kg'
          '${_projectedLoss > 0 && !hasDraft ? "，将记仓前损耗 ${_projectedLoss.toStringAsFixed(2)} kg" : ""}。\n确认后不可再继续分箱。',
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('确认完成')),
        ],
      ),
    );
    if (confirm != true || !mounted) return;

    setState(() => _busy = true);
    final body = <String, dynamic>{'product_id': _productId};
    final pendingOk = _validateBoxDraft(requireLines: true) == null &&
        _boxes.any((b) => (double.tryParse(b.weight.text.trim()) ?? 0) > 0);
    if (pendingOk) {
      body['boxes'] = _boxesPayload();
    }
    final res = await context.read<AuthState>().api.post(
      '/purchase/weigh-tickets/$bizId/box-stock-in/complete',
      body,
    );
    if (!mounted) return;
    setState(() => _busy = false);
    if (!res.ok) {
      _prompt(_errLabel[res.msg] ?? res.msg);
      return;
    }
    await BoxStockinDraftStore.clear(_draftKey);
    var okMsg = '分箱入库完成';
    if (res.data is Map) {
      final data = res.data as Map;
      final loss = data['inbound_loss_kg'];
      final rate = data['inbound_loss_rate'];
      if (loss is num && loss > 0) {
        final pct = rate is num ? (rate * 100) : (loss / (_ticketNet > 0 ? _ticketNet : 1) * 100);
        okMsg = '分箱入库完成（仓前损耗 ${loss.toStringAsFixed(2)} kg，扣损率 ${pct.toStringAsFixed(1)}%）';
      }
      if (data['settlement_id'] != null) okMsg = '$okMsg；已生成结算';
    }
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(okMsg)));
    Navigator.of(context).pop(true);
  }

  Future<void> _confirmGate() async {
    if (!_ready) {
      _prompt('单据未就绪：${_ticket['reason'] ?? _ticket['status'] ?? ''}');
      return;
    }
    final bizId = _bizId;
    if (bizId == null) {
      _prompt('未定位到过磅单');
      return;
    }
    setState(() => _busy = true);
    final res = await context.read<AuthState>().api.post(
      '/purchase/weigh-tickets/$bizId/warehouse-confirm',
      {'verified': true, 'match_confirmed': true},
    );
    if (!mounted) return;
    setState(() => _busy = false);
    final msg = res.ok ? '入厂接收完成' : (_errLabel[res.msg] ?? res.msg);
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
    setState(() => _busy = true);
    final res = await context.read<AuthState>().api.post(
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

  Widget _boxEditor(int i) {
    final line = _boxes[i];
    final api = context.read<AuthState>().api;
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      child: Padding(
        padding: const EdgeInsets.all(10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Text('箱 ${i + 1}', style: const TextStyle(fontWeight: FontWeight.w600)),
                const Spacer(),
                if (_boxes.length > 1)
                  IconButton(
                    onPressed: _busy ? null : () => _removeBox(i),
                    icon: const Icon(Icons.delete_outline, size: 20),
                    tooltip: '删除',
                  ),
              ],
            ),
            FormRow(
              label: '复磅(kg)',
              requiredMark: true,
              child: TextField(
                controller: line.weight,
                textAlign: TextAlign.right,
                keyboardType: const TextInputType.numberWithOptions(decimal: true),
                style: const TextStyle(fontSize: 15),
                decoration: FormRow.fieldDecoration(hint: '真实复磅重量'),
                onChanged: (_) {
                  setState(() {});
                  _scheduleSaveDraft();
                },
              ),
            ),
            const SizedBox(height: 6),
            Row(
              children: [
                const Text('复磅照片', style: TextStyle(fontSize: 13, color: Colors.black54)),
                const Spacer(),
                TextButton.icon(
                  onPressed: _busy ? null : () => _takeBoxPhoto(i),
                  icon: const Icon(Icons.photo_camera, size: 18),
                  label: const Text('拍照'),
                ),
              ],
            ),
            if (line.imageUrls.isEmpty)
              const Text('必填：至少 1 张', style: TextStyle(fontSize: 12, color: Colors.orange))
            else
              SizedBox(
                height: 72,
                child: ListView.separated(
                  scrollDirection: Axis.horizontal,
                  itemCount: line.imageUrls.length,
                  separatorBuilder: (_, __) => const SizedBox(width: 6),
                  itemBuilder: (_, j) {
                    final url = api.resolveMediaUrl(line.imageUrls[j]);
                    return InkWell(
                      onTap: () => _showPhotoPreview(url),
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(6),
                        child: Image.network(
                          url,
                          width: 72,
                          height: 72,
                          fit: BoxFit.cover,
                          errorBuilder: (_, __, ___) => Container(
                            width: 72,
                            height: 72,
                            color: Colors.black12,
                            child: const Icon(Icons.broken_image, size: 18),
                          ),
                        ),
                      ),
                    );
                  },
                ),
              ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final imgs = _sitePhotos();
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;
    final net = _ticketNet;
    final draft = _draftSum;
    final boxed = _boxedWeight;
    final combined = _combinedSum;
    final loss = _projectedLoss;
    final overPct = net > 0 && combined > net ? ((combined - net) / net * 100) : 0.0;
    final title = _isBoxMode && _step == 1 ? '$_kindLabel · 预览' : _kindLabel;
    final api = context.read<AuthState>().api;

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
                      const Text('提交本批仅入库当前箱子；完成本批后才记仓前损耗并结束。', style: TextStyle(fontSize: 12, color: Colors.black54)),
                      const SizedBox(height: 8),
                      _kv('单号', '${_ticket['doc_no'] ?? '-'}'),
                      _kv('溯源码', '${_ticket['trace_code'] ?? '-'}'),
                      _kv('票净重', '${net.toStringAsFixed(2)} kg'),
                      _kv('已入库', '${boxed.toStringAsFixed(2)} kg（$_boxedQty 箱）'),
                      _kv('本批草稿', '${draft.toStringAsFixed(2)} kg（${_boxes.length} 箱）'),
                      _kv('入库产品', _productLabel),
                      for (var i = 0; i < _boxes.length; i++) ...[
                        _kv('箱 ${i + 1}', '${_boxes[i].weight.text.trim()} kg · ${_boxes[i].imageUrls.length} 张图'),
                        if (_boxes[i].imageUrls.isNotEmpty)
                          SizedBox(
                            height: 64,
                            child: ListView.separated(
                              scrollDirection: Axis.horizontal,
                              itemCount: _boxes[i].imageUrls.length,
                              separatorBuilder: (_, __) => const SizedBox(width: 6),
                              itemBuilder: (_, j) {
                                final url = api.resolveMediaUrl(_boxes[i].imageUrls[j]);
                                return InkWell(
                                  onTap: () => _showPhotoPreview(url),
                                  child: ClipRRect(
                                    borderRadius: BorderRadius.circular(6),
                                    child: Image.network(url, width: 64, height: 64, fit: BoxFit.cover),
                                  ),
                                );
                              },
                            ),
                          ),
                      ],
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
                      if (_ticket['gross_weight'] != null) _kv('毛重', '${_ticket['gross_weight']} kg'),
                      _kv(
                        '扣损',
                        _ticket['deduct_weight'] == null || '${_ticket['deduct_weight']}' == ''
                            ? '-'
                            : '${_ticket['deduct_weight']} kg'
                                '${_ticket['deduct_rate'] != null && '${_ticket['deduct_rate']}' != '' ? '（${_ticket['deduct_rate']}%）' : ''}',
                      ),
                      _kv('票净重', '${_ticket['net_weight'] ?? '-'} kg'),
                      if ((_ticket['plate_no'] ?? '').toString().isNotEmpty) _kv('车牌', '${_ticket['plate_no']}'),
                      if (!_ready)
                        Padding(
                          padding: const EdgeInsets.only(top: 8),
                          child: Text(
                            '未就绪：${_ticket['reason'] ?? _ticket['status'] ?? ''}',
                            style: TextStyle(color: Colors.orange.shade800),
                          ),
                        ),
                      if (_isBoxMode) ...[
                        const FormSectionHeader('分箱进度'),
                        _kv('已入库', '${boxed.toStringAsFixed(2)} kg / $_boxedQty 箱'),
                        _kv('剩余可分', '${_remaining.toStringAsFixed(2)} kg'),
                        if (_doneBoxes.isNotEmpty) ...[
                          const Text('已入库箱', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600)),
                          for (final b in _doneBoxes)
                            ListTile(
                              dense: true,
                              contentPadding: EdgeInsets.zero,
                              title: Text('${b['code'] ?? '-'} · ${b['weight'] ?? '-'} kg'),
                              trailing: (b['image_url'] ?? (b['image_urls'] is List && (b['image_urls'] as List).isNotEmpty))
                                      != null &&
                                      '${b['image_url'] ?? ''}${b['image_urls']}' != ''
                                  ? SizedBox(
                                      width: 48,
                                      height: 48,
                                      child: InkWell(
                                        onTap: () {
                                          final u = b['image_url']?.toString();
                                          final list = b['image_urls'];
                                          final url = (u != null && u.isNotEmpty)
                                              ? api.resolveMediaUrl(u)
                                              : (list is List && list.isNotEmpty
                                                  ? api.resolveMediaUrl(list.first.toString())
                                                  : '');
                                          if (url.isNotEmpty) _showPhotoPreview(url);
                                        },
                                        child: Image.network(
                                          api.resolveMediaUrl(
                                            (b['image_url']?.toString().isNotEmpty == true)
                                                ? b['image_url'].toString()
                                                : ((b['image_urls'] as List?)?.first?.toString() ?? ''),
                                          ),
                                          fit: BoxFit.cover,
                                          errorBuilder: (_, __, ___) => const Icon(Icons.image),
                                        ),
                                      ),
                                    )
                                  : null,
                            ),
                        ],
                        const FormSectionHeader('本批复磅（可多日分批提交）'),
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
                              onChanged: (v) {
                                setState(() => _productId = v);
                                _scheduleSaveDraft();
                              },
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
                        for (var i = 0; i < _boxes.length; i++) _boxEditor(i),
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
                            '票净重 ${net.toStringAsFixed(2)} · 已入库 ${boxed.toStringAsFixed(2)} · 本页草稿 ${draft.toStringAsFixed(2)}\n'
                            '合计 ${combined.toStringAsFixed(2)} kg'
                            '${combined > net ? ' · 超重 ${overPct.toStringAsFixed(1)}%' : (loss > 0 ? ' · 完成时预计损耗 ${loss.toStringAsFixed(2)} kg' : '')}\n'
                            '草稿已本地缓存，可改天继续；删箱需确认。',
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
              onPrimary: _busy || !_ready ? null : _confirmGate,
              primaryBusy: _busy,
              busyLabel: '处理中…',
            )
          else if (_step == 0)
            FormStickyActions(
              primaryLabel: '下一步',
              onPrimary: _busy ? null : _goPreview,
            )
          else
            SafeArea(
              top: false,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
                child: Column(
                  children: [
                    Row(
                      children: [
                        OutlinedButton(
                          onPressed: _busy
                              ? null
                              : () => setState(() {
                                    _step = 0;
                                    _msg = '';
                                    _msgIsError = false;
                                  }),
                          child: const Text('修改'),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: FilledButton(
                            onPressed: _busy ? null : _submitBatch,
                            child: Text(_busy ? '处理中…' : '提交本批入库'),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    SizedBox(
                      width: double.infinity,
                      child: OutlinedButton(
                        onPressed: _busy ? null : _completeBatch,
                        child: const Text('完成本批分箱'),
                      ),
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }
}
