import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';

class _ApplyLine {
  int? toolId;
  int qty;

  _ApplyLine({this.toolId, this.qty = 1});
}

class _ReturnLine {
  final int lineId;
  final String name;
  final double remain;
  int qty;

  _ReturnLine({
    required this.lineId,
    required this.name,
    required this.remain,
    required this.qty,
  });
}

/// 物料工具申请/归还 + 指定下一手处理人（支持多行明细）
class ToolIssuePage extends StatefulWidget {
  const ToolIssuePage({super.key});

  @override
  State<ToolIssuePage> createState() => _ToolIssuePageState();
}

class _ToolIssuePageState extends State<ToolIssuePage> {
  List<dynamic> _items = [];
  List<dynamic> _issues = [];
  List<dynamic> _pool = [];
  final List<_ApplyLine> _lines = [_ApplyLine()];
  int? _assignee;
  String _msg = '';
  bool _busy = false;

  static const _statusLabel = {
    'pending': '待审批发放',
    'open': '在用',
    'pending_return': '待确认归还',
    'returned': '已还清',
    'rejected': '已驳回',
  };

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _refresh());
  }

  int? _defaultToolId() {
    if (_items.isEmpty) return null;
    final id = (_items.first as Map)['id'];
    return id is num ? id.toInt() : null;
  }

  void _addLine() {
    setState(() => _lines.add(_ApplyLine(toolId: _defaultToolId())));
  }

  void _removeLine(int idx) {
    if (_lines.length <= 1) return;
    setState(() => _lines.removeAt(idx));
  }

  Future<void> _refresh() async {
    final api = context.read<AuthState>().api;
    final items = await api.get('/hr/tool-items');
    final issues = await api.get('/hr/tool-issues?mine=1');
    final pool = await api.get('/workflow/ticket-handler-pool?category_code=tool_issue');
    if (!mounted) return;
    setState(() {
      _items = ApiClient.listOf(items.data);
      _issues = ApiClient.listOf(issues.data);
      _pool = (pool.data is Map ? (pool.data as Map)['pool'] as List? : null) ?? [];
      final def = _defaultToolId();
      for (final line in _lines) {
        line.toolId ??= def;
      }
      if (_assignee == null && _pool.isNotEmpty) {
        final u = (_pool.first as Map)['user_id'];
        _assignee = u is num ? u.toInt() : null;
      }
    });
  }

  Future<void> _apply() async {
    if (_assignee == null) {
      setState(() => _msg = '请选择下一手处理人');
      return;
    }
    final payload = <Map<String, dynamic>>[];
    final seen = <int>{};
    for (final line in _lines) {
      if (line.toolId == null || line.qty < 1) continue;
      if (!seen.add(line.toolId!)) {
        setState(() => _msg = '同一单内工具不可重复');
        return;
      }
      payload.add({'tool_item_id': line.toolId, 'issue_qty': line.qty});
    }
    if (payload.isEmpty) {
      setState(() => _msg = '请至少选择一种工具');
      return;
    }
    setState(() {
      _busy = true;
      _msg = '';
    });
    final r = await context.read<AuthState>().api.post('/hr/tool-issues', {
      'items': payload,
      'next_assignee_user_id': _assignee,
      'biz_date': DateTime.now().toIso8601String().substring(0, 10),
    });
    if (!mounted) return;
    setState(() {
      _busy = false;
      _msg = r.ok ? '已提交领取申请' : r.msg;
      if (r.ok) {
        _lines
          ..clear()
          ..add(_ApplyLine(toolId: _defaultToolId()));
      }
    });
    if (r.ok) await _refresh();
  }

  Future<void> _returnReq(Map<String, dynamic> row) async {
    final api = context.read<AuthState>().api;
    int? assignee = _assignee;
    final poolRes = await api.get('/workflow/ticket-handler-pool?category_code=tool_return');
    final pool = (poolRes.data is Map ? (poolRes.data as Map)['pool'] as List? : null) ?? [];
    if (pool.isNotEmpty) {
      assignee = ((pool.first as Map)['user_id'] as num?)?.toInt() ?? assignee;
    }
    if (assignee == null) {
      if (!mounted) return;
      setState(() => _msg = '归还处理人池为空，请联系管理员配置');
      return;
    }

    final rawItems = (row['items'] is List) ? row['items'] as List : const [];
    final returnLines = <_ReturnLine>[];
    for (final raw in rawItems) {
      final m = Map<String, dynamic>.from(raw as Map);
      final remain = ((m['issue_qty'] as num?)?.toDouble() ?? 0) - ((m['return_qty'] as num?)?.toDouble() ?? 0);
      if (remain <= 0) continue;
      final lid = (m['id'] as num?)?.toInt() ?? (m['line_id'] as num?)?.toInt() ?? 0;
      if (lid <= 0) continue;
      returnLines.add(_ReturnLine(
        lineId: lid,
        name: '${m['tool_name'] ?? ''}',
        remain: remain,
        qty: remain.round().clamp(1, remain.round()),
      ));
    }
    if (returnLines.isEmpty) {
      if (!mounted) return;
      setState(() => _msg = '无可归还数量');
      return;
    }

    if (!mounted) return;
    final confirmed = await showModalBottomSheet<List<_ReturnLine>>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) {
        return StatefulBuilder(
          builder: (ctx, setModal) {
            return Padding(
              padding: EdgeInsets.only(
                left: 16,
                right: 16,
                top: 16,
                bottom: MediaQuery.of(ctx).viewInsets.bottom + 16,
              ),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text('申请归还', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                  const SizedBox(height: 12),
                  for (var i = 0; i < returnLines.length; i++) ...[
                    Row(
                      children: [
                        Expanded(child: Text(returnLines[i].name)),
                        IconButton(
                          onPressed: returnLines[i].qty <= 1
                              ? null
                              : () => setModal(() => returnLines[i].qty -= 1),
                          icon: const Icon(Icons.remove_circle_outline),
                        ),
                        Text('${returnLines[i].qty}', style: const TextStyle(fontSize: 16)),
                        IconButton(
                          onPressed: returnLines[i].qty >= returnLines[i].remain
                              ? null
                              : () => setModal(() => returnLines[i].qty += 1),
                          icon: const Icon(Icons.add_circle_outline),
                        ),
                      ],
                    ),
                  ],
                  const SizedBox(height: 12),
                  FilledButton(
                    onPressed: () => Navigator.pop(ctx, returnLines),
                    child: const Text('提交归还申请'),
                  ),
                ],
              ),
            );
          },
        );
      },
    );
    if (confirmed == null) return;

    final items = confirmed
        .where((l) => l.qty > 0)
        .map((l) => {'line_id': l.lineId, 'return_qty': l.qty})
        .toList();
    if (items.isEmpty) return;

    final r = await api.post('/hr/tool-issues/${row['id']}/return-request', {
      'items': items,
      'next_assignee_user_id': assignee,
    });
    if (!mounted) return;
    setState(() => _msg = r.ok ? '已提交归还申请' : r.msg);
    if (r.ok) await _refresh();
  }

  Widget _qtyStepper({
    required int qty,
    required VoidCallback? onMinus,
    required VoidCallback? onPlus,
  }) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        IconButton(
          visualDensity: VisualDensity.compact,
          onPressed: onMinus,
          icon: const Icon(Icons.remove_circle_outline),
        ),
        SizedBox(
          width: 28,
          child: Text('$qty', textAlign: TextAlign.center, style: const TextStyle(fontSize: 16)),
        ),
        IconButton(
          visualDensity: VisualDensity.compact,
          onPressed: onPlus,
          icon: const Icon(Icons.add_circle_outline),
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('物料工具')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const Text('申请领取', style: TextStyle(fontWeight: FontWeight.bold)),
          const SizedBox(height: 8),
          for (var i = 0; i < _lines.length; i++) ...[
            Row(
              children: [
                Expanded(
                  child: DropdownButtonFormField<int>(
                    value: _lines[i].toolId,
                    isExpanded: true,
                    decoration: const InputDecoration(
                      labelText: '工具',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                    items: [
                      for (final raw in _items)
                        DropdownMenuItem(
                          value: ((raw as Map)['id'] as num).toInt(),
                          child: Text('${raw['name']}'),
                        ),
                    ],
                    onChanged: (v) => setState(() => _lines[i].toolId = v),
                  ),
                ),
                _qtyStepper(
                  qty: _lines[i].qty,
                  onMinus: _lines[i].qty <= 1
                      ? null
                      : () => setState(() => _lines[i].qty -= 1),
                  onPlus: () => setState(() => _lines[i].qty += 1),
                ),
                IconButton(
                  onPressed: _lines.length <= 1 ? null : () => _removeLine(i),
                  icon: const Icon(Icons.delete_outline),
                ),
              ],
            ),
            const SizedBox(height: 8),
          ],
          Align(
            alignment: Alignment.centerLeft,
            child: TextButton.icon(
              onPressed: _addLine,
              icon: const Icon(Icons.add),
              label: const Text('添加工具'),
            ),
          ),
          const SizedBox(height: 8),
          DropdownButtonFormField<int>(
            value: _assignee,
            decoration: const InputDecoration(labelText: '下一手处理人', border: OutlineInputBorder()),
            items: [
              for (final raw in _pool)
                DropdownMenuItem(
                  value: ((raw as Map)['user_id'] as num).toInt(),
                  child: Text('${raw['name'] ?? raw['login_name']}'),
                ),
            ],
            onChanged: (v) => setState(() => _assignee = v),
          ),
          const SizedBox(height: 12),
          FilledButton(onPressed: _busy ? null : _apply, child: Text(_busy ? '提交中…' : '提交申请')),
          if (_msg.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(_msg, style: const TextStyle(color: Colors.teal)),
            ),
          const Divider(height: 32),
          Row(
            children: [
              const Text('我的单据', style: TextStyle(fontWeight: FontWeight.bold)),
              const Spacer(),
              TextButton(onPressed: _refresh, child: const Text('刷新')),
            ],
          ),
          ..._issues.map((e) {
            final m = Map<String, dynamic>.from(e as Map);
            final st = '${m['status'] ?? ''}';
            final summary = '${m['items_summary'] ?? m['tool_name'] ?? ''}';
            return Card(
              child: ListTile(
                title: Text('$summary · ${_statusLabel[st] ?? st}'),
                subtitle: Text(
                  '日期 ${m['biz_date']} 序号 ${m['seq_no']}\n领 ${m['issue_qty']} 还 ${m['return_qty']} 在用 ${m['total_qty']}',
                ),
                isThreeLine: true,
                trailing: st == 'open'
                    ? TextButton(onPressed: () => _returnReq(m), child: const Text('申请归还'))
                    : null,
              ),
            );
          }),
        ],
      ),
    );
  }
}
