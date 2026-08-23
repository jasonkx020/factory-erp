import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/recent_code_store.dart';
import '../../widgets/trace_code_field.dart';

class _IssueStatusStyle {
  const _IssueStatusStyle({
    required this.label,
    required this.color,
    required this.background,
    required this.icon,
  });

  final String label;
  final Color color;
  final Color background;
  final IconData icon;
}

_IssueStatusStyle _statusStyleFor(Map<String, dynamic> row) {
  final biz = '${row['biz_status'] ?? ''}'.trim();
  final status = '${row['status'] ?? ''}'.trim();
  switch (biz) {
    case 'issue_pending_warehouse':
      return const _IssueStatusStyle(
        label: '待仓管确认',
        color: Color(0xFFE65100),
        background: Color(0xFFFFF3E0),
        icon: Icons.hourglass_top_outlined,
      );
    case 'work_done':
      return const _IssueStatusStyle(
        label: '已结束',
        color: Color(0xFF546E7A),
        background: Color(0xFFECEFF1),
        icon: Icons.check_circle_outline,
      );
    case 'issue_rejected':
      return const _IssueStatusStyle(
        label: '仓管已驳回',
        color: Color(0xFFC62828),
        background: Color(0xFFFFEBEE),
        icon: Icons.cancel_outlined,
      );
    case 'return_pending':
      return const _IssueStatusStyle(
        label: '退库待审',
        color: Color(0xFFEF6C00),
        background: Color(0xFFFBE9E7),
        icon: Icons.undo_outlined,
      );
    default:
      if (status == 'closed') {
        return const _IssueStatusStyle(
          label: '已结束',
          color: Color(0xFF546E7A),
          background: Color(0xFFECEFF1),
          icon: Icons.check_circle_outline,
        );
      }
      return const _IssueStatusStyle(
        label: '进行中',
        color: Color(0xFF0D7A6F),
        background: Color(0xFFE0F2F1),
        icon: Icons.play_circle_outline,
      );
  }
}

class _StatusLegend extends StatelessWidget {
  const _StatusLegend();

  @override
  Widget build(BuildContext context) {
    Widget dot(Color color, String label) => Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 10,
              height: 10,
              decoration: BoxDecoration(color: color, shape: BoxShape.circle),
            ),
            const SizedBox(width: 4),
            Text(label, style: const TextStyle(fontSize: 12, color: Colors.black54)),
          ],
        );
    return Padding(
      padding: const EdgeInsets.only(bottom: 4),
      child: Wrap(
        spacing: 14,
        runSpacing: 4,
        children: [
          dot(const Color(0xFFE65100), '待仓管确认'),
          dot(const Color(0xFF0D7A6F), '进行中'),
          dot(const Color(0xFF546E7A), '已结束'),
        ],
      ),
    );
  }
}

/// 领料历史：本人 / 代领；工牌号与时间过滤；申请退库；主任确认结束
class ProcessIssueHistoryPage extends StatefulWidget {
  const ProcessIssueHistoryPage({super.key, this.scope = 'related'});

  final String scope;

  @override
  State<ProcessIssueHistoryPage> createState() => _ProcessIssueHistoryPageState();
}

class _ProcessIssueHistoryPageState extends State<ProcessIssueHistoryPage> {
  List<Map<String, dynamic>> _items = [];
  bool _loading = false;
  String _scope = 'related';
  final _badge = TextEditingController();
  DateTime? _dateFrom;
  DateTime? _dateTo;
  /// today | d7 | d30 | custom | all
  String _datePreset = 'd7';

  bool get _isForeman {
    final roles = context.read<AuthState>().roles.map((e) => e.toString().toLowerCase()).toList();
    return roles.any((r) =>
        r.contains('foreman') || r.contains('车间主任') || r.contains('主任') || r.contains('生管') || r == 'admin');
  }

  @override
  void initState() {
    super.initState();
    _scope = widget.scope.isEmpty ? 'related' : widget.scope;
    _applyDatePreset('d7');
    _load();
  }

  @override
  void dispose() {
    _badge.dispose();
    super.dispose();
  }

  String _fmtYmd(DateTime d) =>
      '${d.year.toString().padLeft(4, '0')}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';

  String _fmtMd(DateTime d) => '${d.month}/${d.day}';

  void _applyDatePreset(String preset) {
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    _datePreset = preset;
    switch (preset) {
      case 'today':
        _dateFrom = today;
        _dateTo = today;
      case 'd7':
        _dateFrom = today.subtract(const Duration(days: 6));
        _dateTo = today;
      case 'd30':
        _dateFrom = today.subtract(const Duration(days: 29));
        _dateTo = today;
      case 'all':
        _dateFrom = null;
        _dateTo = null;
      default:
        break;
    }
  }

  Future<void> _pickDateRange() async {
    final now = DateTime.now();
    final initial = DateTimeRange(
      start: _dateFrom ?? now.subtract(const Duration(days: 6)),
      end: _dateTo ?? now,
    );
    final picked = await showDateRangePicker(
      context: context,
      firstDate: now.subtract(const Duration(days: 365 * 2)),
      lastDate: now.add(const Duration(days: 1)),
      initialDateRange: initial,
      helpText: '选择领料时间范围',
      saveText: '确定',
    );
    if (picked == null) return;
    setState(() {
      _datePreset = 'custom';
      _dateFrom = DateTime(picked.start.year, picked.start.month, picked.start.day);
      _dateTo = DateTime(picked.end.year, picked.end.month, picked.end.day);
    });
    await _load();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    final q = StringBuffer('/production/process-issues?scope=$_scope&page_size=100&page_num=1');
    final badge = _badge.text.trim();
    if (badge.isNotEmpty) {
      q.write('&badge_code=${Uri.encodeQueryComponent(badge)}');
    }
    if (_dateFrom != null) q.write('&date_from=${_fmtYmd(_dateFrom!)}');
    if (_dateTo != null) q.write('&date_to=${_fmtYmd(_dateTo!)}');
    final r = await context.read<AuthState>().api.get(q.toString());
    if (!mounted) return;
    setState(() => _loading = false);
    if (!r.ok) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.msg)));
      return;
    }
    if (badge.isNotEmpty) {
      await RecentCodeStore.remember(RecentCodeStore.badge, badge, upper: false);
    }
    // PageOK 信封字段为 list（不是 items）
    final items = ApiClient.listOf(r.data);
    setState(() {
      _items = items.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
    });
  }

  Future<void> _openDetail(int id) async {
    final r = await context.read<AuthState>().api.get('/production/process-issues/$id');
    if (!mounted) return;
    if (!r.ok || r.data is! Map) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.msg)));
      return;
    }
    final row = Map<String, dynamic>.from(r.data as Map);
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => _IssueDetailSheet(
        row: row,
        isForeman: _isForeman,
        onChanged: _load,
      ),
    );
  }

  void _setScope(String scope) {
    setState(() => _scope = scope);
    _load();
  }

  void _setDatePreset(String preset) {
    setState(() => _applyDatePreset(preset));
    _load();
  }

  String _dateLabel() {
    if (_dateFrom == null && _dateTo == null) return '自定义日期';
    if (_dateFrom != null && _dateTo != null) {
      if (_fmtYmd(_dateFrom!) == _fmtYmd(_dateTo!)) return _fmtYmd(_dateFrom!);
      return '${_fmtMd(_dateFrom!)}~${_fmtMd(_dateTo!)}';
    }
    if (_dateFrom != null) return '自 ${_fmtYmd(_dateFrom!)}';
    return '至 ${_fmtYmd(_dateTo!)}';
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('领料历史'),
        actions: [IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh))],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 12, 12, 0),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Wrap(
                  spacing: 8,
                  children: [
                    ChoiceChip(
                      label: const Text('相关'),
                      selected: _scope == 'related',
                      onSelected: (_) => _setScope('related'),
                    ),
                    ChoiceChip(
                      label: const Text('归我'),
                      selected: _scope == 'mine',
                      onSelected: (_) => _setScope('mine'),
                    ),
                    ChoiceChip(
                      label: const Text('我代领'),
                      selected: _scope == 'proxy_by_me',
                      onSelected: (_) => _setScope('proxy_by_me'),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                TraceCodeField(
                  controller: _badge,
                  label: '工牌号',
                  hint: '筛选领料人工牌，可点最近使用',
                  scannerTitle: '扫描工牌',
                  textCapitalization: TextCapitalization.none,
                  requiredMark: false,
                  historyKey: RecentCodeStore.badge,
                  onEditingComplete: _load,
                  onScanned: (_) => _load(),
                ),
                const SizedBox(height: 4),
                Wrap(
                  spacing: 8,
                  runSpacing: 4,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: [
                    ChoiceChip(
                      label: const Text('今天'),
                      selected: _datePreset == 'today',
                      onSelected: (_) => _setDatePreset('today'),
                    ),
                    ChoiceChip(
                      label: const Text('近7天'),
                      selected: _datePreset == 'd7',
                      onSelected: (_) => _setDatePreset('d7'),
                    ),
                    ChoiceChip(
                      label: const Text('近30天'),
                      selected: _datePreset == 'd30',
                      onSelected: (_) => _setDatePreset('d30'),
                    ),
                    ChoiceChip(
                      label: const Text('全部'),
                      selected: _datePreset == 'all',
                      onSelected: (_) => _setDatePreset('all'),
                    ),
                    ActionChip(
                      avatar: Icon(Icons.date_range, size: 18, color: _datePreset == 'custom' ? Colors.teal : null),
                      label: Text(_dateLabel()),
                      onPressed: _pickDateRange,
                    ),
                    FilledButton.tonal(
                      onPressed: _loading ? null : _load,
                      child: const Text('查询'),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                const _StatusLegend(),
              ],
            ),
          ),
          if (_loading) const LinearProgressIndicator(),
          Expanded(
            child: _items.isEmpty && !_loading
                ? const Center(child: Text('暂无领料记录', style: TextStyle(color: Colors.black54)))
                : ListView.separated(
                    itemCount: _items.length,
                    separatorBuilder: (_, _) => const Divider(height: 1),
                    itemBuilder: (_, i) {
                      final e = _items[i];
                      final proxy = e['is_proxy'] == true;
                      final worker = '${e['worker_name'] ?? ''}';
                      final issuer = '${e['issuer_name'] ?? ''}';
                      final badge = '${e['badge_code'] ?? ''}'.trim();
                      final created = '${e['created_at'] ?? ''}'.trim();
                      final style = _statusStyleFor(e);
                      return Material(
                        color: style.background.withValues(alpha: 0.55),
                        child: ListTile(
                          leading: CircleAvatar(
                            radius: 18,
                            backgroundColor: style.color.withValues(alpha: 0.14),
                            foregroundColor: style.color,
                            child: Icon(style.icon, size: 20),
                          ),
                          title: Text(
                            '${(e['trace_code'] ?? '').toString().isNotEmpty ? e['trace_code'] : (e['board_code'] ?? '')} · ${e['process_name'] ?? ''}',
                            style: const TextStyle(fontWeight: FontWeight.w600),
                          ),
                          subtitle: Text(
                            '领 ${e['issue_kg']} · 退 ${e['returned_kg']} · 完 ${e['completed_kg']} · 可退 ${e['returnable_kg']}'
                            '${badge.isNotEmpty ? ' · 工牌 $badge' : ''}'
                            '${proxy ? ' · 代领给 $worker（操作人 $issuer）' : (worker.isNotEmpty ? ' · $worker' : '')}'
                            '${created.isNotEmpty ? ' · $created' : ''}',
                          ),
                          trailing: Chip(
                            label: Text(style.label),
                            labelStyle: TextStyle(color: style.color, fontSize: 12, fontWeight: FontWeight.w600),
                            backgroundColor: style.background,
                            side: BorderSide(color: style.color.withValues(alpha: 0.35)),
                            visualDensity: VisualDensity.compact,
                            padding: const EdgeInsets.symmetric(horizontal: 4),
                          ),
                          onTap: () => _openDetail((e['id'] as num).toInt()),
                        ),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}

class _IssueDetailSheet extends StatefulWidget {
  const _IssueDetailSheet({required this.row, required this.isForeman, required this.onChanged});

  final Map<String, dynamic> row;
  final bool isForeman;
  final VoidCallback onChanged;

  @override
  State<_IssueDetailSheet> createState() => _IssueDetailSheetState();
}

class _IssueDetailSheetState extends State<_IssueDetailSheet> {
  late Map<String, dynamic> _row;
  final _kg = TextEditingController();
  String? _photo;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _row = widget.row;
    final ret = (_row['returnable_kg'] as num?)?.toDouble() ?? 0;
    if (ret > 0) _kg.text = ret.toString();
  }

  @override
  void dispose() {
    _kg.dispose();
    super.dispose();
  }

  Future<void> _takePhoto() async {
    try {
      final picker = ImagePicker();
      final file = await picker.pickImage(source: ImageSource.camera, imageQuality: 85);
      if (file == null) return;
      final bytes = await file.readAsBytes();
      if (!mounted) return;
      final r = await context.read<AuthState>().api.postMultipart(
            '/biz/uploads',
            bytes,
            filename: file.name.isEmpty ? 'return.jpg' : file.name,
          );
      if (!mounted) return;
      if (!r.ok || r.data is! Map) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('上传失败：${r.msg}')));
        return;
      }
      final url = (r.data as Map)['url']?.toString() ?? (r.data as Map)['file_url']?.toString() ?? '';
      setState(() => _photo = url);
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('拍照失败：$e')));
    }
  }

  Future<void> _returnApply() async {
    final id = (_row['id'] as num).toInt();
    final kg = double.tryParse(_kg.text) ?? 0;
    if (kg <= 0) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请填写退库重量')));
      return;
    }
    if ((_photo ?? '').isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请拍摄复磅照片')));
      return;
    }
    setState(() => _busy = true);
    final r = await context.read<AuthState>().api.post('/production/process-issues/$id/return-apply', {
      'return_kg': kg,
      'reweigh_kg': kg,
      'photo_url': _photo,
      'image_url': _photo,
    });
    if (!mounted) return;
    setState(() => _busy = false);
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已提交退库申请' : r.msg)));
    if (r.ok) {
      widget.onChanged();
      Navigator.pop(context);
    }
  }

  Future<void> _confirmDone() async {
    final id = (_row['id'] as num).toInt();
    setState(() => _busy = true);
    final r = await context.read<AuthState>().api.post('/production/process-issues/$id/confirm-done', {});
    if (!mounted) return;
    setState(() => _busy = false);
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已确认结束' : r.msg)));
    if (r.ok) {
      widget.onChanged();
      Navigator.pop(context);
    }
  }

  @override
  Widget build(BuildContext context) {
    final ended = _row['biz_status'] == 'work_done';
    final pending = _row['biz_status'] == 'return_pending';
    final whPending = _row['biz_status'] == 'issue_pending_warehouse';
    final rejected = _row['biz_status'] == 'issue_rejected';
    final proxy = _row['is_proxy'] == true;
    final badge = '${_row['badge_code'] ?? ''}'.trim();
    return Padding(
      padding: EdgeInsets.only(left: 16, right: 16, top: 16, bottom: 16 + MediaQuery.viewInsetsOf(context).bottom),
      child: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('领料详情 #${_row['id']}', style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
            Text('${_row['board_code']} · ${_row['trace_code']} · ${_row['process_name']}'),
            Text('工人 ${_row['worker_name']}${badge.isNotEmpty ? '（工牌 $badge）' : ''} · 操作人 ${_row['issuer_name'] ?? '-'}'
                '${proxy ? '（代领）' : ''}'),
            Text('时间 ${_row['created_at'] ?? '-'}'),
            Text(
                '已领 ${_row['issue_kg']} / 已退 ${_row['returned_kg']} / 已产出 ${_row['completed_kg']} / 可退 ${_row['returnable_kg']}'),
            Builder(builder: (context) {
              final style = _statusStyleFor(_row);
              return Padding(
                padding: const EdgeInsets.only(top: 6),
                child: Row(
                  children: [
                    const Text('状态 '),
                    Chip(
                      label: Text(style.label),
                      labelStyle: TextStyle(color: style.color, fontWeight: FontWeight.w600),
                      backgroundColor: style.background,
                      side: BorderSide(color: style.color.withValues(alpha: 0.35)),
                      visualDensity: VisualDensity.compact,
                    ),
                  ],
                ),
              );
            }),
            if (!ended && !pending && !whPending && !rejected) ...[
              const SizedBox(height: 12),
              TextField(
                  controller: _kg,
                  decoration: const InputDecoration(labelText: '申请退库 kg（复磅）'),
                  keyboardType: TextInputType.number),
              ListTile(
                leading: Icon((_photo ?? '').isEmpty ? Icons.camera_alt_outlined : Icons.check_circle,
                    color: (_photo ?? '').isEmpty ? Colors.orange : Colors.teal),
                title: Text((_photo ?? '').isEmpty ? '拍摄复磅照片（必填）' : '复磅照片已上传'),
                onTap: _takePhoto,
              ),
              FilledButton(onPressed: _busy ? null : _returnApply, child: const Text('申请部分退库')),
            ],
            if (widget.isForeman && !ended && !pending && !whPending && !rejected) ...[
              const SizedBox(height: 8),
              OutlinedButton(onPressed: _busy ? null : _confirmDone, child: const Text('确认结束（进日结）')),
            ],
            if (pending)
              const Padding(
                  padding: EdgeInsets.only(top: 12),
                  child: Text('退库待仓管过磅同意…', style: TextStyle(color: Colors.orange))),
          ],
        ),
      ),
    );
  }
}
