import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/auth_state.dart';
import '../../widgets/active_trace_dropdown.dart';

/// 独立入库申请（须溯源生产中）：溯源码 + 工序下拉 + 重量 + 照片
class ProcessStockInApplyPage extends StatefulWidget {
  const ProcessStockInApplyPage({super.key});

  @override
  State<ProcessStockInApplyPage> createState() => _ProcessStockInApplyPageState();
}

class _ProcessStockInApplyPageState extends State<ProcessStockInApplyPage> {
  final _kg = TextEditingController();
  int? _processId;
  String? _selectedTraceCode;
  List<Map<String, dynamic>> _processes = [];
  List<Map<String, dynamic>> _wipSteps = [];
  String? _photo;
  bool _busy = false;
  bool _loadingProc = false;
  bool _loadingWip = false;

  static const _tracePrefKey = 'erp.station.selected_trace';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      final prefs = await SharedPreferences.getInstance();
      _selectedTraceCode = prefs.getString(_tracePrefKey);
      await _loadProcesses();
    });
  }

  @override
  void dispose() {
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
      if (savedProc != null && savedProc > 0) {
        _processId = savedProc;
      }
    });
    final trace = (_selectedTraceCode ?? '').trim();
    if (trace.isNotEmpty) {
      await _loadWipForTrace();
    } else {
      setState(() {
        _processes = [];
        _processId = null;
        _wipSteps = [];
      });
    }
  }

  Future<void> _loadWipForTrace() async {
    final trace = (_selectedTraceCode ?? '').trim();
    if (trace.isEmpty) {
      setState(() {
        _wipSteps = [];
        _processes = [];
        _processId = null;
      });
      return;
    }
    setState(() => _loadingWip = true);
    final r = await context.read<AuthState>().api.get('/production/trace-productions/${Uri.encodeComponent(trace)}/wip');
    if (!mounted) return;
    setState(() => _loadingWip = false);
    if (!r.ok || r.data is! Map) {
      setState(() {
        _wipSteps = [];
        _processes = [];
        _processId = null;
      });
      return;
    }
    final data = Map<String, dynamic>.from(r.data as Map);
    final steps = <Map<String, dynamic>>[];
    final raw = data['steps'];
    if (raw is List) {
      for (final e in raw) {
        if (e is Map) steps.add(Map<String, dynamic>.from(e));
      }
    }
    final routing = data['routing_steps'];
    final filtered = <Map<String, dynamic>>[];
    if (routing is List && routing.isNotEmpty) {
      for (final e in routing) {
        if (e is! Map) continue;
        final m = Map<String, dynamic>.from(e);
        final pid = (m['process_id'] as num?)?.toInt();
        if (pid == null || pid <= 0) continue;
        filtered.add({
          'id': pid,
          'name': m['step_name'] ?? m['process_name'] ?? pid,
          'code': m['step_code'] ?? '',
        });
      }
    }
    final stockable = steps.where((s) => s['can_stock_in'] == true || ((s['wip_kg'] as num?)?.toDouble() ?? 0) > 0).toList();
    int? pid = _processId;
    final candidates = filtered.isNotEmpty ? filtered : _processes;
    if (pid != null && !candidates.any((p) => (p['id'] as num?)?.toInt() == pid)) {
      pid = null;
    }
    if (pid != null && stockable.isNotEmpty && !stockable.any((s) => (s['process_id'] as num?)?.toInt() == pid)) {
      pid = null;
    }
    if (pid == null && stockable.isNotEmpty) {
      final stockIds = stockable.map((s) => (s['process_id'] as num?)?.toInt()).whereType<int>().toSet();
      for (final p in candidates) {
        final id = (p['id'] as num?)?.toInt();
        if (id != null && stockIds.contains(id)) {
          pid = id;
          break;
        }
      }
      pid ??= (stockable.first['process_id'] as num?)?.toInt();
    }
    setState(() {
      _wipSteps = steps;
      _processes = filtered;
      _processId = pid;
    });
  }

  void _onTraceSelected(String? code, Map<String, dynamic>? row) {
    setState(() => _selectedTraceCode = code);
    _loadWipForTrace();
  }

  List<Map<String, dynamic>> get _stockableProcesses {
    if (_wipSteps.isEmpty) return [];
    final ids = _wipSteps
        .where((s) => s['can_stock_in'] == true || ((s['wip_kg'] as num?)?.toDouble() ?? 0) > 0)
        .map((s) => (s['process_id'] as num?)?.toInt())
        .whereType<int>()
        .toSet();
    if (ids.isEmpty) return [];
    return _processes.where((p) => ids.contains((p['id'] as num?)?.toInt())).toList();
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
    final trace = (_selectedTraceCode ?? '').trim();
    final kg = double.tryParse(_kg.text) ?? 0;
    if (trace.isEmpty || (_processId ?? 0) <= 0 || kg <= 0 || (_photo ?? '').isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请选择溯源、工序、重量并拍照')));
      return;
    }
    setState(() => _busy = true);
    final r = await context.read<AuthState>().api.post('/production/process-stock-ins', {
      'trace_code': trace,
      'process_id': _processId,
      'apply_kg': kg,
      'reweigh_kg': kg,
      'photo_url': _photo,
      'image_url': _photo,
    });
    if (!mounted) return;
    setState(() => _busy = false);
    if (r.ok) {
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
          ActiveTraceDropdown(
            value: _selectedTraceCode,
            prefKey: _tracePrefKey,
            onChanged: _onTraceSelected,
          ),
          if (_loadingWip)
            const Padding(
              padding: EdgeInsets.only(top: 8),
              child: LinearProgressIndicator(),
            ),
          const SizedBox(height: 12),
          DropdownButtonFormField<int>(
            key: ValueKey('proc-${_processId ?? 0}-${_stockableProcesses.length}-${_wipSteps.length}'),
            initialValue: _processId != null && _stockableProcesses.any((p) => (p['id'] as num?)?.toInt() == _processId)
                ? _processId
                : null,
            decoration: InputDecoration(
              labelText: '当前工序（本批工艺 · 须有在制）',
              border: const OutlineInputBorder(),
              isDense: true,
              helperText: (_selectedTraceCode ?? '').isEmpty ? '请先选择进行中的溯源生产' : null,
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
            items: _stockableProcesses
                .map((p) {
                  final id = (p['id'] as num?)?.toInt();
                  if (id == null) return null;
                  final name = '${p['name'] ?? p['code'] ?? id}';
                  double? wip;
                  for (final s in _wipSteps) {
                    if ((s['process_id'] as num?)?.toInt() == id) {
                      wip = (s['wip_kg'] as num?)?.toDouble();
                      break;
                    }
                  }
                  final suffix = wip != null && wip > 0 ? ' · 在制 ${wip.toStringAsFixed(2)} kg' : '';
                  return DropdownMenuItem<int>(value: id, child: Text('$name$suffix'));
                })
                .whereType<DropdownMenuItem<int>>()
                .toList(),
            onChanged: _stockableProcesses.isEmpty ? null : (v) => setState(() => _processId = v),
          ),
          if ((_selectedTraceCode ?? '').isNotEmpty && !_loadingWip && _stockableProcesses.isEmpty)
            const Padding(
              padding: EdgeInsets.only(top: 6),
              child: Text('该溯源码暂无工序在制，不能申请入库', style: TextStyle(fontSize: 12, color: Colors.orange)),
            )
          else if (_processes.isEmpty && !_loadingProc)
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
