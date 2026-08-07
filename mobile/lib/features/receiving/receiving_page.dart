import 'dart:async';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';
import '../../widgets/trace_code_field.dart';

/// 现场过磅收货：入厂/入库双模 → 批号+拍照 → 质检 → 确认出码 → 推仓管
/// [initialReceiveKind] / [lockKind]：供主壳「+」快捷入口锁定过磅入厂或入库表单
class ReceivingPage extends StatefulWidget {
  const ReceivingPage({
    super.key,
    this.initialReceiveKind,
    this.lockKind = false,
    this.popOnCreated = false,
  });

  /// `gate` 过磅入厂 · `stockin` 过磅入库
  final String? initialReceiveKind;
  final bool lockKind;
  /// 从「+」快捷创建成功后 pop(true)
  final bool popOnCreated;

  @override
  State<ReceivingPage> createState() => _ReceivingPageState();
}

class _ReceivingPageState extends State<ReceivingPage> with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  List<dynamic> _tickets = [];
  List<dynamic> _farmerHits = [];
  List<dynamic> _purTasks = [];
  List<dynamic> _varieties = [];
  int? _farmerId;
  int? _varietyId;
  int _productId = 1;
  late String _receiveKind;
  String _channel = 'internal';
  String _grade = 'A';
  String _coldStore = 'fresh';
  bool _kindLocked = false;
  final _farmerSearch = TextEditingController();
  final _gross = TextEditingController();
  final _deductRate = TextEditingController(text: '5');
  final _reject = TextEditingController(text: '0');
  final _unitPrice = TextEditingController(text: '1.2');
  final _netWeight = TextEditingController();
  final _bagQty = TextEditingController();
  final _origin = TextEditingController();
  final _plate = TextEditingController();
  final _recvAddr = TextEditingController();
  final _partyName = TextEditingController();
  final _partyMobile = TextEditingController();
  final _batchNo = TextEditingController();
  final _remark = TextEditingController();
  final _freight = TextEditingController(text: '0');
  final _loadingFee = TextEditingController(text: '0');
  final _weighFee = TextEditingController(text: '0');
  final List<String> _photoUrls = [];
  String _msg = '';
  bool _loading = false;
  bool _batchOk = false;
  bool _searchingFarmer = false;
  String _boundFarmerName = '';
  Timer? _searchDebounce;

  @override
  void initState() {
    super.initState();
    final init = widget.initialReceiveKind;
    _receiveKind = (init == 'stockin' || init == 'gate') ? init! : 'gate';
    _kindLocked = widget.lockKind && (init == 'stockin' || init == 'gate');
    _tabs = TabController(length: 3, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) => _boot());
  }

  @override
  void dispose() {
    _searchDebounce?.cancel();
    _tabs.dispose();
    _farmerSearch.dispose();
    _gross.dispose();
    _deductRate.dispose();
    _reject.dispose();
    _unitPrice.dispose();
    _netWeight.dispose();
    _bagQty.dispose();
    _origin.dispose();
    _plate.dispose();
    _recvAddr.dispose();
    _partyName.dispose();
    _partyMobile.dispose();
    _batchNo.dispose();
    _remark.dispose();
    _freight.dispose();
    _loadingFee.dispose();
    _weighFee.dispose();
    super.dispose();
  }

  Future<void> _boot() async {
    final auth = context.read<AuthState>();
    final canRecv = canAccessEmployeeModule(EmployeeModule.receiving, auth.permissions, auth.roles);
    final canWh = canAccessEmployeeModule(EmployeeModule.warehouse, auth.permissions, auth.roles);
    if (!canRecv && !canWh) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('无过磅收货权限')));
        Navigator.of(context).pop();
      }
      return;
    }
    // 仅仓管：快捷入库；无采购过磅权限时不可入厂
    if (!canRecv && canWh) {
      _receiveKind = 'stockin';
      _kindLocked = true;
    }
    if (_kindLocked && _receiveKind == 'gate' && !canRecv) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('无过磅入厂权限')));
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
      api.get('/purchase/tasks?page_size=50'),
      api.get('/purchase/weigh-varieties?status=active'),
    ]);
    if (!mounted) return;
    final varietyRes = results[2];
    setState(() {
      _loading = false;
      _tickets = ApiClient.listOf(results[0].data);
      _purTasks = ApiClient.listOf(results[1].data);
      _varieties = varietyRes.ok ? ApiClient.listOf(varietyRes.data) : [];
      if (!varietyRes.ok && _msg.isEmpty) {
        _msg = '品种加载失败：${varietyRes.msg}';
      }
      if (_varietyId == null && _varieties.isNotEmpty) {
        _applyVariety(Map<String, dynamic>.from(_varieties.first as Map));
      }
    });
  }

  void _applyFarmer(Map<String, dynamic> m) {
    _farmerId = (m['id'] as num?)?.toInt();
    _partyName.text = m['name']?.toString() ?? '';
    _partyMobile.text = m['mobile']?.toString() ?? '';
    _origin.text = m['origin']?.toString() ?? '';
    final price = (m['default_unit_price'] as num?)?.toDouble();
    if (price != null && price > 0) {
      _unitPrice.text = price.toString();
    }
    _farmerSearch.text = '${m['name'] ?? ''} ${m['mobile'] ?? ''}'.trim();
  }

  void _clearFarmerLink() {
    _farmerId = null;
    _farmerSearch.clear();
    _farmerHits = [];
  }

  void _applyVariety(Map<String, dynamic> m) {
    _varietyId = (m['id'] as num?)?.toInt();
    final pid = (m['default_product_id'] as num?)?.toInt() ?? 0;
    _productId = pid > 0 ? pid : 1;
  }

  void _onFarmerSearchChanged(String q) {
    _searchDebounce?.cancel();
    _searchDebounce = Timer(const Duration(milliseconds: 350), () => _searchFarmers(q));
  }

  Future<void> _searchFarmers(String raw) async {
    final q = raw.trim();
    if (q.isEmpty) {
      setState(() {
        _farmerHits = [];
        _searchingFarmer = false;
      });
      return;
    }
    setState(() => _searchingFarmer = true);
    final api = context.read<AuthState>().api;
    // 仅手机号 / 姓名模糊；不再按农户 ID 搜索
    final String path;
    if (RegExp(r'\d').hasMatch(q)) {
      path = '/purchase/farmers?mobile=${Uri.encodeQueryComponent(q)}&page_size=20';
    } else {
      path = '/purchase/farmers?name=${Uri.encodeQueryComponent(q)}&page_size=20';
    }
    final r = await api.get(path);
    if (!mounted) return;
    setState(() {
      _searchingFarmer = false;
      _farmerHits = r.ok ? ApiClient.listOf(r.data) : [];
      if (!r.ok) _msg = '农户搜索失败：${r.msg}';
    });
  }

  Future<void> _showOnsiteFarmerDialog() async {
    final name = TextEditingController(text: _partyName.text);
    final mobile = TextEditingController(text: _partyMobile.text.isNotEmpty ? _partyMobile.text : _farmerSearch.text);
    final origin = TextEditingController(text: _origin.text);
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('现场录入农户'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(controller: name, decoration: const InputDecoration(labelText: '姓名 *')),
              TextField(controller: mobile, decoration: const InputDecoration(labelText: '手机号'), keyboardType: TextInputType.phone),
              TextField(controller: origin, decoration: const InputDecoration(labelText: '产地地址')),
              const SizedBox(height: 8),
              const Text('将写入平台共享农户档案，供全员复用', style: TextStyle(fontSize: 12, color: Colors.black54)),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('保存并关联')),
        ],
      ),
    );
    if (ok != true || !mounted) {
      name.dispose();
      mobile.dispose();
      origin.dispose();
      return;
    }
    if (name.text.trim().isEmpty) {
      setState(() => _msg = '农户姓名必填');
      name.dispose();
      mobile.dispose();
      origin.dispose();
      return;
    }
    final r = await context.read<AuthState>().api.post('/purchase/farmers', {
      'name': name.text.trim(),
      'mobile': mobile.text.trim(),
      'origin': origin.text.trim(),
    });
    name.dispose();
    mobile.dispose();
    origin.dispose();
    if (!mounted) return;
    if (!r.ok || r.data is! Map) {
      setState(() => _msg = '建档失败：${r.msg}');
      return;
    }
    setState(() {
      _applyFarmer(Map<String, dynamic>.from(r.data as Map));
      _farmerHits = [];
      _msg = '已现场建档并关联 #$_farmerId';
    });
  }

  Future<void> _validateBatch() async {
    final code = _batchNo.text.trim().toUpperCase();
    _batchNo.text = code;
    if (code.isEmpty) {
      setState(() {
        _batchOk = false;
        _boundFarmerName = '';
        _msg = '请录入溯源批号';
      });
      return;
    }
    final r = await context.read<AuthState>().api.post('/purchase/trace-batch-codes/validate', {
      'code': code,
      'receive_kind': _receiveKind,
    });
    if (!mounted) return;
    if (!r.ok || r.data is! Map) {
      setState(() {
        _batchOk = false;
        _boundFarmerName = '';
        _msg = r.msg;
      });
      return;
    }
    final m = Map<String, dynamic>.from(r.data as Map);
    setState(() {
      _batchOk = true;
      _applyBatchBinding(m);
      final bound = _boundFarmerName;
      if (_receiveKind == 'stockin') {
        _msg = bound.isEmpty
            ? '批号校验通过（入库）'
            : '批号校验通过 · 已同步农户/产品：$bound';
      } else {
        _msg = bound.isEmpty ? '批号校验通过（可入厂占用）' : '批号校验通过 · 已同步 $bound';
      }
    });
  }

  /// 校验通过后把入厂绑定的农户/产地/品种等写入表单
  void _applyBatchBinding(Map<String, dynamic> m) {
    final name = (m['farmer_name'] ?? m['party_name'] ?? '').toString().trim();
    final mobile = (m['party_mobile'] ?? '').toString().trim();
    final origin = (m['origin'] ?? '').toString().trim();
    final fid = (m['farmer_id'] as num?)?.toInt() ?? 0;
    _boundFarmerName = name;

    if (fid > 0) {
      _farmerId = fid;
      _farmerSearch.text = '$name ${mobile.isNotEmpty ? mobile : ''}'.trim();
    }
    if (name.isNotEmpty) _partyName.text = name;
    if (mobile.isNotEmpty) _partyMobile.text = mobile;
    if (origin.isNotEmpty) _origin.text = origin;

    final plate = (m['plate_no'] ?? '').toString().trim();
    final recv = (m['receive_address'] ?? '').toString().trim();
    if (plate.isNotEmpty) _plate.text = plate;
    if (recv.isNotEmpty) _recvAddr.text = recv;

    final ch = (m['channel'] ?? '').toString();
    if (ch == 'internal' || ch == 'external') _channel = ch;

    final grade = (m['grade'] ?? '').toString();
    if (grade == 'A' || grade == 'B' || grade == 'C') _grade = grade;

    final price = (m['unit_price'] as num?)?.toDouble();
    if (price != null && price > 0) _unitPrice.text = price.toString();

    final pid = (m['product_id'] as num?)?.toInt() ?? 0;
    if (pid > 0) _productId = pid;

    final vid = (m['variety_id'] as num?)?.toInt();
    final vname = (m['variety'] ?? '').toString().trim();
    if (vid != null && vid > 0) {
      final hit = _varieties
          .cast<dynamic>()
          .map((e) => Map<String, dynamic>.from(e as Map))
          .where((x) => (x['id'] as num?)?.toInt() == vid);
      if (hit.isNotEmpty) {
        _applyVariety(hit.first);
      } else {
        _varietyId = vid;
      }
    } else if (vname.isNotEmpty && _varieties.isNotEmpty) {
      final hit = _varieties
          .cast<dynamic>()
          .map((e) => Map<String, dynamic>.from(e as Map))
          .where((x) => '${x['name']}' == vname || '${x['code']}' == vname);
      if (hit.isNotEmpty) _applyVariety(hit.first);
    }
  }

  void _onReceiveKindChanged(String kind) {
    setState(() {
      _receiveKind = kind;
      _batchOk = false;
      _boundFarmerName = '';
      if (kind == 'stockin') {
        _farmerId = null;
        _farmerSearch.clear();
        _farmerHits = [];
      }
    });
  }

  Future<void> _takePhoto() async {
    if (_photoUrls.length >= 3) {
      setState(() => _msg = '最多 3 张现场照片');
      return;
    }
    try {
      final picker = ImagePicker();
      final file = await picker.pickImage(source: ImageSource.camera, imageQuality: 85);
      if (file == null) return;
      final bytes = await file.readAsBytes();
      setState(() => _msg = '上传中…');
      final r = await context.read<AuthState>().api.postMultipart(
            '/biz/uploads',
            bytes,
            filename: file.name.isEmpty ? 'site.jpg' : file.name,
          );
      if (!mounted) return;
      if (!r.ok || r.data is! Map) {
        setState(() => _msg = '上传失败：${r.msg}');
        return;
      }
      final url = (r.data as Map)['url']?.toString() ?? (r.data as Map)['file_url']?.toString() ?? '';
      if (url.isEmpty) {
        setState(() => _msg = '上传无返回 URL');
        return;
      }
      setState(() {
        _photoUrls.add(url);
        _msg = '已上传 ${_photoUrls.length} 张';
      });
    } catch (e) {
      setState(() => _msg = '拍照失败：$e');
    }
  }

  Future<void> _create() async {
    if (!_batchOk) {
      await _validateBatch();
      if (!_batchOk) return;
    }
    if (_photoUrls.isEmpty) {
      setState(() => _msg = '请现场拍照留底');
      return;
    }
    if (_receiveKind == 'gate' && (_farmerId == null || _farmerId! <= 0) && _partyName.text.trim().isEmpty) {
      setState(() => _msg = '入厂须搜索关联农户，或现场录入');
      return;
    }
    String varietyName = '鲜木薯';
    for (final e in _varieties) {
      final m = Map<String, dynamic>.from(e as Map);
      if ((m['id'] as num?)?.toInt() == _varietyId) {
        varietyName = m['name']?.toString() ?? varietyName;
        break;
      }
    }
    final body = <String, dynamic>{
      'receive_kind': _receiveKind,
      'batch_no': _batchNo.text.trim().toUpperCase(),
      'channel': _channel,
      'product_id': _productId,
      if (_varietyId != null) 'variety_id': _varietyId,
      'variety': varietyName,
      'grade': _grade,
      'biz_date': DateTime.now().toIso8601String().substring(0, 10),
      'source_type': 'self',
      'image_url': _photoUrls.first,
      'image_urls': _photoUrls,
      'remark': _remark.text.trim(),
    };
    if (_receiveKind == 'gate') {
      body.addAll({
        'farmer_id': _farmerId ?? 0,
        'party_name': _partyName.text.trim(),
        'party_mobile': _partyMobile.text.trim(),
        'origin': _origin.text.trim(),
      });
    }
    if (_receiveKind == 'gate') {
      final gross = double.tryParse(_gross.text) ?? 0;
      if (gross <= 0) {
        setState(() => _msg = '请输入入场重量');
        return;
      }
      body.addAll({
        'gross_weight': gross,
        'deduct_rate': double.tryParse(_deductRate.text) ?? 0,
        'reject_weight': double.tryParse(_reject.text) ?? 0,
        'unit_price': double.tryParse(_unitPrice.text) ?? 0,
        'plate_no': _plate.text.trim(),
        'receive_address': _recvAddr.text.trim(),
        'freight_fee': double.tryParse(_freight.text) ?? 0,
        'loading_fee': double.tryParse(_loadingFee.text) ?? 0,
        'weigh_fee': double.tryParse(_weighFee.text) ?? 0,
      });
    } else {
      final net = double.tryParse(_netWeight.text) ?? 0;
      if (net <= 0) {
        setState(() => _msg = '请输入入库重量');
        return;
      }
      body.addAll({
        'net_weight': net,
        'bag_qty': double.tryParse(_bagQty.text) ?? 0,
        'cold_store_type': _coldStore,
      });
    }
    final r = await context.read<AuthState>().api.post('/purchase/weigh-tickets', body);
    if (!mounted) return;
    setState(() => _msg = r.ok ? '草稿已创建 #${(r.data is Map) ? (r.data as Map)['doc_no'] : ''}' : r.msg);
    if (r.ok) {
      if (widget.popOnCreated) {
        Navigator.of(context).pop(true);
        return;
      }
      _gross.clear();
      _netWeight.clear();
      _batchNo.clear();
      _photoUrls.clear();
      _batchOk = false;
      _boundFarmerName = '';
      await _refresh();
      _tabs.animateTo(1);
    }
  }

  Future<void> _qc(Map row, {required bool pass}) async {
    final id = (row['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/purchase/weigh-tickets/$id/qc', {
      'qc_result': pass ? 'pass' : 'fail',
      if (pass) 'grade': _grade,
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? (pass ? '质检合格' : '质检不合格') : r.msg)));
    if (r.ok) await _refresh();
  }

  Future<void> _confirm(Map row) async {
    final id = (row['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/purchase/weigh-tickets/$id/confirm', {
      'confirmed': true,
      'gross_weight': row['gross_weight'],
      'deduct_rate': row['deduct_rate'],
      'deduct_weight': row['deduct_weight'],
      'net_weight': row['net_weight'],
      'grade': row['grade'] ?? _grade,
    });
    if (!mounted) return;
    final trace = r.ok && r.data is Map ? (r.data as Map)['trace_code'] : '';
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已出码并推仓管 · $trace' : r.msg)));
    if (r.ok) await _refresh();
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

  String _kindLabel(String k) => k == 'stockin' ? '入库' : '入厂';

  List<Widget> _farmerSection() {
    return [
      const Text('农户（平台共享）', style: TextStyle(fontWeight: FontWeight.w600)),
      const SizedBox(height: 4),
      TextField(
        controller: _farmerSearch,
        decoration: InputDecoration(
          labelText: '手机号 / 姓名（模糊搜索）',
          hintText: '输入后自动匹配共享农户',
          suffixIcon: _searchingFarmer
              ? const Padding(
                  padding: EdgeInsets.all(12),
                  child: SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2)),
                )
              : IconButton(icon: const Icon(Icons.search), onPressed: () => _searchFarmers(_farmerSearch.text)),
        ),
        onChanged: (v) {
          if (_farmerId != null) {
            setState(() => _farmerId = null);
          }
          _onFarmerSearchChanged(v);
        },
        onSubmitted: _searchFarmers,
      ),
      if (_farmerHits.isNotEmpty)
        Card(
          margin: const EdgeInsets.only(top: 6, bottom: 6),
          child: Column(
            children: [
              for (final e in _farmerHits.take(8))
                ListTile(
                  dense: true,
                  title: Text('${(e as Map)['name'] ?? ''}'),
                  subtitle: Text('${e['mobile'] ?? ''} · ${e['origin'] ?? ''}'),
                  trailing: const Icon(Icons.check_circle_outline),
                  onTap: () => setState(() {
                    _applyFarmer(Map<String, dynamic>.from(e));
                    _farmerHits = [];
                  }),
                ),
            ],
          ),
        ),
      if (_farmerSearch.text.trim().isNotEmpty && !_searchingFarmer && _farmerHits.isEmpty && _farmerId == null)
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 6),
          child: Row(
            children: [
              const Expanded(child: Text('未找到匹配农户', style: TextStyle(color: Colors.orange))),
              FilledButton.tonal(onPressed: _showOnsiteFarmerDialog, child: const Text('现场录入')),
            ],
          ),
        ),
      if (_farmerId != null)
        Chip(
          avatar: const Icon(Icons.link, size: 16),
          label: Text('已关联农户 #$_farmerId ${_partyName.text}'),
          onDeleted: () => setState(_clearFarmerLink),
        ),
      TextField(controller: _partyName, decoration: const InputDecoration(labelText: '姓名（可改快照）')),
      TextField(controller: _partyMobile, decoration: const InputDecoration(labelText: '电话'), keyboardType: TextInputType.phone),
      TextField(controller: _origin, decoration: InputDecoration(labelText: _receiveKind == 'stockin' ? '产地地址' : '产地')),
      Align(
        alignment: Alignment.centerLeft,
        child: TextButton.icon(
          onPressed: _showOnsiteFarmerDialog,
          icon: const Icon(Icons.person_add_alt),
          label: const Text('现场新建农户'),
        ),
      ),
    ];
  }

  List<Widget> _batchSection() {
    return [
      const Text('溯源批号', style: TextStyle(fontWeight: FontWeight.w600)),
      const SizedBox(height: 6),
      TraceCodeField(
        controller: _batchNo,
        label: '溯源批号',
        hint: _receiveKind == 'stockin' ? '扫入厂已绑定的批号' : '点击输入，或点右侧图标扫码',
        validated: _batchOk,
        scannerTitle: '扫描溯源批号',
        onChanged: (_) => setState(() {
          _batchOk = false;
          _boundFarmerName = '';
        }),
        onEditingComplete: _validateBatch,
        onScanned: (_) async {
          setState(() {
            _batchOk = false;
            _boundFarmerName = '';
            _msg = '已扫到批号，校验中…';
          });
          await _validateBatch();
        },
      ),
      if (_receiveKind == 'stockin' && _batchOk && _boundFarmerName.isNotEmpty)
        Padding(
          padding: const EdgeInsets.only(top: 8),
          child: Chip(
            avatar: const Icon(Icons.agriculture, size: 16),
            label: Text('关联农户（入厂绑定）$_boundFarmerName'),
          ),
        ),
    ];
  }

  List<Widget> _formFields() {
    return [
      if (!_kindLocked)
        SegmentedButton<String>(
          segments: const [
            ButtonSegment(value: 'gate', label: Text('过磅入厂'), icon: Icon(Icons.login, size: 18)),
            ButtonSegment(value: 'stockin', label: Text('过磅入库'), icon: Icon(Icons.warehouse_outlined, size: 18)),
          ],
          selected: {_receiveKind},
          onSelectionChanged: (s) => _onReceiveKindChanged(s.first),
        )
      else
        ListTile(
          contentPadding: EdgeInsets.zero,
          leading: Icon(_receiveKind == 'stockin' ? Icons.warehouse_outlined : Icons.login),
          title: Text(_receiveKind == 'stockin' ? '过磅入库' : '过磅入厂'),
          subtitle: Text(
            _receiveKind == 'gate' ? '入厂须绑定农户，占用溯源批号' : '入库凭溯源批号跟踪，自动带出入厂关联农户',
          ),
        ),
      if (!_kindLocked)
        Padding(
          padding: const EdgeInsets.only(top: 6, bottom: 4),
          child: Text(
            _receiveKind == 'gate' ? '入厂须绑定农户，占用溯源批号' : '入库凭溯源批号跟踪，自动带出入厂关联农户',
            style: const TextStyle(fontSize: 12, color: Colors.black54),
          ),
        ),
      const SizedBox(height: 8),
      ..._batchSection(),
      if (_receiveKind == 'gate') ...[
        const SizedBox(height: 12),
        ..._farmerSection(),
      ],
      const SizedBox(height: 8),
      if (_varieties.isEmpty)
        const Padding(
          padding: EdgeInsets.symmetric(vertical: 8),
          child: Text('暂无过磅品种，请先在后台配置', style: TextStyle(color: Colors.orange)),
        )
      else
        DropdownButtonFormField<int>(
          value: _varietyId,
          decoration: const InputDecoration(labelText: '品种/产品'),
          items: _varieties.map((e) {
            final m = Map<String, dynamic>.from(e as Map);
            return DropdownMenuItem(value: (m['id'] as num?)?.toInt(), child: Text('${m['name'] ?? m['code']}'));
          }).toList(),
          onChanged: (v) {
            if (v == null) return;
            final hit = _varieties.cast<dynamic>().map((e) => Map<String, dynamic>.from(e as Map)).where((m) => (m['id'] as num?)?.toInt() == v);
            if (hit.isNotEmpty) setState(() => _applyVariety(hit.first));
          },
        ),
      if (_receiveKind == 'gate') ...[
        SegmentedButton<String>(
          segments: const [
            ButtonSegment(value: 'internal', label: Text('厂内秤')),
            ButtonSegment(value: 'external', label: Text('外磅单')),
          ],
          selected: {_channel},
          onSelectionChanged: (s) => setState(() => _channel = s.first),
        ),
        TextField(controller: _gross, decoration: const InputDecoration(labelText: '入场重量(kg)'), keyboardType: TextInputType.number),
        TextField(controller: _deductRate, decoration: const InputDecoration(labelText: '扣损率(% 或小数)'), keyboardType: TextInputType.number),
        TextField(controller: _reject, decoration: const InputDecoration(labelText: '不合格重量(kg)'), keyboardType: TextInputType.number),
        TextField(controller: _unitPrice, decoration: const InputDecoration(labelText: '单价(元/kg)'), keyboardType: TextInputType.number),
        TextField(controller: _plate, decoration: const InputDecoration(labelText: '车牌号')),
        TextField(controller: _recvAddr, decoration: const InputDecoration(labelText: '收货地址')),
        TextField(controller: _freight, decoration: const InputDecoration(labelText: '运费'), keyboardType: TextInputType.number),
        TextField(controller: _loadingFee, decoration: const InputDecoration(labelText: '装卸费'), keyboardType: TextInputType.number),
        TextField(controller: _weighFee, decoration: const InputDecoration(labelText: '过磅费'), keyboardType: TextInputType.number),
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
      ] else ...[
        TextField(controller: _netWeight, decoration: const InputDecoration(labelText: '重量(kg)'), keyboardType: TextInputType.number),
        TextField(controller: _bagQty, decoration: const InputDecoration(labelText: '袋数'), keyboardType: TextInputType.number),
        DropdownButtonFormField<String>(
          value: _coldStore,
          decoration: const InputDecoration(labelText: '冷库类型'),
          items: const [
            DropdownMenuItem(value: 'fresh', child: Text('保鲜库')),
            DropdownMenuItem(value: 'semi', child: Text('半成品库')),
            DropdownMenuItem(value: 'fg', child: Text('成品库')),
          ],
          onChanged: (v) => setState(() => _coldStore = v ?? 'fresh'),
        ),
      ],
      const SizedBox(height: 8),
      if (_receiveKind == 'stockin')
        TextField(controller: _origin, decoration: const InputDecoration(labelText: '产地地址')),
      Row(
        children: [
          FilledButton.tonalIcon(onPressed: _takePhoto, icon: const Icon(Icons.photo_camera), label: const Text('现场拍照')),
          const SizedBox(width: 12),
          Text('已拍 ${_photoUrls.length}/3', style: const TextStyle(color: Colors.black54)),
        ],
      ),
      if (_photoUrls.isNotEmpty)
        Wrap(
          spacing: 8,
          children: [
            for (var i = 0; i < _photoUrls.length; i++)
              Chip(
                label: Text('图${i + 1}'),
                onDeleted: () => setState(() => _photoUrls.removeAt(i)),
              ),
          ],
        ),
      TextField(controller: _remark, decoration: const InputDecoration(labelText: '备注')),
      const SizedBox(height: 12),
      FilledButton(onPressed: _create, child: Text('创建${_kindLabel(_receiveKind)}草稿')),
      if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 12), child: Text(_msg)),
    ];
  }

  @override
  Widget build(BuildContext context) {
    final title = _kindLocked
        ? (_receiveKind == 'stockin' ? '过磅入库' : '过磅入厂')
        : '过磅收货';
    return Scaffold(
      appBar: AppBar(
        title: Text(title),
        bottom: TabBar(controller: _tabs, tabs: const [
          Tab(text: '新建'),
          Tab(text: '单据'),
          Tab(text: '任务'),
        ]),
        actions: [IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh))],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tabs,
              children: [
                ListView(padding: const EdgeInsets.all(16), children: _formFields()),
                RefreshIndicator(
                  onRefresh: _refresh,
                  child: ListView.builder(
                    padding: const EdgeInsets.all(12),
                    itemCount: _tickets.length,
                    itemBuilder: (context, i) {
                      final m = Map<String, dynamic>.from(_tickets[i] as Map);
                      final st = m['status']?.toString() ?? '';
                      final kind = m['receive_kind']?.toString() ?? '';
                      return Card(
                        child: Padding(
                          padding: const EdgeInsets.all(12),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Row(
                                children: [
                                  Expanded(child: Text('${m['doc_no']}', style: const TextStyle(fontWeight: FontWeight.bold))),
                                  Chip(label: Text(_kindLabel(kind), style: const TextStyle(fontSize: 11)), visualDensity: VisualDensity.compact),
                                  Chip(
                                    label: Text(st, style: const TextStyle(fontSize: 11, color: Colors.white)),
                                    backgroundColor: _statusColor(st),
                                    visualDensity: VisualDensity.compact,
                                  ),
                                ],
                              ),
                              Text('${m['party_name'] ?? m['farmer_name'] ?? ''} · ${m['product_name'] ?? m['variety'] ?? ''}'),
                              Text('批号 ${m['batch_no'] ?? '-'}'),
                              if (kind == 'stockin')
                                Text('重 ${m['net_weight']} kg · 冷库 ${m['cold_store_type'] ?? '-'}')
                              else
                                Text('净 ${m['net_weight']} kg · 结算 ${m['settle_amount'] ?? '-'}'),
                              if ((m['trace_code']?.toString() ?? '').isNotEmpty)
                                Text('溯源 ${m['trace_code']}', style: const TextStyle(color: Colors.teal)),
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
                      const Text('现场采购任务', style: TextStyle(fontWeight: FontWeight.bold)),
                      const SizedBox(height: 8),
                      if (_purTasks.isEmpty) const Padding(padding: EdgeInsets.all(24), child: Center(child: Text('暂无采购任务'))),
                      ..._purTasks.map((e) {
                        final m = Map<String, dynamic>.from(e as Map);
                        final st = m['status']?.toString() ?? '';
                        return Card(
                          child: ListTile(
                            title: Text('${m['title'] ?? m['doc_no'] ?? m['id']}'),
                            subtitle: Text('$st · 数量 ${m['qty'] ?? '-'}'),
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
