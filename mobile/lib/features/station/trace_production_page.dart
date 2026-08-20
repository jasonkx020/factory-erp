import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../widgets/trace_code_field.dart';

/// 生管扫溯源码启停生产会话
class TraceProductionPage extends StatefulWidget {
  const TraceProductionPage({super.key});

  @override
  State<TraceProductionPage> createState() => _TraceProductionPageState();
}

class _TraceProductionPageState extends State<TraceProductionPage> {
  final _trace = TextEditingController();
  Map<String, dynamic>? _session;
  List<Map<String, dynamic>> _logs = [];
  bool _busy = false;

  @override
  void dispose() {
    _trace.dispose();
    super.dispose();
  }

  Future<void> _start() async {
    setState(() => _busy = true);
    final r = await context.read<AuthState>().api.post('/production/trace-productions/start', {
      'trace_code': _trace.text.trim(),
    });
    if (!mounted) return;
    setState(() => _busy = false);
    if (!r.ok) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.msg)));
      return;
    }
    setState(() => _session = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : null);
    await _loadLogs();
  }

  Future<void> _complete() async {
    setState(() => _busy = true);
    final r = await context.read<AuthState>().api.post('/production/trace-productions/complete', {
      'trace_code': _trace.text.trim(),
      'id': _session?['id'],
    });
    if (!mounted) return;
    setState(() => _busy = false);
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '生产已完成' : r.msg)));
    if (r.ok) {
      setState(() => _session = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : _session);
      await _loadLogs();
    }
  }

  Future<void> _loadLogs() async {
    final code = _trace.text.trim();
    if (code.isEmpty) return;
    final r = await context.read<AuthState>().api.get(
          '/production/trace-productions/logs?trace_code=${Uri.encodeComponent(code)}',
        );
    if (!mounted || !r.ok || r.data is! Map) return;
    final items = ApiClient.listOf((r.data as Map)['items']);
    setState(() {
      _logs = items.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('溯源生产会话')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TraceCodeField(
            controller: _trace,
            label: '溯源码',
            hint: '扫码后开始/完成生产',
            scannerTitle: '扫描溯源码',
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(child: FilledButton(onPressed: _busy ? null : _start, child: const Text('开始生产'))),
              const SizedBox(width: 12),
              Expanded(child: OutlinedButton(onPressed: _busy ? null : _complete, child: const Text('完成生产'))),
            ],
          ),
          if (_session != null) ...[
            const SizedBox(height: 16),
            Text('状态：${_session!['status']} · 扣损率 ${_session!['loss_rate'] ?? 0}'),
            Text('投入 ${_session!['input_kg'] ?? '-'} / 产出 ${_session!['output_kg'] ?? '-'}'),
          ],
          const SizedBox(height: 16),
          const Text('过程日志', style: TextStyle(fontWeight: FontWeight.w600)),
          for (final l in _logs)
            ListTile(
              dense: true,
              title: Text('${l['event_type']} · ${l['process_name'] ?? ''}'),
              subtitle: Text('${l['created_at'] ?? ''}'),
            ),
        ],
      ),
    );
  }
}
