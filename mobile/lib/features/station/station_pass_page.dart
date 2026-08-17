import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/auth_state.dart';
import '../../core/carrier_code_labels.dart';
import '../../core/notify_service.dart';
import '../../widgets/form_row.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_sticky_actions.dart';
import '../../widgets/hub_entry_tile.dart';
import '../../widgets/trace_code_field.dart';

enum StationPassMode { home, self, proxy, close }

enum StationBoardAction { issue, ret, stockIn }

/// 工序过站：指定工序后扫工牌+载体码，按 kg 领取 / 退库 / 入库换码。
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
  final _kg = TextEditingController();
  StationBoardAction _action = StationBoardAction.issue;
  String _msg = '';
  bool _msgIsError = false;
  bool _busy = false;
  Map<String, dynamic>? _preview;
  List<Map<String, dynamic>> _processes = [];
  int? _processId;

  String get _codeLabel => context.read<CarrierCodeLabels>().code;

  Map<String, String> get _errLabel => {
    'TRACE_CODE_REQUIRED': '该$_codeLabel缺少溯源码，无法领取',
    'BOARD_FINISHED': '该$_codeLabel已完工，不能再操作',
    'QTY_EXCEEDS_AVAILABLE': '领取重量超过$_codeLabel可领',
    'QTY_EXCEEDS_OCCUPANCY': '退库重量超过本人占用',
    'QTY_EXCEEDS_WIP': '重量超过本工序在制',
    'BOX_REQUIRED': '请填写或扫描$_codeLabel',
    'BOX_NOT_FOUND': '未找到该$_codeLabel',
    'PROCESS_REQUIRED': '请先选择工序',
    'AUTO_ROUTING_DISABLED': '已取消按工艺自动进下道，请指定工序后入库或再领取',
    'SHIFT_NOT_AUTHORIZED': '当前班次未授权该工序',
    'ROLE_FORBIDDEN': '仅生管可确认板结束',
    'REMAIN_NEEDS_DECISION': '该$_codeLabel仍有物料，请确认为耗损或返回补工序',
    'APP_ONLY': '请在 App 操作',
  };

  bool get _isSubPage => widget.initialMode != StationPassMode.home;

  bool get _isClose => _mode == StationPassMode.close;

  bool get _canCloseBoard {
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
    });
  }

  Future<void> _loadProcesses() async {
    if (_isClose) return;
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

  @override
  void dispose() {
    try {
      context.read<NotifyService>().removeListener(_onNotify);
    } catch (_) {}
    _badge.dispose();
    _box.dispose();
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
      if (code != null && _box.text.trim().isEmpty) {
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
    final body = <String, dynamic>{
      'board_code': _box.text.trim(),
      'box_code': _box.text.trim(),
      'kg': double.tryParse(_kg.text) ?? 0,
    };
    if (badge.isNotEmpty) body['badge_code'] = badge;
    if (_processId != null && _processId! > 0) body['process_id'] = _processId;
    return body;
  }

  String? _validateScan() {
    if (!_isClose && (_processId == null || _processId! <= 0)) return '请先选择工序';
    if (_box.text.trim().isEmpty) return '请填写或扫描$_codeLabel';
    return null;
  }

  String? _validateAction() {
    if (_isClose) return null;
    if (_processId == null || _processId! <= 0) return '请先选择工序';
    final kg = double.tryParse(_kg.text) ?? 0;
    if (kg <= 0) return '请填写重量(kg)';
    final p = _preview ?? const <String, dynamic>{};
    final avail = (p['available_kg'] as num?)?.toDouble() ?? 0;
    final mine = (p['my_open_kg'] as num?)?.toDouble() ?? 0;
    final wip = (p['wip_kg'] as num?)?.toDouble() ?? (avail + ((p['process_open_kg'] as num?)?.toDouble() ?? 0));
    switch (_action) {
      case StationBoardAction.issue:
        if (kg > avail + 0.0005) return '领取重量不能超过板可领 ${avail.toStringAsFixed(2)} kg';
      case StationBoardAction.ret:
        if (kg > mine + 0.0005) return '退库重量不能超过本人占用 ${mine.toStringAsFixed(2)} kg';
      case StationBoardAction.stockIn:
        if (kg > wip + 0.0005) return '重量不能超过本工序在制 ${wip.toStringAsFixed(2)} kg';
    }
    return null;
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
    if (!mounted) return;
    final api = context.read<AuthState>().api;
    final path = _isClose ? '/production/board-close/preview' : '/production/scan/resolve';
    final r = await api.post(path, _baseBody());
    if (!mounted) return;
    setState(() => _busy = false);
    if (!r.ok) {
      _prompt(r.msg);
      return;
    }
    final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
    final avail = (data['available_kg'] as num?)?.toDouble() ?? 0;
    if (_kg.text.trim().isEmpty && avail > 0) {
      _kg.text = avail.toString();
    }
    setState(() {
      _preview = data;
      _action = StationBoardAction.issue;
      _step = 1;
      _msg = '';
      _msgIsError = false;
    });
  }

  String get _actionPath {
    switch (_action) {
      case StationBoardAction.issue:
        return '/production/board-issues';
      case StationBoardAction.ret:
        return '/production/board-issues/return';
      case StationBoardAction.stockIn:
        return '/production/board-moves';
    }
  }

  Map<String, dynamic> _submitBody() {
    if (_isClose) {
      return {
        'board_code': _box.text.trim(),
        'box_code': _box.text.trim(),
        'confirm_loss': true,
      };
    }
    final body = _baseBody();
    if (_action == StationBoardAction.stockIn) {
      body['move_kind'] = 'stock_in';
    }
    return body;
  }

  String _okMessage(Map<String, dynamic> data) {
    final kg = data['issue_kg'] ?? data['returned_kg'] ?? data['kg'] ?? _kg.text;
    final board = (data['board_code'] ?? _box.text).toString().trim();
    final trace = (data['trace_code'] ?? _preview?['trace_code'] ?? '').toString().trim();
    final loc = [
      if (board.isNotEmpty) '$_codeLabel $board',
      if (trace.isNotEmpty) '溯源 $trace',
    ].join(' · ');
    final locText = loc.isEmpty ? '' : '（$loc）';
    final wage = data['issue_locked_wage_amount'] ?? data['released_locked_wage_amount'];
    final wageText = wage is num && wage != 0 ? ' · 预估¥${wage.toStringAsFixed(2)}' : '';
    switch (_action) {
      case StationBoardAction.issue:
        return '已领取 $kg kg$locText$wageText（预估工钱，日结入账）';
      case StationBoardAction.ret:
        return '已退库 $kg kg$locText$wageText（扣减预估，未日结不入汇总）';
      case StationBoardAction.stockIn:
        final newCode = (data['new_board_code'] ?? data['new_box_code'] ?? '').toString().trim();
        if (newCode.isNotEmpty) {
          return '已入库 $kg kg · 新$_codeLabel $newCode$locText（仅换码，产量工钱请日结）';
        }
        return '已入库 $kg kg$locText（仅换码，产量工钱请日结）';
    }
  }

  String _closeOkMessage(Map<String, dynamic> data) {
    final board = (data['board_code'] ?? _box.text).toString().trim();
    final trace = (data['trace_code'] ?? _preview?['trace_code'] ?? '').toString().trim();
    final loc = [
      if (board.isNotEmpty) '板 $board',
      if (trace.isNotEmpty) '溯源 $trace',
    ].join(' · ');
    return loc.isEmpty ? '该板已结束' : '该板已结束（$loc）';
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
    if (!mounted) return;
    final api = context.read<AuthState>().api;
    final path = _isClose ? '/production/board-close' : _actionPath;
    final r = await api.post(path, _submitBody());
    if (!mounted) return;
    setState(() => _busy = false);
    if (!r.ok) {
      _prompt(r.msg);
      return;
    }
    final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
    final okMsg = _isClose ? _closeOkMessage(data) : _okMessage(data);
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(okMsg)));
    if (_isSubPage && Navigator.of(context).canPop()) {
      Navigator.of(context).pop(true);
      return;
    }
    final newCode = (data['new_board_code'] ?? data['new_box_code'] ?? '').toString().trim();
    _kg.clear();
    if (_action == StationBoardAction.stockIn && newCode.isNotEmpty) {
      _box.text = newCode;
    } else {
      _box.clear();
    }
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
        ? '本人过站'
        : _mode == StationPassMode.proxy
            ? '代人过站'
            : _mode == StationPassMode.close
                ? '板结束'
                : '工序过站';
    if (_mode != StationPassMode.home && _step == 1) {
      return _isClose ? '$base · 核对余量' : '$base · 领料';
    }
    return base;
  }

  String _submitLabel() {
    switch (_action) {
      case StationBoardAction.issue:
        return '确认领取';
      case StationBoardAction.ret:
        return '确认退库';
      case StationBoardAction.stockIn:
        return '确认入库换码';
    }
  }

  String _actionHint() {
    switch (_action) {
      case StationBoardAction.issue:
        return '按所选工序从本$_codeLabel可领池按 kg 领取；计件工钱预锁定，入库后结算。';
      case StationBoardAction.ret:
        return '从未完成占用退回本$_codeLabel本工序可领池；扣减计件锁定，退库部分不入日汇总。';
      case StationBoardAction.stockIn:
        return '按所选工序完工重量入库并分配新$_codeLabel；继续加工时请再选工序扫新码领取。';
    }
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
        const Text('工序过站', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
        const SizedBox(height: 4),
        Text('先选工序，扫$code后按 kg 领取、退库或入库换码', style: const TextStyle(fontSize: 13, color: Colors.black54)),
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
          title: '本人过站',
          subtitle: '锁定本人工牌，指定工序后扫$code领取/退库/入库',
          onTap: () => _openMode(StationPassMode.self),
        ),
        HubEntryTile(
          icon: Icons.group_outlined,
          title: '代人过站',
          subtitle: '指定工序，手输或扫描他人工牌后再扫$code',
          onTap: () => _openMode(StationPassMode.proxy),
        ),
        if (_canCloseBoard)
          HubEntryTile(
            icon: Icons.flag_outlined,
            title: '板结束',
            subtitle: '核对工序/仓库余量，确认为耗损后计算扣损',
            onTap: () => _openMode(StationPassMode.close),
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
        else if (_isClose)
          FormStickyActions(
            secondaryLabel: '补工序',
            onSecondary: _busy ? null : _leaveOrBack,
            primaryLabel: _closeSubmitLabel(),
            onPrimary: _busy ? null : _submit,
            primaryBusy: _busy,
            busyLabel: '提交中…',
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

  String _closeSubmitLabel() {
    final remain = (_preview?['total_remain_kg'] as num?)?.toDouble() ?? 0;
    if (remain > 0.0005) return '全部确认为耗损并结束';
    return '确认结束';
  }

  List<Widget> _formFields() {
    if (_isClose) {
      return [
        FormSectionHeader(_codeLabel),
        TraceCodeField(
          controller: _box,
          label: _codeLabel,
          hint: '手输或扫$_codeLabel',
          scannerTitle: '扫描$_codeLabel',
        ),
      ];
    }
    return [
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
      const FormSectionHeader('过站人'),
      if (_mode == StationPassMode.self)
        FormRow.text(
          label: '工牌',
          controller: _badge,
          hint: (context.read<AuthState>().badgeCode ?? '').trim().isEmpty ? '未绑定工牌（将按当前用户过站）' : '本人工牌（不可改）',
          readOnly: true,
        )
      else
        TraceCodeField(
          controller: _badge,
          label: '工牌',
          hint: '手输或扫描工牌，空则本人',
          scannerTitle: '扫描工牌',
          textCapitalization: TextCapitalization.none,
        ),
      FormSectionHeader(_codeLabel),
      TraceCodeField(
        controller: _box,
        label: _codeLabel,
        hint: '手输或扫$_codeLabel',
        scannerTitle: '扫描$_codeLabel',
      ),
    ];
  }

  Widget _actionBody() {
    if (_isClose) return _closeBody();
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
    return ListView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
      children: [
        Row(
          children: [
            Expanded(child: Text('$code信息', style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13))),
            TextButton(
              onPressed: () => setState(() {
                _step = 0;
                _msg = '';
                _msgIsError = false;
              }),
              child: Text('改$code'),
            ),
          ],
        ),
        const FormSectionHeader('材料'),
        _previewRow(code, _box.text.trim().isEmpty ? '${p['board_code'] ?? '-'}' : _box.text.trim(), emphasize: true),
        _previewRow('溯源码', '${p['trace_code'] ?? '-'}', emphasize: true),
        _previewRow('模式', forOther ? '代人过站' : '本人过站'),
        _previewRow('工牌', passerText.isEmpty ? '当前用户（未填工牌）' : passerText),
        _previewRow('指定工序', stepName),
        _previewRow('$code可领(kg)', _fmtKg(p['available_kg'])),
        _previewRow('本人占用(kg)', _fmtKg(p['my_open_kg'])),
        _previewRow('本工序在制(kg)', _fmtKg(p['wip_kg'])),
        if (p['piecework'] == true) ...[
          _previewRow('计件工价', '${p['rate'] ?? '-'} 元/kg'),
          _previewRow('预估工钱', _fmtMoney(p['locked_wage_amount'])),
          _previewRow('说明', '${p['piecework_hint'] ?? '预估工钱，当日日结入账'}'),
        ],
        const SizedBox(height: 8),
        const FormSectionHeader('操作'),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            ChoiceChip(
              label: const Text('领取'),
              selected: _action == StationBoardAction.issue,
              onSelected: (_) => setState(() => _action = StationBoardAction.issue),
            ),
            ChoiceChip(
              label: const Text('退库'),
              selected: _action == StationBoardAction.ret,
              onSelected: (_) => setState(() => _action = StationBoardAction.ret),
            ),
            ChoiceChip(
              label: const Text('入库'),
              selected: _action == StationBoardAction.stockIn,
              onSelected: (_) => setState(() => _action = StationBoardAction.stockIn),
            ),
          ],
        ),
        const SizedBox(height: 8),
        FormRow.text(
          label: '重量(kg)',
          controller: _kg,
          keyboardType: const TextInputType.numberWithOptions(decimal: true),
          requiredMark: true,
        ),
        const SizedBox(height: 8),
        Text(
          _actionHint(),
          style: const TextStyle(fontSize: 12, color: Colors.black54),
        ),
      ],
    );
  }

  Widget _closeBody() {
    final p = _preview ?? const <String, dynamic>{};
    final remain = (p['total_remain_kg'] as num?)?.toDouble() ?? 0;
    final processes = p['processes'] is List ? List<dynamic>.from(p['processes'] as List) : const [];
    return ListView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
      children: [
        Row(
          children: [
            const Expanded(child: Text('板结束核对', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13))),
            TextButton(
              onPressed: () => setState(() {
                _step = 0;
                _msg = '';
                _msgIsError = false;
              }),
              child: Text('改$_codeLabel'),
            ),
          ],
        ),
        const FormSectionHeader('材料'),
        _previewRow(_codeLabel, _box.text.trim().isEmpty ? '${p['board_code'] ?? '-'}' : _box.text.trim(), emphasize: true),
        _previewRow('溯源码', '${p['trace_code'] ?? '-'}', emphasize: true),
        _previewRow('工序余量(kg)', _fmtKg(p['process_remain_kg'])),
        _previewRow('仓库余量(kg)', _fmtKg(p['warehouse_kg'])),
        _previewRow('合计余量(kg)', _fmtKg(p['total_remain_kg']), emphasize: true),
        if (processes.isNotEmpty) ...[
          const SizedBox(height: 8),
          const FormSectionHeader('各工序余量'),
          for (final raw in processes)
            _previewRow(
              '${(raw is Map ? raw['process_name'] : null) ?? (raw is Map ? raw['process_id'] : '')}',
              _fmtKg(raw is Map ? raw['remain_kg'] : null),
            ),
        ],
        const SizedBox(height: 8),
        Text(
          remain > 0.0005
              ? '仍有物料在工序或仓库。选「补工序」继续生产；选「全部确认为耗损并结束」后才计算各工序扣损率。'
              : '该$_codeLabel工序与仓库已无余量，确认结束后计算各工序扣损率。',
          style: const TextStyle(fontSize: 12, color: Colors.black54),
        ),
      ],
    );
  }
}
