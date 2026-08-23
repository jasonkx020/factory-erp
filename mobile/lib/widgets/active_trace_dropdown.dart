import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../core/api_client.dart';
import '../core/auth_state.dart';

/// 从「生产中」溯源会话列表选取溯源码（与领料页一致，不支持手输/扫码）。
class ActiveTraceDropdown extends StatefulWidget {
  const ActiveTraceDropdown({
    super.key,
    required this.value,
    required this.onChanged,
    this.prefKey,
    this.labelText = '生产中溯源',
    this.emptyHint = '暂无生产中的溯源码，请先在溯源生产台进入生产',
  });

  final String? value;
  final void Function(String? traceCode, Map<String, dynamic>? row) onChanged;
  final String? prefKey;
  final String labelText;
  final String emptyHint;

  @override
  State<ActiveTraceDropdown> createState() => _ActiveTraceDropdownState();
}

class _ActiveTraceDropdownState extends State<ActiveTraceDropdown> {
  List<Map<String, dynamic>> _traces = [];
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  String _traceLabel(Map<String, dynamic> t) {
    final code = '${t['trace_code'] ?? ''}';
    final farmer = '${t['farmer_name'] ?? ''}'.trim();
    if (farmer.isEmpty) return code;
    return '$code · $farmer';
  }

  Map<String, dynamic>? _rowFor(String? code) {
    if (code == null || code.isEmpty) return null;
    for (final t in _traces) {
      if ('${t['trace_code']}' == code) return t;
    }
    return null;
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    final r = await context.read<AuthState>().api.get('/production/trace-productions?status=in_production&page_size=200');
    if (!mounted) return;
    final list = <Map<String, dynamic>>[];
    if (r.ok) {
      final items = ApiClient.listOf(r.data is Map ? r.data as Map : <String, dynamic>{});
      for (final e in items) {
        if (e is Map) list.add(Map<String, dynamic>.from(e));
      }
    }
    String? next = widget.value;
    if ((next ?? '').isEmpty && widget.prefKey != null) {
      final prefs = await SharedPreferences.getInstance();
      next = prefs.getString(widget.prefKey!);
    }
    if (next != null && !list.any((t) => '${t['trace_code']}' == next)) {
      next = null;
    }
    if (next == null && list.isNotEmpty) {
      next = '${list.first['trace_code']}';
    }
    setState(() {
      _loading = false;
      _traces = list;
    });
    if (!mounted) return;
    if (next != widget.value) {
      widget.onChanged(next, _rowFor(next));
      if (widget.prefKey != null && (next ?? '').isNotEmpty) {
        final prefs = await SharedPreferences.getInstance();
        await prefs.setString(widget.prefKey!, next!);
      }
    }
  }

  Future<void> _onSelected(String? code) async {
    if (widget.prefKey != null && code != null && code.isNotEmpty) {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(widget.prefKey!, code);
    }
    widget.onChanged(code, _rowFor(code));
  }

  @override
  Widget build(BuildContext context) {
    if (_loading && _traces.isEmpty) {
      return const Padding(padding: EdgeInsets.all(8), child: LinearProgressIndicator());
    }
    if (_traces.isEmpty) {
      return Padding(
        padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
        child: Row(
          children: [
            Expanded(child: Text(widget.emptyHint, style: const TextStyle(fontSize: 12, color: Colors.orange))),
            IconButton(tooltip: '刷新', onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh)),
          ],
        ),
      );
    }
    final selected = (widget.value ?? '').isNotEmpty && _traces.any((t) => '${t['trace_code']}' == widget.value)
        ? widget.value
        : null;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
      child: DropdownButtonFormField<String>(
        key: ValueKey('active-trace-${selected ?? ''}-${_traces.length}'),
        initialValue: selected,
        decoration: InputDecoration(
          labelText: widget.labelText,
          border: const OutlineInputBorder(),
          isDense: true,
          suffixIcon: IconButton(
            tooltip: '刷新',
            onPressed: _loading ? null : _load,
            icon: const Icon(Icons.refresh),
          ),
        ),
        items: _traces
            .map((t) {
              final code = '${t['trace_code'] ?? ''}';
              if (code.isEmpty) return null;
              return DropdownMenuItem<String>(value: code, child: Text(_traceLabel(t)));
            })
            .whereType<DropdownMenuItem<String>>()
            .toList(),
        onChanged: _onSelected,
      ),
    );
  }
}
