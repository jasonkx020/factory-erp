import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';

/// 物料工具申请/归还 + 指定下一手处理人
class ToolIssuePage extends StatefulWidget {
  const ToolIssuePage({super.key});

  @override
  State<ToolIssuePage> createState() => _ToolIssuePageState();
}

class _ToolIssuePageState extends State<ToolIssuePage> {
  List<dynamic> _items = [];
  List<dynamic> _issues = [];
  List<dynamic> _pool = [];
  int? _toolId;
  int? _assignee;
  final _qty = TextEditingController(text: '1');
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

  @override
  void dispose() {
    _qty.dispose();
    super.dispose();
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
      if (_toolId == null && _items.isNotEmpty) {
        _toolId = (_items.first as Map)['id'] is num ? ((_items.first as Map)['id'] as num).toInt() : null;
      }
      if (_assignee == null && _pool.isNotEmpty) {
        final u = (_pool.first as Map)['user_id'];
        _assignee = u is num ? u.toInt() : null;
      }
    });
  }

  Future<void> _apply() async {
    if (_toolId == null || _assignee == null) {
      setState(() => _msg = '请选择工具与下一手处理人');
      return;
    }
    setState(() {
      _busy = true;
      _msg = '';
    });
    final r = await context.read<AuthState>().api.post('/hr/tool-issues', {
      'tool_item_id': _toolId,
      'issue_qty': double.tryParse(_qty.text) ?? 1,
      'next_assignee_user_id': _assignee,
      'biz_date': DateTime.now().toIso8601String().substring(0, 10),
    });
    if (!mounted) return;
    setState(() {
      _busy = false;
      _msg = r.ok ? '已提交领取申请' : r.msg;
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
    final qty = (row['total_qty'] as num?)?.toDouble() ?? 0;
    final r = await api.post('/hr/tool-issues/${row['id']}/return-request', {
      'return_qty': qty,
      'next_assignee_user_id': assignee,
    });
    if (!mounted) return;
    setState(() => _msg = r.ok ? '已提交归还申请' : r.msg);
    if (r.ok) await _refresh();
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
          DropdownButtonFormField<int>(
            initialValue: _toolId,
            decoration: const InputDecoration(labelText: '工具', border: OutlineInputBorder()),
            items: [
              for (final raw in _items)
                DropdownMenuItem(
                  value: ((raw as Map)['id'] as num).toInt(),
                  child: Text('${raw['name']}'),
                ),
            ],
            onChanged: (v) => setState(() => _toolId = v),
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _qty,
            decoration: const InputDecoration(labelText: '领取数量', border: OutlineInputBorder()),
            keyboardType: TextInputType.number,
          ),
          const SizedBox(height: 8),
          DropdownButtonFormField<int>(
            initialValue: _assignee,
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
          if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_msg, style: const TextStyle(color: Colors.teal))),
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
            return Card(
              child: ListTile(
                title: Text('${m['tool_name']} · ${_statusLabel[st] ?? st}'),
                subtitle: Text(
                  '日期 ${m['biz_date']} 序号 ${m['seq_no']}\n领 ${m['issue_qty']} 还 ${m['return_qty']} 合计 ${m['total_qty']}',
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
