import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/auth_state.dart';
import '../../core/carrier_code_labels.dart';
import '../../core/notify_service.dart';
import '../../core/recent_code_store.dart';
import '../../widgets/form_row.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_sticky_actions.dart';
import '../../widgets/hub_entry_tile.dart';
import '../../widgets/trace_code_field.dart';
import 'trace_production_page.dart';
import 'process_issue_history_page.dart';
import 'process_stock_in_apply_page.dart';

enum StationPassMode { home, self, proxy }

/// 生产：仓库出库领料（板+溯源）或工序领料（仅溯源）；退库走历史，入库走独立申请。
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
  /// 0 扫码 · 1 操作
  int _step = 0;
  final _badge = TextEditingController();
  final _box = TextEditingController();
  final _trace = TextEditingController();
  final _kg = TextEditingController();
  String _msg = '';
  bool _msgIsError = false;
  bool _busy = false;
  Map<String, dynamic>? _preview;
  List<Map<String, dynamic>> _processes = [];
  int? _processId;
  String? _reweighPhotoUrl;
  /// warehouse | process
  String _issueSource = 'warehouse';

  static const _issueSourcePrefKey = 'erp.station.issue_source';

  String get _codeLabel => context.read<CarrierCodeLabels>().code;
  bool get _fromWarehouse => _issueSource == 'warehouse';

  Map<String, String> get _errLabel => {
    'TRACE_CODE_REQUIRED': '请填写溯源码',
    'PRODUCT_REQUIRED': '请选择物料品类',
    'BOARD_FINISHED': '该$_codeLabel已完工，不能再操作',
    'QTY_EXCEEDS_AVAILABLE': '领取重量超过可领量',
    'QTY_EXCEEDS_OCCUPANCY': '退库重量超过本人占用',
    'QTY_EXCEEDS_WIP': '重量超过本工序在制',
    'BOX_REQUIRED': '仓库出库须扫板码',
    'BOX_NOT_FOUND': '未找到该$_codeLabel',
    'PROCESS_REQUIRED': '请先选择工序',
    'AUTO_ROUTING_DISABLED': '已取消按工艺自动进下道，请指定工序后入库或再领取',
    'SHIFT_NOT_AUTHORIZED': '当前班次未授权该工序',
    'ROLE_FORBIDDEN': '仅生管可代领',
    'FEATURE_REMOVED:board_close': '板结束已取消，请在溯源生产台结束生产',
    'REWEIGH_REQUIRED': '请填写复磅重量',
    'REWEIGH_PHOTO_REQUIRED': '请拍摄复磅照片',
    'TRACE_MISMATCH': '板码与溯源码不一致',
    'STOCK_OUT_FAILED': '仓库出库记账失败',
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
      final savedSource = prefs.getString(_issueSourcePrefKey);
      if (savedSource == 'warehouse' || savedSource == 'process') {
        _issueSource = savedSource!;
      }
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
      context.read<NotifyService>().addListener(_onNotify);
      await _loadProcesses();
      if (_fromWarehouse && _box.text.trim().isEmpty) {
        final boards = await RecentCodeStore.list(RecentCodeStore.board);
        if (mounted && boards.isNotEmpty && _box.text.trim().isEmpty) {
          setState(() => _box.text = boards.first);
        }
      }
      if (_trace.text.trim().isEmpty) {
        final traces = await RecentCodeStore.list(RecentCodeStore.trace);
        if (mounted && traces.isNotEmpty && _trace.text.trim().isEmpty) {
          setState(() => _trace.text = traces.first);
        }
      }
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
      _processes = list;
      if (_processId != null && !list.any((p) => (p['id'] as num?)?.toInt() == _processId)) {
        _processId = null;
      }
      if (_processId == null && list.isNotEmpty) {
        _processId = (list.first['id'] as num?)?.toInt();
      }
    });
  }

  Future<void> _setIssueSource(String source) async {
    if (source != 'warehouse' && source != 'process') return;
    setState(() => _issueSource = source);
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_issueSourcePrefKey, source);
  }

  @override
  void dispose() {
    try {
      context.read<NotifyService>().removeListener(_onNotify);
    } catch (_) {}
    _badge.dispose();
    _box.dispose();
    _trace.dispose();
    _kg.dispose();
    super.dispose();
  }

  void _onNotify() {
    if (!mounted) return;
    final notify = context.read<NotifyService>();
    for (final raw in notify.inbox) {
      if (raw is! Map) continue;
      final p = NotifyService.parsePayload(raw['payload'] ?? raw['payload_json']);
      final next = p['next'] is Map ? Map<String, dynamic>.from(p['next'] as Map) : p;
      final code = next['new_box_code'] ?? next['board_code'];
      if (code != null && _fromWarehouse && _box.text.trim().isEmpty) {
        setState(() => _box.text = code.toString());
        break;
      }
    }
  }

  void _fillSelfBadgeIfNeeded() {
    if (_mode != StationPassMode.self) return;
    final auth = context.read<AuthState>();
    final mine = (auth.badgeCode ?? '').trim();
    _badge.text = mine;
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
      MaterialPageRoute(
        builder: (_) => StationPassPage(initialMode: mode),
      ),
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
    final badge = _badge.text.trim();
    final kg = double.tryParse(_kg.text) ?? 0;
    final body = <String, dynamic>{
      'source': _issueSource,
      'kg': kg,
      'reweigh_kg': kg,
    };
    if (badge.isNotEmpty) body['badge_code'] = badge;
    if (_processId != null && _processId! > 0) body['process_id'] = _processId;
    if ((_reweighPhotoUrl ?? '').isNotEmpty) {
      body['photo_url'] = _reweighPhotoUrl;
      body['image_url'] = _reweighPhotoUrl;
    }
    final trace = _trace.text.trim();
    if (trace.isNotEmpty) body['trace_code'] = trace;
    if (_fromWarehouse) {
      final board = _box.text.trim();
      body['board_code'] = board;
      body['box_code'] = board;
    }
    return body;
  }

  String? _validateScan() {
    if (_processId == null || _processId! <= 0) return '请先选择工序';
    if (_fromWarehouse) {
      if (_box.text.trim().isEmpty) return '请填写或扫描$_codeLabel';
      if (_trace.text.trim().isEmpty) return '请填写或扫描溯源码';
    } else {
      if (_trace.text.trim().isEmpty) return '请填写或扫描溯源码';
    }
    return null;
  }

  String? _validateAction() {
    if (_processId == null || _processId! <= 0) return '请先选择工序';
    final kg = double.tryParse(_kg.text) ?? 0;
    if (kg <= 0) return '请填写重量(kg)';
    if ((_reweighPhotoUrl ?? '').isEmpty) return '请拍摄复磅照片';
    if (_fromWarehouse) {
      final p = _preview ?? const <String, dynamic>{};
      final avail = (p['available_kg'] as num?)?.toDouble() ?? 0;
      if (kg > avail + 0.0005) return '领取重量不能超过板可领 ${avail.toStringAsFixed(2)} kg';
    }
    return null;
  }

  Future<void> _takeReweighPhoto() async {
    try {
      final picker = ImagePicker();
      final file = await picker.pickImage(source: ImageSource.camera, imageQuality: 85);
      if (file == null) return;
      final bytes = await file.readAsBytes();
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
    if (badge.isNotEmpty) {
      await prefs.setString('erp.worker.badge', badge);
    }
    if (_processId != null && _processId! > 0) {
      await prefs.setInt('erp.station.process_id', _processId!);
    }
    await prefs.setString(_issueSourcePrefKey, _issueSource);
    if (!mounted) return;

    // 工序领料：跳过 scan/resolve，本地轻量预览（不校验可领量上限）
    if (!_fromWarehouse) {
      setState(() {
        _busy = false;
        _preview = {
          'trace_code': _trace.text.trim(),
          'process_id': _processId,
          'available_kg': 999999,
          'source': 'process',
        };
        _step = 1;
        _msg = '';
        _msgIsError = false;
      });
      return;
    }

    final api = context.read<AuthState>().api;
    final r = await api.post('/production/scan/resolve', _baseBody());
    if (!mounted) return;
    setState(() => _busy = false);
    if (!r.ok) {
      _prompt(r.msg);
      return;
    }
    final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
    final previewTrace = (data['trace_code'] ?? '').toString().trim();
    if (_trace.text.trim().isEmpty && previewTrace.isNotEmpty) {
      _trace.text = previewTrace;
    }
    final avail = (data['available_kg'] as num?)?.toDouble() ?? 0;
    if (_kg.text.trim().isEmpty && avail > 0) {
      _kg.text = avail.toString();
    }
    setState(() {
      _preview = data;
      _step = 1;
      _msg = '';
      _msgIsError = false;
    });
  }

  Map<String, dynamic> _submitBody() {
    final body = _baseBody();
    final kg = double.tryParse(_kg.text) ?? 0;
    body['kg'] = kg;
    body['reweigh_kg'] = kg;
    if ((_reweighPhotoUrl ?? '').isNotEmpty) {
      body['photo_url'] = _reweighPhotoUrl;
      body['image_url'] = _reweighPhotoUrl;
    }
    return body;
  }

  String _okMessage(Map<String, dynamic> data) {
    final kg = data['issue_kg'] ?? data['kg'] ?? _kg.text;
    final board = (data['board_code'] ?? (_fromWarehouse ? _box.text : '')).toString().trim();
    final fromField = _trace.text.trim();
    final fromPreview = (_preview?['trace_code'] ?? '').toString().trim();
    final trace = (data['trace_code'] ?? (fromField.isNotEmpty ? fromField : fromPreview)).toString().trim();
    final loc = [
      if (board.isNotEmpty) '$_codeLabel $board',
      if (trace.isNotEmpty) '溯源 $trace',
    ].join(' · ');
    final locText = loc.isEmpty ? '' : '（$loc）';
    final wage = data['issue_locked_wage_amount'];
    final wageText = wage is num && wage != 0 ? ' · 预估¥${wage.toStringAsFixed(2)}' : '';
    final src = _fromWarehouse ? '仓库出库' : '工序领料';
    return '已领取 $kg kg$locText$wageText（$src，预估工钱，日结入账）';
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
    final prefs = await SharedPreferences.getInstance();
    final badge = _badge.text.trim();
    if (badge.isNotEmpty) {
      await prefs.setString('erp.worker.badge', badge);
    }
    if (_processId != null && _processId! > 0) {
      await prefs.setInt('erp.station.process_id', _processId!);
    }
    await prefs.setString(_issueSourcePrefKey, _issueSource);
    if (!mounted) return;
    final api = context.read<AuthState>().api;
    final r = await api.post('/production/board-issues', _submitBody());
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
    if (_fromWarehouse && _box.text.trim().isNotEmpty) {
      await RecentCodeStore.remember(RecentCodeStore.board, _box.text);
    }
    if (_trace.text.trim().isNotEmpty) {
      await RecentCodeStore.remember(RecentCodeStore.trace, _trace.text);
    }
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
    if (_mode == StationPassMode.self) {
      _fillSelfBadgeIfNeeded();
    }
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

  String _fmtKg(dynamic v) {
    if (v is num) return v.toStringAsFixed(2);
    return '-';
  }

  String _fmtMoney(dynamic v) {
    if (v is num) return '¥${v.toStringAsFixed(2)}';
    return '-';
  }

  String get _pageTitle {
    final base = _mode == StationPassMode.self
        ? '本人领料'
        : _mode == StationPassMode.proxy
            ? '代人领料'
            : '生产';
    if (_mode != StationPassMode.home && _step == 1) {
      return '$base · 领料';
    }
    return base;
  }

  String _submitLabel() => '确认领取';

  String _actionHint() {
    if (_fromWarehouse) {
      return '仓库出库：按所选工序从本$_codeLabel可领池按 kg 领取；计件工钱预锁定，确认结束后日结。退库请到「领料历史」，入库请到生产首页「入库申请」。';
    }
    return '工序领料：仅需溯源码，按所选工序领取；计件工钱预锁定，确认结束后日结。退库请到「领料历史」，入库请到生产首页「入库申请」。';
  }

  String get _hubIssueHint {
    if (_fromWarehouse) {
      return '仓库出库须扫$_codeLabel+溯源；工序领料仅溯源';
    }
    return '当前为工序领料（仅溯源）；可切换为仓库出库（$_codeLabel+溯源）';
  }

  @override
  Widget build(BuildContext context) {
    final showAppBar = _isSubPage || !widget.asTab;
    return Scaffold(
      appBar: showAppBar
          ? AppBar(
              title: Text(_pageTitle),
              leading: _isSubPage
                  ? IconButton(icon: const Icon(Icons.arrow_back), onPressed: _leaveOrBack)
                  : null,
            )
          : AppBar(toolbarHeight: 0),
      body: _mode == StationPassMode.home ? _buildHome() : _buildFormFlow(),
    );
  }

  Widget _buildHome() {
    final code = context.watch<CarrierCodeLabels>().code;
    return ListView(
      padding: EdgeInsets.fromLTRB(16, widget.asTab ? 40 : 16, 16, 16),
      children: [
        const Text('生产', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
        const SizedBox(height: 4),
        Text(
          '先选工序；仓库出库扫$code+溯源，工序领料仅扫溯源。退库/入库走申请+仓管过磅；计件以「确认结束」后日结为准',
          style: const TextStyle(fontSize: 13, color: Colors.black54),
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
          subtitle: _fromWarehouse
              ? '锁定本人工牌，指定工序后扫$code+溯源领取'
              : '锁定本人工牌，指定工序后仅扫溯源领取',
          onTap: () => _openMode(StationPassMode.self),
        ),
        HubEntryTile(
          icon: Icons.group_outlined,
          title: '代人领料',
          subtitle: _fromWarehouse
              ? '指定工序，扫他人工牌后再扫$code+溯源（须生管角色）'
              : '指定工序，扫他人工牌后再扫溯源（须生管角色）',
          onTap: () => _openMode(StationPassMode.proxy),
        ),
        HubEntryTile(
          icon: Icons.history,
          title: '领料历史 / 退库',
          subtitle: '查看已领、申请部分退库；主任可确认结束',
          onTap: () => Navigator.of(context).push(
            MaterialPageRoute(builder: (_) => const ProcessIssueHistoryPage()),
          ),
        ),
        HubEntryTile(
          icon: Icons.inventory_2_outlined,
          title: '入库申请',
          subtitle: '独立建单，须溯源+工序，仓管过磅后过账',
          onTap: () => Navigator.of(context).push(
            MaterialPageRoute(builder: (_) => const ProcessStockInApplyPage()),
          ),
        ),
        if (_canForeman)
          HubEntryTile(
            icon: Icons.qr_code_2_outlined,
            title: '溯源生产台',
            subtitle: '库中/生产中/已结束；工序分布；进入/结束生产',
            onTap: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => const TraceProductionPage()),
            ),
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
      const FormSectionHeader('领料来源'),
      Padding(
        padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
        child: SegmentedButton<String>(
          segments: const [
            ButtonSegment(value: 'warehouse', label: Text('仓库出库'), icon: Icon(Icons.warehouse_outlined, size: 18)),
            ButtonSegment(value: 'process', label: Text('工序领料'), icon: Icon(Icons.precision_manufacturing_outlined, size: 18)),
          ],
          selected: {_issueSource},
          onSelectionChanged: (s) {
            if (s.isEmpty) return;
            _setIssueSource(s.first);
          },
        ),
      ),
      Padding(
        padding: const EdgeInsets.fromLTRB(8, 0, 8, 4),
        child: Text(_hubIssueHint, style: const TextStyle(fontSize: 12, color: Colors.black54)),
      ),
      const FormSectionHeader('工序'),
      Padding(
        padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
        child: DropdownButtonFormField<int>(
          key: ValueKey('proc-${_processId ?? 0}-${_processes.length}'),
          initialValue: _processId != null && _processes.any((p) => (p['id'] as num?)?.toInt() == _processId) ? _processId : null,
          decoration: const InputDecoration(
            labelText: '本站工序',
            border: OutlineInputBorder(),
            isDense: true,
          ),
          items: _processes
              .map((p) {
                final id = (p['id'] as num?)?.toInt();
                if (id == null) return null;
                final name = '${p['name'] ?? p['code'] ?? id}';
                return DropdownMenuItem<int>(value: id, child: Text(name));
              })
              .whereType<DropdownMenuItem<int>>()
              .toList(),
          onChanged: (v) => setState(() => _processId = v),
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
      if (_fromWarehouse) ...[
        FormSectionHeader(_codeLabel),
        TraceCodeField(
          controller: _box,
          label: _codeLabel,
          hint: '手输、扫码或点最近使用',
          scannerTitle: '扫描$_codeLabel',
          historyKey: RecentCodeStore.board,
        ),
      ],
      const FormSectionHeader('溯源码'),
      TraceCodeField(
        controller: _trace,
        label: '溯源码',
        hint: _fromWarehouse ? '须与板一致；可点最近使用' : '手输、扫码或点最近使用',
        scannerTitle: '扫描溯源码',
        historyKey: RecentCodeStore.trace,
      ),
    ];
  }

  Widget _actionBody() {
    final p = _preview ?? const <String, dynamic>{};
    final code = context.watch<CarrierCodeLabels>().code;
    final workerName = (p['worker_name'] ?? '').toString();
    final badge = (p['badge_code'] ?? _badge.text).toString().trim();
    final forOther = p['pass_for_other'] == true || _mode == StationPassMode.proxy;
    final stepName = (p['step_name'] ?? p['process_name'] ?? '-').toString();
    final passerText = [
      if (workerName.isNotEmpty) workerName,
      if (badge.isNotEmpty) badge,
    ].join(' · ');
    final boardText = _box.text.trim().isEmpty ? '${p['board_code'] ?? '-'}' : _box.text.trim();
    final traceText = _trace.text.trim().isEmpty ? '${p['trace_code'] ?? '-'}' : _trace.text.trim();
    return ListView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
      children: [
        Row(
          children: [
            Expanded(
              child: Text(
                _fromWarehouse ? '$code信息' : '领料核对',
                style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13),
              ),
            ),
            TextButton(
              onPressed: () => setState(() {
                _step = 0;
                _msg = '';
                _msgIsError = false;
              }),
              child: Text(_fromWarehouse ? '改$code' : '修改'),
            ),
          ],
        ),
        const FormSectionHeader('材料'),
        _previewRow('来源', _fromWarehouse ? '仓库出库' : '工序领料'),
        if (_fromWarehouse) _previewRow(code, boardText, emphasize: true),
        _previewRow('溯源码', traceText, emphasize: true),
        if ((p['product_name'] ?? '').toString().isNotEmpty)
          _previewRow('物料', '${p['product_name']}${p['product_category'] != null && '${p['product_category']}' != '' ? ' · ${p['product_category']}' : ''}'),
        _previewRow('模式', forOther ? '代人领料' : '本人领料'),
        _previewRow('工牌', passerText.isEmpty ? '当前用户（未填工牌）' : passerText),
        _previewRow('指定工序', stepName),
        if (_fromWarehouse) ...[
          _previewRow('$code可领(kg)', _fmtKg(p['available_kg'])),
          _previewRow('本人占用(kg)', _fmtKg(p['my_open_kg'])),
          _previewRow('本工序在制(kg)', _fmtKg(p['wip_kg'])),
        ],
        if (p['piecework'] == true) ...[
          _previewRow('计件工价', '${p['rate'] ?? '-'} 元/kg'),
          _previewRow('预估工钱', _fmtMoney(p['locked_wage_amount'])),
          _previewRow('说明', '${p['piecework_hint'] ?? '预估工钱，当日日结入账'}'),
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
        Text(
          _actionHint(),
          style: const TextStyle(fontSize: 12, color: Colors.black54),
        ),
      ],
    );
  }
}
