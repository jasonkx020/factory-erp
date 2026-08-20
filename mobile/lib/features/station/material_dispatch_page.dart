import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../widgets/form_row.dart';
import '../../widgets/trace_code_field.dart';

/// 生管：派料（板码+工牌）列表与完工确认；工人可看自己名下记录。
class MaterialDispatchPage extends StatefulWidget {
  const MaterialDispatchPage({super.key, this.scope = 'mine_dispatch'});

  /// mine_dispatch | mine_work | all
  final String scope;

  @override
  State<MaterialDispatchPage> createState() => _MaterialDispatchPageState();
}

class _MaterialDispatchPageState extends State<MaterialDispatchPage> {
  List<Map<String, dynamic>> _rows = [];
  bool _busy = false;
  String _msg = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    setState(() => _busy = true);
    final q = widget.scope == 'all' ? '' : '?scope=${widget.scope}';
    final r = await context.read<AuthState>().api.get('/production/material-dispatches$q');
    if (!mounted) return;
    setState(() {
      _busy = false;
      if (r.ok && r.data is Map) {
        final items = ApiClient.listOf((r.data as Map)['items']);
        _rows = items.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
      } else {
        _msg = r.msg;
      }
    });
  }

  Future<void> _complete(Map<String, dynamic> row) async {
    try {
      final picker = ImagePicker();
      final file = await picker.pickImage(source: ImageSource.camera, imageQuality: 85);
      if (file == null) return;
      final bytes = await file.readAsBytes();
      final up = await context.read<AuthState>().api.postMultipart(
            '/biz/uploads',
            bytes,
            filename: file.name.isEmpty ? 'confirm.jpg' : file.name,
          );
      if (!mounted) return;
      if (!up.ok || up.data is! Map) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('上传失败：${up.msg}')));
        return;
      }
      final url = (up.data as Map)['url']?.toString() ?? (up.data as Map)['file_url']?.toString() ?? '';
      final id = row['id'];
      final r = await context.read<AuthState>().api.post(
            '/production/material-dispatches/$id/complete',
            {'confirm_photo_url': url},
          );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已确认完工' : r.msg)));
      if (r.ok) await _load();
    } catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    }
  }

  Future<void> _openCreate() async {
    final ok = await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => const _CreateDispatchPage()),
    );
    if (ok == true) await _load();
  }

  @override
  Widget build(BuildContext context) {
    final isForeman = widget.scope != 'mine_work';
    return Scaffold(
      appBar: AppBar(
        title: Text(isForeman ? '派料记录' : '我的派料'),
        actions: [
          IconButton(onPressed: _busy ? null : _load, icon: const Icon(Icons.refresh)),
          if (isForeman)
            IconButton(onPressed: _openCreate, icon: const Icon(Icons.add)),
        ],
      ),
      body: _busy && _rows.isEmpty
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(12),
              children: [
                if (_msg.isNotEmpty) Text(_msg, style: const TextStyle(color: Colors.orange)),
                if (_rows.isEmpty) const Padding(padding: EdgeInsets.all(24), child: Center(child: Text('暂无记录'))),
                for (final r in _rows)
                  Card(
                    child: ListTile(
                      title: Text('${r['board_code'] ?? '-'} · ${r['process_name'] ?? '-'}'),
                      subtitle: Text(
                        '${r['worker_name'] ?? '-'} · ${r['weight_kg'] ?? '-'} kg\n'
                        '${r['source_kind'] == 'process' ? '工序出' : '仓库出'} · ${r['status']}\n'
                        '工价 ¥${r['wage_amount'] ?? 0}',
                      ),
                      isThreeLine: true,
                      trailing: r['status'] == 'in_progress' && isForeman
                          ? TextButton(onPressed: () => _complete(r), child: const Text('完工'))
                          : Text('${r['status']}'),
                    ),
                  ),
              ],
            ),
    );
  }
}

class _CreateDispatchPage extends StatefulWidget {
  const _CreateDispatchPage();

  @override
  State<_CreateDispatchPage> createState() => _CreateDispatchPageState();
}

class _CreateDispatchPageState extends State<_CreateDispatchPage> {
  final _badge = TextEditingController();
  final _board = TextEditingController();
  final _kg = TextEditingController();
  List<Map<String, dynamic>> _processes = [];
  int? _processId;
  String _source = 'warehouse';
  String? _photoUrl;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadProcesses());
  }

  @override
  void dispose() {
    _badge.dispose();
    _board.dispose();
    _kg.dispose();
    super.dispose();
  }

  Future<void> _loadProcesses() async {
    final r = await context.read<AuthState>().api.get('/production/processes?page_size=200');
    if (!mounted || !r.ok) return;
    final list = ApiClient.listOf(r.data);
    setState(() {
      _processes = list.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
      if (_processes.isNotEmpty) {
        _processId = (_processes.first['id'] as num?)?.toInt();
      }
    });
  }

  Future<void> _takePhoto() async {
    final picker = ImagePicker();
    final file = await picker.pickImage(source: ImageSource.camera, imageQuality: 85);
    if (file == null) return;
    final bytes = await file.readAsBytes();
    final up = await context.read<AuthState>().api.postMultipart(
          '/biz/uploads',
          bytes,
          filename: file.name.isEmpty ? 'reweigh.jpg' : file.name,
        );
    if (!mounted) return;
    if (!up.ok || up.data is! Map) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(up.msg)));
      return;
    }
    setState(() {
      _photoUrl = (up.data as Map)['url']?.toString() ?? (up.data as Map)['file_url']?.toString();
    });
  }

  Future<void> _submit() async {
    final kg = double.tryParse(_kg.text) ?? 0;
    if (_board.text.trim().isEmpty || _processId == null || kg <= 0) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请填写板码、工序与重量')));
      return;
    }
    if ((_photoUrl ?? '').isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请拍摄复磅照片')));
      return;
    }
    setState(() => _busy = true);
    final r = await context.read<AuthState>().api.post('/production/material-dispatches', {
      'board_code': _board.text.trim(),
      'badge_code': _badge.text.trim(),
      'process_id': _processId,
      'weight_kg': kg,
      'reweigh_kg': kg,
      'source_kind': _source,
      'photo_url': _photoUrl,
    });
    if (!mounted) return;
    setState(() => _busy = false);
    if (!r.ok) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.msg)));
      return;
    }
    Navigator.of(context).pop(true);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('新建派料')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TraceCodeField(controller: _badge, label: '工牌', hint: '扫员工工牌', scannerTitle: '扫描工牌'),
          TraceCodeField(controller: _board, label: '板码', hint: '扫板码', scannerTitle: '扫描板码'),
          FormRow(
            label: '工序',
            requiredMark: true,
            child: DropdownButtonHideUnderline(
              child: DropdownButton<int>(
                isExpanded: true,
                value: _processId,
                items: [
                  for (final p in _processes)
                    DropdownMenuItem(
                      value: (p['id'] as num?)?.toInt(),
                      child: Text('${p['name'] ?? p['code'] ?? p['id']}'),
                    ),
                ],
                onChanged: (v) => setState(() => _processId = v),
              ),
            ),
          ),
          FormRow.text(
            label: '重量(kg)',
            controller: _kg,
            keyboardType: const TextInputType.numberWithOptions(decimal: true),
            requiredMark: true,
          ),
          const SizedBox(height: 8),
          const Text('出料来源（优先仓库）', style: TextStyle(fontWeight: FontWeight.w600)),
          SegmentedButton<String>(
            segments: const [
              ButtonSegment(value: 'warehouse', label: Text('仓库')),
              ButtonSegment(value: 'process', label: Text('工序')),
            ],
            selected: {_source},
            onSelectionChanged: (s) => setState(() => _source = s.first),
          ),
          ListTile(
            contentPadding: EdgeInsets.zero,
            title: Text((_photoUrl ?? '').isEmpty ? '复磅照片（必填）' : '复磅照片已上传'),
            trailing: TextButton(onPressed: _takePhoto, child: const Text('拍照')),
          ),
          const SizedBox(height: 16),
          FilledButton(onPressed: _busy ? null : _submit, child: Text(_busy ? '提交中…' : '派料')),
        ],
      ),
    );
  }
}

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
