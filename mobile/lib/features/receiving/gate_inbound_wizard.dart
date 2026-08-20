import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../widgets/form_row.dart';
import '../../widgets/trace_code_field.dart';
import 'gate_inbound_prefs.dart';

Future<void> _showPhotoPreview(BuildContext context, String url) async {
  await showDialog<void>(
    context: context,
    barrierColor: Colors.black87,
    builder: (ctx) => Dialog(
      backgroundColor: Colors.transparent,
      insetPadding: const EdgeInsets.all(12),
      child: Stack(
        children: [
          Center(
            child: InteractiveViewer(
              minScale: 0.8,
              maxScale: 4,
              child: Image.network(
                url,
                fit: BoxFit.contain,
                errorBuilder: (_, __, ___) => const Padding(
                  padding: EdgeInsets.all(24),
                  child: Text('图片加载失败', style: TextStyle(color: Colors.white)),
                ),
              ),
            ),
          ),
          Positioned(
            top: 0,
            right: 0,
            child: IconButton(
              onPressed: () => Navigator.of(ctx).pop(),
              icon: const Icon(Icons.close, color: Colors.white),
            ),
          ),
        ],
      ),
    ),
  );
}

typedef GateSubmitFn = Future<bool> Function({
  required String? nextRole,
  String? nextNodeId,
  int? nextAssigneeUserId,
});

/// 过磅入厂四步向导（仅 gate）：农户 → 溯源码 → 过磅照片 → 预览确认。
class GateInboundWizard extends StatefulWidget {
  const GateInboundWizard({
    super.key,
    required this.batchNo,
    required this.unitPrice,
    required this.deductRate,
    required this.freight,
    required this.loadingFee,
    required this.weighFee,
    required this.gross,
    required this.plate,
    required this.recvAddr,
    required this.remark,
    required this.partyName,
    required this.partyMobile,
    required this.origin,
    required this.batchOk,
    this.bindingLocked = false,
    required this.photoMaterial,
    required this.photoScale,
    required this.photoCloseup,
    required this.varieties,
    required this.varietyId,
    required this.channel,
    required this.coldStore,
    required this.grade,
    required this.farmerId,
    required this.farmerHits,
    required this.searchingFarmer,
    required this.msg,
    this.msgIsError = false,
    required this.onBatchChanged,
    required this.onValidateBatch,
    required this.onNameChanged,
    required this.onMobileChanged,
    required this.onApplyFarmer,
    required this.onClearFarmer,
    required this.onTapManualTrace,
    required this.onGenerateTraceCode,
    required this.onShowGeneratedQr,
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
  final TextEditingController freight;
  final TextEditingController loadingFee;
  final TextEditingController weighFee;
  final TextEditingController gross;
  final TextEditingController plate;
  final TextEditingController recvAddr;
  final TextEditingController remark;
  final TextEditingController partyName;
  final TextEditingController partyMobile;
  final TextEditingController origin;

  final bool batchOk;
  /// 溯源码已过站中：农户/品种锁定为首单关联信息
  final bool bindingLocked;
  final String? photoMaterial;
  final String? photoScale;
  final String? photoCloseup;
  final List<dynamic> varieties;
  final int? varietyId;
  final String channel;
  final String coldStore;
  final String grade;
  final int? farmerId;
  final List<dynamic> farmerHits;
  final bool searchingFarmer;
  final String msg;
  final bool msgIsError;

  final ValueChanged<String> onBatchChanged;
  final Future<bool> Function() onValidateBatch;
  final ValueChanged<String> onNameChanged;
  final ValueChanged<String> onMobileChanged;
  final Future<void> Function(Map<String, dynamic>) onApplyFarmer;
  final VoidCallback onClearFarmer;
  final Future<void> Function() onTapManualTrace;
  final Future<bool> Function() onGenerateTraceCode;
  final Future<void> Function() onShowGeneratedQr;
  final ValueChanged<Map<String, dynamic>> onApplyVariety;
  final ValueChanged<String> onChannelChanged;
  final ValueChanged<String> onColdStoreChanged;
  final ValueChanged<String> onGradeChanged;
  final Future<void> Function(String slot) onTakePhoto;
  final ValueChanged<String> onRemovePhoto;
  final GateSubmitFn onSubmit;
  final ValueChanged<String> onMsg;

  @override
  State<GateInboundWizard> createState() => _GateInboundWizardState();
}

class _GateInboundWizardState extends State<GateInboundWizard> {
  final _pages = PageController();
  int _step = 0;
  bool _busy = false;
  bool _prefsLoaded = false;

  List<dynamic> _options = [];
  String? _nextRole;
  String? _nextNodeId;
  int? _nextAssignee;
  bool _loadingOptions = false;

  static const _titles = ['农户与溯源', '费用确认', '照片与过磅', '预览确认'];

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
    apply(widget.deductRate, 'deduct_rate', '0');
    apply(widget.freight, 'freight_fee', '0');
    apply(widget.loadingFee, 'loading_fee', '0');
    apply(widget.weighFee, 'weigh_fee', '0');
    final lastRole = m['next_role']?.trim() ?? '';
    setState(() {
      _prefsLoaded = true;
      if (lastRole.isNotEmpty) _nextRole = lastRole;
    });
  }

  int _myUserId() {
    final auth = context.read<AuthState>();
    if (auth.userId <= 0) auth.syncUserIdFromToken();
    return auth.userId;
  }

  Future<void> _loadNextOptions() async {
    if (_loadingOptions) return;
    setState(() => _loadingOptions = true);
    final r = await context.read<AuthState>().api.get(
          '/purchase/weigh-flow/next-options?receive_kind=gate&from_action=submit',
        );
    if (!mounted) return;
    final myId = _myUserId();
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
          _nextAssignee = _firstAssigneeExcluding(first['users'], myId);
        }
      } else {
        _syncAssigneeForRole(_nextRole!);
      }
      if (!r.ok) widget.onMsg('下一部门加载失败：${r.msg}');
    });
  }

  int? _firstAssigneeExcluding(dynamic usersRaw, int myId) {
    final users = (usersRaw as List?) ?? [];
    for (final u in users) {
      final uid = ((u as Map)['user_id'] as num?)?.toInt() ?? 0;
      if (uid <= 0) continue;
      if (myId > 0 && uid == myId) continue;
      return uid;
    }
    return null;
  }

  void _syncAssigneeForRole(String role) {
    final myId = _myUserId();
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
    final stillValid = users.any((u) {
      final uid = ((u as Map)['user_id'] as num?)?.toInt() ?? 0;
      if (uid <= 0 || uid != _nextAssignee) return false;
      if (myId > 0 && uid == myId) return false;
      return true;
    });
    if (!stillValid) {
      _nextAssignee = _firstAssigneeExcluding(users, myId);
    }
  }

  List<Map<String, dynamic>> get _usersForRole {
    final myId = _myUserId();
    for (final e in _options) {
      final m = Map<String, dynamic>.from(e as Map);
      if (m['role_code']?.toString() == _nextRole) {
        return ((m['users'] as List?) ?? [])
            .map((u) => Map<String, dynamic>.from(u as Map))
            .where((u) {
              final uid = (u['user_id'] as num?)?.toInt() ?? 0;
              if (uid <= 0) return false;
              if (myId > 0 && uid == myId) return false;
              return true;
            })
            .toList();
      }
    }
    return [];
  }

  String? _validateStep(int step) {
    switch (step) {
      case 0:
        // 溯源码与农户/品种强关联：如果溯源码已校验通过，则允许跳过手工填写农户姓名
        if (widget.partyName.text.trim().isEmpty && !widget.batchOk) return '请填写农户姓名';
        if (widget.varieties.isEmpty) return '暂无过磅品种，请先在后台配置';
        if (widget.varietyId == null) return '请选择品种';
        return null;
      case 1:
        if (widget.varieties.isEmpty) return '暂无过磅品种，请先在后台配置';
        if (widget.varietyId == null) return '请选择品种';
        return null;
      case 2:
        if ((widget.photoMaterial ?? '').isEmpty) return '请拍摄材料过磅照片';
        if ((widget.photoScale ?? '').isEmpty) return '请拍摄磅显数据特写';
        if ((widget.photoCloseup ?? '').isEmpty) return '请拍摄近距离照片';
        if ((double.tryParse(widget.gross.text) ?? 0) <= 0) return '请填写入场重量（kg）';
        return null;
      case 3:
        if (_nextRole == null || _nextRole!.isEmpty) return '请选择下一处理部门';
        final myId = _myUserId();
        if (_nextAssignee != null && myId > 0 && _nextAssignee == myId) {
          dynamic usersRaw;
          for (final e in _options) {
            final m = Map<String, dynamic>.from(e as Map);
            if (m['role_code']?.toString() == _nextRole) {
              usersRaw = m['users'];
              break;
            }
          }
          _nextAssignee = _firstAssigneeExcluding(usersRaw, myId);
          if (_nextAssignee == null || _nextAssignee == myId) {
            return '不能指派自己为下一处理人，请选择其他仓管';
          }
        }
        if (_nextAssignee == null || _nextAssignee! <= 0) {
          if (_usersForRole.isEmpty) {
            return '该部门没有其他可指派人员（不能指派自己），请先配置仓管账号';
          }
          return '请选择处理人';
        }
        return null;
    }
    return null;
  }

  Future<void> _goNext() async {
    FormRow.dismissKeyboard();
    // 第 1 步：用户可能已经在溯源码输入框里填了值，但尚未触发校验完成
    // 这里兜底：如果溯源码不为空但 batchOk=false，则先校验后再走下一步校验
    if (_step == 0) {
      final code = widget.batchNo.text.trim().toUpperCase();
      if (code.isNotEmpty && !widget.batchOk) {
        final ok = await widget.onValidateBatch();
        if (!ok) return;
        await Future<void>.delayed(const Duration(milliseconds: 0));
      }
    }
    final err = _validateStep(_step);
    if (err != null) {
      widget.onMsg(err);
      return;
    }
    if (_step == 1 && widget.batchNo.text.trim().isEmpty) {
      final generate = await showDialog<bool>(
        context: context,
        builder: (ctx) => AlertDialog(
          title: const Text('生成溯源码'),
          content: const Text('未填溯源码，是否新生成？'),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('否')),
            FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('是')),
          ],
        ),
      );
      if (generate != true) return;
      final ok = await widget.onGenerateTraceCode();
      if (!ok) return;
      await widget.onShowGeneratedQr();
    } else if (_step == 1 && !widget.batchOk) {
      final ok = await widget.onValidateBatch();
      if (!ok) return;
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
      setState(() => _step = 0);
      _pages.jumpToPage(0);
    }
  }

  Future<void> _goPrev() async {
    if (_step <= 0) return;
    FormRow.dismissKeyboard();
    setState(() => _step--);
    await _pages.animateToPage(_step, duration: const Duration(milliseconds: 250), curve: Curves.easeOut);
  }

  Future<void> _jumpToStep(int step) async {
    if (step < 0 || step > 3 || step == _step) return;
    setState(() => _step = step);
    await _pages.animateToPage(_step, duration: const Duration(milliseconds: 250), curve: Curves.easeOut);
  }

  String _varietyLabel() {
    for (final e in widget.varieties) {
      final m = Map<String, dynamic>.from(e as Map);
      if ((m['id'] as num?)?.toInt() == widget.varietyId) {
        return m['name']?.toString() ?? m['code']?.toString() ?? '-';
      }
    }
    return '-';
  }

  String _coldLabel(String v) {
    switch (v) {
      case 'semi':
        return '半成品库';
      case 'fg':
        return '成品库';
      default:
        return '保鲜库';
    }
  }

  double _previewNet() {
    final gross = double.tryParse(widget.gross.text) ?? 0;
    var rate = double.tryParse(widget.deductRate.text) ?? 0;
    if (rate > 1) rate = rate / 100;
    final net = gross - gross * rate;
    return net < 0 ? 0 : net;
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
              _stepFarmer(),
              _stepTrace(),
              _stepWeighPhotos(),
              _stepNextDept(),
            ],
          ),
        ),
        if (widget.msg.isNotEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Align(
              alignment: Alignment.centerLeft,
              child: Text(
                widget.msg,
                style: TextStyle(
                  color: widget.msgIsError ? Theme.of(context).colorScheme.error : Colors.black54,
                  fontSize: 13,
                  fontWeight: widget.msgIsError ? FontWeight.w600 : FontWeight.normal,
                ),
              ),
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
                    child: Text(_busy ? '提交中…' : (_step < 3 ? '下一步' : '确认创建并绑定')),
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
    return FormRow.text(
      label: label,
      controller: c,
      hint: hint,
      keyboardType: keyboardType,
      requiredMark: requiredMark,
    );
  }

  Widget _stepFarmer() {
    return ListView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 16),
      children: [
        const Padding(
          padding: EdgeInsets.fromLTRB(4, 0, 4, 6),
          child: Text(
            '先填农户与产地，再绑定或生成溯源码并选择品种；溯源与农户强绑定，输入已有码可自动带回档案',
            style: TextStyle(fontSize: 13, color: Colors.black54),
          ),
        ),
        ..._farmerFields(),
        TraceCodeField(
          controller: widget.batchNo,
          label: '溯源码',
          hint: '扫码绑定；空则可下一步生成（现场无法打印时用）',
          validated: widget.batchOk,
          scannerTitle: '扫描溯源批号',
          compact: true,
          requiredMark: false,
          onChanged: widget.onBatchChanged,
          onEditingComplete: () {
            widget.onValidateBatch();
          },
          onTapManual: () {
            widget.onTapManualTrace();
          },
          onScanned: (_) async {
            widget.onBatchChanged(widget.batchNo.text);
            await widget.onValidateBatch();
          },
        ),
        if (widget.bindingLocked)
          Padding(
            padding: const EdgeInsets.only(top: 6),
            child: Text(
              '已选过站中码：农户与品种锁定为首单信息，本单可追加重量等',
              style: TextStyle(fontSize: 12, color: Colors.blue.shade800),
            ),
          ),
        if (widget.varieties.isEmpty)
          const Padding(
            padding: EdgeInsets.symmetric(vertical: 8),
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
                onChanged: widget.bindingLocked
                    ? null
                    : (v) {
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
      ],
    );
  }

  Widget _stepTrace() {
    return ListView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 16),
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(4, 0, 4, 6),
          child: widget.batchOk
              ? Text(
                  '溯源码已校验：如需更换请返回上一步修改',
                  style: const TextStyle(fontSize: 13, color: Colors.black54),
                )
              : const Text(
                  '扫码绑定，或点输入框手输；可不填，下一步将询问是否新生成',
                  style: TextStyle(fontSize: 13, color: Colors.black54),
                ),
        ),
        if (!widget.batchOk)
          TraceCodeField(
            controller: widget.batchNo,
            label: '溯源码',
            hint: '扫码或点击手输，可空',
            validated: widget.batchOk,
            scannerTitle: '扫描溯源批号',
            compact: true,
            requiredMark: false,
            onChanged: widget.onBatchChanged,
            onEditingComplete: () {
              widget.onValidateBatch();
            },
            onTapManual: () {
              widget.onTapManualTrace();
            },
            onScanned: (_) async {
              widget.onBatchChanged(widget.batchNo.text);
              await widget.onValidateBatch();
            },
          ),
        if (widget.bindingLocked)
          Padding(
            padding: const EdgeInsets.only(top: 6),
            child: Text(
              '已选过站中码：农户与品种锁定为首单信息，本单可追加重量等',
              style: TextStyle(fontSize: 12, color: Colors.blue.shade800),
            ),
          ),
        if (!widget.batchOk) ...[
          if (widget.varieties.isEmpty)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 8),
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
                  onChanged: widget.bindingLocked
                      ? null
                      : (v) {
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
        ] else if (widget.varietyId != null)
          Padding(
            padding: const EdgeInsets.only(top: 10),
            child: Text(
              '当前品种：${_varietyLabel()}',
              style: const TextStyle(fontSize: 13, color: Colors.black54),
            ),
          ),
        const Padding(
          padding: EdgeInsets.fromLTRB(4, 14, 4, 6),
          child: Text('常用项（本地记忆）', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
        ),
        _textRow('单价（元）', widget.unitPrice, hint: '0', keyboardType: const TextInputType.numberWithOptions(decimal: true)),
        _textRow('扣损率（%）', widget.deductRate, hint: '0', keyboardType: const TextInputType.numberWithOptions(decimal: true)),
        _textRow('运费（元）', widget.freight, hint: '0', keyboardType: const TextInputType.numberWithOptions(decimal: true)),
        _textRow('装卸费（元）', widget.loadingFee, hint: '0', keyboardType: const TextInputType.numberWithOptions(decimal: true)),
        _textRow('过磅费（元）', widget.weighFee, hint: '0', keyboardType: const TextInputType.numberWithOptions(decimal: true)),
      ],
    );
  }

  List<Widget> _farmerFields() {
    if (widget.bindingLocked) {
      return [
        FormRow(
          label: '已锁定农户',
          child: Align(
            alignment: Alignment.centerRight,
            child: InputChip(
              avatar: const Icon(Icons.lock_outline, size: 16),
              label: Text(
                '${widget.farmerId != null ? '#${widget.farmerId} ' : ''}${widget.partyName.text}'.trim(),
              ),
            ),
          ),
        ),
        _previewRow('电话', widget.partyMobile.text.trim().isEmpty ? '-' : widget.partyMobile.text.trim()),
        _previewRow('产地', widget.origin.text.trim().isEmpty ? '-' : widget.origin.text.trim()),
      ];
    }
    final typed = widget.partyName.text.trim().isNotEmpty || widget.partyMobile.text.trim().isNotEmpty;
    return [
      FormRow.text(
        label: '姓名',
        controller: widget.partyName,
        hint: '输入后自动搜索',
        requiredMark: true,
        onChanged: widget.onNameChanged,
      ),
      FormRow.text(
        label: '手机号',
        controller: widget.partyMobile,
        hint: '选填，输入后自动搜索',
        keyboardType: TextInputType.phone,
        onChanged: widget.onMobileChanged,
      ),
      if (widget.searchingFarmer)
        const Padding(
          padding: EdgeInsets.symmetric(vertical: 8),
          child: Center(child: SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))),
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
      if (typed && !widget.searchingFarmer && widget.farmerHits.isEmpty && widget.farmerId == null)
        const Padding(
          padding: EdgeInsets.symmetric(vertical: 6),
          child: Text('未找到档案，提交入厂单时将自动建档', style: TextStyle(color: Colors.black45, fontSize: 13)),
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
      _textRow('产地', widget.origin, hint: '选填'),
    ];
  }

  Widget _stepWeighPhotos() {
    return ListView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 16),
      children: [
        const Text('先拍三张照片，再填写重量', style: TextStyle(fontSize: 12, color: Colors.black54)),
        const SizedBox(height: 8),
        _photoSlot(
          slot: 'material',
          title: '1. 材料过磅照片',
          hint: '拍物料上磅的全景',
          url: widget.photoMaterial,
        ),
        _photoSlot(
          slot: 'scale_display',
          title: '2. 磅显数据特写',
          hint: '拍秤上数字，清晰可读（后续用于识别重量）',
          url: widget.photoScale,
        ),
        _photoSlot(
          slot: 'closeup',
          title: '3. 近距离照片',
          hint: '拍物料近景',
          url: widget.photoCloseup,
        ),
        const SizedBox(height: 8),
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
          hint: '对照磅显特写填写',
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
        _textRow('备注', widget.remark, hint: '选填'),
      ],
    );
  }

  Widget _photoSlot({
    required String slot,
    required String title,
    required String hint,
    required String? url,
  }) {
    final api = context.read<AuthState>().api;
    final resolved = (url ?? '').isEmpty ? '' : api.resolveMediaUrl(url!);
    final taken = resolved.isNotEmpty;
    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.fromLTRB(10, 10, 10, 8),
      decoration: BoxDecoration(
        border: Border.all(color: taken ? const Color(0xFFB7D8C8) : const Color(0xFFE2E8F0)),
        borderRadius: BorderRadius.circular(10),
        color: taken ? const Color(0xFFF3FAF6) : const Color(0xFFFAFBFC),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text.rich(
                  TextSpan(
                    style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14),
                    children: [
                      TextSpan(text: title),
                      const TextSpan(text: ' *', style: TextStyle(color: Colors.redAccent)),
                    ],
                  ),
                ),
                const SizedBox(height: 2),
                Text(hint, style: const TextStyle(fontSize: 12, color: Colors.black54)),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  children: [
                    TextButton.icon(
                      onPressed: () => widget.onTakePhoto(slot),
                      icon: Icon(taken ? Icons.cameraswitch_outlined : Icons.photo_camera, size: 18),
                      label: Text(taken ? '重拍' : '拍照'),
                    ),
                    if (taken)
                      TextButton(
                        onPressed: () => widget.onRemovePhoto(slot),
                        child: const Text('删除'),
                      ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Material(
            color: Colors.black12,
            borderRadius: BorderRadius.circular(8),
            clipBehavior: Clip.antiAlias,
            child: InkWell(
              onTap: taken ? () => _showPhotoPreview(context, resolved) : () => widget.onTakePhoto(slot),
              child: taken
                  ? Image.network(
                      resolved,
                      width: 84,
                      height: 84,
                      fit: BoxFit.cover,
                      errorBuilder: (_, __, ___) => const SizedBox(
                        width: 84,
                        height: 84,
                        child: Center(child: Icon(Icons.broken_image_outlined)),
                      ),
                    )
                  : const SizedBox(
                      width: 84,
                      height: 84,
                      child: Center(child: Icon(Icons.add_a_photo_outlined, color: Colors.black38)),
                    ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _previewSection(String title, int editStep, List<Widget> rows) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(child: Text(title, style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13))),
            TextButton(
              onPressed: () => _jumpToStep(editStep),
              child: const Text('修改'),
            ),
          ],
        ),
        ...rows,
        const Divider(height: 20),
      ],
    );
  }

  Widget _previewRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 108,
            child: Text(label, style: TextStyle(fontSize: 13, color: Colors.black.withValues(alpha: 0.6))),
          ),
          Expanded(child: Text(value, textAlign: TextAlign.right, style: const TextStyle(fontSize: 14))),
        ],
      ),
    );
  }

  Widget _stepNextDept() {
    if (_loadingOptions) {
      return const Center(child: CircularProgressIndicator());
    }
    final users = _usersForRole;
    final net = _previewNet();
    final gross = double.tryParse(widget.gross.text) ?? 0;
    final rate = double.tryParse(widget.deductRate.text) ?? 0;
    final unit = double.tryParse(widget.unitPrice.text) ?? 0;
    final freight = double.tryParse(widget.freight.text) ?? 0;
    final loading = double.tryParse(widget.loadingFee.text) ?? 0;
    final weigh = double.tryParse(widget.weighFee.text) ?? 0;
    final settle = net * unit + freight + loading + weigh;
    return ListView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 16),
      children: [
        const Text('请核对单据，有误请点「修改」或底栏「上一步」', style: TextStyle(fontSize: 12, color: Colors.black54)),
        const SizedBox(height: 8),
        _previewSection('农户', 0, [
          _previewRow('农户ID', widget.farmerId == null ? '提交时自动建档' : '#${widget.farmerId}'),
          _previewRow('姓名', widget.partyName.text.trim().isEmpty ? '-' : widget.partyName.text.trim()),
          _previewRow('电话', widget.partyMobile.text.trim().isEmpty ? '-' : widget.partyMobile.text.trim()),
          _previewRow('产地', widget.origin.text.trim().isEmpty ? '-' : widget.origin.text.trim()),
        ]),
        _previewSection('溯源码与品种', 1, [
          _previewRow('溯源批号', widget.batchNo.text.trim().toUpperCase()),
          _previewRow('品种', _varietyLabel()),
          _previewRow('扣损率(%)', widget.deductRate.text),
          _previewRow('单价', unit.toString()),
          _previewRow('运/装/磅费', '$freight / $loading / $weigh'),
        ]),
        _previewSection('照片与过磅', 2, [
          _previewRow('渠道', widget.channel == 'external' ? '外磅' : '厂内'),
          _previewRow('材料过磅照', (widget.photoMaterial ?? '').isEmpty ? '未拍' : '已拍'),
          _previewRow('磅显特写', (widget.photoScale ?? '').isEmpty ? '未拍' : '已拍'),
          _previewRow('近距离照片', (widget.photoCloseup ?? '').isEmpty ? '未拍' : '已拍'),
          _previewRow('入场重量(kg)', gross.toString()),
          _previewRow('预估净重(kg)', net.toStringAsFixed(2)),
          _previewRow('预估结算', settle.toStringAsFixed(2)),
          _previewRow('车牌', widget.plate.text.trim().isEmpty ? '-' : widget.plate.text.trim()),
          _previewRow('收货地址', widget.recvAddr.text.trim().isEmpty ? '-' : widget.recvAddr.text.trim()),
          _previewRow('目标库', _coldLabel(widget.coldStore)),
          _previewRow('等级', widget.grade),
          _previewRow('备注', widget.remark.text.trim().isEmpty ? '-' : widget.remark.text.trim()),
        ]),
        Text.rich(
          TextSpan(
            style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13, color: Colors.black87),
            children: const [
              TextSpan(text: '下一处理部门'),
              TextSpan(text: ' *', style: TextStyle(color: Colors.redAccent)),
            ],
          ),
        ),
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
        const SizedBox(height: 12),
        Text.rich(
          TextSpan(
            style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13, color: Colors.black87),
            children: const [
              TextSpan(text: '处理人'),
              TextSpan(text: ' *', style: TextStyle(color: Colors.redAccent)),
            ],
          ),
        ),
        const SizedBox(height: 8),
        if (users.isEmpty)
          const Text('该部门暂无其他可指派用户（不能指派自己）', style: TextStyle(color: Colors.orange))
        else
          ...users.map((u) {
            final uid = (u['user_id'] as num?)?.toInt();
            final selected = uid != null && uid == _nextAssignee;
            return ListTile(
              dense: true,
              contentPadding: EdgeInsets.zero,
              leading: Icon(selected ? Icons.radio_button_checked : Icons.radio_button_off),
              title: Text('${u['name'] ?? u['login_name'] ?? uid}'),
              subtitle: Text('${u['login_name'] ?? ''}'),
              selected: selected,
              onTap: uid == null
                  ? null
                  : () {
                      final myId = _myUserId();
                      if (myId > 0 && uid == myId) {
                        widget.onMsg('不能指派自己为下一处理人');
                        return;
                      }
                      setState(() => _nextAssignee = uid);
                    },
            );
          }),
        const SizedBox(height: 8),
        Text(
          '确认后本张过磅单独立绑定并推仓管（结算按单）；同码追加须同农户同产品。扣损率按 $rate${rate > 1 ? '%' : ''} 估算净重。',
          style: const TextStyle(fontSize: 12, color: Colors.black54),
        ),
      ],
    );
  }
}
