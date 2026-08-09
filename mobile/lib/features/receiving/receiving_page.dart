import 'dart:async';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';
import '../../widgets/form_row.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_sticky_actions.dart';
import '../../widgets/trace_code_field.dart';
import 'gate_inbound_wizard.dart';

/// 现场过磅收货：选类型 → 填表预览 → 确认创建（批号即溯源码，绑定农户并推仓管）
/// [initialReceiveKind] / [lockKind]：供主壳「+」快捷入口锁定过磅入厂或入库表单
class ReceivingPage extends StatefulWidget {
  const ReceivingPage({
    super.key,
    this.initialReceiveKind,
    this.lockKind = false,
    this.popOnCreated = false,
    this.asTab = false,
  });

  /// `gate` 过磅入厂 · `stockin` 过磅入库
  final String? initialReceiveKind;
  final bool lockKind;
  /// 从「+」快捷创建成功后 pop(true)
  final bool popOnCreated;
  /// 作为产线壳 Tab 时隐藏标题栏。
  final bool asTab;

  @override
  State<ReceivingPage> createState() => _ReceivingPageState();
}

/// 收货首页四入口：入厂 / 入库 / 单据 / 任务（无顶栏 Tab）
enum _RecvSection { home, gate, stockin, tickets, tasks }

class _ReceivingPageState extends State<ReceivingPage> {
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
  _RecvSection _section = _RecvSection.home;
  /// 入库：0 填表 · 1 预览
  int _stockinStep = 0;
  /// 重建入厂向导（回选择页 / 提交成功后）
  int _formEpoch = 0;
  bool _canRecv = true;
  bool _canWh = true;
  final _farmerSearch = TextEditingController();
  final _gross = TextEditingController();
  final _deductRate = TextEditingController(text: '5');
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
  /// 底部提示是否为错误（必填未填、校验失败等）
  bool _msgIsError = false;
  bool _loading = false;
  bool _batchOk = false;
  bool _searchingFarmer = false;
  String _boundFarmerName = '';
  Timer? _searchDebounce;
  /// 单据列表日期范围（默认近 3 日：今天-2 ～ 今天）
  late DateTime _ticketDateFrom;
  late DateTime _ticketDateTo;
  /// 单据列表展开详情
  final Set<int> _expandedTicketIds = {};
  final Map<int, Map<String, dynamic>> _ticketDetails = {};
  final Set<int> _ticketDetailLoading = {};

  /// 必填未填 / 校验拦截：底部文案 + SnackBar，避免用户漏看。
  void _promptRequired(String msg) {
    if (!mounted) return;
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
        duration: const Duration(seconds: 3),
      ),
    );
  }

  static DateTime _dateOnly(DateTime d) => DateTime(d.year, d.month, d.day);

  void _resetTicketDateDefault() {
    final today = _dateOnly(DateTime.now());
    _ticketDateFrom = today.subtract(const Duration(days: 2));
    _ticketDateTo = today;
  }

  String _fmtYmd(DateTime d) =>
      '${d.year.toString().padLeft(4, '0')}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';

  String _fmtMd(DateTime d) =>
      '${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';

  /// 结算金额展示，默认单位「元人民币」。
  String _fmtSettleMoney(dynamic v, {String unit = '元人民币'}) {
    if (v == null) return '-';
    final s = v.toString().trim();
    if (s.isEmpty || s == '-') return '-';
    final n = v is num ? v.toDouble() : double.tryParse(s);
    if (n == null) return '$s $unit';
    final fixed = n == n.roundToDouble() ? n.toStringAsFixed(0) : n.toStringAsFixed(2);
    return '$fixed $unit';
  }

  String _processActionLabel(Map log) {
    final labeled = log['action_label']?.toString().trim() ?? '';
    if (labeled.isNotEmpty) return labeled;
    switch ((log['action'] ?? '').toString().toLowerCase()) {
      case 'create':
        return '建单提交';
      case 'assign':
        return '指派/认领';
      case 'warehouse_confirm':
      case 'stock_in':
        return '仓管确认入库';
      case 'warehouse_return':
        return '仓管退回采购';
      case 'settle_pay':
      case 'settle_paid':
        return '财务付款关单';
      case 'confirm':
        return '确认出码';
      case 'reject':
        return '驳回';
      case 'comment':
        return '备注';
      default:
        final a = log['action']?.toString() ?? '';
        return a.isEmpty ? '处理' : a;
    }
  }

  /// 操作人（谁做了这一步）
  String _processOperator(Map log) {
    final from = (log['from_name'] ?? '').toString().trim();
    final to = (log['to_name'] ?? '').toString().trim();
    final act = (log['action'] ?? '').toString().toLowerCase();
    // 指派类：from 是操作者；建单：from 是提交人
    if (from.isNotEmpty) return from;
    if (act == 'create' && to.isNotEmpty) return to;
    if (to.isNotEmpty) return to;
    return '系统';
  }

  /// 本步交予的下一处理人（若有）
  String? _processHandoff(Map log) {
    final from = (log['from_name'] ?? '').toString().trim();
    final to = (log['to_name'] ?? '').toString().trim();
    if (to.isEmpty || to == from) return null;
    return to;
  }

  Future<void> _toggleTicketExpand(Map<String, dynamic> row) async {
    final id = (row['id'] as num?)?.toInt();
    if (id == null) return;
    final expanding = !_expandedTicketIds.contains(id);
    setState(() {
      if (expanding) {
        _expandedTicketIds.add(id);
      } else {
        _expandedTicketIds.remove(id);
      }
    });
    if (!expanding || _ticketDetails.containsKey(id) || _ticketDetailLoading.contains(id)) return;
    setState(() => _ticketDetailLoading.add(id));
    final r = await context.read<AuthState>().api.get('/purchase/weigh-tickets/$id');
    if (!mounted) return;
    setState(() {
      _ticketDetailLoading.remove(id);
      if (r.ok && r.data is Map) {
        _ticketDetails[id] = Map<String, dynamic>.from(r.data as Map);
      }
    });
    if (!r.ok && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(_batchErrorMessage(r.msg))));
    }
  }

  String _batchErrorMessage(String code) {
    final c = code.trim();
    switch (c) {
      case 'BATCH_CODE_USED':
        return '该溯源码已被绑定，不可再用';
      case 'BATCH_CODE_RESERVED':
        return '该溯源码正被他人占用，请换码或稍后再试';
      case 'BATCH_CODE_VOID':
        return '该溯源码已作废';
      case 'BATCH_CODE_INVALID':
        return '溯源码格式无效';
      case 'BATCH_CODE_NOT_FOUND':
        return '溯源码不存在';
      case 'BATCH_CODE_UNAVAILABLE':
        return '溯源码当前不可用';
      case 'GATE_BINDING_REQUIRED':
        return '请先完成入厂绑定后再入库';
      case 'DATE_RANGE_TOO_LARGE':
        return '查询跨度不能超过 31 天';
      case 'DATE_RANGE_INVALID':
        return '日期范围无效';
      default:
        return c.isEmpty ? '操作失败' : c;
    }
  }

  @override
  void initState() {
    super.initState();
    _resetTicketDateDefault();
    final init = widget.initialReceiveKind;
    final locked = widget.lockKind && (init == 'stockin' || init == 'gate');
    _receiveKind = (init == 'stockin' || init == 'gate') ? init! : 'gate';
    _kindLocked = locked;
    if (locked) {
      _section = init == 'stockin' ? _RecvSection.stockin : _RecvSection.gate;
    }
    WidgetsBinding.instance.addPostFrameCallback((_) => _boot());
  }

  @override
  void dispose() {
    _searchDebounce?.cancel();
    _farmerSearch.dispose();
    _gross.dispose();
    _deductRate.dispose();
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
    _canRecv = canRecv;
    _canWh = canWh;
    if (!canRecv && !canWh) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('无过磅收货权限')));
        Navigator.of(context).pop();
      }
      return;
    }
    // 仅仓管：快捷入库；无采购过磅权限时不可入厂
    if (!canRecv && canWh) {
      setState(() {
        _receiveKind = 'stockin';
        _kindLocked = true;
        _section = _RecvSection.stockin;
      });
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
    final q = 'page_size=50&date_from=${_fmtYmd(_ticketDateFrom)}&date_to=${_fmtYmd(_ticketDateTo)}';
    final results = await Future.wait([
      api.get('/purchase/weigh-tickets?$q'),
      api.get('/purchase/tasks?page_size=50'),
      api.get('/purchase/weigh-varieties?status=active'),
    ]);
    if (!mounted) return;
    final varietyRes = results[2];
    final ticketRes = results[0];
    setState(() {
      _loading = false;
      if (ticketRes.ok) {
        _tickets = ApiClient.listOf(ticketRes.data);
        // 列表刷新后详情按需重拉，避免处理流水过期
        _ticketDetails.clear();
      } else {
        _tickets = [];
      }
      _purTasks = ApiClient.listOf(results[1].data);
      _varieties = varietyRes.ok ? ApiClient.listOf(varietyRes.data) : [];
      if (!varietyRes.ok && _msg.isEmpty) {
        _msg = '品种加载失败：${varietyRes.msg}';
      }
      if (_varietyId == null && _varieties.isNotEmpty) {
        _applyVariety(Map<String, dynamic>.from(_varieties.first as Map));
      }
    });
    // 单据列表失败仅提示，不写入首页/建单表单的 _msg（日期筛选只在「单据」页）
    if (!ticketRes.ok && ticketRes.msg.isNotEmpty && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(_batchErrorMessage(ticketRes.msg))),
      );
    }
  }

  Future<void> _pickTicketDateRange() async {
    final now = _dateOnly(DateTime.now());
    final picked = await showDateRangePicker(
      context: context,
      firstDate: now.subtract(const Duration(days: 365)),
      lastDate: now,
      initialDateRange: DateTimeRange(start: _ticketDateFrom, end: _ticketDateTo),
      helpText: '选择单据日期（最长 31 天）',
      saveText: '确定',
    );
    if (picked == null || !mounted) return;
    final from = _dateOnly(picked.start);
    final to = _dateOnly(picked.end);
    if (to.difference(from).inDays > 30) {
      _promptRequired('查询跨度不能超过 31 天');
      return;
    }
    setState(() {
      _ticketDateFrom = from;
      _ticketDateTo = to;
    });
    await _refresh();
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
    final name = '${m['name'] ?? m['code'] ?? ''}'.toLowerCase();
    if (name.contains('半成品') || name.contains('semi')) {
      _coldStore = 'semi';
    } else if (_receiveKind == 'gate') {
      _coldStore = 'fresh';
    }
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
      _promptRequired('请填写农户姓名');
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
      });
      _promptRequired('请填写溯源批号');
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
      });
      _promptRequired(_batchErrorMessage(r.msg.isNotEmpty ? r.msg : '溯源批号校验失败'));
      return;
    }
    final m = Map<String, dynamic>.from(r.data as Map);
    setState(() {
      _batchOk = true;
      _applyBatchBinding(m);
      _msgIsError = false;
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
      _stockinStep = 0;
      if (kind == 'stockin') {
        _farmerId = null;
        _farmerSearch.clear();
        _farmerHits = [];
      }
    });
  }

  void _chooseReceiveKind(String kind) {
    _onReceiveKindChanged(kind);
    setState(() {
      _section = kind == 'stockin' ? _RecvSection.stockin : _RecvSection.gate;
      _msg = '';
      _msgIsError = false;
      _formEpoch++;
    });
  }

  void _openSection(_RecvSection section) {
    setState(() {
      _section = section;
      _msg = '';
      _msgIsError = false;
    });
    if (section == _RecvSection.tickets || section == _RecvSection.tasks) {
      _refresh();
    }
  }

  void _backToKindChooser() {
    if (_kindLocked) return;
    setState(() {
      _section = _RecvSection.home;
      _stockinStep = 0;
      _msg = '';
      _msgIsError = false;
      _formEpoch++;
      _batchOk = false;
      _boundFarmerName = '';
    });
  }

  void _clearCreateForm() {
    _gross.clear();
    _netWeight.clear();
    _batchNo.clear();
    _photoUrls.clear();
    _batchOk = false;
    _boundFarmerName = '';
    _farmerId = null;
    _farmerHits = [];
    _farmerSearch.clear();
    _bagQty.clear();
    _remark.clear();
    _plate.clear();
    _recvAddr.clear();
    _partyName.clear();
    _partyMobile.clear();
    _origin.clear();
    _stockinStep = 0;
  }

  String _varietyName() {
    for (final e in _varieties) {
      final m = Map<String, dynamic>.from(e as Map);
      if ((m['id'] as num?)?.toInt() == _varietyId) {
        return m['name']?.toString() ?? m['code']?.toString() ?? '-';
      }
    }
    return '-';
  }

  String _coldStoreLabel(String v) {
    switch (v) {
      case 'semi':
        return '半成品库';
      case 'fg':
        return '成品库';
      default:
        return '原料库';
    }
  }

  String? _validateStockinForm() {
    if (_batchNo.text.trim().isEmpty) return '请填写溯源批号';
    if (!_batchOk) return '请先校验溯源批号（输入后点完成或扫码）';
    if (_varieties.isNotEmpty && _varietyId == null) return '请选择品种';
    final net = double.tryParse(_netWeight.text) ?? 0;
    if (net <= 0) return '请填写入库净重（kg）';
    if (_photoUrls.isEmpty) return '请拍摄现场照片';
    return null;
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

  Future<bool> _create({
    String? nextRole,
    String? nextNodeId,
    int? nextAssigneeUserId,
  }) async {
    if (!_batchOk) {
      await _validateBatch();
      if (!_batchOk) return false;
    }
    if (_photoUrls.isEmpty) {
      _promptRequired('请拍摄现场照片');
      return false;
    }
    if (_receiveKind == 'gate' && (_farmerId == null || _farmerId! <= 0) && _partyName.text.trim().isEmpty) {
      _promptRequired('请关联农户或填写农户姓名');
      return false;
    }
    final varietyName = _varietyName() == '-' ? '鲜木薯' : _varietyName();
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
      'activate': true,
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
        _promptRequired('请填写入场重量（kg）');
        return false;
      }
      body.addAll({
        'gross_weight': gross,
        'deduct_rate': double.tryParse(_deductRate.text) ?? 0,
        'reject_weight': 0,
        'unit_price': double.tryParse(_unitPrice.text) ?? 0,
        'plate_no': _plate.text.trim(),
        'receive_address': _recvAddr.text.trim(),
        'freight_fee': double.tryParse(_freight.text) ?? 0,
        'loading_fee': double.tryParse(_loadingFee.text) ?? 0,
        'weigh_fee': double.tryParse(_weighFee.text) ?? 0,
        'cold_store_type': _coldStore,
        if (nextRole != null && nextRole.isNotEmpty) 'next_role': nextRole,
        if (nextNodeId != null && nextNodeId.isNotEmpty) 'next_node_id': nextNodeId,
        if (nextAssigneeUserId != null && nextAssigneeUserId > 0) 'next_assignee_user_id': nextAssigneeUserId,
      });
    } else {
      final net = double.tryParse(_netWeight.text) ?? 0;
      if (net <= 0) {
        _promptRequired('请填写入库净重（kg）');
        return false;
      }
      body.addAll({
        'net_weight': net,
        'bag_qty': double.tryParse(_bagQty.text) ?? 0,
        'cold_store_type': _coldStore,
      });
    }
    final r = await context.read<AuthState>().api.post('/purchase/weigh-tickets', body);
    if (!mounted) return false;
    if (!r.ok) {
      _promptRequired(_batchErrorMessage(r.msg.isNotEmpty ? r.msg : '创建失败'));
      return false;
    }
    final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
    final docNo = data['doc_no']?.toString() ?? '';
    final trace = data['trace_code']?.toString() ?? '';
    final okMsg = trace.isNotEmpty ? '已创建并绑定 · $docNo · 溯源码 $trace' : '已创建并绑定 · $docNo';
    if (widget.popOnCreated) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(okMsg)));
        Navigator.of(context).pop(true);
      }
      return true;
    }
    _clearCreateForm();
    setState(() {
      _msg = okMsg;
      _msgIsError = false;
      if (!_kindLocked) _section = _RecvSection.home;
      _stockinStep = 0;
      _formEpoch++;
    });
    await _refresh();
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(okMsg)));
    }
    return true;
  }

  Widget _gateWizard() {
    return GateInboundWizard(
      key: ValueKey('gate-$_formEpoch'),
      batchNo: _batchNo,
      unitPrice: _unitPrice,
      deductRate: _deductRate,
      freight: _freight,
      loadingFee: _loadingFee,
      weighFee: _weighFee,
      gross: _gross,
      plate: _plate,
      recvAddr: _recvAddr,
      remark: _remark,
      farmerSearch: _farmerSearch,
      partyName: _partyName,
      partyMobile: _partyMobile,
      origin: _origin,
      batchOk: _batchOk,
      photoUrls: _photoUrls,
      varieties: _varieties,
      varietyId: _varietyId,
      channel: _channel,
      coldStore: _coldStore,
      grade: _grade,
      farmerId: _farmerId,
      farmerHits: _farmerHits,
      searchingFarmer: _searchingFarmer,
      msg: _msg,
      msgIsError: _msgIsError,
      onBatchChanged: (_) => setState(() {
        _batchOk = false;
        _boundFarmerName = '';
      }),
      onValidateBatch: _validateBatch,
      onFarmerSearchChanged: (v) {
        if (_farmerId != null) setState(() => _farmerId = null);
        _onFarmerSearchChanged(v);
      },
      onSearchFarmers: _searchFarmers,
      onApplyFarmer: (m) => setState(() {
        _applyFarmer(m);
        _farmerHits = [];
      }),
      onClearFarmer: () => setState(_clearFarmerLink),
      onShowOnsiteFarmer: _showOnsiteFarmerDialog,
      onApplyVariety: (m) => setState(() => _applyVariety(m)),
      onChannelChanged: (v) => setState(() => _channel = v),
      onColdStoreChanged: (v) => setState(() => _coldStore = v),
      onGradeChanged: (v) => setState(() => _grade = v),
      onTakePhoto: _takePhoto,
      onRemovePhoto: (i) => setState(() => _photoUrls.removeAt(i)),
      onMsg: _promptRequired,
      onSubmit: ({required nextRole, nextNodeId, nextAssigneeUserId}) => _create(
        nextRole: nextRole,
        nextNodeId: nextNodeId,
        nextAssigneeUserId: nextAssigneeUserId,
      ),
    );
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

  Color _phaseColor(String phase, String st) {
    switch (phase) {
      case 'await_warehouse':
        return Colors.orange.shade700;
      case 'await_finance':
        return Colors.teal;
      case 'settled':
      case 'stocked_done':
        return Colors.green.shade700;
      case 'pending_bind':
        return Colors.blue;
      case 'returned_by_warehouse':
        return Colors.red.shade700;
      default:
        break;
    }
    switch (st) {
      case 'draft':
        return Colors.orange;
      case 'qc_pass':
      case 'pending_confirm':
        return Colors.blue;
      case 'weighed':
        return Colors.orange.shade700;
      case 'returned':
        return Colors.red.shade700;
      case 'stocked':
      case 'posted':
        return Colors.green;
      case 'qc_fail':
        return Colors.red;
      default:
        return Colors.grey;
    }
  }

  /// 优先展示处理进度；无 process_phase 时回退到 status。
  String _phaseLabel(Map<String, dynamic> m) {
    final phase = m['process_phase']?.toString() ?? '';
    final kind = m['receive_kind']?.toString() ?? '';
    switch (phase) {
      case 'await_warehouse':
        return kind == 'stockin' ? '待仓管确认' : '待仓管入库';
      case 'await_finance':
        return '已入仓·待结算';
      case 'settled':
        return '已结清';
      case 'stocked_done':
        return '已入库';
      case 'pending_bind':
        return '待绑定';
      case 'returned_by_warehouse':
        return '仓管已退回';
    }
    final st = m['status']?.toString() ?? '';
    switch (st) {
      case 'draft':
        return '草稿';
      case 'qc_pending':
        return '待质检';
      case 'qc_pass':
        return '质检合格';
      case 'qc_fail':
        return '质检不合格';
      case 'pending_confirm':
        return '待绑定';
      case 'weighed':
        return kind == 'stockin' ? '待仓管确认' : '待仓管入库';
      case 'returned':
        return '仓管已退回';
      case 'stocked':
        return kind == 'gate' ? '已入仓·待结算' : '已入库';
      case 'posted':
        return '已过账';
      default:
        return st.isEmpty ? '未知' : st;
    }
  }

  String? _phaseHint(Map<String, dynamic> m) {
    final phase = m['process_phase']?.toString() ?? '';
    final kind = m['receive_kind']?.toString() ?? '';
    final assignee = (m['current_assignee_name']?.toString() ?? '').trim();
    final st = m['status']?.toString() ?? '';
    if (phase == 'returned_by_warehouse' || st == 'returned') {
      final remark = (m['remark']?.toString() ?? '').trim();
      final base = remark.isEmpty ? '请核对修正后重新推仓管' : '退回原因：$remark';
      return assignee.isEmpty ? base : '$base · 处理人 $assignee';
    }
    if (phase == 'await_warehouse' || st == 'weighed') {
      final base = kind == 'gate' ? '仓管确认入仓后，财务才可给农户结单' : '等待仓管确认入库';
      return assignee.isEmpty ? base : '$base · 处理人 $assignee';
    }
    if (phase == 'await_finance') {
      return assignee.isEmpty ? '已入仓，待财务结算' : '已入仓，待财务结算 · 处理人 $assignee';
    }
    return null;
  }

  Future<void> _repushReturned(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/purchase/weigh-tickets/$id/confirm', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已重新推送仓管' : _batchErrorMessage(r.msg))));
    if (r.ok) await _refresh();
  }

  String _kindLabel(String k) => k == 'stockin' ? '入库' : '入厂';

  List<Widget> _batchSection() {
    return [
      const FormSectionHeader('溯源批号'),
      TraceCodeField(
        controller: _batchNo,
        label: '溯源批号',
        hint: _receiveKind == 'stockin' ? '扫入厂已绑定的批号' : '手输或扫码',
        validated: _batchOk,
        compact: true,
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
            label: Text('关联农户（入厂绑定）：$_boundFarmerName'),
          ),
        ),
    ];
  }

  List<Widget> _stockinFormFields() {
    return [
      const Padding(
        padding: EdgeInsets.only(bottom: 4),
        child: Text(
          '凭据及溯源批号关联，自动带出入厂绑定农户',
          style: TextStyle(fontSize: 12, color: Colors.black54),
        ),
      ),
      ..._batchSection(),
      const FormSectionHeader('过磅信息'),
      if (_varieties.isEmpty)
        const Padding(
          padding: EdgeInsets.symmetric(vertical: 8),
          child: Text('暂无可用品种，请先在后台配置', style: TextStyle(color: Colors.orange)),
        )
      else
        FormRow(
          label: '品种/货品',
          requiredMark: true,
          child: DropdownButtonHideUnderline(
            child: DropdownButton<int>(
              isExpanded: true,
              value: _varietyId,
              alignment: Alignment.centerRight,
              hint: const Text('请选择', textAlign: TextAlign.right),
              items: _varieties.map((e) {
                final map = Map<String, dynamic>.from(e as Map);
                return DropdownMenuItem(
                  value: (map['id'] as num?)?.toInt(),
                  child: Text('${map['name'] ?? map['code']}', textAlign: TextAlign.right),
                );
              }).toList(),
              onChanged: (v) {
                if (v == null) return;
                final hit = _varieties
                    .cast<dynamic>()
                    .map((e) => Map<String, dynamic>.from(e as Map))
                    .where((map) => (map['id'] as num?)?.toInt() == v);
                if (hit.isNotEmpty) setState(() => _applyVariety(hit.first));
              },
            ),
          ),
        ),
      FormRow.text(label: '净重(kg)', controller: _netWeight, keyboardType: TextInputType.number, requiredMark: true),
      FormRow.text(label: '袋数', controller: _bagQty, keyboardType: TextInputType.number),
      FormRow(
        label: '冷藏库区',
        child: DropdownButtonHideUnderline(
          child: DropdownButton<String>(
            isExpanded: true,
            value: _coldStore,
            alignment: Alignment.centerRight,
            items: const [
              DropdownMenuItem(value: 'fresh', child: Text('原料库', textAlign: TextAlign.right)),
              DropdownMenuItem(value: 'semi', child: Text('半成品库', textAlign: TextAlign.right)),
              DropdownMenuItem(value: 'fg', child: Text('成品库', textAlign: TextAlign.right)),
            ],
            onChanged: (v) => setState(() => _coldStore = v ?? 'fresh'),
          ),
        ),
      ),
      FormRow.text(label: '产地地址', controller: _origin),
      const FormSectionHeader('现场照片'),
      FormRow(
        label: '现场照片',
        requiredMark: true,
        child: Row(
          mainAxisAlignment: MainAxisAlignment.end,
          children: [
            Text('已拍 ${_photoUrls.length}/3', style: const TextStyle(fontSize: 13, color: Colors.black54)),
            const SizedBox(width: 8),
            FilledButton.tonalIcon(
              onPressed: _takePhoto,
              icon: const Icon(Icons.photo_camera, size: 18),
              label: const Text('拍照'),
            ),
          ],
        ),
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
      FormRow.text(label: '备注', controller: _remark, maxLines: 2),
    ];
  }

  Widget _kindChooser() {
    final topPad = widget.asTab ? 48.0 : 24.0;
    return ListView(
      padding: EdgeInsets.fromLTRB(20, topPad, 20, 24),
      children: [
        const Text('过磅收货', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
        const SizedBox(height: 8),
        const Text('选择业务入口；创建生效后回到本页', style: TextStyle(fontSize: 13, color: Colors.black54)),
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
        const SizedBox(height: 20),
        _kindChoiceCard(
          enabled: _canRecv,
          icon: Icons.login,
          title: '过磅入厂',
          subtitle: '扫溯源码过磅，建单即与农户绑定并推仓管',
          onTap: () => _chooseReceiveKind('gate'),
        ),
        const SizedBox(height: 12),
        _kindChoiceCard(
          enabled: _canRecv || _canWh,
          icon: Icons.warehouse_outlined,
          title: '过磅入库',
          subtitle: '凭入厂已绑定溯源码入库过磅',
          onTap: () => _chooseReceiveKind('stockin'),
        ),
        const SizedBox(height: 12),
        _kindChoiceCard(
          enabled: true,
          icon: Icons.receipt_long_outlined,
          title: '单据',
          subtitle: '查看过磅单与绑定状态',
          onTap: () => _openSection(_RecvSection.tickets),
        ),
        const SizedBox(height: 12),
        _kindChoiceCard(
          enabled: true,
          icon: Icons.assignment_outlined,
          title: '任务',
          subtitle: '现场采购任务认领与完成',
          onTap: () => _openSection(_RecvSection.tasks),
        ),
      ],
    );
  }

  Widget _kindChoiceCard({
    required bool enabled,
    required IconData icon,
    required String title,
    required String subtitle,
    required VoidCallback onTap,
  }) {
    return Card(
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: enabled ? onTap : null,
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              CircleAvatar(
                backgroundColor: enabled ? Colors.teal.shade50 : Colors.black12,
                child: Icon(icon, color: enabled ? Colors.teal.shade700 : Colors.black38),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(title, style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600, color: enabled ? null : Colors.black38)),
                    const SizedBox(height: 4),
                    Text(subtitle, style: TextStyle(fontSize: 12, color: enabled ? Colors.black54 : Colors.black26)),
                    if (!enabled)
                      const Padding(
                        padding: EdgeInsets.only(top: 4),
                        child: Text('当前账号无此权限', style: TextStyle(fontSize: 11, color: Colors.orange)),
                      ),
                  ],
                ),
              ),
              Icon(Icons.chevron_right, color: enabled ? Colors.black45 : Colors.black26),
            ],
          ),
        ),
      ),
    );
  }

  Widget _backHomeBar({String label = '返回首页'}) {
    if (_kindLocked) return const SizedBox.shrink();
    return Material(
      color: Theme.of(context).colorScheme.surfaceContainerHighest.withValues(alpha: 0.5),
      child: InkWell(
        onTap: _backToKindChooser,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          child: Row(
            children: [
              const Icon(Icons.arrow_back_ios_new, size: 16),
              const SizedBox(width: 6),
              Text(label, style: const TextStyle(fontWeight: FontWeight.w500)),
            ],
          ),
        ),
      ),
    );
  }

  Widget _stockinPreviewRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          SizedBox(width: 108, child: Text(label, style: TextStyle(fontSize: 13, color: Colors.black.withValues(alpha: 0.6)))),
          Expanded(child: Text(value, textAlign: TextAlign.right, style: const TextStyle(fontSize: 14))),
        ],
      ),
    );
  }

  Widget _stockinPreviewBody() {
    return ListView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
      children: [
        Row(
          children: [
            const Expanded(child: Text('单据预览', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13))),
            TextButton(onPressed: () => setState(() => _stockinStep = 0), child: const Text('修改')),
          ],
        ),
        const Text('请核对以下信息，有误请返回修改', style: TextStyle(fontSize: 12, color: Colors.black54)),
        const SizedBox(height: 8),
        _stockinPreviewRow('溯源批号', _batchNo.text.trim().toUpperCase()),
        if (_boundFarmerName.isNotEmpty) _stockinPreviewRow('关联农户', _boundFarmerName),
        _stockinPreviewRow('品种', _varietyName()),
        _stockinPreviewRow('净重(kg)', _netWeight.text),
        _stockinPreviewRow('袋数', _bagQty.text.trim().isEmpty ? '0' : _bagQty.text),
        _stockinPreviewRow('冷藏库区', _coldStoreLabel(_coldStore)),
        _stockinPreviewRow('产地', _origin.text.trim().isEmpty ? '-' : _origin.text.trim()),
        _stockinPreviewRow('现场照片', '${_photoUrls.length} 张'),
        _stockinPreviewRow('备注', _remark.text.trim().isEmpty ? '-' : _remark.text.trim()),
        const SizedBox(height: 8),
        const Text('确认后单据生效：溯源码与本单/农户唯一绑定，并推仓管。', style: TextStyle(fontSize: 12, color: Colors.black54)),
      ],
    );
  }

  Widget _gateFormBody() {
    return Column(
      children: [
        _backHomeBar(label: '返回首页'),
        Expanded(child: _gateWizard()),
      ],
    );
  }

  Widget _stockinFormBody() {
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;
    return Column(
      children: [
        _backHomeBar(label: '返回首页'),
        Expanded(
          child: _stockinStep == 0
              ? ListView(
                  keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
                  padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + bottomInset),
                  children: _stockinFormFields(),
                )
              : _stockinPreviewBody(),
        ),
        if (_msg.isNotEmpty)
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 4),
            child: Text(
              _msg,
              style: TextStyle(
                fontSize: 13,
                color: _msgIsError ? Theme.of(context).colorScheme.error : null,
                fontWeight: _msgIsError ? FontWeight.w600 : FontWeight.normal,
              ),
            ),
          ),
        if (_stockinStep == 0)
          FormStickyActions(
            primaryLabel: '下一步',
            onPrimary: () {
              final err = _validateStockinForm();
              if (err != null) {
                _promptRequired(err);
                return;
              }
              setState(() {
                _msg = '';
                _msgIsError = false;
                _stockinStep = 1;
              });
            },
          )
        else
          FormStickyActions(
            secondaryLabel: '上一步',
            onSecondary: () => setState(() => _stockinStep = 0),
            primaryLabel: '确认创建并绑定',
            onPrimary: () => _create(),
          ),
      ],
    );
  }

  Widget _ticketDetailPanel(Map<String, dynamic> d) {
    final kind = d['receive_kind']?.toString() ?? '';
    final currency = (d['currency_label'] ?? '元人民币').toString();
    final logs = d['process_logs'] is List ? d['process_logs'] as List : const [];
    final nextHint = (d['next_handler_hint'] ?? '').toString().trim();
    final nextName = (d['next_handler_name'] ?? d['current_assignee_name'] ?? '').toString().trim();
    final api = context.read<AuthState>().api;
    final imgs = <String>[];
    void addImg(dynamic v) {
      final s = api.resolveMediaUrl(v?.toString() ?? '');
      if (s.isNotEmpty && !imgs.contains(s)) imgs.add(s);
    }
    addImg(d['image_url']);
    for (final k in ['verify_images', 'image_urls', 'site_photos']) {
      final raw = d[k];
      if (raw is List) {
        for (final e in raw) {
          addImg(e);
        }
      }
    }
    final evidences = d['evidences'];
    if (evidences is List) {
      for (final e in evidences) {
        if (e is Map) addImg(e['file_url'] ?? e['url']);
      }
    }

    Widget kv(String k, String v) => Padding(
          padding: const EdgeInsets.symmetric(vertical: 2),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              SizedBox(width: 76, child: Text(k, style: const TextStyle(color: Colors.black54, fontSize: 12))),
              Expanded(child: Text(v, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500))),
            ],
          ),
        );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          width: double.infinity,
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
          decoration: BoxDecoration(
            color: Colors.teal.shade50,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                nextName.isNotEmpty ? '下一步处理人：$nextName' : '下一步',
                style: TextStyle(fontWeight: FontWeight.w600, color: Colors.teal.shade900, fontSize: 13),
              ),
              if (nextHint.isNotEmpty)
                Text(nextHint, style: TextStyle(fontSize: 12, color: Colors.teal.shade800)),
              Text(
                '进度 ${_phaseLabel(d)}',
                style: const TextStyle(fontSize: 12, color: Colors.black54),
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        const Text('单据明细', style: TextStyle(fontWeight: FontWeight.w600)),
        const SizedBox(height: 6),
        kv('业务日', '${d['biz_date'] ?? '-'}'),
        kv('模式', _kindLabel(kind)),
        kv('农户', '${d['party_name'] ?? d['farmer_name'] ?? '-'}'),
        kv('品种', '${d['product_name'] ?? d['variety'] ?? '-'}'),
        kv('溯源码', '${d['trace_code'] ?? d['batch_no'] ?? '-'}'),
        kv('毛重', '${d['gross_weight'] ?? '-'} kg'),
        kv('扣损', '${d['deduct_weight'] ?? '-'} kg（${d['deduct_rate'] ?? '-'}%）'),
        kv('净重', '${d['net_weight'] ?? '-'} kg'),
        kv('单价', _fmtSettleMoney(d['unit_price'], unit: '$currency/kg')),
        kv('结算', _fmtSettleMoney(d['settle_amount'] ?? d['settlement_amount'], unit: currency)),
        if ((d['settlement_doc_no'] ?? '').toString().isNotEmpty)
          kv('结算单', '${d['settlement_doc_no']} · ${d['settlement_status'] ?? ''}'),
        if ((d['box_code'] ?? '').toString().isNotEmpty) kv('箱码', '${d['box_code']}'),
        if ((d['applicant_name'] ?? '').toString().isNotEmpty) kv('建单人', '${d['applicant_name']}'),
        if (imgs.isNotEmpty) ...[
          const SizedBox(height: 8),
          const Text('现场照片', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
          const SizedBox(height: 6),
          SizedBox(
            height: 72,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              itemCount: imgs.length,
              separatorBuilder: (_, __) => const SizedBox(width: 6),
              itemBuilder: (_, i) => ClipRRect(
                borderRadius: BorderRadius.circular(6),
                child: Image.network(
                  imgs[i],
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
            ),
          ),
        ],
        const SizedBox(height: 12),
        const Text('处理记录（每步经办人）', style: TextStyle(fontWeight: FontWeight.w600)),
        const SizedBox(height: 6),
        if (logs.isEmpty)
          const Text('暂无处理流水', style: TextStyle(color: Colors.black54, fontSize: 12))
        else
          ...logs.toList().asMap().entries.map((entry) {
            final idx = entry.key + 1;
            final log = Map<String, dynamic>.from(entry.value as Map);
            final when = (log['created_at'] ?? '').toString();
            final comment = (log['comment'] ?? '').toString().trim();
            final handoff = _processHandoff(log);
            return Padding(
              padding: const EdgeInsets.only(bottom: 10),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    width: 22,
                    height: 22,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: Colors.teal.shade100,
                      shape: BoxShape.circle,
                    ),
                    child: Text('$idx', style: TextStyle(fontSize: 11, fontWeight: FontWeight.w600, color: Colors.teal.shade900)),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          _processActionLabel(log),
                          style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13),
                        ),
                        Text(
                          '操作人：${_processOperator(log)}',
                          style: const TextStyle(fontSize: 12, color: Colors.black87),
                        ),
                        if (handoff != null)
                          Text(
                            '交予：$handoff',
                            style: TextStyle(fontSize: 12, color: Colors.indigo.shade700),
                          ),
                        if (when.isNotEmpty)
                          Text(when, style: const TextStyle(fontSize: 11, color: Colors.black45)),
                        if (comment.isNotEmpty && !comment.startsWith('WT-') && !comment.startsWith('flow:'))
                          Text(comment, style: const TextStyle(fontSize: 12, color: Colors.black54))
                        else if (comment.startsWith('flow:'))
                          Text(
                            comment.replaceFirst('flow:', '流转 '),
                            style: const TextStyle(fontSize: 11, color: Colors.black45),
                          ),
                      ],
                    ),
                  ),
                ],
              ),
            );
          }),
      ],
    );
  }

  Widget _ticketsBody() {
    return Column(
      children: [
        _backHomeBar(),
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  '业务日 ${_fmtMd(_ticketDateFrom)} ~ ${_fmtMd(_ticketDateTo)}',
                  style: const TextStyle(fontSize: 13, color: Colors.black54),
                ),
              ),
              TextButton(
                onPressed: () {
                  setState(_resetTicketDateDefault);
                  _refresh();
                },
                child: const Text('近3日'),
              ),
              FilledButton.tonal(
                onPressed: _pickTicketDateRange,
                child: const Text('选择日期'),
              ),
            ],
          ),
        ),
        Expanded(
          child: RefreshIndicator(
            onRefresh: _refresh,
            child: _tickets.isEmpty
                ? ListView(
                    physics: const AlwaysScrollableScrollPhysics(),
                    children: const [
                      SizedBox(height: 80),
                      Center(child: Text('暂无过磅单据')),
                    ],
                  )
                : ListView.builder(
                    padding: const EdgeInsets.all(12),
                    itemCount: _tickets.length,
                    itemBuilder: (context, i) {
                      final m = Map<String, dynamic>.from(_tickets[i] as Map);
                      final id = (m['id'] as num?)?.toInt();
                      final st = m['status']?.toString() ?? '';
                      final phase = m['process_phase']?.toString() ?? '';
                      final kind = m['receive_kind']?.toString() ?? '';
                      final hint = _phaseHint(m);
                      final expanded = id != null && _expandedTicketIds.contains(id);
                      final detail = id != null ? _ticketDetails[id] : null;
                      final loadingDetail = id != null && _ticketDetailLoading.contains(id);
                      final batch = (m['batch_no']?.toString() ?? '').toUpperCase();
                      final trace = (m['trace_code']?.toString() ?? '').toUpperCase();
                      final code = trace.isNotEmpty ? trace : batch;
                      final settleText = _fmtSettleMoney(m['settle_amount']);
                      return Card(
                        child: InkWell(
                          onTap: () => _toggleTicketExpand(m),
                          borderRadius: BorderRadius.circular(12),
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
                                      label: Text(_phaseLabel(m), style: const TextStyle(fontSize: 11, color: Colors.white)),
                                      backgroundColor: _phaseColor(phase, st),
                                      visualDensity: VisualDensity.compact,
                                    ),
                                    Icon(expanded ? Icons.expand_less : Icons.expand_more, color: Colors.black45),
                                  ],
                                ),
                                Text('${m['party_name'] ?? m['farmer_name'] ?? ''} · ${m['product_name'] ?? m['variety'] ?? ''}'),
                                Text(
                                  code.isEmpty ? '溯源码 -' : '溯源码 $code',
                                  style: const TextStyle(color: Colors.teal),
                                ),
                                if (kind == 'stockin')
                                  Text('净重 ${m['net_weight'] ?? '-'} kg · 冷库 ${m['cold_store_type'] ?? '-'} · 结算 $settleText')
                                else
                                  Text('净重 ${m['net_weight'] ?? '-'} kg · 结算 $settleText'),
                                Builder(builder: (_) {
                                  final next = (m['current_assignee_name'] ?? '').toString().trim();
                                  if (next.isEmpty) return const SizedBox.shrink();
                                  return Padding(
                                    padding: const EdgeInsets.only(top: 2),
                                    child: Text(
                                      '下一步：$next',
                                      style: TextStyle(fontSize: 12, color: Colors.indigo.shade700),
                                    ),
                                  );
                                }),
                                if (hint != null)
                                  Padding(
                                    padding: const EdgeInsets.only(top: 4),
                                    child: Text(hint, style: TextStyle(fontSize: 12, color: Colors.orange.shade800)),
                                  ),
                                if (st == 'returned' || phase == 'returned_by_warehouse')
                                  Align(
                                    alignment: Alignment.centerRight,
                                    child: FilledButton.tonal(
                                      onPressed: () => _repushReturned(m),
                                      child: const Text('重新推仓管'),
                                    ),
                                  ),
                                if (expanded) ...[
                                  const Divider(height: 20),
                                  if (loadingDetail)
                                    const Padding(
                                      padding: EdgeInsets.symmetric(vertical: 12),
                                      child: Center(child: SizedBox(width: 22, height: 22, child: CircularProgressIndicator(strokeWidth: 2))),
                                    )
                                  else if (detail == null)
                                    const Text('详情加载失败，请再点一次重试', style: TextStyle(color: Colors.black54, fontSize: 12))
                                  else
                                    _ticketDetailPanel(detail),
                                ],
                              ],
                            ),
                          ),
                        ),
                      );
                    },
                  ),
          ),
        ),
      ],
    );
  }

  Widget _tasksBody() {
    return Column(
      children: [
        _backHomeBar(),
        Expanded(
          child: RefreshIndicator(
            onRefresh: _refresh,
            child: ListView(
              physics: const AlwaysScrollableScrollPhysics(),
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
        ),
      ],
    );
  }

  Widget _sectionBody() {
    switch (_section) {
      case _RecvSection.home:
        return _kindChooser();
      case _RecvSection.gate:
        return _gateFormBody();
      case _RecvSection.stockin:
        return _stockinFormBody();
      case _RecvSection.tickets:
        return _ticketsBody();
      case _RecvSection.tasks:
        return _tasksBody();
    }
  }

  String get _sectionTitle {
    switch (_section) {
      case _RecvSection.home:
        return '过磅收货';
      case _RecvSection.gate:
        return '过磅入厂';
      case _RecvSection.stockin:
        return '过磅入库';
      case _RecvSection.tickets:
        return '单据';
      case _RecvSection.tasks:
        return '任务';
    }
  }

  @override
  Widget build(BuildContext context) {
    final showBack = !widget.asTab && _section != _RecvSection.home && !_kindLocked;
    return Scaffold(
      appBar: AppBar(
        title: widget.asTab ? null : Text(_sectionTitle),
        toolbarHeight: widget.asTab ? 0 : kToolbarHeight,
        leading: showBack ? IconButton(icon: const Icon(Icons.arrow_back), onPressed: _backToKindChooser) : null,
        actions: [IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh))],
      ),
      body: _loading && _section == _RecvSection.home && _tickets.isEmpty && _varieties.isEmpty
          ? const Center(child: CircularProgressIndicator())
          : _sectionBody(),
    );
  }
}
