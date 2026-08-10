import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/auth_state.dart';
import '../../widgets/form_row.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_sticky_actions.dart';
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
  const StationPassPage({super.key, this.asTab = false});

  /// 作为产线壳 Tab 时隐藏标题栏，把高度留给表单。
  final bool asTab;

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
    final bottomInset = MediaQuery.viewInsetsOf(context).bottom;
    final topPad = widget.asTab ? 40.0 : 8.0;
    return Scaffold(
      appBar: widget.asTab
          ? null
          : AppBar(
              title: Text(_isCheckpoint ? '工序过站 · 卡点复核' : '工序过站'),
            ),
      body: Column(
        children: [
          Expanded(
            child: ListView(
              keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
              padding: EdgeInsets.fromLTRB(12, topPad, 12, 16 + bottomInset),
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
                const FormSectionHeader('扫码过站'),
                FormRow.text(label: '工牌码', controller: _badge, requiredMark: true),
                FormRow.text(label: '箱码', controller: _box, requiredMark: true),
                FormRow.text(label: '投料重(kg)', controller: _in, keyboardType: TextInputType.number),
                FormRow.text(label: '完工重(kg)', controller: _out, keyboardType: TextInputType.number, requiredMark: true),
                FormRow.text(label: '袋数', controller: _bag, keyboardType: TextInputType.number, hint: '装袋工序'),
                const FormSectionHeader('次品类型'),
                FormRow(
                  label: '次品',
                  child: Wrap(
                    alignment: WrapAlignment.end,
                    spacing: 6,
                    runSpacing: 4,
                    children: _scrapOptions
                        .map((e) => ChoiceChip(
                              label: Text(e.value, style: const TextStyle(fontSize: 12)),
                              selected: _scrapType == e.key,
                              visualDensity: VisualDensity.compact,
                              onSelected: (_) => setState(() => _scrapType = e.key),
                            ))
                        .toList(),
                  ),
                ),
                if (_msg.isNotEmpty)
                  Padding(padding: const EdgeInsets.only(top: 12), child: Text(_msg)),
                if (_last != null && _last!['utilization'] != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: Text(
                      '利用率 ${(_last!['utilization'] as num?)?.toStringAsFixed(3) ?? '-'} · 损耗 ${_last!['loss'] ?? '-'} kg',
                      style: const TextStyle(color: Colors.black54, fontSize: 13),
                    ),
                  ),
              ],
            ),
          ),
          FormStickyButtonBar(
            children: [
              OutlinedButton(onPressed: () => _scan(resolveOnly: true), child: const Text('预览')),
              FilledButton(onPressed: () => _scan(resolveOnly: false), child: const Text('提交草稿')),
            ],
          ),
          if (_pendingReportId != null)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  FilledButton(onPressed: () => _confirm(qcPass: true), child: const Text('确认过站（QC 合格）')),
                  OutlinedButton(
                    onPressed: () async {
                      final id = _pendingReportId ?? (_last?['id'] as num?)?.toInt();
                      if (id == null) {
                        setState(() => _msg = '无待作废草稿');
                        return;
                      }
                      final api = context.read<AuthState>().api;
                      final r = await api.post('/production/report-works/$id/void', {'remark': 'void_draft'});
                      setState(() {
                        if (r.ok) {
                          _pendingReportId = null;
                          _msg = '草稿已作废';
                        } else {
                          _msg = r.msg;
                        }
                      });
                    },
                    child: const Text('作废草稿'),
                  ),
                  if (_isCheckpoint)
                    OutlinedButton(
                      onPressed: () => _confirm(qcPass: false),
                      child: const Text('QC 不合格（阻断）'),
                    ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}
