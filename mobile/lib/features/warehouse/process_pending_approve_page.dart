import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/recent_code_store.dart';
import '../../widgets/trace_code_field.dart';

/// 仓管：领料待审 + 退库待审 + 入库待审过磅同意
class ProcessPendingApprovePage extends StatefulWidget {
  const ProcessPendingApprovePage({super.key});

  @override
  State<ProcessPendingApprovePage> createState() => _ProcessPendingApprovePageState();
}

class _ProcessPendingApprovePageState extends State<ProcessPendingApprovePage> with SingleTickerProviderStateMixin {
  late TabController _tab;
  List<Map<String, dynamic>> _issues = [];
  List<Map<String, dynamic>> _returns = [];
  List<Map<String, dynamic>> _stockIns = [];
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    _tab = TabController(length: 3, vsync: this);
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

  String _normTrace(String? raw) => (raw ?? '').trim().toUpperCase();

  String _traceMismatchMessage({
    required String boardCode,
    required String expectedTrace,
    String? boardTrace,
  }) {
    final boardLabel = boardTrace == null || boardTrace.trim().isEmpty ? '未绑定溯源' : boardTrace.trim();
    return '板码 $boardCode 属于溯源 $boardLabel\n本单申请溯源为 ${expectedTrace.trim()}，请更换正确板码';
  }

  String _apiErrorMessage(ApiEnvelope r) {
    if (r.msg.contains('TimeoutException')) {
      return '过账请求超时，请刷新待审列表确认是否已成功；勿重复提交';
    }
    if (r.msg == 'TRACE_MISMATCH' && r.data is Map) {
      final d = Map<String, dynamic>.from(r.data as Map);
      final expected = '${d['expected_trace_code'] ?? ''}'.trim();
      final boardCode = '${d['board_code'] ?? ''}'.trim();
      final boardTrace = '${d['board_trace_code'] ?? ''}'.trim();
      if (expected.isNotEmpty && boardCode.isNotEmpty) {
        return _traceMismatchMessage(
          boardCode: boardCode,
          expectedTrace: expected,
          boardTrace: boardTrace.isEmpty ? null : boardTrace,
        );
      }
    }
    return r.msg.isEmpty ? '操作失败' : r.msg;
  }

  String _qtyExceedsMessage(ApiEnvelope r, double requestedKg) {
    if (r.msg.contains('TimeoutException')) return _apiErrorMessage(r);
    if (r.msg != 'QTY_EXCEEDS_AVAILABLE') return _apiErrorMessage(r);
    if (r.data is Map) {
      final d = Map<String, dynamic>.from(r.data as Map);
      final avail = (d['available_kg'] as num?)?.toDouble();
      if (avail != null) {
        return '板码可领 ${avail.toStringAsFixed(2)} kg，本次过账 ${requestedKg.toStringAsFixed(2)} kg 超出可用量';
      }
    }
    return '板码可用量不足，请核对库存或减少过账重量';
  }

  Future<void> _showTraceMismatchDialog(String message) async {
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('溯源不一致'),
        content: Text(message),
        actions: [FilledButton(onPressed: () => Navigator.pop(ctx), child: const Text('知道了'))],
      ),
    );
  }

  Future<bool> _validateBoardTrace(String boardCode, String expectedTrace) async {
    final expected = _normTrace(expectedTrace);
    if (expected.isEmpty) return true;
    final r = await context.read<AuthState>().api.get(
          '/inventory/box-codes/trace/${Uri.encodeComponent(boardCode)}',
        );
    if (!mounted) return false;
    if (!r.ok) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.msg.isEmpty ? '板码查询失败' : r.msg)));
      return false;
    }
    if (r.data is! Map) return false;
    final rawBoardTrace = (r.data as Map)['trace_code']?.toString();
    if (_normTrace(rawBoardTrace) != expected) {
      await _showTraceMismatchDialog(_traceMismatchMessage(
        boardCode: boardCode,
        expectedTrace: expectedTrace,
        boardTrace: rawBoardTrace,
      ));
      return false;
    }
    return true;
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    final api = context.read<AuthState>().api;
    final r0 = await api.get('/production/process-issues?biz_status=issue_pending_warehouse&scope=warehouse_queue&page_size=100');
    final r1 = await api.get('/production/process-issues?biz_status=return_pending&scope=warehouse_queue&page_size=100');
    final r2 = await api.get('/production/process-stock-ins?status=pending_warehouse&page_size=100');
    if (!mounted) return;
    setState(() {
      _loading = false;
      _issues = r0.ok ? _parseList(r0.data) : [];
      _returns = r1.ok ? _parseList(r1.data) : [];
      _stockIns = r2.ok ? _parseList(r2.data) : [];
    });
    if (!r0.ok || !r1.ok || !r2.ok) {
      final msgs = <String>[
        if (!r0.ok) '领料待审：${r0.msg}',
        if (!r1.ok) '退库待审：${r1.msg}',
        if (!r2.ok) '入库待审：${r2.msg}',
      ];
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msgs.join('\n'))));
    }
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

  Future<String?> _askBoardCode(
    String initial, {
    required String expectedTrace,
    String? farmerName,
  }) async {
    final ctrl = TextEditingController(text: initial);
    final traceHint = expectedTrace.trim();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('分配板码'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (traceHint.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: Text(
                  farmerName != null && farmerName.trim().isNotEmpty
                      ? '本单溯源：$traceHint · ${farmerName.trim()}'
                      : '本单溯源：$traceHint',
                  style: const TextStyle(color: Colors.black54, fontSize: 13),
                ),
              ),
            TraceCodeField(
              controller: ctrl,
              label: '板码',
              hint: '领料/入库过账必填，可扫码或点最近使用',
              scannerTitle: '扫描板码',
              historyKey: RecentCodeStore.board,
              compact: false,
            ),
          ],
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

  Future<void> _approveIssue(Map<String, dynamic> row) async {
    final expectedTrace = '${row['trace_code'] ?? ''}';
    final farmerName = '${row['farmer_name'] ?? ''}'.trim();
    final board = await _askBoardCode(
      '',
      expectedTrace: expectedTrace,
      farmerName: farmerName.isEmpty ? null : farmerName,
    );
    if (board == null || board.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('过账需要板码')));
      return;
    }
    if (!await _validateBoardTrace(board, expectedTrace)) return;
    final photo = await _capturePhoto();
    if (photo == null || photo.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('需拍摄仓管复磅照片')));
      return;
    }
    final kg = (row['issue_kg'] as num?)?.toDouble() ?? (row['pending_reweigh_kg'] as num?)?.toDouble() ?? 0;
    final id = (row['id'] as num).toInt();
    if (!mounted) return;
    final r = await context.read<AuthState>().api.post(
      '/production/process-issues/$id/issue-approve',
      {
        'reweigh_kg': kg,
        'photo_url': photo,
        'image_url': photo,
        'board_code': board,
        'box_code': board,
      },
      timeout: const Duration(seconds: 45),
    );
    if (!mounted) return;
    if (r.ok) {
      await RecentCodeStore.remember(RecentCodeStore.board, board);
    }
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '领料已过账' : _qtyExceedsMessage(r, kg))));
    if (r.ok) _load();
  }

  Future<void> _rejectIssue(Map<String, dynamic> row) async {
    final id = (row['id'] as num).toInt();
    final r = await context.read<AuthState>().api.post('/production/process-issues/$id/issue-reject', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已驳回' : r.msg)));
    if (r.ok) _load();
  }

  Future<void> _approveStockIn(Map<String, dynamic> row) async {
    final expectedTrace = '${row['trace_code'] ?? ''}';
    var board = '${row['board_code'] ?? ''}'.trim();
    if (board.isEmpty) {
      final asked = await _askBoardCode('', expectedTrace: expectedTrace);
      if (asked == null || asked.isEmpty) {
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('过账需要板码')));
        return;
      }
      board = asked;
    }
    if (!await _validateBoardTrace(board, expectedTrace)) return;
    final photo = await _capturePhoto();
    if (photo == null || photo.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('需拍摄仓管复磅照片')));
      return;
    }
    final kg = (row['apply_kg'] as num?)?.toDouble() ?? (row['reweigh_kg'] as num?)?.toDouble() ?? 0;
    final id = (row['id'] as num).toInt();
    if (!mounted) return;
    final r = await context.read<AuthState>().api.post(
      '/production/process-stock-ins/$id/approve',
      {
        'reweigh_kg': kg,
        'photo_url': photo,
        'image_url': photo,
        'board_code': board,
      },
      timeout: const Duration(seconds: 45),
    );
    if (!mounted) return;
    if (r.ok) {
      await RecentCodeStore.remember(RecentCodeStore.board, board);
    }
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '入库已过账' : _apiErrorMessage(r))));
    if (r.ok) _load();
  }

  String _traceFarmerLabel(Map<String, dynamic> e) {
    final trace = '${e['trace_code'] ?? ''}';
    final farmer = '${e['farmer_name'] ?? ''}'.trim();
    if (farmer.isEmpty) return trace;
    return '$trace · $farmer';
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('领退入库待审'),
        bottom: TabBar(
          controller: _tab,
          tabs: const [Tab(text: '领料待审'), Tab(text: '退库待审'), Tab(text: '入库待审')],
        ),
        actions: [IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh))],
      ),
      body: TabBarView(
        controller: _tab,
        children: [
          ListView(
            children: [
              if (_loading) const LinearProgressIndicator(),
              for (final e in _issues)
                ListTile(
                  title: Text('${_traceFarmerLabel(e)} · ${e['process_name'] ?? ''}'),
                  subtitle: Text(
                    '申请 ${e['issue_kg']} kg · ${e['worker_name']}'
                    '${(e['pending_photo_url'] ?? '').toString().isNotEmpty ? ' · 有工人复磅照' : ''}',
                  ),
                  trailing: Wrap(
                    spacing: 4,
                    children: [
                      TextButton(onPressed: () => _rejectIssue(e), child: const Text('驳回')),
                      TextButton(onPressed: () => _approveIssue(e), child: const Text('过账')),
                    ],
                  ),
                ),
              if (!_loading && _issues.isEmpty) const ListTile(title: Text('暂无领料待审')),
            ],
          ),
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
