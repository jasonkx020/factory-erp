import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/recent_code_store.dart';
import '../../widgets/trace_code_field.dart';

/// 仓管：退库待审 + 入库待审过磅同意
class ProcessPendingApprovePage extends StatefulWidget {
  const ProcessPendingApprovePage({super.key});

  @override
  State<ProcessPendingApprovePage> createState() => _ProcessPendingApprovePageState();
}

class _ProcessPendingApprovePageState extends State<ProcessPendingApprovePage> with SingleTickerProviderStateMixin {
  late TabController _tab;
  List<Map<String, dynamic>> _returns = [];
  List<Map<String, dynamic>> _stockIns = [];
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    _tab = TabController(length: 2, vsync: this);
    _load();
  }

  @override
  void dispose() {
    _tab.dispose();
    super.dispose();
  }

  List<Map<String, dynamic>> _parseList(dynamic data) {
    final items = ApiClient.listOf(data);
    return items.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    final api = context.read<AuthState>().api;
    final r1 = await api.get('/production/process-issues?biz_status=return_pending&page_size=100');
    final r2 = await api.get('/production/process-stock-ins?status=pending_warehouse&page_size=100');
    if (!mounted) return;
    setState(() {
      _loading = false;
      if (r1.ok) _returns = _parseList(r1.data);
      if (r2.ok) _stockIns = _parseList(r2.data);
    });
  }

  Future<String?> _capturePhoto() async {
    try {
      final picker = ImagePicker();
      final file = await picker.pickImage(source: ImageSource.camera, imageQuality: 85);
      if (file == null) return null;
      final bytes = await file.readAsBytes();
      if (!mounted) return null;
      final r = await context.read<AuthState>().api.postMultipart('/biz/uploads', bytes, filename: 'wh_reweigh.jpg');
      if (!r.ok || r.data is! Map) return null;
      return (r.data as Map)['url']?.toString() ?? (r.data as Map)['file_url']?.toString();
    } catch (_) {
      return null;
    }
  }

  Future<void> _approveReturn(Map<String, dynamic> row) async {
    final photo = await _capturePhoto();
    if (photo == null || photo.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('需拍摄仓管复磅照片')));
      return;
    }
    final kg = (row['pending_return_kg'] as num?)?.toDouble() ?? 0;
    final id = (row['id'] as num).toInt();
    if (!mounted) return;
    final r = await context.read<AuthState>().api.post('/production/process-issues/$id/return-approve', {
      'reweigh_kg': kg,
      'photo_url': photo,
      'image_url': photo,
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '退库已同意' : r.msg)));
    if (r.ok) _load();
  }

  Future<String?> _askBoardCode(String initial) async {
    final ctrl = TextEditingController(text: initial);
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('分配板码'),
        content: TraceCodeField(
          controller: ctrl,
          label: '板码',
          hint: '入库过账必填，可扫码或点最近使用',
          scannerTitle: '扫描板码',
          historyKey: RecentCodeStore.board,
          compact: false,
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('确定')),
        ],
      ),
    );
    final code = ctrl.text.trim();
    ctrl.dispose();
    if (ok != true || code.isEmpty) return null;
    return code;
  }

  Future<void> _approveStockIn(Map<String, dynamic> row) async {
    var board = '${row['board_code'] ?? ''}'.trim();
    if (board.isEmpty) {
      final asked = await _askBoardCode('');
      if (asked == null || asked.isEmpty) {
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('过账需要板码')));
        return;
      }
      board = asked;
    }
    final photo = await _capturePhoto();
    if (photo == null || photo.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('需拍摄仓管复磅照片')));
      return;
    }
    final kg = (row['apply_kg'] as num?)?.toDouble() ?? (row['reweigh_kg'] as num?)?.toDouble() ?? 0;
    final id = (row['id'] as num).toInt();
    if (!mounted) return;
    final r = await context.read<AuthState>().api.post('/production/process-stock-ins/$id/approve', {
      'reweigh_kg': kg,
      'photo_url': photo,
      'image_url': photo,
      'board_code': board,
    });
    if (!mounted) return;
    if (r.ok) {
      await RecentCodeStore.remember(RecentCodeStore.board, board);
    }
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '入库已过账' : r.msg)));
    if (r.ok) _load();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('领退入库待审'),
        bottom: TabBar(controller: _tab, tabs: const [Tab(text: '退库待审'), Tab(text: '入库待审')]),
        actions: [IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh))],
      ),
      body: TabBarView(
        controller: _tab,
        children: [
          ListView(
            children: [
              if (_loading) const LinearProgressIndicator(),
              for (final e in _returns)
                ListTile(
                  title: Text('${e['board_code']} · ${e['process_name']}'),
                  subtitle: Text('申请退 ${e['pending_return_kg']} kg · ${e['worker_name']} · ${e['trace_code']}'),
                  trailing: TextButton(onPressed: () => _approveReturn(e), child: const Text('同意')),
                ),
              if (!_loading && _returns.isEmpty) const ListTile(title: Text('暂无退库待审')),
            ],
          ),
          ListView(
            children: [
              if (_loading) const LinearProgressIndicator(),
              for (final e in _stockIns)
                ListTile(
                  title: Text('${e['doc_no'] ?? ''} · ${e['process_name'] ?? ''}'),
                  subtitle: Text(
                    '申请 ${e['apply_kg']} kg · ${e['trace_code']}'
                    '${'${e['board_code'] ?? ''}'.trim().isEmpty ? ' · 待分配板码' : ' · ${e['board_code']}'}',
                  ),
                  trailing: TextButton(onPressed: () => _approveStockIn(e), child: const Text('过账')),
                ),
              if (!_loading && _stockIns.isEmpty) const ListTile(title: Text('暂无入库待审')),
            ],
          ),
        ],
      ),
    );
  }
}
