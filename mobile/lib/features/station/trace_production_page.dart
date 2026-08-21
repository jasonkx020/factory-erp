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
  bool _busy = false;
  String? _err;

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
  }

  Future<void> _start() async {
    setState(() => _busy = true);
    final r = await context.read<AuthState>().api.post('/production/trace-productions/start', {
      'trace_code': widget.traceCode,
    });
    if (!mounted) return;
    setState(() => _busy = false);
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已进入生产' : r.msg)));
    if (r.ok) {
      widget.onChanged();
      await _load();
    }
  }

  Future<void> _complete() async {
    setState(() => _busy = true);
    final r = await context.read<AuthState>().api.post('/production/trace-productions/complete', {
      'trace_code': widget.traceCode,
      'id': _wip?['session_id'],
    });
    if (!mounted) return;
    setState(() => _busy = false);
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已结束生产' : r.msg)));
    if (r.ok) {
      widget.onChanged();
      await _load();
    }
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
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(child: FilledButton(onPressed: _busy ? null : _start, child: const Text('进入生产'))),
                  const SizedBox(width: 12),
                  Expanded(child: OutlinedButton(onPressed: _busy ? null : _complete, child: const Text('结束生产'))),
                ],
              ),
              const SizedBox(height: 16),
              const Text('工序分布', style: TextStyle(fontWeight: FontWeight.w600)),
              for (final s in ((_wip!['steps'] is List ? _wip!['steps'] as List : const [])).whereType<Map>())
                ListTile(
                  dense: true,
                  contentPadding: EdgeInsets.zero,
                  title: Text('${s['process_name'] ?? s['process_id']}'),
                  subtitle: Text('可领 ${s['available_kg']} · 占用 ${s['occupied_kg']} · 在制 ${s['wip_kg']}'),
                ),
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
