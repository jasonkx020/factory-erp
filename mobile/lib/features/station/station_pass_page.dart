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

const _scrapOptions = <MapEntry<String, String>>[
  MapEntry('', '无次品'),
  MapEntry('cut_defect', '切断次品'),
  MapEntry('core_defect', '去芯次品'),
  MapEntry('dice_defect', '切块次品'),
  MapEntry('sieve_bag_defect', '过筛装袋次品'),
];

enum StationPassMode { home, self, proxy }

/// 工序过站：入口区分本人/代人；工牌与箱码均可手输或扫码；预览通过后直接提交。
/// 与收货一致：Hub 选入口后 [Navigator.push] 全屏子页。
class StationPassPage extends StatefulWidget {
  const StationPassPage({
    super.key,
    this.asTab = false,
    this.initialMode = StationPassMode.home,
  });

  /// 作为产线壳 Tab 时隐藏标题栏，仅展示入口首页。
  final bool asTab;

  /// 非 home 时作为独立子页（Navigator.push），返回即销毁。
  final StationPassMode initialMode;

  @override
  State<StationPassPage> createState() => _StationPassPageState();
}

class _StationPassPageState extends State<StationPassPage> {
  late StationPassMode _mode;
  /// 0 填表 · 1 预览
  int _step = 0;
  final _badge = TextEditingController();
  final _box = TextEditingController();
  final _in = TextEditingController();
  final _out = TextEditingController();
  final _bag = TextEditingController(text: '0');
  String _scrapType = '';
  String _msg = '';
  bool _msgIsError = false;
  bool _busy = false;
  Map<String, dynamic>? _preview;
  bool _isCheckpoint = false;

  bool get _isSubPage => widget.initialMode != StationPassMode.home;

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
    _in.dispose();
    _out.dispose();
    _bag.dispose();
    super.dispose();
  }

  void _onNotify() {
    if (!mounted) return;
    final notify = context.read<NotifyService>();
    for (final raw in notify.inbox) {
      if (raw is! Map) continue;
      final p = NotifyService.parsePayload(raw['payload'] ?? raw['payload_json']);
      final next = p['next'] is Map ? Map<String, dynamic>.from(p['next'] as Map) : p;
      final code = next['new_box_code'];
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
    // 本人过站：始终回填并锁定为本人工牌
    _badge.text = mine;
  }

  void _prompt(String msg, {bool error = true}) {
    if (!mounted) return;
    setState(() {
      _msg = msg;
      _msgIsError = error;
    });
    if (!error) return;
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
        _msg = '过站已提交';
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
      return;
    }
  }

  Map<String, dynamic> _scanBody({required bool autoConfirm, String qc = 'pass'}) {
    if (_mode == StationPassMode.self) {
      _fillSelfBadgeIfNeeded();
    }
    final badge = _badge.text.trim();
    final body = <String, dynamic>{
      'box_code': _box.text.trim(),
      'input_weight': double.tryParse(_in.text) ?? 0,
      'output_weight': double.tryParse(_out.text) ?? 0,
      'net_weight': double.tryParse(_out.text) ?? 0,
      'bag_qty': double.tryParse(_bag.text) ?? 0,
      'auto_confirm': autoConfirm,
      'process_qc_result': qc,
    };
    // 有工牌则按工牌解析过站人；空则后端默认当前登录员工。操作人由后端按登录态自动写入。
    if (badge.isNotEmpty) body['badge_code'] = badge;
    if (_scrapType.isNotEmpty) body['scrap_type'] = _scrapType;
    return body;
  }

  String? _validateForm() {
    if (_box.text.trim().isEmpty) return '请填写或扫描箱码';
    final outW = double.tryParse(_out.text) ?? 0;
    if (outW <= 0) return '请填写完工重(kg)';
    return null;
  }

  Future<void> _goPreview() async {
    final err = _validateForm();
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
    final r = await api.post('/production/scan/resolve', _scanBody(autoConfirm: false));
    if (!mounted) return;
    setState(() => _busy = false);
    if (!r.ok) {
      _prompt(r.msg);
      return;
    }
    final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
    if (data['input_weight'] != null && (_in.text.trim().isEmpty || (double.tryParse(_in.text) ?? 0) <= 0)) {
      _in.text = '${data['input_weight']}';
    }
    if (data['output_weight'] != null && (_out.text.trim().isEmpty || (double.tryParse(_out.text) ?? 0) <= 0)) {
      _out.text = '${data['output_weight']}';
    }
    setState(() {
      _preview = data;
      _isCheckpoint = data['is_inbound_checkpoint'] == true;
      _step = 1;
      _msg = '';
      _msgIsError = false;
    });
  }

  Future<void> _submit({required bool qcPass}) async {
    final err = _validateForm();
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
    final r = await api.post(
      '/production/scan',
      _scanBody(autoConfirm: true, qc: qcPass ? 'pass' : 'fail'),
    );
    if (!mounted) return;
    setState(() => _busy = false);
    if (!r.ok) {
      _prompt(r.msg);
      return;
    }
    final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
    final wage = data['wage_amount'] ?? 0;
    final okMsg = qcPass ? '过站成功 工钱¥$wage' : '已记录 QC 不合格（未过账）';
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(okMsg)));
    if (_isSubPage && Navigator.of(context).canPop()) {
      Navigator.of(context).pop(true);
      return;
    }
    _in.clear();
    _out.clear();
    _bag.text = '0';
    _scrapType = '';
    _box.clear();
    if (_mode == StationPassMode.self) {
      _fillSelfBadgeIfNeeded();
    }
    setState(() {
      _preview = data;
      _step = 0;
      _msg = okMsg;
      _msgIsError = false;
      _isCheckpoint = false;
    });
  }

  Widget _previewRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          SizedBox(
            width: 108,
            child: Text(label, style: TextStyle(fontSize: 13, color: Colors.black.withValues(alpha: 0.6))),
          ),
          Expanded(child: Text(value, textAlign: TextAlign.right, style: const TextStyle(fontSize: 14))),
        ],
      ),
    );
  }

  String _scrapLabel() {
    for (final e in _scrapOptions) {
      if (e.key == _scrapType) return e.value;
    }
    return '无次品';
  }

  String get _pageTitle {
    final base = _mode == StationPassMode.self
        ? '本人过站'
        : _mode == StationPassMode.proxy
            ? '代人过站'
            : '工序过站';
    if (_mode != StationPassMode.home && _step == 1) return '$base · 预览';
    return base;
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
        const Text('选择本人或代人过站；预览核对后直接提交', style: TextStyle(fontSize: 13, color: Colors.black54)),
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
          subtitle: '系统锁定本人工牌，扫箱码即可提交',
          onTap: () => _openMode(StationPassMode.self),
        ),
        HubEntryTile(
          icon: Icons.group_outlined,
          title: '代人过站',
          subtitle: '手输或扫描他人工牌后再扫箱码',
          onTap: () => _openMode(StationPassMode.proxy),
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
              : _previewBody(),
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
          Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              FormStickyActions(
                secondaryLabel: '修改',
                onSecondary: _busy
                    ? null
                    : () => setState(() {
                          _step = 0;
                          _msg = '';
                          _msgIsError = false;
                        }),
                primaryLabel: _isCheckpoint ? '确认过站（QC合格）' : '确认过站',
                onPrimary: _busy ? null : () => _submit(qcPass: true),
                primaryBusy: _busy,
                busyLabel: '提交中…',
              ),
              if (_isCheckpoint)
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
                  child: OutlinedButton(
                    onPressed: _busy ? null : () => _submit(qcPass: false),
                    child: const Text('QC 不合格（阻断过账）'),
                  ),
                ),
            ],
          ),
      ],
    );
  }

  List<Widget> _formFields() {
    return [
      if (_isCheckpoint)
        Card(
          color: Colors.amber.shade50,
          child: const ListTile(
            leading: Icon(Icons.verified_user),
            title: Text('收货卡点模式'),
            subtitle: Text('复核重量与外观，QC 不合格将阻断过账'),
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
      const FormSectionHeader('箱码与重量'),
      TraceCodeField(
        controller: _box,
        label: '箱码',
        hint: '手输或扫码',
        scannerTitle: '扫描箱码',
      ),
      FormRow.text(label: '投料重(kg)', controller: _in, keyboardType: TextInputType.number),
      FormRow.text(
        label: '完工重(kg)',
        controller: _out,
        keyboardType: TextInputType.number,
        requiredMark: true,
      ),
      FormRow.text(label: '袋数', controller: _bag, keyboardType: TextInputType.number, hint: '装袋工序'),
      const FormSectionHeader('次品类型'),
      FormRow(
        label: '次品',
        child: Wrap(
          alignment: WrapAlignment.end,
          spacing: 6,
          runSpacing: 8,
          children: _scrapOptions.map((e) {
            final selected = _scrapType == e.key;
            return Stack(
              clipBehavior: Clip.none,
              children: [
                ChoiceChip(
                  label: Text(e.value, style: const TextStyle(fontSize: 12)),
                  selected: selected,
                  showCheckmark: false,
                  visualDensity: VisualDensity.compact,
                  onSelected: (_) => setState(() => _scrapType = e.key),
                ),
                if (selected)
                  Positioned(
                    top: -4,
                    right: -2,
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
                      decoration: BoxDecoration(
                        color: Theme.of(context).colorScheme.primary,
                        borderRadius: BorderRadius.circular(6),
                      ),
                      child: Text(
                        '已选',
                        style: TextStyle(
                          fontSize: 9,
                          height: 1.1,
                          color: Theme.of(context).colorScheme.onPrimary,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                  ),
              ],
            );
          }).toList(),
        ),
      ),
    ];
  }

  Widget _previewBody() {
    final p = _preview ?? const <String, dynamic>{};
    final workerName = (p['worker_name'] ?? '').toString();
    final badge = (p['badge_code'] ?? _badge.text).toString().trim();
    final forOther = p['pass_for_other'] == true || _mode == StationPassMode.proxy;
    final stepName = (p['step_name'] ?? '-').toString();
    final util = p['utilization'];
    final utilText = util is num ? util.toStringAsFixed(3) : '-';
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
            const Expanded(child: Text('过站预览', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13))),
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
        const Text('请核对以下信息，有误请返回修改', style: TextStyle(fontSize: 12, color: Colors.black54)),
        const SizedBox(height: 8),
        _previewRow('模式', forOther ? '代人过站' : '本人过站'),
        _previewRow('工牌', passerText.isEmpty ? '当前用户（未填工牌）' : passerText),
        _previewRow('工序', stepName),
        _previewRow('箱码', _box.text.trim().isEmpty ? '-' : _box.text.trim()),
        _previewRow('投料重(kg)', _in.text.trim().isEmpty ? '${p['input_weight'] ?? '-'}' : _in.text.trim()),
        _previewRow('完工重(kg)', _out.text.trim().isEmpty ? '${p['output_weight'] ?? '-'}' : _out.text.trim()),
        _previewRow('损耗(kg)', '${p['loss'] ?? '-'}'),
        _previewRow('利用率', utilText),
        _previewRow('袋数', _bag.text.trim().isEmpty ? '0' : _bag.text.trim()),
        _previewRow('次品', _scrapLabel()),
        if (_isCheckpoint) ...[
          const SizedBox(height: 8),
          Text('本步为收货卡点，提交时将做 QC 判定。', style: TextStyle(fontSize: 12, color: Colors.amber.shade900)),
        ],
        const SizedBox(height: 8),
        const Text('确认后直接过账；操作人由系统按当前登录账号自动记录。', style: TextStyle(fontSize: 12, color: Colors.black54)),
      ],
    );
  }
}
