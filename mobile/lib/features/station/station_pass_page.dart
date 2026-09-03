import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/auth_state.dart';
import '../../core/carrier_code_labels.dart';
import '../../core/recent_code_store.dart';
import '../../widgets/form_row.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_sticky_actions.dart';
import '../../widgets/hub_entry_tile.dart';
import '../../widgets/tab_safe_padding.dart';
import '../../widgets/active_trace_dropdown.dart';
import '../../widgets/trace_code_field.dart';
import 'trace_production_page.dart';
import 'process_issue_history_page.dart';
import 'process_stock_in_apply_page.dart';

enum StationPassMode { home, self, proxy }

class _MaterialSource {
  const _MaterialSource({
    required this.key,
    required this.label,
    required this.source,
    this.processId,
    this.limitKg,
  });

  final String key;
  final String label;
  final String source; // warehouse | process
  final int? processId;
  final double? limitKg;

  bool get isWarehouse => source == 'warehouse';
}

/// 生产领料：选生产中溯源 → 选来源（仓库/工序）→ 确认领取
class StationPassPage extends StatefulWidget {
  const StationPassPage({
    super.key,
    this.asTab = false,
    this.initialMode = StationPassMode.home,
  });

  final bool asTab;
  final StationPassMode initialMode;

  @override
  State<StationPassPage> createState() => _StationPassPageState();
}

class _StationPassPageState extends State<StationPassPage> {
  late StationPassMode _mode;
  int _step = 0;
  final _badge = TextEditingController();
  final _kg = TextEditingController();
  String _msg = '';
  bool _msgIsError = false;
  bool _busy = false;
  Map<String, dynamic>? _preview;
  List<Map<String, dynamic>> _allProcesses = [];
  List<Map<String, dynamic>> _processes = [];
  int? _processId;
  String? _selectedTraceCode;
  Map<String, dynamic>? _selectedTraceRowData;
  List<_MaterialSource> _sources = [];
  String? _selectedSourceKey;
  bool _loadingSources = false;
  String? _reweighPhotoUrl;

  static const _sourcePrefKey = 'erp.station.material_source_key';
  static const _tracePrefKey = 'erp.station.selected_trace';

  String get _codeLabel => context.read<CarrierCodeLabels>().code;

  _MaterialSource? get _selectedSource {
    if (_selectedSourceKey == null) return null;
    for (final s in _sources) {
      if (s.key == _selectedSourceKey) return s;
    }
    return null;
  }

  bool get _fromWarehouse => _selectedSource?.isWarehouse ?? false;

  Map<String, String> get _errLabel => {
        'TRACE_CODE_REQUIRED': '请选择溯源码',
        'PRODUCT_REQUIRED': '请选择物料品类',
        'BOARD_FINISHED': '该$_codeLabel已完工，不能再操作',
        'QTY_EXCEEDS_AVAILABLE': '领取重量超过可领量',
        'QTY_EXCEEDS_OCCUPANCY': '退库重量超过本人占用',
        'QTY_EXCEEDS_WIP': '重量超过本工序在制',
        'PROCESS_WIP_EMPTY': '来源工序在制不足',
        'BOX_REQUIRED': '仓库出库须扫板码',
        'BOX_NOT_FOUND': '未找到该$_codeLabel',
        'PROCESS_REQUIRED': '请先选择工序',
        'SHIFT_NOT_AUTHORIZED': '当前班次未授权该工序',
        'ROLE_FORBIDDEN': '仅生管可代领',
        'REWEIGH_REQUIRED': '请填写复磅重量',
        'REWEIGH_PHOTO_REQUIRED': '请拍摄复磅照片',
        'TRACE_MISMATCH': '板码与溯源码不一致',
        'STOCK_OUT_FAILED': '仓库出库记账失败',
        'ISSUE_PENDING_EXISTS': '该溯源已有待仓管确认的领料申请',
        'ROUTING_TRANSITION_FORBIDDEN': '该工序不在工艺路线允许范围内',
        'APP_ONLY': '请在 App 操作',
      };

  bool get _isSubPage => widget.initialMode != StationPassMode.home;

  bool get _canForeman {
    final roles = context.read<AuthState>().roles.map((e) => e.toString().toLowerCase()).toList();
    return roles.contains('foreman') ||
        roles.contains('车间主任') ||
        roles.contains('主任') ||
        roles.contains('生管') ||
        roles.contains('sys_admin') ||
        roles.contains('admin') ||
        roles.contains('系统管理员');
  }

  @override
  void initState() {
    super.initState();
    _mode = widget.initialMode;
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      final prefs = await SharedPreferences.getInstance();
      if (!mounted) return;
      _selectedSourceKey = prefs.getString(_sourcePrefKey);
      _selectedTraceCode = prefs.getString(_tracePrefKey);
      if (_mode == StationPassMode.proxy) {
        final saved = prefs.getString('erp.worker.badge') ?? '';
        if (saved.isNotEmpty && _badge.text.isEmpty) {
          _badge.text = saved;
        }
      }
      final savedProc = prefs.getInt('erp.station.process_id');
      if (savedProc != null && savedProc > 0) {
        _processId = savedProc;
      }
      _fillSelfBadgeIfNeeded();
      await _loadProcesses();
      if (mounted) setState(() {});
    });
  }

  Future<void> _loadProcesses() async {
    final api = context.read<AuthState>().api;
    final r = await api.get('/production/processes');
    if (!mounted) return;
    if (!r.ok) return;
    final list = <Map<String, dynamic>>[];
    final raw = r.data;
    final items = raw is Map ? (raw['list'] ?? raw['items']) : (raw is List ? raw : null);
    if (items is List) {
      for (final e in items) {
        if (e is Map) list.add(Map<String, dynamic>.from(e));
      }
    }
    setState(() {
      _allProcesses = list;
    });
    final trace = (_selectedTraceCode ?? '').trim();
    if (trace.isNotEmpty) {
      await _applyRoutingProcessHint(trace);
    } else {
      setState(() {
        _processes = [];
        _processId = null;
      });
    }
  }

  Map<String, dynamic>? _selectedTraceRow() => _selectedTraceRowData;

  Future<void> _loadMaterialSources() async {
    final trace = (_selectedTraceCode ?? '').trim();
    if (trace.isEmpty) {
      setState(() {
        _sources = [];
        _selectedSourceKey = null;
      });
      return;
    }
    setState(() => _loadingSources = true);
    final api = context.read<AuthState>().api;
    final enc = Uri.encodeComponent(trace);
    final r = await api.get('/production/trace-productions/$enc/material-locations');
    if (!mounted) return;
    setState(() => _loadingSources = false);
    final sources = <_MaterialSource>[];
    if (r.ok && r.data is Map) {
      final data = Map<String, dynamic>.from(r.data as Map);
      final wh = data['warehouse'];
      if (wh is Map && wh['selectable'] == true) {
        sources.add(const _MaterialSource(key: 'warehouse', label: '仓库', source: 'warehouse'));
      }
      final procs = data['processes'];
      if (procs is List) {
        for (final e in procs) {
          if (e is! Map) continue;
          final m = Map<String, dynamic>.from(e);
          final pid = (m['process_id'] as num?)?.toInt();
          if (pid == null || pid <= 0) continue;
          final limit = (m['source_limit_kg'] as num?)?.toDouble() ?? (m['wip_kg'] as num?)?.toDouble() ?? 0;
          final name = '${m['process_name'] ?? pid}';
          sources.add(_MaterialSource(
            key: 'process:$pid',
            label: '$name · 在制 ${limit.toStringAsFixed(2)} kg',
            source: 'process',
            processId: pid,
            limitKg: limit,
          ));
        }
      }
    }
    final prefs = await SharedPreferences.getInstance();
    var key = _selectedSourceKey ?? prefs.getString(_sourcePrefKey);
    if (key != null && !sources.any((s) => s.key == key)) {
      key = null;
    }
    if (key == null) {
      final procBest = sources.where((s) => !s.isWarehouse).toList()
        ..sort((a, b) => (b.limitKg ?? 0).compareTo(a.limitKg ?? 0));
      if (procBest.isNotEmpty && (procBest.first.limitKg ?? 0) > 0) {
        key = procBest.first.key;
      } else if (sources.any((s) => s.isWarehouse)) {
        key = 'warehouse';
      } else if (sources.isNotEmpty) {
        key = sources.first.key;
      }
    }
    setState(() {
      _sources = sources;
      _selectedSourceKey = key;
    });
    await _applyRoutingProcessHint(trace);
  }

  Future<void> _applyRoutingProcessHint(String trace) async {
    final r = await context.read<AuthState>().api.get('/production/trace-productions/${Uri.encodeComponent(trace)}/wip');
    if (!mounted || !r.ok || r.data is! Map) {
      setState(() {
        _processes = [];
        _processId = null;
      });
      return;
    }
    final data = Map<String, dynamic>.from(r.data as Map);
    final steps = data['routing_steps'];
    final filtered = <Map<String, dynamic>>[];
    int? hint;
    if (steps is List && steps.isNotEmpty) {
      for (final e in steps) {
        if (e is! Map) continue;
        final m = Map<String, dynamic>.from(e);
        final pid = (m['process_id'] as num?)?.toInt();
        if (pid == null || pid <= 0) continue;
        final st = '${m['step_status']}';
        if (st == 'in_progress' || st == 'ready') {
          hint ??= pid;
        } else if (st == 'pending' && hint == null) {
          hint = pid;
        }
        final fromAll = _allProcesses.where((p) => (p['id'] as num?)?.toInt() == pid).toList();
        if (fromAll.isNotEmpty) {
          filtered.add(fromAll.first);
        } else {
          filtered.add({
            'id': pid,
            'name': m['step_name'] ?? m['process_name'] ?? pid,
            'code': m['step_code'] ?? '',
          });
        }
      }
    }
    setState(() {
      _processes = filtered;
      if (_processId != null && !filtered.any((p) => (p['id'] as num?)?.toInt() == _processId)) {
        _processId = null;
      }
      if (hint != null && filtered.any((p) => (p['id'] as num?)?.toInt() == hint)) {
        _processId = hint;
      } else if (_processId == null && filtered.isNotEmpty) {
        _processId = (filtered.first['id'] as num?)?.toInt();
      }
    });
  }

  void _onTraceSelected(String? code, Map<String, dynamic>? row) {
    setState(() {
      _selectedTraceCode = code;
      _selectedTraceRowData = row;
    });
    if ((code ?? '').trim().isEmpty) {
      setState(() {
        _processes = [];
        _processId = null;
        _sources = [];
        _selectedSourceKey = null;
      });
      return;
    }
    _loadMaterialSources();
  }

  Future<void> _onSourceChanged(String? key) async {
    setState(() => _selectedSourceKey = key);
    if (key != null) {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_sourcePrefKey, key);
    }
  }

  @override
  void dispose() {
    _badge.dispose();
    _kg.dispose();
    super.dispose();
  }

  void _fillSelfBadgeIfNeeded() {
    if (_mode != StationPassMode.self) return;
    final auth = context.read<AuthState>();
    _badge.text = (auth.badgeCode ?? '').trim();
  }

  void _prompt(String msg, {bool error = true}) {
    if (!mounted) return;
    final shown = _errLabel[msg] ?? msg;
    setState(() {
      _msg = shown;
      _msgIsError = error;
    });
    if (!error) return;
    final messenger = ScaffoldMessenger.of(context);
    messenger.clearSnackBars();
    final scheme = Theme.of(context).colorScheme;
    messenger.showSnackBar(
      SnackBar(
        content: Text(shown, style: TextStyle(color: scheme.onError)),
        backgroundColor: scheme.error,
        behavior: SnackBarBehavior.floating,
        duration: const Duration(seconds: 3),
      ),
    );
  }

  Future<void> _openMode(StationPassMode mode) async {
    if (mode == StationPassMode.home) return;
    final ok = await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => StationPassPage(initialMode: mode)),
    );
    if (!mounted) return;
    if (ok == true) {
      setState(() {
        _msg = '已提交';
        _msgIsError = false;
      });
    }
  }

  void _leaveOrBack() {
    if (_step == 1) {
      setState(() {
        _step = 0;
        _msg = '';
        _msgIsError = false;
      });
      return;
    }
    if (Navigator.of(context).canPop()) {
      Navigator.of(context).pop();
    }
  }

  Map<String, dynamic> _baseBody() {
    if (_mode == StationPassMode.self) {
      _fillSelfBadgeIfNeeded();
    }
    final src = _selectedSource;
    final kg = double.tryParse(_kg.text) ?? 0;
    final body = <String, dynamic>{
      'source': src?.source ?? 'process',
      'kg': kg,
      'reweigh_kg': kg,
    };
    final badge = _badge.text.trim();
    if (badge.isNotEmpty) body['badge_code'] = badge;
    if (_processId != null && _processId! > 0) body['process_id'] = _processId;
    if (src != null && !src.isWarehouse && (src.processId ?? 0) > 0) {
      body['from_process_id'] = src.processId;
      body['to_process_id'] = _processId;
    }
    if ((_reweighPhotoUrl ?? '').isNotEmpty) {
      body['photo_url'] = _reweighPhotoUrl;
      body['image_url'] = _reweighPhotoUrl;
    }
    final trace = (_selectedTraceCode ?? '').trim();
    if (trace.isNotEmpty) body['trace_code'] = trace;
    return body;
  }

  String? _validateScan() {
    if (_processId == null || _processId! <= 0) return '请先选择工序';
    if ((_selectedTraceCode ?? '').trim().isEmpty) return '请选择溯源码';
    if (_selectedSource == null) return '请选择领料来源';
    if (_sources.isEmpty) return '该溯源暂无可领来源';
    return null;
  }

  String? _validateAction() {
    if (_processId == null || _processId! <= 0) return '请先选择工序';
    final kg = double.tryParse(_kg.text) ?? 0;
    if (kg <= 0) return '请填写重量(kg)';
    if ((_reweighPhotoUrl ?? '').isEmpty) return '请拍摄复磅照片';
    final src = _selectedSource;
    if (src != null && !src.isWarehouse) {
      final limit = src.limitKg ?? 0;
      if (limit <= 0) return '来源工序无在制';
      if (kg > limit + 0.0005) return '领取重量不能超过在制 ${limit.toStringAsFixed(2)} kg';
    }
    return null;
  }

  Future<void> _takeReweighPhoto() async {
    try {
      final picker = ImagePicker();
      final file = await picker.pickImage(source: ImageSource.camera, imageQuality: 85);
      if (file == null) return;
      final bytes = await file.readAsBytes();
      if (!mounted) return;
      setState(() {
        _msg = '上传复磅照片…';
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
        _reweighPhotoUrl = url;
        _msg = '复磅照片已上传';
        _msgIsError = false;
      });
    } catch (e) {
      _prompt('拍照失败：$e');
    }
  }

  Future<void> _goPreview() async {
    final err = _validateScan();
    if (err != null) {
      _prompt(err);
      return;
    }
    setState(() {
      _busy = true;
      _msg = '';
      _msgIsError = false;
    });
    final prefs = await SharedPreferences.getInstance();
    final badge = _badge.text.trim();
    if (badge.isNotEmpty) await prefs.setString('erp.worker.badge', badge);
    if (_processId != null && _processId! > 0) {
      await prefs.setInt('erp.station.process_id', _processId!);
    }
    if (_selectedSourceKey != null) {
      await prefs.setString(_sourcePrefKey, _selectedSourceKey!);
    }
    if (!mounted) return;

    final traceRow = _selectedTraceRow();
    final src = _selectedSource!;
    final procName = () {
      for (final p in _processes) {
        if ((p['id'] as num?)?.toInt() == _processId) {
          return '${p['name'] ?? p['code'] ?? _processId}';
        }
      }
      return '';
    }();

    if (src.isWarehouse) {
      setState(() {
        _busy = false;
        _preview = {
          'trace_code': _selectedTraceCode,
          'farmer_name': traceRow?['farmer_name'] ?? '',
          'process_id': _processId,
          'process_name': procName,
          'source': 'warehouse',
          'source_label': src.label,
          'pending': true,
        };
        _step = 1;
      });
      return;
    }

    final limit = src.limitKg ?? 0;
    if (limit <= 0) {
      setState(() => _busy = false);
      _prompt('该来源工序无在制');
      return;
    }
    setState(() {
      _busy = false;
      _preview = {
        'trace_code': _selectedTraceCode,
        'farmer_name': traceRow?['farmer_name'] ?? '',
        'from_process_id': src.processId,
        'from_process_name': src.label.split(' · ').first,
        'process_id': _processId,
        'process_name': procName,
        'available_kg': limit,
        'wip_kg': limit,
        'source_limit_kg': limit,
        'source': 'process',
        'source_label': src.label,
      };
      _step = 1;
      if (_kg.text.trim().isEmpty && limit > 0) {
        _kg.text = limit.toString();
      }
    });
  }

  Map<String, dynamic> _submitBody() => _baseBody();

  String _okMessage(Map<String, dynamic> data) {
    final kg = data['issue_kg'] ?? data['kg'] ?? _kg.text;
    final trace = (data['trace_code'] ?? _selectedTraceCode ?? '').toString();
    final pending = data['pending'] == true || data['biz_status'] == 'issue_pending_warehouse';
    if (pending) {
      return '已提交 $kg kg（溯源 $trace），待仓管确认后领料完成';
    }
    final wage = data['issue_locked_wage_amount'];
    final wageText = wage is num && wage != 0 ? ' · 预估¥${wage.toStringAsFixed(2)}' : '';
    return '已领取 $kg kg（溯源 $trace）$wageText';
  }

  Future<void> _submit() async {
    final err = _validateAction();
    if (err != null) {
      _prompt(err);
      return;
    }
    setState(() {
      _busy = true;
      _msg = '';
      _msgIsError = false;
    });
    if (!mounted) return;
    final r = await context.read<AuthState>().api.post('/production/board-issues', _submitBody());
    if (!mounted) return;
    setState(() => _busy = false);
    if (!r.ok) {
      var msg = r.msg;
      if (msg.contains('TRACE_PRODUCTION_NOT_STARTED')) {
        msg = '该溯源码未进入生产状态，不允许领取';
      }
      _prompt(msg);
      return;
    }
    final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
    final okMsg = _okMessage(data);
    if (_mode == StationPassMode.proxy && _badge.text.trim().isNotEmpty) {
      await RecentCodeStore.remember(RecentCodeStore.badge, _badge.text, upper: false);
    }
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(okMsg)));
    if (_isSubPage && Navigator.of(context).canPop()) {
      Navigator.of(context).pop(true);
      return;
    }
    _kg.clear();
    _reweighPhotoUrl = null;
    if (_mode == StationPassMode.self) _fillSelfBadgeIfNeeded();
    setState(() {
      _preview = data;
      _step = 0;
      _msg = okMsg;
      _msgIsError = false;
    });
  }

  Widget _previewRow(String label, String value, {bool emphasize = false}) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          SizedBox(
            width: 108,
            child: Text(label, style: TextStyle(fontSize: 13, color: Colors.black.withValues(alpha: 0.6))),
          ),
          Expanded(
            child: Text(
              value,
              textAlign: TextAlign.right,
              style: TextStyle(fontSize: emphasize ? 16 : 14, fontWeight: emphasize ? FontWeight.w700 : FontWeight.normal),
            ),
          ),
        ],
      ),
    );
  }

  String _fmtKg(dynamic v) => v is num ? v.toStringAsFixed(2) : '-';
  String _fmtMoney(dynamic v) => v is num ? '¥${v.toStringAsFixed(2)}' : '-';

  String get _pageTitle {
    final base = _mode == StationPassMode.self
        ? '本人领料'
        : _mode == StationPassMode.proxy
            ? '代人领料'
            : '生产';
    if (_mode != StationPassMode.home && _step == 1) return '$base · 领料';
    return base;
  }

  String _submitLabel() => _fromWarehouse ? '提交申请' : '确认领取';

  String _actionHint() {
    if (_fromWarehouse) {
      return '仓库领料：提交后待仓管指定板码并复磅过账，确认后才算领取完成。';
    }
    return '工序领料：条件满足即领取完成；退库请到「领料历史」，入库请到「入库申请」。';
  }

  @override
  Widget build(BuildContext context) {
    final showAppBar = _isSubPage || !widget.asTab;
    return Scaffold(
      appBar: showAppBar
          ? AppBar(
              title: Text(_pageTitle),
              leading: _isSubPage ? IconButton(icon: const Icon(Icons.arrow_back), onPressed: _leaveOrBack) : null,
            )
          : AppBar(toolbarHeight: 0),
      body: _mode == StationPassMode.home ? _buildHome() : _buildFormFlow(),
    );
  }

  Widget _buildHome() {
    return ListView(
      padding: EdgeInsets.fromLTRB(16, widget.asTab ? 40 : 16, 16, tabShellBottomPadding(context, asTab: widget.asTab)),
      children: [
        const Text('生产', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
        const SizedBox(height: 4),
        const Text(
          '先选工序 → 选生产中溯源 → 选来源领取。仓库来源须仓管确认；工序来源即时完成。',
          style: TextStyle(fontSize: 13, color: Colors.black54),
        ),
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
          icon: Icons.person_outline,
          title: '本人领料',
          subtitle: '选生产中溯源与来源，本人工牌领取',
          onTap: () => _openMode(StationPassMode.self),
        ),
        HubEntryTile(
          icon: Icons.group_outlined,
          title: '代人领料',
          subtitle: '扫他人工牌后代领（须生管角色）',
          onTap: () => _openMode(StationPassMode.proxy),
        ),
        HubEntryTile(
          icon: Icons.history,
          title: '领料历史 / 退库',
          subtitle: '查看已领、待仓管确认、申请退库',
          onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const ProcessIssueHistoryPage())),
        ),
        HubEntryTile(
          icon: Icons.inventory_2_outlined,
          title: '入库申请',
          subtitle: '独立建单，须溯源+工序，仓管过磅后过账',
          onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const ProcessStockInApplyPage())),
        ),
        if (_canForeman)
          HubEntryTile(
            icon: Icons.qr_code_2_outlined,
            title: '溯源生产台',
            subtitle: '库中/生产中/已结束；进入/结束生产',
            onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const TraceProductionPage())),
          ),
      ],
    );
  }

  Widget _buildFormFlow() {
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;
    return Column(
      children: [
        Expanded(
          child: _step == 0
              ? ListView(
                  keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
                  padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + bottomInset),
                  children: _formFields(),
                )
              : _actionBody(),
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
        if (_step == 0)
          FormStickyActions(
            primaryLabel: _busy ? '解析中…' : '下一步',
            onPrimary: _busy ? null : _goPreview,
            primaryBusy: _busy,
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
            primaryLabel: _submitLabel(),
            onPrimary: _busy ? null : _submit,
            primaryBusy: _busy,
            busyLabel: '提交中…',
          ),
      ],
    );
  }

  List<Widget> _formFields() {
    return [
      const FormSectionHeader('工序'),
      Padding(
        padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
        child: DropdownButtonFormField<int>(
          key: ValueKey('proc-${_processId ?? 0}-${_processes.length}'),
          initialValue: _processId != null && _processes.any((p) => (p['id'] as num?)?.toInt() == _processId) ? _processId : null,
          decoration: InputDecoration(
            labelText: '领入工序（本批工艺）',
            border: const OutlineInputBorder(),
            isDense: true,
            helperText: (_selectedTraceCode ?? '').isEmpty
                ? '请先选择进行中的溯源生产'
                : (_processes.isEmpty ? '该批次未锁定工艺或无可选工序' : null),
          ),
          items: _processes
              .map((p) {
                final id = (p['id'] as num?)?.toInt();
                if (id == null) return null;
                return DropdownMenuItem<int>(value: id, child: Text('${p['name'] ?? p['code'] ?? id}'));
              })
              .whereType<DropdownMenuItem<int>>()
              .toList(),
          onChanged: _processes.isEmpty ? null : (v) => setState(() => _processId = v),
        ),
      ),
      const FormSectionHeader('领料人'),
      if (_mode == StationPassMode.self)
        FormRow.text(
          label: '工牌',
          controller: _badge,
          hint: (context.read<AuthState>().badgeCode ?? '').trim().isEmpty ? '未绑定工牌（将按当前用户领料）' : '本人工牌（不可改）',
          readOnly: true,
        )
      else
        TraceCodeField(
          controller: _badge,
          label: '工牌',
          hint: '手输、扫码或点最近使用',
          scannerTitle: '扫描工牌',
          textCapitalization: TextCapitalization.none,
          historyKey: RecentCodeStore.badge,
        ),
      const FormSectionHeader('溯源码'),
      ActiveTraceDropdown(
        value: _selectedTraceCode,
        prefKey: _tracePrefKey,
        onChanged: _onTraceSelected,
      ),
      const FormSectionHeader('领料来源'),
      if (_loadingSources)
        const Padding(padding: EdgeInsets.all(8), child: LinearProgressIndicator())
      else if ((_selectedTraceCode ?? '').isNotEmpty && _sources.isEmpty)
        const Padding(
          padding: EdgeInsets.symmetric(horizontal: 8, vertical: 4),
          child: Text('该溯源暂无可领来源', style: TextStyle(fontSize: 12, color: Colors.orange)),
        )
      else if (_sources.isNotEmpty)
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
          child: DropdownButtonFormField<String>(
            key: ValueKey('src-${_selectedSourceKey ?? ''}-${_sources.length}'),
            initialValue: _selectedSourceKey != null && _sources.any((s) => s.key == _selectedSourceKey) ? _selectedSourceKey : null,
            decoration: const InputDecoration(labelText: '从哪领取', border: OutlineInputBorder(), isDense: true),
            items: _sources.map((s) => DropdownMenuItem<String>(value: s.key, child: Text(s.label))).toList(),
            onChanged: _onSourceChanged,
          ),
        ),
      if (_fromWarehouse)
        const Padding(
          padding: EdgeInsets.fromLTRB(8, 4, 8, 0),
          child: Text('仓库来源：提交后由仓管指定板码并复磅过账', style: TextStyle(fontSize: 12, color: Colors.black54)),
        ),
    ];
  }

  Widget _actionBody() {
    final p = _preview ?? const <String, dynamic>{};
    final workerName = (p['worker_name'] ?? '').toString();
    final badge = (p['badge_code'] ?? _badge.text).toString().trim();
    final forOther = p['pass_for_other'] == true || _mode == StationPassMode.proxy;
    final stepName = (p['process_name'] ?? '-').toString();
    final passerText = [if (workerName.isNotEmpty) workerName, if (badge.isNotEmpty) badge].join(' · ');
    final traceText = '${p['trace_code'] ?? _selectedTraceCode ?? '-'}';
    final farmer = '${p['farmer_name'] ?? _selectedTraceRow()?['farmer_name'] ?? ''}'.trim();
    final pending = p['pending'] == true || _fromWarehouse;
    return ListView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
      children: [
        Row(
          children: [
            const Expanded(child: Text('领料核对', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13))),
            TextButton(
              onPressed: () => setState(() {
                _step = 0;
                _msg = '';
                _msgIsError = false;
              }),
              child: const Text('修改'),
            ),
          ],
        ),
        const FormSectionHeader('材料'),
        _previewRow('来源', '${p['source_label'] ?? (_fromWarehouse ? '仓库' : '工序')}'),
        _previewRow('溯源码', traceText, emphasize: true),
        if (farmer.isNotEmpty) _previewRow('农户', farmer),
        if (!_fromWarehouse && (p['from_process_name'] ?? '').toString().isNotEmpty)
          _previewRow('来源工序', '${p['from_process_name']}'),
        if (!_fromWarehouse) _previewRow('在制(kg)', _fmtKg(p['source_limit_kg'] ?? p['wip_kg'] ?? p['available_kg'])),
        if (pending) _previewRow('说明', '提交后待仓管确认'),
        _previewRow('模式', forOther ? '代人领料' : '本人领料'),
        _previewRow('工牌', passerText.isEmpty ? '当前用户（未填工牌）' : passerText),
        _previewRow('领入工序', stepName),
        if (p['piecework'] == true) ...[
          _previewRow('计件工价', '${p['rate'] ?? '-'} 元/kg'),
          _previewRow('预估工钱', _fmtMoney(p['locked_wage_amount'])),
        ],
        const SizedBox(height: 8),
        const FormSectionHeader('领取'),
        FormRow.text(
          label: '重量(kg)',
          controller: _kg,
          keyboardType: const TextInputType.numberWithOptions(decimal: true),
          requiredMark: true,
        ),
        const SizedBox(height: 8),
        ListTile(
          contentPadding: EdgeInsets.zero,
          leading: Icon(
            (_reweighPhotoUrl ?? '').isEmpty ? Icons.camera_alt_outlined : Icons.check_circle,
            color: (_reweighPhotoUrl ?? '').isEmpty ? Colors.orange : Colors.teal,
          ),
          title: Text((_reweighPhotoUrl ?? '').isEmpty ? '拍摄复磅照片（必填）' : '复磅照片已上传'),
          trailing: TextButton(onPressed: _busy ? null : _takeReweighPhoto, child: const Text('拍照')),
        ),
        const SizedBox(height: 8),
        Text(_actionHint(), style: const TextStyle(fontSize: 12, color: Colors.black54)),
      ],
    );
  }
}
