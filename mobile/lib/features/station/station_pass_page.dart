import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/notify_service.dart';

const _scrapOptions = <MapEntry<String, String>>[
  MapEntry('', '无次品'),
  MapEntry('cut_defect', '切断次品'),
  MapEntry('core_defect', '去芯次品'),
  MapEntry('dice_defect', '切块次品'),
  MapEntry('sieve_bag_defect', '过筛装袋次品'),
];

/// 工序过站：扫工牌 + 扫箱 + 投料/完工 + 确认（合并原 worker/workshop 双扫）。
class StationPassPage extends StatefulWidget {
  const StationPassPage({super.key});

  @override
  State<StationPassPage> createState() => _StationPassPageState();
}

class _StationPassPageState extends State<StationPassPage> {
  final _badge = TextEditingController();
  final _box = TextEditingController();
  final _in = TextEditingController();
  final _out = TextEditingController();
  final _bag = TextEditingController(text: '0');
  String _scrapType = '';
  String _msg = '';
  Map<String, dynamic>? _last;
  int? _pendingReportId;
  bool _isCheckpoint = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      final prefs = await SharedPreferences.getInstance();
      if (!mounted) return;
      _badge.text = prefs.getString('erp.worker.badge') ?? '';
      context.read<NotifyService>().addListener(_onNotify);
    });
  }

  @override
  void dispose() {
    try {
      context.read<NotifyService>().removeListener(_onNotify);
    } catch (_) {}
    _badge.dispose();
    _box.dispose();
    _in.dispose();
    _out.dispose();
    _bag.dispose();
    super.dispose();
  }

  void _onNotify() {
    if (!mounted) return;
    final notify = context.read<NotifyService>();
    for (final raw in notify.inbox) {
      if (raw is! Map) continue;
      final p = NotifyService.parsePayload(raw['payload'] ?? raw['payload_json']);
      final next = p['next'] is Map ? Map<String, dynamic>.from(p['next'] as Map) : p;
      final code = next['new_box_code'];
      if (code != null && _box.text.trim().isEmpty) {
        setState(() => _box.text = code.toString());
        break;
      }
    }
  }

  Future<void> _scan({required bool resolveOnly}) async {
    final prefs = await SharedPreferences.getInstance();
    if (_badge.text.trim().isNotEmpty) {
      await prefs.setString('erp.worker.badge', _badge.text.trim());
    }
    if (!mounted) return;
    final api = context.read<AuthState>().api;
    final path = resolveOnly ? '/production/scan/resolve' : '/production/scan';
    final outW = double.tryParse(_out.text) ?? 0;
    final r = await api.post(path, {
      'badge_code': _badge.text.trim(),
      'box_code': _box.text.trim(),
      'input_weight': double.tryParse(_in.text) ?? 0,
      'output_weight': outW,
      'net_weight': outW,
    });
    setState(() {
      if (r.ok) {
        final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
        _last = data;
        _isCheckpoint = data['is_inbound_checkpoint'] == true;
        if (resolveOnly) {
          _msg = '已解析 ${data['worker_name'] ?? ''} · ${data['step_name'] ?? ''}';
          if (data['input_weight'] != null && _in.text.isEmpty) {
            _in.text = '${data['input_weight']}';
          }
          if (data['output_weight'] != null && _out.text.isEmpty) {
            _out.text = '${data['output_weight']}';
          }
        } else if (data['needs_confirm'] == true || data['status'] == 'confirm_pending') {
          _pendingReportId = (data['id'] as num?)?.toInt();
          _msg = '草稿已建，请确认过站';
        } else {
          _msg = '过站成功 工钱¥${data['wage_amount'] ?? 0}';
          _pendingReportId = null;
        }
        final next = data['next'];
        if (next is Map && next['new_box_code'] != null) {
          _box.text = next['new_box_code'].toString();
        }
      } else {
        _msg = r.msg;
      }
    });
  }

  Future<void> _confirm({required bool qcPass}) async {
    final id = _pendingReportId ?? (_last?['id'] as num?)?.toInt();
    if (id == null) {
      setState(() => _msg = '请先提交过站草稿');
      return;
    }
    final api = context.read<AuthState>().api;
    final outW = double.tryParse(_out.text) ?? 0;
    final body = <String, dynamic>{
      'input_weight': double.tryParse(_in.text) ?? 0,
      'output_weight': outW,
      'process_qc_result': qcPass ? 'pass' : 'fail',
      'bag_qty': double.tryParse(_bag.text) ?? 0,
    };
    if (_scrapType.isNotEmpty) body['scrap_type'] = _scrapType;
    final r = await api.post('/production/report-works/$id/confirm', body);
    setState(() {
      if (r.ok) {
        final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
        _last = data;
        _pendingReportId = null;
        _msg = qcPass ? '已过站确认 工钱¥${data['wage_amount'] ?? 0}' : 'QC 未通过，未过账';
        final next = data['next'];
        if (next is Map && next['new_box_code'] != null) {
          _box.text = next['new_box_code'].toString();
        }
      } else {
        _msg = r.msg;
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(_isCheckpoint ? '工序过站 · 卡点复核' : '工序过站'),
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          if (_isCheckpoint)
            Card(
              color: Colors.amber.shade50,
              child: const ListTile(
                leading: Icon(Icons.verified_user),
                title: Text('收货卡点模式'),
                subtitle: Text('复核重量与外观，QC 不合格将阻断过账'),
              ),
            ),
          TextField(controller: _badge, decoration: const InputDecoration(labelText: '工牌码')),
          TextField(controller: _box, decoration: const InputDecoration(labelText: '箱码')),
          TextField(
            controller: _in,
            decoration: const InputDecoration(labelText: '投料重 (kg)'),
            keyboardType: TextInputType.number,
          ),
          TextField(
            controller: _out,
            decoration: const InputDecoration(labelText: '完工重 (kg)'),
            keyboardType: TextInputType.number,
          ),
          TextField(
            controller: _bag,
            decoration: const InputDecoration(labelText: '袋数（装袋工序）'),
            keyboardType: TextInputType.number,
          ),
          const SizedBox(height: 8),
          const Text('次品类型', style: TextStyle(fontSize: 12, color: Colors.black54)),
          Wrap(
            spacing: 8,
            children: _scrapOptions
                .map((e) => ChoiceChip(
                      label: Text(e.value),
                      selected: _scrapType == e.key,
                      onSelected: (_) => setState(() => _scrapType = e.key),
                    ))
                .toList(),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(child: OutlinedButton(onPressed: () => _scan(resolveOnly: true), child: const Text('预览'))),
              const SizedBox(width: 8),
              Expanded(child: FilledButton(onPressed: () => _scan(resolveOnly: false), child: const Text('提交草稿'))),
            ],
          ),
          if (_pendingReportId != null) ...[
            const SizedBox(height: 8),
            FilledButton(onPressed: () => _confirm(qcPass: true), child: const Text('确认过站（QC 合格）')),
            if (_isCheckpoint)
              OutlinedButton(
                onPressed: () => _confirm(qcPass: false),
                child: const Text('QC 不合格（阻断）'),
              ),
          ],
          if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 12), child: Text(_msg)),
          if (_last != null && _last!['utilization'] != null)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(
                '利用率 ${(_last!['utilization'] as num?)?.toStringAsFixed(3) ?? '-'} · 损耗 ${_last!['loss'] ?? '-'} kg',
                style: const TextStyle(color: Colors.black54),
              ),
            ),
        ],
      ),
    );
  }
}
