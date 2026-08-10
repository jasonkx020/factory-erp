import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/auth_state.dart';
import '../../widgets/form_row.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_sticky_actions.dart';
import '../../widgets/hub_entry_tile.dart';

/// 工序退库：领出未用完还仓（班组长审批 → 仓管确认；不回冲已确认计件）。
class ProcessReturnPage extends StatefulWidget {
  const ProcessReturnPage({super.key, this.asTab = false, this.warehouseMode = false});

  final bool asTab;
  /// 仓管端：优先展示待确认列表。
  final bool warehouseMode;

  @override
  State<ProcessReturnPage> createState() => _ProcessReturnPageState();
}

class _ProcessReturnPageState extends State<ProcessReturnPage> {
  final _box = TextEditingController();
  final _weight = TextEditingController(text: '30');
  final _reason = TextEditingController(text: '提前下班');
  final _transferTo = TextEditingController();
  List<dynamic> _list = [];
  String? _error;
  bool _loading = false;
  String _msg = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  @override
  void dispose() {
    _box.dispose();
    _weight.dispose();
    _reason.dispose();
    _transferTo.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    final api = context.read<AuthState>().api;
    final qs = widget.warehouseMode ? 'status=pending_warehouse' : '';
    final r = await api.get('/production/process-returns${qs.isEmpty ? '' : '?$qs'}');
    if (!mounted) return;
    setState(() {
      _loading = false;
      if (r.ok) {
        final data = r.data;
        if (data is Map && data['list'] is List) {
          _list = data['list'] as List;
        } else if (data is List) {
          _list = data;
        } else {
          _list = [];
        }
      } else {
        _error = r.msg;
      }
    });
  }

  Future<void> _create() async {
    final api = context.read<AuthState>().api;
    final r = await api.post('/production/process-returns', {
      'box_code': _box.text.trim(),
      'return_weight': double.tryParse(_weight.text) ?? 0,
      'reason': _reason.text.trim(),
      'warehouse_id': 1,
    });
    if (!mounted) return;
    setState(() => _msg = r.ok ? '已创建 ${(r.data is Map ? r.data['doc_no'] : '')}' : r.msg);
    if (r.ok) await _load();
  }

  Future<void> _act(int id, String path, [Map<String, dynamic>? body]) async {
    final api = context.read<AuthState>().api;
    final r = await api.post('/production/process-returns/$id/$path', body ?? {});
    if (!mounted) return;
    setState(() => _msg = r.ok ? '已处理' : r.msg);
    if (r.ok) await _load();
  }

  @override
  Widget build(BuildContext context) {
    final topPad = widget.asTab ? 40.0 : 8.0;
    return Scaffold(
      appBar: widget.asTab ? null : AppBar(title: Text(widget.warehouseMode ? '退库确认' : '剩余料退库')),
      body: Column(
        children: [
          Expanded(
            child: ListView(
              padding: EdgeInsets.fromLTRB(12, topPad, 12, 16),
              children: [
                if (!widget.warehouseMode) ...[
                  const FormSectionHeader('申请退未用完料'),
                  FormRow.text(label: '箱码', controller: _box, requiredMark: true),
                  FormRow.text(label: '退回kg', controller: _weight, keyboardType: TextInputType.number, requiredMark: true),
                  FormRow.text(label: '原因', controller: _reason),
                  FormStickyButtonBar(
                    children: [
                      FilledButton(onPressed: _create, child: const Text('新建并保存')),
                    ],
                  ),
                ],
                const FormSectionHeader('退库单'),
                FormRow.text(label: '转交用户ID', controller: _transferTo, keyboardType: TextInputType.number),
                if (_loading) const Center(child: Padding(padding: EdgeInsets.all(24), child: CircularProgressIndicator())),
                if (_error != null) Text(_error!, style: const TextStyle(color: Colors.red)),
                if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(bottom: 8), child: Text(_msg)),
                ..._list.map((raw) {
                  final m = raw is Map ? Map<String, dynamic>.from(raw) : <String, dynamic>{};
                  final id = (m['id'] as num?)?.toInt() ?? 0;
                  final st = '${m['status'] ?? ''}';
                  return Card(
                    child: ListTile(
                      title: Text('${m['doc_no'] ?? id} · ${m['box_code'] ?? ''}'),
                      subtitle: Text('${m['return_weight'] ?? ''} kg · $st · ${m['reason'] ?? ''}'),
                      isThreeLine: true,
                      trailing: Wrap(
                        spacing: 4,
                        children: [
                          if (st == 'draft')
                            TextButton(onPressed: () => _act(id, 'submit'), child: const Text('提交')),
                          if (st == 'pending_foreman')
                            TextButton(onPressed: () => _act(id, 'approve'), child: const Text('班组通过')),
                          if (st == 'pending_warehouse')
                            TextButton(onPressed: () => _act(id, 'warehouse-confirm'), child: const Text('仓管确认')),
                          if (st == 'pending_foreman' || st == 'pending_warehouse')
                            TextButton(
                              onPressed: () {
                                final to = int.tryParse(_transferTo.text.trim()) ?? 0;
                                if (to <= 0) {
                                  setState(() => _msg = '请填写转交用户ID');
                                  return;
                                }
                                _act(id, 'transfer', {'to_user_id': to});
                              },
                              child: const Text('转交'),
                            ),
                        ],
                      ),
                    ),
                  );
                }),
                if (!_loading && _list.isEmpty)
                  const Padding(
                    padding: EdgeInsets.all(24),
                    child: Text('暂无退库单', textAlign: TextAlign.center),
                  ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
            child: OutlinedButton(onPressed: _load, child: const Text('刷新')),
          ),
        ],
      ),
    );
  }
}

/// 班组 hub 入口卡片（也可单独 push）。
class ProcessReturnHubTile extends StatelessWidget {
  const ProcessReturnHubTile({super.key, this.warehouseMode = false});

  final bool warehouseMode;

  @override
  Widget build(BuildContext context) {
    return HubEntryTile(
      icon: Icons.undo_outlined,
      title: warehouseMode ? '退库确认' : '剩余料退库',
      subtitle: warehouseMode ? '仓管确认未用完还仓' : '提前下班退未用完料',
      onTap: () {
        Navigator.of(context).push(
          MaterialPageRoute(
            builder: (_) => ProcessReturnPage(warehouseMode: warehouseMode),
          ),
        );
      },
    );
  }
}
