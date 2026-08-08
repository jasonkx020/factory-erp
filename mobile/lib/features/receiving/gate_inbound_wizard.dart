import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../widgets/form_row.dart';
import '../../widgets/trace_code_field.dart';
import 'gate_inbound_prefs.dart';

typedef GateSubmitFn = Future<bool> Function({
  required String? nextRole,
  String? nextNodeId,
  int? nextAssigneeUserId,
  required bool qualified,
});

/// 过磅入厂四步向导（仅 gate）。
class GateInboundWizard extends StatefulWidget {
  const GateInboundWizard({
    super.key,
    required this.batchNo,
    required this.unitPrice,
    required this.deductRate,
    required this.reject,
    required this.freight,
    required this.loadingFee,
    required this.weighFee,
    required this.gross,
    required this.plate,
    required this.recvAddr,
    required this.remark,
    required this.farmerSearch,
    required this.partyName,
    required this.partyMobile,
    required this.origin,
    required this.batchOk,
    required this.photoUrls,
    required this.varieties,
    required this.varietyId,
    required this.channel,
    required this.coldStore,
    required this.grade,
    required this.farmerId,
    required this.farmerHits,
    required this.searchingFarmer,
    required this.msg,
    required this.onBatchChanged,
    required this.onValidateBatch,
    required this.onFarmerSearchChanged,
    required this.onSearchFarmers,
    required this.onApplyFarmer,
    required this.onClearFarmer,
    required this.onShowOnsiteFarmer,
    required this.onApplyVariety,
    required this.onChannelChanged,
    required this.onColdStoreChanged,
    required this.onGradeChanged,
    required this.onTakePhoto,
    required this.onRemovePhoto,
    required this.onSubmit,
    required this.onMsg,
  });

  final TextEditingController batchNo;
  final TextEditingController unitPrice;
  final TextEditingController deductRate;
  final TextEditingController reject;
  final TextEditingController freight;
  final TextEditingController loadingFee;
  final TextEditingController weighFee;
  final TextEditingController gross;
  final TextEditingController plate;
  final TextEditingController recvAddr;
  final TextEditingController remark;
  final TextEditingController farmerSearch;
  final TextEditingController partyName;
  final TextEditingController partyMobile;
  final TextEditingController origin;

  final bool batchOk;
  final List<String> photoUrls;
  final List<dynamic> varieties;
  final int? varietyId;
  final String channel;
  final String coldStore;
  final String grade;
  final int? farmerId;
  final List<dynamic> farmerHits;
  final bool searchingFarmer;
  final String msg;

  final ValueChanged<String> onBatchChanged;
  final Future<void> Function() onValidateBatch;
  final ValueChanged<String> onFarmerSearchChanged;
  final Future<void> Function(String) onSearchFarmers;
  final ValueChanged<Map<String, dynamic>> onApplyFarmer;
  final VoidCallback onClearFarmer;
  final Future<void> Function() onShowOnsiteFarmer;
  final ValueChanged<Map<String, dynamic>> onApplyVariety;
  final ValueChanged<String> onChannelChanged;
  final ValueChanged<String> onColdStoreChanged;
  final ValueChanged<String> onGradeChanged;
  final Future<void> Function() onTakePhoto;
  final ValueChanged<int> onRemovePhoto;
  final GateSubmitFn onSubmit;
  final ValueChanged<String> onMsg;

  @override
  State<GateInboundWizard> createState() => _GateInboundWizardState();
}

class _GateInboundWizardState extends State<GateInboundWizard> {
  final _pages = PageController();
  int _step = 0;
  bool _qualified = true;
  bool _busy = false;
  bool _prefsLoaded = false;

  List<dynamic> _options = [];
  String? _nextRole;
  String? _nextNodeId;
  int? _nextAssignee;
  bool _loadingOptions = false;

  static const _titles = ['扫码与常用项', '照片与过磅', '关联农户', '下一部门'];

  @override
  void initState() {
    super.initState();
    _loadPrefs();
  }

  @override
  void dispose() {
    _pages.dispose();
    super.dispose();
  }

  Future<void> _loadPrefs() async {
    final m = await GateInboundPrefs.load();
    if (!mounted) return;
    void apply(TextEditingController c, String key, String fallback) {
      final v = m[key]?.trim() ?? '';
      if (v.isNotEmpty) {
        c.text = v;
      } else if (c.text.isEmpty) {
        c.text = fallback;
      }
    }

    apply(widget.unitPrice, 'unit_price', '1.2');
    apply(widget.deductRate, 'deduct_rate', '5');
    apply(widget.freight, 'freight_fee', '0');
    apply(widget.loadingFee, 'loading_fee', '0');
    apply(widget.weighFee, 'weigh_fee', '0');
    final lastRole = m['next_role']?.trim() ?? '';
    setState(() {
      _prefsLoaded = true;
      if (lastRole.isNotEmpty) _nextRole = lastRole;
    });
  }

  Future<void> _loadNextOptions() async {
    if (_loadingOptions) return;
    setState(() => _loadingOptions = true);
    final r = await context.read<AuthState>().api.get(
          '/purchase/weigh-flow/next-options?receive_kind=gate&from_action=submit',
        );
    if (!mounted) return;
    final data = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : <String, dynamic>{};
    final opts = r.ok ? ApiClient.listOf(data['options']) : <dynamic>[];
    setState(() {
      _loadingOptions = false;
      _options = opts;
      if (_nextRole == null || _nextRole!.isEmpty) {
        if (opts.isNotEmpty) {
          final first = Map<String, dynamic>.from(opts.first as Map);
          _nextRole = first['role_code']?.toString();
          _nextNodeId = first['node_id']?.toString();
          final users = (first['users'] as List?) ?? [];
          if (users.isNotEmpty) {
            _nextAssignee = ((users.first as Map)['user_id'] as num?)?.toInt();
          }
        }
      } else {
        _syncAssigneeForRole(_nextRole!);
      }
      if (!r.ok) widget.onMsg('下一部门加载失败：${r.msg}');
    });
  }

  void _syncAssigneeForRole(String role) {
    Map<String, dynamic>? hit;
    for (final e in _options) {
      final m = Map<String, dynamic>.from(e as Map);
      if (m['role_code']?.toString() == role) {
        hit = m;
        break;
      }
    }
    if (hit == null) return;
    _nextNodeId = hit['node_id']?.toString();
    final users = (hit['users'] as List?) ?? [];
    final stillValid = users.any((u) => ((u as Map)['user_id'] as num?)?.toInt() == _nextAssignee);
    if (!stillValid) {
      _nextAssignee = users.isNotEmpty ? ((users.first as Map)['user_id'] as num?)?.toInt() : null;
    }
  }

  List<Map<String, dynamic>> get _usersForRole {
    final myId = context.read<AuthState>().userId;
    for (final e in _options) {
      final m = Map<String, dynamic>.from(e as Map);
      if (m['role_code']?.toString() == _nextRole) {
        return ((m['users'] as List?) ?? [])
            .map((u) => Map<String, dynamic>.from(u as Map))
            .where((u) => ((u['user_id'] as num?)?.toInt() ?? 0) != myId)
            .toList();
      }
    }
    return [];
  }

  String? _validateStep(int step) {
    switch (step) {
      case 0:
        if (!widget.batchOk) return '请先校验溯源批号';
        if (!_qualified) {
          final rj = double.tryParse(widget.reject.text) ?? 0;
          if (rj <= 0) return '不合格时请填写不合格重量';
        }
        return null;
      case 1:
        if (widget.photoUrls.isEmpty) return '请现场拍照留底';
        if ((double.tryParse(widget.gross.text) ?? 0) <= 0) return '请输入入场重量';
        if (widget.varietyId == null && widget.varieties.isNotEmpty) return '请选择品种';
        return null;
      case 2:
        if ((widget.farmerId == null || widget.farmerId! <= 0) && widget.partyName.text.trim().isEmpty) {
          return '请关联农户或现场录入姓名';
        }
        return null;
      case 3:
        if (_nextRole == null || _nextRole!.isEmpty) return '请选择下一处理部门';
        if (_nextAssignee == null || _nextAssignee! <= 0) return '请选择处理人';
        return null;
    }
    return null;
  }

  Future<void> _goNext() async {
    final err = _validateStep(_step);
    if (err != null) {
      widget.onMsg(err);
      return;
    }
    if (_step < 3) {
      if (_step + 1 == 3) await _loadNextOptions();
      setState(() => _step++);
      await _pages.animateToPage(_step, duration: const Duration(milliseconds: 250), curve: Curves.easeOut);
      return;
    }
    setState(() => _busy = true);
    final ok = await widget.onSubmit(
      nextRole: _nextRole,
      nextNodeId: (_nextNodeId != null && _nextNodeId!.isNotEmpty) ? _nextNodeId : null,
      nextAssigneeUserId: _nextAssignee,
      qualified: _qualified,
    );
    if (!mounted) return;
    setState(() => _busy = false);
    if (ok) {
      await GateInboundPrefs.save(
        unitPrice: widget.unitPrice.text,
        deductRate: widget.deductRate.text,
        freight: widget.freight.text,
        loadingFee: widget.loadingFee.text,
        weighFee: widget.weighFee.text,
        nextRole: _nextRole,
      );
      setState(() {
        _step = 0;
        _qualified = true;
      });
      _pages.jumpToPage(0);
    }
  }

  Future<void> _goPrev() async {
    if (_step <= 0) return;
    setState(() => _step--);
    await _pages.animateToPage(_step, duration: const Duration(milliseconds: 250), curve: Curves.easeOut);
  }

  @override
  Widget build(BuildContext context) {
    if (!_prefsLoaded) {
      return const Center(child: CircularProgressIndicator());
    }
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('步骤 ${_step + 1}/4 · ${_titles[_step]}', style: const TextStyle(fontWeight: FontWeight.w600)),
              const SizedBox(height: 8),
              Row(
                children: List.generate(4, (i) {
                  final active = i <= _step;
                  return Expanded(
                    child: Container(
                      height: 4,
                      margin: EdgeInsets.only(right: i < 3 ? 4 : 0),
                      decoration: BoxDecoration(
                        color: active ? Theme.of(context).colorScheme.primary : Colors.black12,
                        borderRadius: BorderRadius.circular(2),
                      ),
                    ),
                  );
                }),
              ),
            ],
          ),
        ),
        Expanded(
          child: PageView(
            controller: _pages,
            physics: const NeverScrollableScrollPhysics(),
            children: [
              _stepScanPrefs(),
              _stepWeighPhotos(),
              _stepFarmer(),
              _stepNextDept(),
            ],
          ),
        ),
        if (widget.msg.isNotEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Align(
              alignment: Alignment.centerLeft,
              child: Text(widget.msg, style: const TextStyle(color: Colors.black54, fontSize: 13)),
            ),
          ),
        SafeArea(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
            child: Row(
              children: [
                if (_step > 0)
                  OutlinedButton(onPressed: _busy ? null : _goPrev, child: const Text('上一步')),
                if (_step > 0) const SizedBox(width: 12),
                Expanded(
                  child: FilledButton(
                    onPressed: _busy ? null : _goNext,
                    child: Text(_busy ? '提交中…' : (_step < 3 ? '下一步' : '创建入厂草稿')),
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _textRow(
    String label,
    TextEditingController c, {
    String? hint,
    TextInputType? keyboardType,
    bool requiredMark = false,
  }) {
    return FormRow(
      label: label,
      requiredMark: requiredMark,
      child: TextField(
        controller: c,
        textAlign: TextAlign.right,
        keyboardType: keyboardType,
        style: const TextStyle(fontSize: 15),
        decoration: FormRow.fieldDecoration(hint: hint),
        onTap: () => FormRow.moveCursorToEnd(c),
      ),
    );
  }

  Widget _stepScanPrefs() {
    return ListView(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 16),
      children: [
        TraceCodeField(
          controller: widget.batchNo,
          label: '溯源批号',
          hint: '输入或扫码',
          validated: widget.batchOk,
          scannerTitle: '扫描溯源批号',
          compact: true,
          onChanged: widget.onBatchChanged,
          onEditingComplete: widget.onValidateBatch,
          onScanned: (_) async {
            widget.onBatchChanged(widget.batchNo.text);
            await widget.onValidateBatch();
          },
        ),
        const Padding(
          padding: EdgeInsets.fromLTRB(4, 14, 4, 6),
          child: Text('常用项（本地记忆）', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
        ),
        _textRow('单价（元）', widget.unitPrice, hint: '0', keyboardType: const TextInputType.numberWithOptions(decimal: true)),
        _textRow('扣损率（%）', widget.deductRate, hint: '0', keyboardType: const TextInputType.numberWithOptions(decimal: true)),
        FormRow(
          label: '是否合格',
          child: Align(
            alignment: Alignment.centerRight,
            child: SegmentedButton<bool>(
              style: const ButtonStyle(visualDensity: VisualDensity.compact, tapTargetSize: MaterialTapTargetSize.shrinkWrap),
              segments: const [
                ButtonSegment(value: true, label: Text('合格')),
                ButtonSegment(value: false, label: Text('不合格')),
              ],
              selected: {_qualified},
              onSelectionChanged: (s) {
                setState(() {
                  _qualified = s.first;
                  if (_qualified) widget.reject.text = '0';
                });
              },
            ),
          ),
        ),
        if (!_qualified)
          _textRow(
            '不合格重（kg）',
            widget.reject,
            hint: '0',
            requiredMark: true,
            keyboardType: const TextInputType.numberWithOptions(decimal: true),
          ),
        _textRow('运费（元）', widget.freight, hint: '0', keyboardType: const TextInputType.numberWithOptions(decimal: true)),
        _textRow('装卸费（元）', widget.loadingFee, hint: '0', keyboardType: const TextInputType.numberWithOptions(decimal: true)),
        _textRow('过磅费（元）', widget.weighFee, hint: '0', keyboardType: const TextInputType.numberWithOptions(decimal: true)),
      ],
    );
  }

  Widget _stepWeighPhotos() {
    return ListView(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 16),
      children: [
        if (widget.varieties.isEmpty)
          const Padding(
            padding: EdgeInsets.all(8),
            child: Text('暂无过磅品种，请先在后台配置', style: TextStyle(color: Colors.orange)),
          )
        else
          FormRow(
            label: '品种',
            requiredMark: true,
            child: DropdownButtonHideUnderline(
              child: DropdownButton<int>(
                isExpanded: true,
                value: widget.varietyId,
                alignment: AlignmentDirectional.centerEnd,
                hint: const Text('请选择', textAlign: TextAlign.right),
                items: widget.varieties.map((e) {
                  final m = Map<String, dynamic>.from(e as Map);
                  return DropdownMenuItem(
                    value: (m['id'] as num?)?.toInt(),
                    alignment: AlignmentDirectional.centerEnd,
                    child: Text('${m['name'] ?? m['code']}', textAlign: TextAlign.right),
                  );
                }).toList(),
                onChanged: (v) {
                  if (v == null) return;
                  final hit = widget.varieties
                      .cast<dynamic>()
                      .map((e) => Map<String, dynamic>.from(e as Map))
                      .where((m) => (m['id'] as num?)?.toInt() == v);
                  if (hit.isNotEmpty) widget.onApplyVariety(hit.first);
                },
              ),
            ),
          ),
        FormRow(
          label: '过磅方式',
          child: Align(
            alignment: Alignment.centerRight,
            child: SegmentedButton<String>(
              style: const ButtonStyle(visualDensity: VisualDensity.compact, tapTargetSize: MaterialTapTargetSize.shrinkWrap),
              segments: const [
                ButtonSegment(value: 'internal', label: Text('厂内秤')),
                ButtonSegment(value: 'external', label: Text('外磅单')),
              ],
              selected: {widget.channel},
              onSelectionChanged: (s) => widget.onChannelChanged(s.first),
            ),
          ),
        ),
        _textRow(
          '入场重量（kg）',
          widget.gross,
          hint: '0',
          requiredMark: true,
          keyboardType: const TextInputType.numberWithOptions(decimal: true),
        ),
        _textRow('车牌号', widget.plate, hint: '选填'),
        _textRow('收货地址', widget.recvAddr, hint: '选填'),
        FormRow(
          label: '目标库',
          child: DropdownButtonHideUnderline(
            child: DropdownButton<String>(
              isExpanded: true,
              value: widget.coldStore,
              alignment: AlignmentDirectional.centerEnd,
              items: const [
                DropdownMenuItem(value: 'fresh', alignment: AlignmentDirectional.centerEnd, child: Text('保鲜库', textAlign: TextAlign.right)),
                DropdownMenuItem(value: 'semi', alignment: AlignmentDirectional.centerEnd, child: Text('半成品库', textAlign: TextAlign.right)),
              ],
              onChanged: (v) => widget.onColdStoreChanged(v ?? 'fresh'),
            ),
          ),
        ),
        FormRow(
          label: '等级',
          child: DropdownButtonHideUnderline(
            child: DropdownButton<String>(
              isExpanded: true,
              value: widget.grade,
              alignment: AlignmentDirectional.centerEnd,
              items: const [
                DropdownMenuItem(value: 'A', alignment: AlignmentDirectional.centerEnd, child: Text('A', textAlign: TextAlign.right)),
                DropdownMenuItem(value: 'B', alignment: AlignmentDirectional.centerEnd, child: Text('B', textAlign: TextAlign.right)),
                DropdownMenuItem(value: 'C', alignment: AlignmentDirectional.centerEnd, child: Text('C', textAlign: TextAlign.right)),
              ],
              onChanged: (v) => widget.onGradeChanged(v ?? 'A'),
            ),
          ),
        ),
        FormRow(
          label: '现场照片',
          requiredMark: true,
          child: Row(
            mainAxisAlignment: MainAxisAlignment.end,
            children: [
              Text('已拍 ${widget.photoUrls.length}/3', style: const TextStyle(color: Colors.black54, fontSize: 13)),
              const SizedBox(width: 8),
              TextButton.icon(
                onPressed: widget.onTakePhoto,
                icon: const Icon(Icons.photo_camera, size: 18),
                label: const Text('拍照'),
              ),
            ],
          ),
        ),
        if (widget.photoUrls.isNotEmpty)
          Padding(
            padding: const EdgeInsets.only(left: 108, top: 4, bottom: 4),
            child: Wrap(
              spacing: 8,
              alignment: WrapAlignment.end,
              children: [
                for (var i = 0; i < widget.photoUrls.length; i++)
                  Chip(label: Text('图${i + 1}'), onDeleted: () => widget.onRemovePhoto(i), visualDensity: VisualDensity.compact),
              ],
            ),
          ),
        _textRow('备注', widget.remark, hint: '选填'),
      ],
    );
  }

  Widget _stepFarmer() {
    return ListView(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 16),
      children: [
        FormRow(
          label: '搜索农户',
          child: TextField(
            controller: widget.farmerSearch,
            textAlign: TextAlign.right,
            style: const TextStyle(fontSize: 15),
            decoration: FormRow.fieldDecoration(
              hint: '手机号/姓名',
              suffixIcon: widget.searchingFarmer
                  ? const Padding(
                      padding: EdgeInsets.all(10),
                      child: SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2)),
                    )
                  : IconButton(
                      icon: const Icon(Icons.search, size: 20),
                      onPressed: () => widget.onSearchFarmers(widget.farmerSearch.text),
                    ),
            ),
            onTap: () => FormRow.moveCursorToEnd(widget.farmerSearch),
            onChanged: widget.onFarmerSearchChanged,
            onSubmitted: widget.onSearchFarmers,
          ),
        ),
        if (widget.farmerHits.isNotEmpty)
          Card(
            margin: const EdgeInsets.only(top: 6, bottom: 6),
            child: Column(
              children: [
                for (final e in widget.farmerHits.take(8))
                  ListTile(
                    dense: true,
                    title: Text('${(e as Map)['name'] ?? ''}'),
                    subtitle: Text('${e['mobile'] ?? ''} · ${e['origin'] ?? ''}'),
                    trailing: const Icon(Icons.check_circle_outline),
                    onTap: () => widget.onApplyFarmer(Map<String, dynamic>.from(e)),
                  ),
              ],
            ),
          ),
        if (widget.farmerSearch.text.trim().isNotEmpty && !widget.searchingFarmer && widget.farmerHits.isEmpty && widget.farmerId == null)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 6),
            child: Row(
              children: [
                const Expanded(child: Text('未找到匹配农户', style: TextStyle(color: Colors.orange))),
                FilledButton.tonal(onPressed: widget.onShowOnsiteFarmer, child: const Text('现场录入')),
              ],
            ),
          ),
        if (widget.farmerId != null)
          FormRow(
            label: '已关联',
            child: Align(
              alignment: Alignment.centerRight,
              child: InputChip(
                avatar: const Icon(Icons.link, size: 16),
                label: Text('#${widget.farmerId} ${widget.partyName.text}'),
                onDeleted: widget.onClearFarmer,
              ),
            ),
          ),
        _textRow('姓名', widget.partyName, hint: '可改快照', requiredMark: true),
        _textRow('电话', widget.partyMobile, hint: '选填', keyboardType: TextInputType.phone),
        _textRow('产地', widget.origin, hint: '选填'),
        Align(
          alignment: Alignment.centerRight,
          child: TextButton.icon(
            onPressed: widget.onShowOnsiteFarmer,
            icon: const Icon(Icons.person_add_alt),
            label: const Text('现场新建农户'),
          ),
        ),
      ],
    );
  }

  Widget _stepNextDept() {
    if (_loadingOptions) {
      return const Center(child: CircularProgressIndicator());
    }
    final users = _usersForRole;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        const Text('选择下一处理部门', style: TextStyle(fontWeight: FontWeight.w600)),
        const SizedBox(height: 8),
        if (_options.isEmpty)
          const Text('暂无部门选项，请检查流程图或角色用户', style: TextStyle(color: Colors.orange))
        else
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final e in _options)
                Builder(builder: (_) {
                  final m = Map<String, dynamic>.from(e as Map);
                  final code = m['role_code']?.toString() ?? '';
                  final name = m['role_name']?.toString() ?? code;
                  final selected = _nextRole == code;
                  return ChoiceChip(
                    label: Text(name),
                    selected: selected,
                    onSelected: (_) => setState(() {
                      _nextRole = code;
                      _syncAssigneeForRole(code);
                    }),
                  );
                }),
            ],
          ),
        const SizedBox(height: 16),
        const Text('处理人', style: TextStyle(fontWeight: FontWeight.w600)),
        const SizedBox(height: 8),
        if (users.isEmpty)
          const Text('该部门暂无可指派用户', style: TextStyle(color: Colors.orange))
        else
          ...users.map((u) {
            final uid = (u['user_id'] as num?)?.toInt();
            final selected = uid != null && uid == _nextAssignee;
            return ListTile(
              leading: Icon(selected ? Icons.radio_button_checked : Icons.radio_button_off),
              title: Text('${u['name'] ?? u['login_name'] ?? uid}'),
              subtitle: Text('${u['login_name'] ?? ''}'),
              selected: selected,
              onTap: uid == null ? null : () => setState(() => _nextAssignee = uid),
            );
          }),
        const SizedBox(height: 12),
        const Text('确认后将创建入厂草稿，并指派协作工单给所选处理人。', style: TextStyle(fontSize: 12, color: Colors.black54)),
      ],
    );
  }
}
