import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';

/// 按工单类型动态填表创建
class TicketCreatePage extends StatefulWidget {
  const TicketCreatePage({super.key});

  @override
  State<TicketCreatePage> createState() => _TicketCreatePageState();
}

class _TicketCreatePageState extends State<TicketCreatePage> {
  List<dynamic> _categories = [];
  List<dynamic> _pool = [];
  int? _categoryId;
  int? _assignee;
  List<Map<String, dynamic>> _schema = [];
  final Map<String, TextEditingController> _ctrls = {};
  final Map<String, dynamic> _selects = {};
  String _msg = '';
  bool _busy = false;
  final _title = TextEditingController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadCats());
  }

  @override
  void dispose() {
    _title.dispose();
    for (final c in _ctrls.values) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _loadCats() async {
    final r = await context.read<AuthState>().api.get('/workflow/ticket-categories');
    if (!mounted) return;
    final list = ApiClient.listOf(r.data);
    // skip pure tool types — use dedicated tool page; still allow if user picks
    setState(() {
      _categories = list;
      if (_categories.isNotEmpty) {
        _categoryId = ((_categories.first as Map)['id'] as num?)?.toInt();
      }
    });
    if (_categoryId != null) await _onCat(_categoryId!);
  }

  Future<void> _onCat(int id) async {
    for (final c in _ctrls.values) {
      c.dispose();
    }
    _ctrls.clear();
    _selects.clear();
    final cat = _categories.cast<Map>().firstWhere((e) => (e['id'] as num?)?.toInt() == id, orElse: () => {});
    final schemaRaw = cat['form_schema'];
    final schema = <Map<String, dynamic>>[];
    if (schemaRaw is List) {
      for (final x in schemaRaw) {
        schema.add(Map<String, dynamic>.from(x as Map));
      }
    }
    final today = DateTime.now().toIso8601String().substring(0, 10);
    for (final f in schema) {
      final key = '${f['key']}';
      final type = '${f['type'] ?? 'text'}';
      if (type == 'select') {
        final opts = (f['options'] as List?)?.map((e) => '$e').toList() ?? [];
        _selects[key] = opts.isNotEmpty ? opts.first : null;
      } else if (type == 'date') {
        _ctrls[key] = TextEditingController(text: today);
      } else {
        _ctrls[key] = TextEditingController();
      }
    }
    final poolRes = await context.read<AuthState>().api.get('/workflow/ticket-handler-pool?category_id=$id');
    final pool = (poolRes.data is Map ? (poolRes.data as Map)['pool'] as List? : null) ?? [];
    if (!mounted) return;
    setState(() {
      _categoryId = id;
      _schema = schema;
      _pool = pool;
      _assignee = pool.isNotEmpty ? ((pool.first as Map)['user_id'] as num?)?.toInt() : null;
    });
  }

  Future<void> _submit() async {
    if (_categoryId == null || _assignee == null) {
      setState(() => _msg = '请选择类型与下一手');
      return;
    }
    final payload = <String, dynamic>{};
    for (final f in _schema) {
      final key = '${f['key']}';
      final type = '${f['type'] ?? 'text'}';
      if (type == 'select') {
        payload[key] = _selects[key];
      } else if (type == 'number') {
        payload[key] = double.tryParse(_ctrls[key]?.text ?? '');
      } else {
        payload[key] = _ctrls[key]?.text.trim() ?? '';
      }
    }
    setState(() {
      _busy = true;
      _msg = '';
    });
    final body = <String, dynamic>{
      'category_id': _categoryId,
      'next_assignee_user_id': _assignee,
      'payload': payload,
    };
    if (_title.text.trim().isNotEmpty) body['title'] = _title.text.trim();
    final r = await context.read<AuthState>().api.post('/workflow/tickets', body);
    if (!mounted) return;
    setState(() {
      _busy = false;
      _msg = r.ok ? '工单已创建' : r.msg;
    });
    if (r.ok && mounted) {
      Navigator.of(context).pop(true);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('新建工单')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          DropdownButtonFormField<int>(
            initialValue: _categoryId,
            decoration: const InputDecoration(labelText: '工单类型', border: OutlineInputBorder()),
            items: [
              for (final raw in _categories)
                DropdownMenuItem(
                  value: ((raw as Map)['id'] as num).toInt(),
                  child: Text('${raw['name']}'),
                ),
            ],
            onChanged: (v) {
              if (v != null) _onCat(v);
            },
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _title,
            decoration: const InputDecoration(labelText: '标题（可空）', border: OutlineInputBorder()),
          ),
          const SizedBox(height: 12),
          ..._schema.map((f) {
            final key = '${f['key']}';
            final type = '${f['type'] ?? 'text'}';
            final label = '${f['label']}${f['required'] == true ? ' *' : ''}';
            if (type == 'select') {
              final opts = (f['options'] as List?)?.map((e) => '$e').toList() ?? [];
              return Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: DropdownButtonFormField<String>(
                  initialValue: _selects[key]?.toString(),
                  decoration: InputDecoration(labelText: label, border: const OutlineInputBorder()),
                  items: [for (final o in opts) DropdownMenuItem(value: o, child: Text(o))],
                  onChanged: (v) => setState(() => _selects[key] = v),
                ),
              );
            }
            return Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: TextField(
                controller: _ctrls[key],
                decoration: InputDecoration(
                  labelText: label,
                  border: const OutlineInputBorder(),
                  suffixText: f['unit']?.toString(),
                ),
                keyboardType: type == 'number' ? TextInputType.number : TextInputType.text,
                maxLines: type == 'textarea' ? 3 : 1,
              ),
            );
          }),
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
          const SizedBox(height: 16),
          FilledButton(onPressed: _busy ? null : _submit, child: Text(_busy ? '提交中…' : '提交工单')),
          if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_msg)),
        ],
      ),
    );
  }
}
