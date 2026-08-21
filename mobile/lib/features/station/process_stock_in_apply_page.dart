import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/auth_state.dart';
import '../../core/recent_code_store.dart';
import '../../widgets/trace_code_field.dart';

/// 独立入库申请（须溯源生产中）：溯源码 + 工序下拉 + 重量 + 照片
class ProcessStockInApplyPage extends StatefulWidget {
  const ProcessStockInApplyPage({super.key});

  @override
  State<ProcessStockInApplyPage> createState() => _ProcessStockInApplyPageState();
}

class _ProcessStockInApplyPageState extends State<ProcessStockInApplyPage> {
  final _trace = TextEditingController();
  final _kg = TextEditingController();
  int? _processId;
  List<Map<String, dynamic>> _processes = [];
  String? _photo;
  bool _busy = false;
  bool _loadingProc = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      await _loadProcesses();
      await _prefillRecent();
    });
  }

  Future<void> _prefillRecent() async {
    final traces = await RecentCodeStore.list(RecentCodeStore.trace);
    if (!mounted) return;
    if (_trace.text.trim().isEmpty && traces.isNotEmpty) {
      setState(() => _trace.text = traces.first);
    }
  }

  @override
  void dispose() {
    _trace.dispose();
    _kg.dispose();
    super.dispose();
  }

  Future<void> _loadProcesses() async {
    setState(() => _loadingProc = true);
    final prefs = await SharedPreferences.getInstance();
    final savedProc = prefs.getInt('erp.station.process_id');
    if (!mounted) return;
    final r = await context.read<AuthState>().api.get('/production/processes?page_size=200');
    if (!mounted) return;
    setState(() => _loadingProc = false);
    if (!r.ok) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('工序加载失败：${r.msg}')));
      return;
    }
    final raw = r.data;
    final items = raw is Map ? (raw['list'] ?? raw['items']) : (raw is List ? raw : null);
    final list = <Map<String, dynamic>>[];
    if (items is List) {
      for (final e in items) {
        if (e is Map) list.add(Map<String, dynamic>.from(e));
      }
    }
    setState(() {
      _processes = list;
      if (_processId != null && !list.any((p) => (p['id'] as num?)?.toInt() == _processId)) {
        _processId = null;
      }
      if (_processId == null && savedProc != null && savedProc > 0 && list.any((p) => (p['id'] as num?)?.toInt() == savedProc)) {
        _processId = savedProc;
      }
      if (_processId == null && list.isNotEmpty) {
        _processId = (list.first['id'] as num?)?.toInt();
      }
    });
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
            filename: file.name.isEmpty ? 'stock_in.jpg' : file.name,
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

  Future<void> _submit() async {
    final kg = double.tryParse(_kg.text) ?? 0;
    if (_trace.text.trim().isEmpty || (_processId ?? 0) <= 0 || kg <= 0 || (_photo ?? '').isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请填写溯源、工序、重量并拍照')));
      return;
    }
    setState(() => _busy = true);
    final r = await context.read<AuthState>().api.post('/production/process-stock-ins', {
      'trace_code': _trace.text.trim(),
      'process_id': _processId,
      'apply_kg': kg,
      'reweigh_kg': kg,
      'photo_url': _photo,
      'image_url': _photo,
    });
    if (!mounted) return;
    setState(() => _busy = false);
    if (r.ok) {
      await RecentCodeStore.remember(RecentCodeStore.trace, _trace.text);
      final prefs = await SharedPreferences.getInstance();
      await prefs.setInt('erp.station.process_id', _processId!);
      if (!mounted) return;
    }
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '入库申请已提交，待仓管过磅' : r.msg)));
    if (r.ok) Navigator.pop(context, true);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('入库申请')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TraceCodeField(
            controller: _trace,
            label: '溯源码',
            hint: '须已进入生产；可点最近使用',
            historyKey: RecentCodeStore.trace,
          ),
          const SizedBox(height: 12),
          DropdownButtonFormField<int>(
            key: ValueKey('proc-${_processId ?? 0}-${_processes.length}'),
            initialValue: _processId != null && _processes.any((p) => (p['id'] as num?)?.toInt() == _processId) ? _processId : null,
            decoration: InputDecoration(
              labelText: '当前工序（必选）',
              border: const OutlineInputBorder(),
              isDense: true,
              suffixIcon: _loadingProc
                  ? const Padding(
                      padding: EdgeInsets.all(12),
                      child: SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2)),
                    )
                  : IconButton(
                      tooltip: '刷新工序',
                      icon: const Icon(Icons.refresh),
                      onPressed: _loadingProc ? null : _loadProcesses,
                    ),
            ),
            items: _processes
                .map((p) {
                  final id = (p['id'] as num?)?.toInt();
                  if (id == null) return null;
                  final name = '${p['name'] ?? p['code'] ?? id}';
                  return DropdownMenuItem<int>(value: id, child: Text(name));
                })
                .whereType<DropdownMenuItem<int>>()
                .toList(),
            onChanged: (v) => setState(() => _processId = v),
          ),
          if (_processes.isEmpty && !_loadingProc)
            const Padding(
              padding: EdgeInsets.only(top: 6),
              child: Text('暂无工序可选，请点右侧刷新', style: TextStyle(fontSize: 12, color: Colors.orange)),
            ),
          const SizedBox(height: 12),
          TextField(
            controller: _kg,
            decoration: const InputDecoration(
              labelText: '申请入库 kg',
              border: OutlineInputBorder(),
              isDense: true,
            ),
            keyboardType: const TextInputType.numberWithOptions(decimal: true),
          ),
          const SizedBox(height: 8),
          ListTile(
            contentPadding: EdgeInsets.zero,
            leading: Icon(
              (_photo ?? '').isEmpty ? Icons.camera_alt_outlined : Icons.check_circle,
              color: (_photo ?? '').isEmpty ? Colors.orange : Colors.teal,
            ),
            title: Text((_photo ?? '').isEmpty ? '拍摄复磅照片（必填）' : '复磅照片已上传'),
            onTap: _takePhoto,
          ),
          const SizedBox(height: 16),
          FilledButton(onPressed: _busy ? null : _submit, child: Text(_busy ? '提交中…' : '提交入库申请')),
        ],
      ),
    );
  }
}
