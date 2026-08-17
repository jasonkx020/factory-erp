import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/auth_state.dart';
import '../../core/notify_service.dart';
import '../../widgets/form_row.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_sticky_actions.dart';
import '../../widgets/hub_entry_tile.dart';
import '../../widgets/trace_code_field.dart';

enum StationPassMode { home, self, proxy, close }

enum StationBoardAction { issue, ret, next }

/// 工序过站：扫工牌+板码后按 kg 领取 / 退库 / 领入下道（末道完工入库）。
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
      _fillSelfBadgeIfNeeded();
      context.read<NotifyService>().addListener(_onNotify);
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

  static const _errLabel = {
    'TRACE_CODE_REQUIRED': '该板缺少溯源码，无法领取',
    'BOARD_FINISHED': '该板已完工，不能再操作',
    'QTY_EXCEEDS_AVAILABLE': '领取重量超过板可领',
    'QTY_EXCEEDS_OCCUPANCY': '退库重量超过本人占用',
    'QTY_EXCEEDS_WIP': '重量超过本工序在制',
    'BOX_REQUIRED': '请填写或扫描板码',
    'BOX_NOT_FOUND': '未找到该板码',
    'SHIFT_NOT_AUTHORIZED': '当前班次未授权该工序',
    'ROLE_FORBIDDEN': '仅生管可确认板结束',
    'REMAIN_NEEDS_DECISION': '该板仍有物料，请确认为耗损或返回补工序',
    'APP_ONLY': '请在 App 操作',
  };

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
    return body;
  }

  String? _validateScan() {
    if (_box.text.trim().isEmpty) return '请填写或扫描板码';
    return null;
  }

  String? _validateAction() {
    if (_isClose) return null;
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
      case StationBoardAction.next:
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
      case StationBoardAction.next:
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
    if (_action == StationBoardAction.next) {
      final hasNext = _preview?['has_next'] == true;
      body['move_kind'] = hasNext ? 'next' : 'finish_in';
    }
    return body;
  }

  String _okMessage(Map<String, dynamic> data) {
    final kg = data['issue_kg'] ?? data['returned_kg'] ?? data['kg'] ?? _kg.text;
    final board = (data['board_code'] ?? _box.text).toString().trim();
    final trace = (data['trace_code'] ?? _preview?['trace_code'] ?? '').toString().trim();
    final loc = [
      if (board.isNotEmpty) '板 $board',
      if (trace.isNotEmpty) '溯源 $trace',
    ].join(' · ');
    final locText = loc.isEmpty ? '' : '（$loc）';
    final wage = data['issue_locked_wage_amount'] ?? data['released_locked_wage_amount'] ?? data['settled_wage_amount'];
    final wageText = wage is num ? ' · 工钱¥${wage.toStringAsFixed(2)}' : '';
    switch (_action) {
      case StationBoardAction.issue:
        return '已领取 $kg kg$locText$wageText（预锁定，完工结算）';
      case StationBoardAction.ret:
        return '已退库 $kg kg$locText$wageText（扣减锁定，未结算）';
      case StationBoardAction.next:
        if (data['move_kind'] == 'finish_in' || _preview?['has_next'] != true) {
          return '已完工入库 $kg kg$locText$wageText';
        }
        final nextName = '${data['to_step_name'] ?? _preview?['next_step_name'] ?? '下道'}';
        final settled = data['settled_wage_amount'];
        final settledText = settled is num ? ' · 本道结算¥${settled.toStringAsFixed(2)}' : '';
        return '已领入$nextName $kg kg$locText$settledText';
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
    _kg.clear();
    _box.clear();
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

  String get _nextLabel {
    if (_preview?['has_next'] == true) {
      final name = '${_preview?['next_step_name'] ?? '下道'}';
      return '领入$name';
    }
    return '完工入库';
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
    return ListView(
      padding: EdgeInsets.fromLTRB(16, widget.asTab ? 40 : 16, 16, 16),
      children: [
        const Text('工序过站', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
        const SizedBox(height: 4),
        const Text('扫板码后按 kg 领取、退库或领入下道', style: TextStyle(fontSize: 13, color: Colors.black54)),
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
          subtitle: '锁定本人工牌，扫板码领取/退库/进下道',
          onTap: () => _openMode(StationPassMode.self),
        ),
        HubEntryTile(
          icon: Icons.group_outlined,
          title: '代人过站',
          subtitle: '手输或扫描他人工牌后再扫板码',
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

  String _submitLabel() {
    switch (_action) {
      case StationBoardAction.issue:
        return '确认领取';
      case StationBoardAction.ret:
        return '确认退库';
      case StationBoardAction.next:
        return _preview?['has_next'] == true ? '确认领入下道' : '确认完工入库';
    }
  }

  List<Widget> _formFields() {
    if (_isClose) {
      return [
        const FormSectionHeader('板码'),
        TraceCodeField(
          controller: _box,
          label: '板码',
          hint: '手输或扫板码',
          scannerTitle: '扫描板码',
        ),
      ];
    }
    return [
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
      const FormSectionHeader('板码'),
      TraceCodeField(
        controller: _box,
        label: '板码',
        hint: '手输或扫板码',
        scannerTitle: '扫描板码',
      ),
    ];
  }

  Widget _actionBody() {
    if (_isClose) return _closeBody();
    final p = _preview ?? const <String, dynamic>{};
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
            const Expanded(child: Text('板信息', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13))),
            TextButton(
              onPressed: () => setState(() {
                _step = 0;
                _msg = '';
                _msgIsError = false;
              }),
              child: const Text('改板码'),
            ),
          ],
        ),
        const FormSectionHeader('材料'),
        _previewRow('板码', _box.text.trim().isEmpty ? '${p['board_code'] ?? '-'}' : _box.text.trim(), emphasize: true),
        _previewRow('溯源码', '${p['trace_code'] ?? '-'}', emphasize: true),
        _previewRow('模式', forOther ? '代人过站' : '本人过站'),
        _previewRow('工牌', passerText.isEmpty ? '当前用户（未填工牌）' : passerText),
        _previewRow('当前工序', stepName),
        _previewRow('板可领(kg)', _fmtKg(p['available_kg'])),
        _previewRow('本人占用(kg)', _fmtKg(p['my_open_kg'])),
        _previewRow('本工序在制(kg)', _fmtKg(p['wip_kg'])),
        if (p['piecework'] == true) ...[
          _previewRow('计件工价', '${p['rate'] ?? '-'} 元/kg'),
          _previewRow('预锁定工钱', _fmtMoney(p['locked_wage_amount'])),
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
              label: Text(_nextLabel),
              selected: _action == StationBoardAction.next,
              onSelected: (_) => setState(() => _action = StationBoardAction.next),
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
              child: const Text('改板码'),
            ),
          ],
        ),
        const FormSectionHeader('材料'),
        _previewRow('板码', _box.text.trim().isEmpty ? '${p['board_code'] ?? '-'}' : _box.text.trim(), emphasize: true),
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
              : '该板工序与仓库已无余量，确认结束后计算各工序扣损率。',
          style: const TextStyle(fontSize: 12, color: Colors.black54),
        ),
      ],
    );
  }

  String _actionHint() {
    switch (_action) {
      case StationBoardAction.issue:
        return '从本板当前工序可领池按 kg 领取；计件工钱预锁定，进下道或完工入库后结算。';
      case StationBoardAction.ret:
        return '从未完成占用退回本板本工序可领池；扣减计件锁定，退库部分不入日汇总。';
      case StationBoardAction.next:
        if (_preview?['has_next'] == true) {
          return '下道领取重量即本道完工重，本道不再称重。整板扣损由生管在「板结束」确认。';
        }
        return '末道完工入库。整板扣损由生管在「板结束」确认；未结束前可继续入库。';
    }
  }
}
