import 'dart:async';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';
import '../../core/role_codes.dart';
import '../../widgets/form_row.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_sticky_actions.dart';
import '../../widgets/hub_entry_tile.dart';
import '../../widgets/trace_code_field.dart';
import 'gate_inbound_wizard.dart';
import 'trace_code_qr_sheet.dart';
import 'weigh_ticket_local_store.dart';

/// 现场过磅收货：选类型 → 填表预览 → 确认创建（批号即溯源码，绑定农户并推仓管）
/// [initialReceiveKind] / [lockKind]：供主壳「+」快捷入口锁定过磅入厂或入库表单
class ReceivingPage extends StatefulWidget {
  const ReceivingPage({
    super.key,
    this.initialReceiveKind,
    this.lockKind = false,
    this.popOnCreated = false,
    this.asTab = false,
    this.initialSection = RecvHubSection.home,
  });

  /// `gate` 过磅入厂 · `stockin` 过磅入库
  final String? initialReceiveKind;
  final bool lockKind;
  /// 从「+」快捷创建成功后 pop(true)
  final bool popOnCreated;
  /// 作为产线壳 Tab 时隐藏标题栏，仅展示入口首页。
  final bool asTab;
  /// 非 home 时作为独立子页（Navigator.push），返回即销毁。
  final RecvHubSection initialSection;

  @override
  State<ReceivingPage> createState() => _ReceivingPageState();
}

/// 收货首页三入口：入厂 / 单据 / 任务（无顶栏 Tab）
enum RecvHubSection { home, gate, stockin, tickets, tasks }

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
  RecvHubSection _section = RecvHubSection.home;
  /// 入库：0 填表 · 1 预览
  int _stockinStep = 0;
  /// 重建入厂向导（回选择页 / 提交成功后）
  int _formEpoch = 0;
  bool _canRecv = true;
  bool _canWh = true;
  final _gross = TextEditingController();
  final _deductRate = TextEditingController(text: '0');
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
  String? _photoMaterial;
  String? _photoScale;
  String? _photoCloseup;
  String _msg = '';
  /// 底部提示是否为错误（必填未填、校验失败等）
  bool _msgIsError = false;
  bool _loading = false;
  bool _batchOk = false;
  bool _searchingFarmer = false;
  /// 溯源码过站中：农户/产品锁定
  bool _bindingLocked = false;
  String _boundFarmerName = '';
  List<dynamic> _farmerCodes = [];
  Timer? _searchDebounce;
  bool _applyingFarmer = false;
  bool _generatedTraceThisTicket = false;
  bool _reusePrompted = false;
  bool _ticketsFromLocal = false;
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
    final id = _ticketExpandId(row);
    if (id == 0) return;
    final expanding = !_expandedTicketIds.contains(id);
    setState(() {
      if (expanding) {
        _expandedTicketIds.add(id);
      } else {
        _expandedTicketIds.remove(id);
      }
    });
    if (!expanding || _ticketDetails.containsKey(id) || _ticketDetailLoading.contains(id)) return;
    final serverId = (row['id'] as num?)?.toInt() ?? 0;
    final localOnly = row['local_backup'] == true;
    if (localOnly && serverId <= 0) {
      setState(() => _ticketDetails[id] = Map<String, dynamic>.from(row));
      return;
    }
    if (localOnly) {
      setState(() => _ticketDetails[id] = Map<String, dynamic>.from(row));
    }
    if (serverId <= 0) return;
    setState(() => _ticketDetailLoading.add(id));
    final r = await context.read<AuthState>().api.get('/purchase/weigh-tickets/$serverId');
    if (!mounted) return;
    setState(() {
      _ticketDetailLoading.remove(id);
      if (r.ok && r.data is Map) {
        _ticketDetails[id] = Map<String, dynamic>.from(r.data as Map);
      } else if (localOnly && !_ticketDetails.containsKey(id)) {
        _ticketDetails[id] = Map<String, dynamic>.from(row);
      }
    });
    if (!r.ok && !localOnly && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(_batchErrorMessage(r.msg))));
    }
  }

  String _batchErrorMessage(String code) {
    final c = code.trim();
    switch (c) {
      case 'SELF_ASSIGN_FORBIDDEN':
        return '不能指派自己为下一处理人，请选择其他仓管账号';
      case 'NO_ASSIGNEE_AVAILABLE':
        return '没有可指派的下一处理人，请先配置仓管人员';
      case 'BATCH_CODE_USED':
        return '该溯源码状态异常，请刷新后重试';
      case 'BATCH_CODE_ENDED':
        return '该溯源已入库完成，不可再入厂';
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
      case 'TRACE_FARMER_LOCKED':
        return '该溯源码已锁定农户，不可更换';
      case 'TRACE_PRODUCT_LOCKED':
        return '该溯源码已锁定产品/品种，不可更换';
      case 'GATE_BINDING_REQUIRED':
        return '请先完成入厂绑定后再入库';
      case 'DATE_RANGE_TOO_LARGE':
        return '查询跨度不能超过 31 天';
      case 'DATE_RANGE_INVALID':
        return '日期范围无效';
      case 'FARMER_CREATE_FAILED':
        return '自动建农户档案失败，请检查姓名后重试';
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
    if (widget.initialSection != RecvHubSection.home) {
      _section = widget.initialSection;
      if (widget.initialSection == RecvHubSection.gate) {
        _receiveKind = 'gate';
        _kindLocked = true;
      } else if (widget.initialSection == RecvHubSection.stockin) {
        _receiveKind = 'stockin';
        _kindLocked = true;
      }
    } else if (locked) {
      _section = init == 'stockin' ? RecvHubSection.stockin : RecvHubSection.gate;
    }
    WidgetsBinding.instance.addPostFrameCallback((_) => _boot());
  }

  @override
  void dispose() {
    _searchDebounce?.cancel();
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
    if (rolesPreferQcShell(auth.roles)) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('质检请从质检待办处理，无需进入采购建单')),
        );
        Navigator.of(context).pushReplacementNamed('/home');
      }
      return;
    }
    if (!canRecv && !canWh) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('无采购权限')));
        Navigator.of(context).pop();
      }
      return;
    }
    // 仅仓管：快捷入库；无采购过磅权限时不可入厂
    if (!canRecv && canWh) {
      setState(() {
        _receiveKind = 'stockin';
        _kindLocked = true;
        _section = RecvHubSection.stockin;
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
    final local = await _loadLocalTickets();
    setState(() {
      _loading = false;
      if (ticketRes.ok) {
        _ticketsFromLocal = false;
        _tickets = _mergeTicketsWithLocal(ApiClient.listOf(ticketRes.data), local);
        _ticketDetails.clear();
      } else {
        _ticketsFromLocal = true;
        _tickets = local;
        _ticketDetails.clear();
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
    if (!ticketRes.ok && ticketRes.msg.isNotEmpty && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            _ticketsFromLocal
                ? '服务端刷新失败，当前为手机本地备份'
                : _batchErrorMessage(ticketRes.msg),
          ),
        ),
      );
    }
  }

  int _localUserId() {
    final auth = context.read<AuthState>();
    if (auth.userId <= 0) auth.syncUserIdFromToken();
    return auth.userId;
  }

  Future<List<Map<String, dynamic>>> _loadLocalTickets() async {
    return WeighTicketLocalStore.load(_localUserId());
  }

  bool _localInDateRange(Map<String, dynamic> m) {
    final raw = (m['biz_date'] ?? m['saved_at'] ?? m['created_at'] ?? '').toString();
    final t = DateTime.tryParse(raw);
    if (t == null) return true;
    final d = DateTime(t.year, t.month, t.day);
    return !d.isBefore(_ticketDateFrom) && !d.isAfter(_ticketDateTo);
  }

  List<dynamic> _mergeTicketsWithLocal(List<dynamic> server, List<Map<String, dynamic>> local) {
    final seenIds = <int>{};
    final seenDocs = <String>{};
    final out = <dynamic>[];
    for (final e in server) {
      final m = Map<String, dynamic>.from(e as Map);
      final id = (m['id'] as num?)?.toInt() ?? 0;
      final doc = (m['doc_no'] ?? '').toString().trim();
      if (id > 0) seenIds.add(id);
      if (doc.isNotEmpty) seenDocs.add(doc);
      out.add(m);
    }
    for (final loc in local) {
      if (!_localInDateRange(loc)) continue;
      final id = (loc['id'] as num?)?.toInt() ?? 0;
      final doc = (loc['doc_no'] ?? '').toString().trim();
      if (id > 0 && seenIds.contains(id)) continue;
      if (doc.isNotEmpty && seenDocs.contains(doc)) continue;
      final row = Map<String, dynamic>.from(loc);
      row['local_backup'] = true;
      out.add(row);
    }
    return out;
  }

  int _ticketExpandId(Map<String, dynamic> m) {
    final id = (m['id'] as num?)?.toInt() ?? 0;
    if (id > 0) return id;
    final key = '${m['doc_no'] ?? ''}|${m['saved_at'] ?? ''}';
    return -(key.hashCode.abs() % 0x7fffffff);
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
    _applyingFarmer = true;
    _farmerId = (m['id'] as num?)?.toInt();
    _partyName.text = m['name']?.toString() ?? '';
    _partyMobile.text = m['mobile']?.toString() ?? '';
    _origin.text = m['origin']?.toString() ?? '';
    final price = (m['default_unit_price'] as num?)?.toDouble();
    if (price != null && price > 0) {
      _unitPrice.text = price.toString();
    }
    _batchOk = false;
    _bindingLocked = false;
    _boundFarmerName = '';
    _batchNo.clear();
    _reusePrompted = false;
    _applyingFarmer = false;
  }

  void _clearFarmerLink() {
    _farmerId = null;
    _farmerHits = [];
    _farmerCodes = [];
    _batchOk = false;
    _bindingLocked = false;
    _boundFarmerName = '';
    _batchNo.clear();
    _reusePrompted = false;
  }

  Future<void> _loadFarmerCodes() async {
    final fid = _farmerId;
    if (fid == null || fid <= 0) {
      setState(() => _farmerCodes = []);
      return;
    }
    final r = await context.read<AuthState>().api.get(
          '/purchase/trace-batch-codes?farmer_id=$fid&page_size=50',
        );
    if (!mounted) return;
    final list = r.ok && r.data is Map ? ApiClient.listOf((r.data as Map)['list']) : <dynamic>[];
    setState(() {
      _farmerCodes = list;
      if (!r.ok) {
        _msgIsError = true;
        _msg = _batchErrorMessage(r.msg.isNotEmpty ? r.msg : '倒查溯源码失败');
      }
    });
  }

  Future<void> _pickFarmerCode(Map<String, dynamic> m) async {
    final code = (m['code'] ?? '').toString().trim().toUpperCase();
    if (code.isEmpty) return;
    final selectable = m['can_append'] == true || m['selectable'] == true ||
        m['status'] == 'in_progress' || m['status'] == 'used';
    if (!selectable) {
      _promptRequired('该溯源码不可用（${m['status_label'] ?? m['status'] ?? ''}）');
      return;
    }
    _batchNo.text = code;
    await _validateBatch();
  }

  Future<bool> _generateTraceCodeForGate() async {
    final r = await context.read<AuthState>().api.post('/purchase/trace-batch-codes/generate', {
      'biz_date': DateTime.now().toIso8601String().substring(0, 10),
      'lot_no': '01',
      'qty': 1,
    });
    if (!mounted) return false;
    if (!r.ok || r.data is! Map) {
      _promptRequired(_batchErrorMessage(r.msg.isNotEmpty ? r.msg : '生成溯源码失败'));
      return false;
    }
    final data = Map<String, dynamic>.from(r.data as Map);
    final list = ApiClient.listOf(data['list']);
    String code = '';
    if (list.isNotEmpty) {
      code = (Map<String, dynamic>.from(list.first as Map)['code'] ?? '').toString();
    }
    if (code.isEmpty) {
      _promptRequired('生成成功但未返回批号');
      return false;
    }
    _batchNo.text = code.toUpperCase();
    setState(() {
      _generatedTraceThisTicket = true;
      _msgIsError = false;
      _msg = '已生成新溯源码 $code，校验中…';
    });
    return _validateBatch();
  }

  Future<void> _showGeneratedQr() async {
    if (!mounted) return;
    await showTraceCodeQrSheet(
      context,
      code: _batchNo.text,
      farmerName: _partyName.text,
    );
  }

  Future<void> _onTapManualTrace() async {
    if (_batchNo.text.trim().isNotEmpty) return;
    if (_reusePrompted) return;
    final fid = _farmerId;
    if (fid == null || fid <= 0) return;
    _reusePrompted = true;
    await _loadFarmerCodes();
    if (!mounted) return;
    final usable = _farmerCodes
        .cast<dynamic>()
        .map((e) => Map<String, dynamic>.from(e as Map))
        .where((m) =>
            m['can_append'] == true ||
            m['selectable'] == true ||
            m['status'] == 'in_progress' ||
            m['status'] == 'used')
        .toList();
    if (usable.isEmpty) return;
    final first = usable.first;
    final code = (first['code'] ?? '').toString();
    if (code.isEmpty) return;
    final reuse = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('复用溯源码'),
        content: Text('该农户有可用溯源码 $code，是否继续使用？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('否')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('是')),
        ],
      ),
    );
    if (reuse == true) {
      await _pickFarmerCode(first);
    }
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

  void _onPartyFieldChanged(String q, {required bool byMobile}) {
    if (_applyingFarmer || _bindingLocked) return;
    setState(() {
      if (_farmerId != null) _farmerId = null;
      _farmerHits = [];
    });
    _searchDebounce?.cancel();
    _searchDebounce = Timer(const Duration(milliseconds: 350), () => _searchFarmers(q, byMobile: byMobile));
  }

  Future<void> _searchFarmers(String raw, {bool byMobile = false}) async {
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
    final useMobile = byMobile || RegExp(r'\d').hasMatch(q);
    final path = useMobile
        ? '/purchase/farmers?mobile=${Uri.encodeQueryComponent(q)}&page_size=20'
        : '/purchase/farmers?name=${Uri.encodeQueryComponent(q)}&page_size=20';
    final r = await api.get(path);
    if (!mounted) return;
    setState(() {
      _searchingFarmer = false;
      _farmerHits = r.ok ? ApiClient.listOf(r.data) : [];
      if (!r.ok) _msg = '农户搜索失败：${r.msg}';
    });
  }

  Future<bool> _validateBatch() async {
    final code = _batchNo.text.trim().toUpperCase();
    _batchNo.text = code;
    if (code.isEmpty) {
      setState(() {
        _batchOk = false;
        _bindingLocked = false;
        _boundFarmerName = '';
      });
      _promptRequired('请填写溯源批号');
      return false;
    }
    final r = await context.read<AuthState>().api.post('/purchase/trace-batch-codes/validate', {
      'code': code,
      'receive_kind': _receiveKind,
    });
    if (!mounted) return false;
    if (!r.ok || r.data is! Map) {
      setState(() {
        _batchOk = false;
        _bindingLocked = false;
        _boundFarmerName = '';
      });
      _promptRequired(_batchErrorMessage(r.msg.isNotEmpty ? r.msg : '溯源批号校验失败'));
      return false;
    }
    final m = Map<String, dynamic>.from(r.data as Map);
    final lockedFid = (m['farmer_id'] as num?)?.toInt() ?? 0;
    if (_receiveKind == 'gate' &&
        lockedFid > 0 &&
        _farmerId != null &&
        _farmerId! > 0 &&
        lockedFid != _farmerId &&
        (m['binding_locked'] == true || m['status'] == 'in_progress' || m['can_append'] == true)) {
      setState(() {
        _batchOk = false;
        _bindingLocked = false;
      });
      _promptRequired('该溯源码已绑定其他农户，请换码或改选农户');
      return false;
    }
    setState(() {
      _batchOk = true;
      _applyBatchBinding(m);
      _msgIsError = false;
      final bound = _boundFarmerName;
      final st = (m['status'] ?? '').toString();
      if (_receiveKind == 'stockin') {
        _msg = bound.isEmpty
            ? '批号校验通过（入库）'
            : '批号校验通过 · 已同步农户/产品：$bound';
      } else if (st == 'in_progress' || m['can_append'] == true) {
        _msg = bound.isEmpty
            ? '批号过站中，可追加同农户同产品采购单'
            : '批号过站中 · 已锁定 $bound，可追加本单';
      } else {
        _msg = bound.isEmpty ? '批号校验通过（可入厂占用）' : '批号校验通过 · 已同步 $bound';
      }
    });
    return true;
  }

  /// 校验通过后把入厂绑定的农户/产地/品种等写入表单
  void _applyBatchBinding(Map<String, dynamic> m) {
    final name = (m['farmer_name'] ?? m['party_name'] ?? '').toString().trim();
    final mobile = (m['party_mobile'] ?? '').toString().trim();
    final origin = (m['origin'] ?? '').toString().trim();
    final fid = (m['farmer_id'] as num?)?.toInt() ?? 0;
    _boundFarmerName = name;
    final st = (m['status'] ?? '').toString();
    _bindingLocked = m['binding_locked'] == true || st == 'in_progress' || m['can_append'] == true;

    if (fid > 0) {
      _farmerId = fid;
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
      _bindingLocked = false;
      _boundFarmerName = '';
      _stockinStep = 0;
      if (kind == 'stockin') {
        _farmerId = null;
        _farmerHits = [];
      }
    });
  }

  bool get _isSubPage =>
      widget.initialSection != RecvHubSection.home || widget.lockKind;

  void _chooseReceiveKind(String kind) {
    _onReceiveKindChanged(kind);
    final section = kind == 'stockin' ? RecvHubSection.stockin : RecvHubSection.gate;
    _pushSection(section);
  }

  Future<void> _pushSection(RecvHubSection section) async {
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => ReceivingPage(
          initialSection: section,
          initialReceiveKind: section == RecvHubSection.stockin
              ? 'stockin'
              : section == RecvHubSection.gate
                  ? 'gate'
                  : null,
          lockKind: section == RecvHubSection.gate || section == RecvHubSection.stockin,
        ),
      ),
    );
    if (mounted) await _refresh();
  }

  void _openSection(RecvHubSection section) {
    _pushSection(section);
  }

  void _backToKindChooser() {
    if (Navigator.of(context).canPop()) {
      Navigator.of(context).pop();
      return;
    }
    if (_kindLocked) return;
    setState(() {
      _section = RecvHubSection.home;
      _stockinStep = 0;
      _msg = '';
      _msgIsError = false;
      _formEpoch++;
      _batchOk = false;
      _bindingLocked = false;
      _boundFarmerName = '';
    });
  }

  void _clearCreateForm() {
    _gross.clear();
    _netWeight.clear();
    _batchNo.clear();
    _photoUrls.clear();
    _photoMaterial = null;
    _photoScale = null;
    _photoCloseup = null;
    _batchOk = false;
    _bindingLocked = false;
    _boundFarmerName = '';
    _farmerId = null;
    _farmerHits = [];
    _farmerCodes = [];
    _bagQty.clear();
    _remark.clear();
    _plate.clear();
    _recvAddr.clear();
    _partyName.clear();
    _partyMobile.clear();
    _origin.clear();
    _stockinStep = 0;
    _generatedTraceThisTicket = false;
    _reusePrompted = false;
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

  Future<void> _takePhoto({String? slot}) async {
    if (slot == null && _photoUrls.length >= 3) {
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
        if (slot == 'material') {
          _photoMaterial = url;
          _msg = '已拍材料过磅照片';
        } else if (slot == 'scale_display') {
          _photoScale = url;
          _msg = '已拍磅显数据特写';
        } else if (slot == 'closeup') {
          _photoCloseup = url;
          _msg = '已拍近距离照片';
        } else {
          _photoUrls.add(url);
          _msg = '已上传 ${_photoUrls.length} 张';
        }
        _msgIsError = false;
      });
    } catch (e) {
      setState(() => _msg = '拍照失败：$e');
    }
  }

  void _removeGatePhoto(String slot) {
    setState(() {
      if (slot == 'material') _photoMaterial = null;
      if (slot == 'scale_display') _photoScale = null;
      if (slot == 'closeup') _photoCloseup = null;
    });
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
    if (_receiveKind == 'gate') {
      if ((_photoMaterial ?? '').isEmpty) {
        _promptRequired('请拍摄材料过磅照片');
        return false;
      }
      if ((_photoScale ?? '').isEmpty) {
        _promptRequired('请拍摄磅显数据特写');
        return false;
      }
      if ((_photoCloseup ?? '').isEmpty) {
        _promptRequired('请拍摄近距离照片');
        return false;
      }
    } else if (_photoUrls.isEmpty) {
      _promptRequired('请拍摄现场照片');
      return false;
    }
    if (_receiveKind == 'gate' && (_farmerId == null || _farmerId! <= 0) && _partyName.text.trim().isEmpty) {
      _promptRequired('请关联农户或填写农户姓名');
      return false;
    }
    final varietyName = _varietyName() == '-' ? '鲜木薯' : _varietyName();
    final gatePhotos = _receiveKind == 'gate'
        ? <String, String>{
            'material': _photoMaterial!,
            'scale_display': _photoScale!,
            'closeup': _photoCloseup!,
          }
        : null;
    final imageUrls = _receiveKind == 'gate'
        ? <String>[_photoMaterial!, _photoScale!, _photoCloseup!]
        : List<String>.from(_photoUrls);
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
      'image_url': _receiveKind == 'gate' ? _photoCloseup! : imageUrls.first,
      'image_urls': imageUrls,
      if (gatePhotos != null) 'photos': gatePhotos,
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
      final myId = context.read<AuthState>().userId;
      if (myId <= 0) context.read<AuthState>().syncUserIdFromToken();
      final selfId = context.read<AuthState>().userId;
      if (nextAssigneeUserId != null && selfId > 0 && nextAssigneeUserId == selfId) {
        _promptRequired('不能指派自己为下一处理人，请选择其他仓管账号');
        return false;
      }
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
    if (_receiveKind == 'gate') {
      await WeighTicketLocalStore.save(_localUserId(), {
        ...data,
        'doc_no': docNo.isNotEmpty ? docNo : data['doc_no'],
        'trace_code': trace.isNotEmpty ? trace : _batchNo.text.trim().toUpperCase(),
        'batch_no': _batchNo.text.trim().toUpperCase(),
        'farmer_id': data['farmer_id'] ?? _farmerId,
        'party_name': data['party_name'] ?? _partyName.text.trim(),
        'party_mobile': data['party_mobile'] ?? _partyMobile.text.trim(),
        'origin': data['origin'] ?? _origin.text.trim(),
        'variety': data['variety'] ?? varietyName,
        'gross_weight': data['gross_weight'] ?? body['gross_weight'],
        'net_weight': data['net_weight'],
        'status': data['status'],
        'created_at': data['created_at'],
        'biz_date': data['biz_date'] ?? body['biz_date'],
        'photo_urls': imageUrls,
        'image_urls': imageUrls,
        'photos': gatePhotos,
        'receive_kind': 'gate',
        'unit_price': body['unit_price'],
        'deduct_rate': body['deduct_rate'],
        'plate_no': body['plate_no'],
      });
    }
    final showQr = _generatedTraceThisTicket && _receiveKind == 'gate';
    final qrCode = _batchNo.text.trim().toUpperCase();
    final farmerName = _partyName.text.trim();
    if (widget.popOnCreated || _isSubPage) {
      if (mounted) {
        if (showQr) {
          await showTraceCodeQrSheet(context, code: qrCode, farmerName: farmerName);
        }
        if (!mounted) return true;
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(okMsg)));
        Navigator.of(context).pop(true);
      }
      return true;
    }
    _clearCreateForm();
    setState(() {
      _msg = okMsg;
      _msgIsError = false;
      if (!_kindLocked) _section = RecvHubSection.home;
      _stockinStep = 0;
      _formEpoch++;
    });
    await _refresh();
    if (mounted) {
      if (showQr) {
        await showTraceCodeQrSheet(context, code: qrCode, farmerName: farmerName);
      }
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(okMsg)));
      }
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
      partyName: _partyName,
      partyMobile: _partyMobile,
      origin: _origin,
      batchOk: _batchOk,
      bindingLocked: _bindingLocked,
      photoMaterial: _photoMaterial,
      photoScale: _photoScale,
      photoCloseup: _photoCloseup,
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
        _bindingLocked = false;
        _boundFarmerName = '';
      }),
      onValidateBatch: _validateBatch,
      onNameChanged: (v) => _onPartyFieldChanged(v, byMobile: false),
      onMobileChanged: (v) => _onPartyFieldChanged(v, byMobile: true),
      onApplyFarmer: (m) async {
        if (_bindingLocked) return;
        setState(() {
          _applyFarmer(m);
          _farmerHits = [];
        });
      },
      onClearFarmer: () {
        if (_bindingLocked) return;
        setState(_clearFarmerLink);
      },
      onTapManualTrace: _onTapManualTrace,
      onGenerateTraceCode: _generateTraceCodeForGate,
      onShowGeneratedQr: _showGeneratedQr,
      onApplyVariety: (m) {
        if (_bindingLocked) {
          _promptRequired('该溯源码已锁定品种，不可更换');
          return;
        }
        setState(() => _applyVariety(m));
      },
      onChannelChanged: (v) => setState(() => _channel = v),
      onColdStoreChanged: (v) => setState(() => _coldStore = v),
      onGradeChanged: (v) => setState(() => _grade = v),
      onTakePhoto: (slot) => _takePhoto(slot: slot),
      onRemovePhoto: _removeGatePhoto,
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
      case 'await_gate':
        return Colors.orange.shade700;
      case 'await_stockin':
        return Colors.indigo.shade700;
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
      case 'gate_accepted':
        return Colors.indigo.shade700;
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
      case 'await_gate':
        return '待入厂';
      case 'await_stockin':
        return '待入库';
      case 'await_warehouse':
        return kind == 'stockin' ? '待仓管确认' : '待入厂';
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
        return kind == 'stockin' ? '待仓管确认' : '待入厂';
      case 'gate_accepted':
        return '待入库';
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
    if (phase == 'await_gate' || (st == 'weighed' && kind != 'stockin')) {
      final base = '等待仓管接收入厂';
      return assignee.isEmpty ? base : '$base · 处理人 $assignee';
    }
    if (phase == 'await_stockin' || st == 'gate_accepted') {
      final base = '已入厂，待仓管分板入库';
      return assignee.isEmpty ? base : '$base · 处理人 $assignee';
    }
    if (phase == 'await_warehouse' || st == 'weighed') {
      final base = '等待仓管确认入库';
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
          _bindingLocked = false;
          _boundFarmerName = '';
        }),
        onEditingComplete: () {
          _validateBatch();
        },
        onScanned: (_) async {
          setState(() {
            _batchOk = false;
            _bindingLocked = false;
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
              onPressed: () => _takePhoto(),
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
    return ListView(
      padding: EdgeInsets.fromLTRB(16, widget.asTab ? 40 : 16, 16, 16),
      children: [
        const Text('采购', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
        const SizedBox(height: 4),
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
        const SizedBox(height: 16),
        const Text('常用', style: TextStyle(fontWeight: FontWeight.bold)),
        const SizedBox(height: 8),
        HubEntryTile(
          enabled: _canRecv,
          icon: Icons.login,
          title: '过磅入厂',
          subtitle: '扫溯源码过磅，建单即与农户绑定并推仓管',
          onTap: () => _chooseReceiveKind('gate'),
        ),
        HubEntryTile(
          icon: Icons.receipt_long_outlined,
          title: '单据',
          subtitle: '查看过磅单与绑定状态',
          onTap: () => _openSection(RecvHubSection.tickets),
        ),
        HubEntryTile(
          icon: Icons.assignment_outlined,
          title: '任务',
          subtitle: '现场采购任务认领与完成',
          onTap: () => _openSection(RecvHubSection.tasks),
        ),
      ],
    );
  }

  Widget _backHomeBar({String? title}) {
    if (_isSubPage || _kindLocked) return const SizedBox.shrink();
    final heading = title ?? _sectionTitle;
    return Material(
      color: Theme.of(context).colorScheme.surface,
      elevation: 0.5,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
        child: Row(
          children: [
            IconButton(
              tooltip: '返回',
              onPressed: _backToKindChooser,
              icon: const Icon(Icons.arrow_back),
            ),
            Expanded(
              child: Text(heading, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
            ),
            IconButton(tooltip: '刷新', onPressed: _refresh, icon: const Icon(Icons.refresh)),
          ],
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
        _backHomeBar(),
        Expanded(child: _gateWizard()),
      ],
    );
  }

  Widget _stockinFormBody() {
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;
    return Column(
      children: [
        _backHomeBar(),
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

  Widget _ticketDetailPanel(Map<String, dynamic> d, {bool localBackup = false}) {
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
    final photos = d['photos'];
    final labeledPhotos = <(String, String)>[];
    void addLabeled(String key, String label) {
      if (photos is! Map) return;
      final s = api.resolveMediaUrl((photos[key] ?? '').toString());
      if (s.isEmpty) return;
      labeledPhotos.add((label, s));
      addImg(s);
    }
    addLabeled('material', '材料过磅');
    addLabeled('scale_display', '磅显特写');
    addLabeled('closeup', '近距离');
    for (final k in ['verify_images', 'image_urls', 'site_photos', 'photo_urls']) {
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
        if (localBackup)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Text(
              '本机备份：仅可查看文字快照，不可改单。照片为当时上传的链接，服务端丢失后可能无法打开。',
              style: TextStyle(fontSize: 12, color: Colors.orange.shade800),
            ),
          ),
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
        if ((d['box_code'] ?? '').toString().isNotEmpty) kv('板码', '${d['box_code']}'),
        if ((d['applicant_name'] ?? '').toString().isNotEmpty) kv('建单人', '${d['applicant_name']}'),
        if (labeledPhotos.isNotEmpty) ...[
          const SizedBox(height: 8),
          const Text('现场照片', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
          const SizedBox(height: 6),
          SizedBox(
            height: 98,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              itemCount: labeledPhotos.length,
              separatorBuilder: (_, __) => const SizedBox(width: 8),
              itemBuilder: (_, i) {
                final p = labeledPhotos[i];
                return Column(
                  children: [
                    ClipRRect(
                      borderRadius: BorderRadius.circular(6),
                      child: Image.network(
                        p.$2,
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
                    Text(p.$1, style: const TextStyle(fontSize: 10, color: Colors.black54)),
                  ],
                );
              },
            ),
          ),
        ] else if (imgs.isNotEmpty) ...[
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
        if (_ticketsFromLocal)
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
            child: Text(
              '当前为手机本地备份（近 3 个月），服务端暂时不可用',
              style: TextStyle(fontSize: 12, color: Colors.orange.shade800),
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
                      final id = _ticketExpandId(m);
                      final st = m['status']?.toString() ?? '';
                      final phase = m['process_phase']?.toString() ?? '';
                      final kind = m['receive_kind']?.toString() ?? '';
                      final hint = _phaseHint(m);
                      final expanded = _expandedTicketIds.contains(id);
                      final detail = _ticketDetails[id];
                      final loadingDetail = _ticketDetailLoading.contains(id);
                      final batch = (m['batch_no']?.toString() ?? '').toUpperCase();
                      final trace = (m['trace_code']?.toString() ?? '').toUpperCase();
                      final code = trace.isNotEmpty ? trace : batch;
                      final settleText = _fmtSettleMoney(m['settle_amount']);
                      final localBackup = m['local_backup'] == true;
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
                                    Expanded(child: Text('${m['doc_no'] ?? '-'}', style: const TextStyle(fontWeight: FontWeight.bold))),
                                    if (localBackup)
                                      Chip(
                                        label: const Text('本机备份', style: TextStyle(fontSize: 11)),
                                        visualDensity: VisualDensity.compact,
                                        backgroundColor: Colors.orange.shade50,
                                      ),
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
                                    _ticketDetailPanel(detail, localBackup: localBackup || detail['local_backup'] == true),
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
      case RecvHubSection.home:
        return _kindChooser();
      case RecvHubSection.gate:
        return _gateFormBody();
      case RecvHubSection.stockin:
        return _stockinFormBody();
      case RecvHubSection.tickets:
        return _ticketsBody();
      case RecvHubSection.tasks:
        return _tasksBody();
    }
  }

  String get _sectionTitle {
    switch (_section) {
      case RecvHubSection.home:
        return '采购';
      case RecvHubSection.gate:
        return '过磅入厂';
      case RecvHubSection.stockin:
        return '过磅入库';
      case RecvHubSection.tickets:
        return '单据';
      case RecvHubSection.tasks:
        return '任务';
    }
  }

  @override
  Widget build(BuildContext context) {
    final showAppBar = _isSubPage || !widget.asTab;
    return Scaffold(
      appBar: showAppBar
          ? AppBar(
              title: Text(_sectionTitle),
              actions: [IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh))],
            )
          : AppBar(toolbarHeight: 0),
      body: _loading && _section == RecvHubSection.home && _tickets.isEmpty && _varieties.isEmpty
          ? const Center(child: CircularProgressIndicator())
          : _sectionBody(),
    );
  }
}
