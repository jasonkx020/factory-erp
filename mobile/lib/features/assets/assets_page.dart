import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';

/// 固定资产查询 + 内部转移申请
class AssetsPage extends StatefulWidget {
  const AssetsPage({super.key});

  @override
  State<AssetsPage> createState() => _AssetsPageState();
}

class _AssetsPageState extends State<AssetsPage> with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  List<dynamic> _assets = [];
  List<dynamic> _transfers = [];
  int? _assetId;
  final _toDept = TextEditingController(text: '维修间');
  final _toLoc = TextEditingController(text: '维修间');
  final _remark = TextEditingController();
  String _msg = '';
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 2, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) => _boot());
  }

  @override
  void dispose() {
    _tabs.dispose();
    _toDept.dispose();
    _toLoc.dispose();
    _remark.dispose();
    super.dispose();
  }

  Future<void> _boot() async {
    final auth = context.read<AuthState>();
    if (!canAccessEmployeeModule(EmployeeModule.assets, auth.permissions, auth.roles)) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('无固定资产权限')));
        Navigator.of(context).pop();
      }
      return;
    }
    await _refresh();
  }

  Future<void> _refresh() async {
    setState(() => _loading = true);
    final api = context.read<AuthState>().api;
    final results = await Future.wait([
      api.get('/asset/fixed-assets?page_size=100'),
      api.get('/asset/transfers?page_size=50'),
    ]);
    if (!mounted) return;
    setState(() {
      _loading = false;
      _assets = ApiClient.listOf(results[0].data);
      _transfers = ApiClient.listOf(results[1].data);
      if (_assetId == null && _assets.isNotEmpty) {
        _assetId = (_assets.first as Map)['id'] is num ? ((_assets.first as Map)['id'] as num).toInt() : null;
      }
    });
  }

  Future<void> _createTransfer() async {
    if (_assetId == null) {
      setState(() => _msg = '请选择资产');
      return;
    }
    final r = await context.read<AuthState>().api.post('/asset/transfers', {
      'asset_id': _assetId,
      'to_dept_name': _toDept.text.trim(),
      'to_location': _toLoc.text.trim(),
      'remark': _remark.text.trim().isEmpty ? '手机端转移申请' : _remark.text.trim(),
    });
    setState(() => _msg = r.ok ? '转移申请已提交（草稿）' : r.msg);
    if (r.ok) {
      _tabs.animateTo(1);
      await _refresh();
    }
  }

  Future<void> _confirmTransfer(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/asset/transfers/$id/confirm', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '转移已确认' : r.msg)));
    if (r.ok) await _refresh();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('固定资产'),
        bottom: TabBar(controller: _tabs, tabs: const [
          Tab(text: '资产查询'),
          Tab(text: '转移申请'),
        ]),
        actions: [IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh))],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tabs,
              children: [
                ListView(
                  padding: const EdgeInsets.all(12),
                  children: [
                    if (_assets.isEmpty) const Padding(padding: EdgeInsets.all(24), child: Center(child: Text('暂无资产'))),
                    ..._assets.map((e) {
                      final m = Map<String, dynamic>.from(e as Map);
                      return Card(
                        child: ListTile(
                          title: Text('${m['code'] ?? ''} ${m['name'] ?? ''}'),
                          subtitle: Text('${m['dept_name'] ?? ''} · ${m['location_text'] ?? ''}\n净值 ¥${m['net_value'] ?? m['original_value'] ?? 0}'),
                          isThreeLine: true,
                          trailing: Text('${m['status'] ?? ''}'),
                          onTap: () {
                            setState(() => _assetId = (m['id'] as num?)?.toInt());
                            _tabs.animateTo(1);
                          },
                        ),
                      );
                    }),
                  ],
                ),
                ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    DropdownButtonFormField<int>(
                      value: _assetId,
                      decoration: const InputDecoration(labelText: '资产'),
                      items: _assets.map((e) {
                        final m = Map<String, dynamic>.from(e as Map);
                        return DropdownMenuItem(
                          value: (m['id'] as num?)?.toInt(),
                          child: Text('${m['code'] ?? ''} ${m['name'] ?? m['id']}'),
                        );
                      }).toList(),
                      onChanged: (v) => setState(() => _assetId = v),
                    ),
                    TextField(controller: _toDept, decoration: const InputDecoration(labelText: '调入部门')),
                    TextField(controller: _toLoc, decoration: const InputDecoration(labelText: '调入位置')),
                    TextField(controller: _remark, decoration: const InputDecoration(labelText: '备注')),
                    const SizedBox(height: 8),
                    FilledButton(onPressed: _createTransfer, child: const Text('提交转移申请')),
                    if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_msg)),
                    const Divider(),
                    ..._transfers.map((e) {
                      final m = Map<String, dynamic>.from(e as Map);
                      final st = m['status']?.toString() ?? '';
                      return Card(
                        child: ListTile(
                          title: Text('${m['doc_no']} · ${m['asset_name'] ?? m['asset_code'] ?? ''}'),
                          subtitle: Text('${m['from_location'] ?? ''} → ${m['to_location'] ?? m['to_dept_name'] ?? ''}\n$st'),
                          isThreeLine: true,
                          trailing: st == 'draft'
                              ? FilledButton.tonal(onPressed: () => _confirmTransfer(m), child: const Text('确认'))
                              : null,
                        ),
                      );
                    }),
                  ],
                ),
              ],
            ),
    );
  }
}
