import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/recent_code_store.dart';
import '../../widgets/trace_code_field.dart';

/// 生管溯源台：默认列出全部可用溯源码，点选查看/启停；扫码为辅助。
class TraceProductionPage extends StatefulWidget {
  const TraceProductionPage({super.key});

  @override
  State<TraceProductionPage> createState() => _TraceProductionPageState();
}

class _TraceProductionPageState extends State<TraceProductionPage> {
  final _search = TextEditingController();
  final _scan = TextEditingController();
  List<Map<String, dynamic>> _list = [];
  String _statusFilter = '';
  bool _loading = false;
  int _total = 0;

  @override
  void initState() {
    super.initState();
    _loadList();
  }

  @override
  void dispose() {
    _search.dispose();
    _scan.dispose();
    super.dispose();
  }

  String _statusLabel(String? s) {
    switch ((s ?? '').toLowerCase()) {
      case 'in_production':
        return '生产中';
      case 'ended':
        return '已结束';
      default:
        return '库中';
    }
  }

  Color _statusColor(String? s) {
    switch ((s ?? '').toLowerCase()) {
      case 'in_production':
        return Colors.orange.shade700;
      case 'ended':
        return Colors.blueGrey;
      default:
        return Colors.teal;
    }
  }

  List<Map<String, dynamic>> get _filtered {
    final q = _search.text.trim().toUpperCase();
    if (q.isEmpty) return _list;
    return _list.where((e) => '${e['trace_code']}'.toUpperCase().contains(q)).toList();
  }

  Future<void> _loadList() async {
    setState(() => _loading = true);
    final qs = <String>['page_size=200', 'page_num=1'];
    if (_statusFilter.isNotEmpty) {
      qs.add('status=${Uri.encodeComponent(_statusFilter)}');
    }
    final r = await context.read<AuthState>().api.get('/production/trace-productions?${qs.join('&')}');
    if (!mounted) return;
    setState(() => _loading = false);
    if (!r.ok) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.msg)));
      return;
    }
    final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
    final items = ApiClient.listOf(data);
    setState(() {
      _list = items.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
      _total = (data['total'] as num?)?.toInt() ?? _list.length;
    });
  }

  Future<void> _openDetail(String code) async {
    code = code.trim();
    if (code.isEmpty) return;
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (_) => _TraceDetailSheet(
        traceCode: code,
        statusLabel: _statusLabel,
        onChanged: _loadList,
      ),
    );
    if (mounted) await _loadList();
  }

  void _setFilter(String v) {
    setState(() => _statusFilter = v);
    _loadList();
  }

  @override
  Widget build(BuildContext context) {
    final rows = _filtered;
    return Scaffold(
      appBar: AppBar(
        title: const Text('溯源生产台'),
        actions: [
          IconButton(onPressed: _loading ? null : _loadList, icon: const Icon(Icons.refresh)),
        ],
      ),
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  '共 $_total 个溯源码，点选查看工序分布并进入/结束生产',
                  style: const TextStyle(fontSize: 13, color: Colors.black54),
                ),
                const SizedBox(height: 8),
                TextField(
                  controller: _search,
                  decoration: InputDecoration(
                    labelText: '筛选溯源码',
                    hintText: '输入关键字过滤列表',
                    prefixIcon: const Icon(Icons.search),
                    suffixIcon: _search.text.isEmpty
                        ? null
                        : IconButton(
                            icon: const Icon(Icons.clear),
                            onPressed: () {
                              _search.clear();
                              setState(() {});
                            },
                          ),
                    border: const OutlineInputBorder(),
                    isDense: true,
                  ),
                  onChanged: (_) => setState(() {}),
                ),
                const SizedBox(height: 8),
                TraceCodeField(
                  controller: _scan,
                  label: '扫码定位（可选）',
                  hint: '扫码或点最近使用后打开详情',
                  scannerTitle: '扫描溯源码',
                  historyKey: RecentCodeStore.trace,
                  onScanned: (v) {
                    _search.text = v;
                    _openDetail(v);
                  },
                  onChanged: (v) {
                    if (v.trim().length >= 4) _search.text = v;
                  },
                ),
                const SizedBox(height: 8),
                Wrap(
                  spacing: 8,
                  children: [
                    ChoiceChip(label: const Text('全部'), selected: _statusFilter.isEmpty, onSelected: (_) => _setFilter('')),
                    ChoiceChip(label: const Text('库中'), selected: _statusFilter == 'in_stock', onSelected: (_) => _setFilter('in_stock')),
                    ChoiceChip(
                        label: const Text('生产中'),
                        selected: _statusFilter == 'in_production',
                        onSelected: (_) => _setFilter('in_production')),
                    ChoiceChip(label: const Text('已结束'), selected: _statusFilter == 'ended', onSelected: (_) => _setFilter('ended')),
                  ],
                ),
              ],
            ),
          ),
          if (_loading) const LinearProgressIndicator(),
          Expanded(
            child: rows.isEmpty && !_loading
                ? const Center(child: Text('暂无可用溯源码', style: TextStyle(color: Colors.black54)))
                : ListView.separated(
                    padding: const EdgeInsets.fromLTRB(8, 8, 8, 24),
                    itemCount: rows.length,
                    separatorBuilder: (_, __) => const Divider(height: 1),
                    itemBuilder: (_, i) {
                      final row = rows[i];
                      final code = '${row['trace_code'] ?? ''}';
                      final st = row['ui_status']?.toString();
                      return ListTile(
                        leading: CircleAvatar(
                          backgroundColor: _statusColor(st).withValues(alpha: 0.15),
                          child: Icon(Icons.qr_code_2, color: _statusColor(st), size: 20),
                        ),
                        title: Text(code, style: const TextStyle(fontWeight: FontWeight.w600)),
                        subtitle: Text('库存 ${row['stock_kg'] ?? 0} kg · 板 ${row['board_count'] ?? 0}'),
                        trailing: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Text(
                              _statusLabel(st),
                              style: TextStyle(color: _statusColor(st), fontWeight: FontWeight.w600, fontSize: 13),
                            ),
                            const Icon(Icons.chevron_right, color: Colors.black38),
                          ],
                        ),
                        onTap: () => _openDetail(code),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}

class _TraceDetailSheet extends StatefulWidget {
  const _TraceDetailSheet({
    required this.traceCode,
    required this.statusLabel,
    required this.onChanged,
  });

  final String traceCode;
  final String Function(String?) statusLabel;
  final VoidCallback onChanged;

  @override
  State<_TraceDetailSheet> createState() => _TraceDetailSheetState();
}

class _TraceDetailSheetState extends State<_TraceDetailSheet> {
  Map<String, dynamic>? _wip;
  Map<String, dynamic>? _report;
  bool _busy = false;
  String? _err;

  Color _stepColor(String st) {
    switch (st) {
      case 'done':
        return Colors.teal;
      case 'in_progress':
        return Colors.orange.shade700;
      case 'ready':
        return Colors.blue;
      default:
        return Colors.blueGrey;
    }
  }

  String _stepStatusLabel(String st) {
    switch (st) {
      case 'done':
        return '已完成';
      case 'in_progress':
        return '进行中';
      case 'ready':
        return '可结束';
      default:
        return '待做';
    }
  }

  String _stepSubtitle(Map s) {
    final inName = '${s['input_product_name'] ?? ''}'.trim();
    final outName = '${s['output_product_name'] ?? ''}'.trim();
    final productHint = [
      if (inName.isNotEmpty) '领 $inName',
      if (outName.isNotEmpty) '产 $outName',
    ].join(' · ');
    if (s['step_status'] == 'done') {
      final base = '扣损 ${s['loss_kg'] ?? '-'} kg · 产出 ${s['output_kg'] ?? '-'} kg';
      return productHint.isEmpty ? base : '$productHint · $base';
    }
    final base = '在制 ${s['wip_kg'] ?? s['available_kg'] ?? 0} kg · ${_stepStatusLabel('${s['step_status']}')}';
    return productHint.isEmpty ? base : '$productHint · $base';
  }

  Future<void> _loadReport() async {
    final code = Uri.encodeComponent(widget.traceCode);
    final r = await context.read<AuthState>().api.get('/production/trace-productions/$code/report');
    if (!mounted || !r.ok) return;
    setState(() => _report = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : null);
  }

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _busy = true;
      _err = null;
    });
    final code = Uri.encodeComponent(widget.traceCode);
    final r = await context.read<AuthState>().api.get('/production/trace-productions/$code/wip');
    if (!mounted) return;
    setState(() => _busy = false);
    if (!r.ok) {
      setState(() => _err = r.msg);
      return;
    }
    setState(() => _wip = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : null);
    if (_wip?['ui_status']?.toString() == 'ended') {
      await _loadReport();
    } else {
      setState(() => _report = null);
    }
  }

  Future<void> _completeProcess(Map step) async {
    setState(() => _busy = true);
    final r = await context.read<AuthState>().api.post('/production/trace-productions/process-complete', {
      'trace_code': widget.traceCode,
      'process_id': step['process_id'],
    });
    if (!mounted) return;
    setState(() => _busy = false);
    final auto = r.data is Map && (r.data as Map)['auto_finalized'] == true;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? (auto ? '已自动结案' : '工序已结束') : r.msg)));
    if (r.ok) {
      widget.onChanged();
      await _load();
    }
  }

  Future<void> _start() async {
    final api = context.read<AuthState>().api;
    setState(() => _busy = true);
    final code = Uri.encodeComponent(widget.traceCode);
    final optRes = await api.get('/production/trace-productions/$code/routing-options');
    if (!mounted) return;
    setState(() => _busy = false);
    if (!optRes.ok) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(optRes.msg)));
      return;
    }
    final data = optRes.data is Map ? Map<String, dynamic>.from(optRes.data as Map) : <String, dynamic>{};
    final rawOpts = data['routing_options'];
    final options = rawOpts is List
        ? rawOpts.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList()
        : <Map<String, dynamic>>[];
    if (options.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('该原料暂无可用工艺，请先在管理端配置工艺流程')));
      return;
    }
    final suggested = (data['suggested_routing_id'] as num?)?.toInt();
    int? selected = suggested ?? (options.first['id'] as num?)?.toInt();
    final productName = '${data['product_name'] ?? ''}';
    final confirmed = await showDialog<int>(
      context: context,
      builder: (ctx) {
        return StatefulBuilder(
          builder: (ctx, setLocal) {
            final preview = () {
              for (final o in options) {
                if ((o['id'] as num?)?.toInt() == selected && o['steps_preview'] is List) {
                  return (o['steps_preview'] as List).map((e) => '$e').join(' → ');
                }
              }
              return '';
            }();
            return AlertDialog(
              title: const Text('选择工艺流程'),
              content: SingleChildScrollView(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    if (productName.isNotEmpty) Text('原料：$productName', style: const TextStyle(fontSize: 13, color: Colors.black54)),
                    const SizedBox(height: 12),
                    DropdownButtonFormField<int>(
                      value: selected,
                      decoration: const InputDecoration(labelText: '工艺流程', border: OutlineInputBorder(), isDense: true),
                      items: options.map((o) {
                        final id = (o['id'] as num?)?.toInt() ?? 0;
                        final label = '${o['code'] ?? ''} · ${o['name'] ?? ''}（${o['step_count'] ?? 0}道）';
                        return DropdownMenuItem(value: id, child: Text(label, overflow: TextOverflow.ellipsis));
                      }).toList(),
                      onChanged: (v) => setLocal(() => selected = v),
                    ),
                    if (preview.isNotEmpty) ...[
                      const SizedBox(height: 12),
                      Text('工序：$preview', style: const TextStyle(fontSize: 12, color: Colors.black54)),
                    ],
                  ],
                ),
              ),
              actions: [
                TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
                FilledButton(onPressed: selected == null ? null : () => Navigator.pop(ctx, selected), child: const Text('确认进入生产')),
              ],
            );
          },
        );
      },
    );
    if (confirmed == null || !mounted) return;
    setState(() => _busy = true);
    final r = await api.post('/production/trace-productions/start', {
      'trace_code': widget.traceCode,
      'routing_id': confirmed,
    });
    if (!mounted) return;
    setState(() => _busy = false);
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已进入生产，工艺已锁定' : r.msg)));
    if (r.ok) {
      widget.onChanged();
      await _load();
    }
  }

  bool _canCompleteStep(Map<String, dynamic> s) {
    final st = _wip?['ui_status']?.toString();
    if (st != 'in_progress') return false;
    final action = '${s['action'] ?? s['step_status'] ?? ''}';
    if (action != 'complete' && s['step_status'] != 'ready') return false;
    return _wip?['can_complete_process_id'] == s['process_id'];
  }

  List<Map<String, dynamic>> _timelineSteps() {
    if (_wip == null) return const [];
    final routing = _wip!['routing_steps'];
    if (routing is List && routing.isNotEmpty) {
      return routing.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
    }
    final steps = _wip!['steps'];
    if (steps is List) {
      return steps.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
    }
    return const [];
  }

  Widget? _stepTrailing(Map<String, dynamic> s) {
    if (_canCompleteStep(s)) {
      return TextButton(onPressed: _busy ? null : () => _completeProcess(s), child: const Text('结束'));
    }
    final st = '${s['step_status'] ?? s['action'] ?? ''}';
    if ((st == 'ready' || st == 'complete') && _wip?['ui_status'] != 'in_progress') {
      return const Text('待启动', style: TextStyle(fontSize: 12, color: Colors.black45));
    }
    if (st == 'done') return const Text('已完成', style: TextStyle(fontSize: 12, color: Colors.teal));
    if (st == 'in_progress') {
      return Text('${s['action_hint'] ?? '在制中'}', style: TextStyle(fontSize: 12, color: Colors.orange.shade800));
    }
    return Text('${s['action_hint'] ?? '待做'}', style: const TextStyle(fontSize: 12, color: Colors.black45));
  }

  @override
  Widget build(BuildContext context) {
    final bottom = MediaQuery.viewInsetsOf(context).bottom;
    final st = _wip?['ui_status']?.toString();
    return Padding(
      padding: EdgeInsets.fromLTRB(16, 16, 16, 16 + bottom),
      child: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(widget.traceCode, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700)),
            if (_busy && _wip == null) const Padding(padding: EdgeInsets.symmetric(vertical: 24), child: Center(child: CircularProgressIndicator())),
            if (_err != null) Text(_err!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
            if (_wip != null) ...[
              Text(
                '${widget.statusLabel(st)} · 可领 ${_wip!['total_available_kg'] ?? 0} kg · 占用 ${_wip!['total_occupied_kg'] ?? 0} kg',
                style: const TextStyle(fontSize: 13, color: Colors.black54),
              ),
              if (_wip!['product_name'] != null && '${_wip!['product_name']}'.isNotEmpty)
                Text('物料 ${_wip!['product_name']}', style: const TextStyle(fontSize: 13, color: Colors.black54)),
              if (_wip!['routing_code'] != null && '${_wip!['routing_code']}'.isNotEmpty)
                Text(
                  '工艺 ${_wip!['routing_code']}${_wip!['routing_name'] != null && '${_wip!['routing_name']}'.isNotEmpty ? ' · ${_wip!['routing_name']}' : ''}',
                  style: const TextStyle(fontSize: 13, color: Colors.black54),
                ),
              const SizedBox(height: 12),
              if (st == 'in_stock')
                FilledButton(onPressed: _busy ? null : _start, child: const Text('进入生产'))
              else if (st == 'in_production')
                const Text('已启动溯源生产，请按工序顺序结束', style: TextStyle(fontSize: 13, color: Colors.orange))
              else
                const Text('全部工序结束后已自动结案', style: TextStyle(fontSize: 13, color: Colors.teal)),
              if (st != 'ended' && _timelineSteps().isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: Text('按工艺顺序结束各工序；末道工序完成后自动结案', style: TextStyle(fontSize: 12, color: Colors.grey.shade600)),
                ),
              const SizedBox(height: 16),
              const Text('工序时间线', style: TextStyle(fontWeight: FontWeight.w600)),
              for (final s in _timelineSteps())
                ListTile(
                  dense: true,
                  contentPadding: EdgeInsets.zero,
                  leading: CircleAvatar(
                    radius: 14,
                    backgroundColor: _stepColor('${s['step_status']}').withValues(alpha: 0.15),
                    child: Text('${s['seq_no'] ?? ''}', style: TextStyle(fontSize: 11, color: _stepColor('${s['step_status']}'))),
                  ),
                  title: Text('${s['process_name'] ?? s['process_id']}'),
                  subtitle: Text(_stepSubtitle(s)),
                  trailing: _stepTrailing(s),
                ),
              if (_report != null) ...[
                const SizedBox(height: 16),
                const Text('生产报表', style: TextStyle(fontWeight: FontWeight.w600)),
                Builder(builder: (context) {
                  final summary = _report!['summary'] is Map ? Map<String, dynamic>.from(_report!['summary'] as Map) : <String, dynamic>{};
                  final rate = ((summary['trace_loss_rate'] as num?) ?? 0) * 100;
                  return Text(
                    '投入 ${summary['trace_input_kg'] ?? '-'} kg · 产出 ${summary['trace_output_kg'] ?? '-'} kg · 耗损率 ${rate.toStringAsFixed(1)}%',
                    style: const TextStyle(fontSize: 13),
                  );
                }),
                for (final y in ((_report!['process_yields'] is List ? _report!['process_yields'] as List : const [])).whereType<Map>())
                  ListTile(
                    dense: true,
                    contentPadding: EdgeInsets.zero,
                    title: Text('${y['process_name'] ?? y['process_id']}'),
                    subtitle: Text('投 ${y['input_kg']} · 产 ${y['output_kg']} · 损 ${y['loss_kg']} kg'),
                  ),
              ],
              if ((_wip!['boards'] is List ? _wip!['boards'] as List : const []).isNotEmpty) ...[
                const SizedBox(height: 8),
                const Text('板明细', style: TextStyle(fontWeight: FontWeight.w600)),
                for (final b in ((_wip!['boards'] is List ? _wip!['boards'] as List : const [])).whereType<Map>())
                  ListTile(
                    dense: true,
                    contentPadding: EdgeInsets.zero,
                    title: Text('${b['code']}'),
                    subtitle: Text('${b['process_name'] ?? ''} · ${b['weight_kg']} kg · ${b['status'] ?? ''}'),
                  ),
              ],
            ],
          ],
        ),
      ),
    );
  }
}
