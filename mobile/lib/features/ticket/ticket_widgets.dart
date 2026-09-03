import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';

const ticketStatusLabel = {
  'open': '待处理',
  'in_progress': '处理中',
  'done': '已办结',
  'rejected': '已驳回',
  'cancelled': '已取消',
};

bool ticketIsClosed(String status) =>
    status == 'done' || status == 'rejected' || status == 'cancelled';

bool ticketCanAct(String status) => status == 'open' || status == 'in_progress';

/// 仅当前处理人（或系统管理员）可操作工单。
bool ticketIsCurrentAssignee(AuthState auth, Map<String, dynamic> row) {
  if (row['can_act'] == true) return true;
  if (row['can_act'] == false) {
    final roles = auth.roles.map((e) => e.toString().toLowerCase()).toList();
    if (roles.contains('sys_admin') || roles.contains('admin') || roles.contains('系统管理员')) {
      return true;
    }
    return false;
  }
  final roles = auth.roles.map((e) => e.toString().toLowerCase()).toList();
  if (roles.contains('sys_admin') || roles.contains('admin') || roles.contains('系统管理员')) {
    return true;
  }
  final assignee = (row['current_assignee_user_id'] as num?)?.toInt() ?? 0;
  return assignee > 0 && assignee == auth.userId;
}

bool ticketCanActByMe(AuthState auth, Map<String, dynamic> row) =>
    ticketCanAct('${row['status'] ?? ''}') && ticketIsCurrentAssignee(auth, row);

bool _hasRole(AuthState auth, String role) {
  final roles = auth.roles.map((e) => e.toString().toLowerCase()).toList();
  return roles.contains(role.toLowerCase()) ||
      roles.contains('sys_admin') ||
      roles.contains('admin') ||
      (role == 'warehouse' && (roles.contains('仓管') || roles.contains('仓管员'))) ||
      (role == 'qc' && (roles.contains('质检') || roles.contains('质检员')));
}

bool _isQcOnly(AuthState auth) {
  if (_hasRole(auth, 'warehouse') || _hasRole(auth, 'finance') || _hasRole(auth, 'purchase')) {
    return false;
  }
  return _hasRole(auth, 'qc');
}

List<String> _weighVerifyImages(Map<String, dynamic> payload, {String Function(String)? resolve}) {
  final out = <String>[];
  void add(dynamic v) {
    var s = v?.toString().trim() ?? '';
    if (s.isEmpty) return;
    if (resolve != null) s = resolve(s);
    if (s.isEmpty || out.contains(s)) return;
    out.add(s);
  }

  add(payload['image_url']);
  final photos = payload['photos'];
  if (photos is Map) {
    add(photos['material']);
    add(photos['scale_display']);
    add(photos['closeup']);
  }
  for (final key in ['verify_images', 'site_photos', 'image_urls']) {
    final raw = payload[key];
    if (raw is List) {
      for (final e in raw) {
        add(e);
      }
    }
  }
  final evidences = payload['evidences'];
  if (evidences is List) {
    for (final e in evidences) {
      if (e is Map) {
        add(e['file_url'] ?? e['url']);
      }
    }
  }
  return out;
}

Future<void> _previewImage(BuildContext context, String url) async {
  await showDialog<void>(
    context: context,
    builder: (ctx) => Dialog(
      insetPadding: const EdgeInsets.all(12),
      child: InteractiveViewer(
        child: AspectRatio(
          aspectRatio: 1,
          child: Image.network(
            url,
            fit: BoxFit.contain,
            errorBuilder: (_, __, ___) => const Center(child: Text('图片加载失败')),
          ),
        ),
      ),
    ),
  );
}

class TicketRefreshBus extends ChangeNotifier {
  void bump() => notifyListeners();
}

Future<void> ticketAct(
  BuildContext context,
  Map<String, dynamic> row,
  String action, {
  Map<String, dynamic>? extra,
  VoidCallback? onDone,
  bool requireHandoff = false,
}) async {
  final id = (row['id'] as num?)?.toInt();
  if (id == null) return;
  final body = <String, dynamic>{'action': action, ...?extra};
  if (requireHandoff &&
      body['next_assignee_user_id'] == null &&
      (body['next_role_code'] == null || '${body['next_role_code']}'.isEmpty)) {
    final next = await pickNextHandler(context, id);
    if (next == null) return;
    body.addAll(next);
  }
  final r = await context.read<AuthState>().api.post('/workflow/tickets/$id/action', body);
  if (!context.mounted) return;
  ScaffoldMessenger.of(context).showSnackBar(
    SnackBar(content: Text(r.ok ? '已$action' : r.msg)),
  );
  if (r.ok) {
    onDone?.call();
    try {
      context.read<TicketRefreshBus>().bump();
    } catch (_) {}
  }
}

/// 选择下一处理人（或按角色交办）。取消返回 null。
Future<Map<String, dynamic>?> pickNextHandler(BuildContext context, int ticketId) async {
  final api = context.read<AuthState>().api;
  final r = await api.get('/workflow/tickets/$ticketId/handlers-pool');
  if (!context.mounted) return null;
  if (!r.ok) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.msg)));
    return null;
  }
  final raw = r.data is Map ? (r.data as Map)['pool'] : null;
  final pool = <Map<String, dynamic>>[];
  if (raw is List) {
    for (final e in raw) {
      if (e is Map) pool.add(Map<String, dynamic>.from(e));
    }
  }
  if (pool.isEmpty) {
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('处理人池为空，无法交办')));
    return null;
  }

  final roleSet = <String>{};
  for (final p in pool) {
    final codes = p['role_codes'];
    if (codes is List) {
      for (final c in codes) {
        final s = c.toString().trim();
        if (s.isNotEmpty) roleSet.add(s);
      }
    }
  }
  final roles = roleSet.toList()..sort();

  return showModalBottomSheet<Map<String, dynamic>>(
    context: context,
    isScrollControlled: true,
    builder: (ctx) {
      String? filterRole;
      return StatefulBuilder(
        builder: (ctx, setLocal) {
          final filtered = filterRole == null || filterRole!.isEmpty
              ? pool
              : pool.where((p) {
                  final codes = p['role_codes'];
                  if (codes is! List) return false;
                  return codes.any((c) => c.toString().toLowerCase() == filterRole!.toLowerCase());
                }).toList();
          return SafeArea(
            child: SizedBox(
              height: MediaQuery.of(ctx).size.height * 0.65,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Padding(
                    padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                    child: Text('指定下一处理人', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                  ),
                  if (roles.isNotEmpty)
                    SingleChildScrollView(
                      scrollDirection: Axis.horizontal,
                      padding: const EdgeInsets.symmetric(horizontal: 12),
                      child: Row(
                        children: [
                          ChoiceChip(
                            label: const Text('全部'),
                            selected: filterRole == null,
                            onSelected: (_) => setLocal(() => filterRole = null),
                          ),
                          ...roles.map(
                            (role) => Padding(
                              padding: const EdgeInsets.only(left: 6),
                              child: ChoiceChip(
                                label: Text(role),
                                selected: filterRole == role,
                                onSelected: (_) => setLocal(() => filterRole = role),
                              ),
                            ),
                          ),
                          if (filterRole != null)
                            Padding(
                              padding: const EdgeInsets.only(left: 8),
                              child: TextButton(
                                onPressed: () => Navigator.pop(ctx, {'next_role_code': filterRole}),
                                child: Text('交办给角色 $filterRole'),
                              ),
                            ),
                        ],
                      ),
                    ),
                  const Divider(height: 1),
                  Expanded(
                    child: ListView.builder(
                      itemCount: filtered.length,
                      itemBuilder: (_, i) {
                        final p = filtered[i];
                        final uid = (p['user_id'] as num?)?.toInt() ?? 0;
                        final name = '${p['name'] ?? p['login_name'] ?? uid}';
                        final codes = (p['role_codes'] is List)
                            ? (p['role_codes'] as List).map((e) => e.toString()).join(',')
                            : '';
                        return ListTile(
                          title: Text(name),
                          subtitle: codes.isEmpty ? null : Text(codes),
                          onTap: () => Navigator.pop(ctx, {'next_assignee_user_id': uid}),
                        );
                      },
                    ),
                  ),
                ],
              ),
            ),
          );
        },
      );
    },
  );
}

Future<void> _qcJudgmentFlow(
  BuildContext context,
  Map<String, dynamic> d, {
  VoidCallback? onActed,
}) async {
  final bizId = (d['biz_id'] as num?)?.toInt() ?? 0;
  final ticketId = (d['id'] as num?)?.toInt() ?? 0;
  if (bizId <= 0) return;

  String? result;
  String grade = 'A';
  final remarkCtrl = TextEditingController();
  final ok = await showDialog<bool>(
    context: context,
    builder: (ctx) => StatefulBuilder(
      builder: (ctx, setLocal) => AlertDialog(
        title: const Text('质检判定'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            SegmentedButton<String>(
              segments: const [
                ButtonSegment(value: 'pass', label: Text('合格')),
                ButtonSegment(value: 'fail', label: Text('不合格')),
              ],
              emptySelectionAllowed: true,
              selected: {if (result != null) result!},
              onSelectionChanged: (s) => setLocal(() => result = s.isEmpty ? null : s.first),
            ),
            if (result == 'pass') ...[
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                // ignore: deprecated_member_use — StatefulBuilder 需受控 value
                value: grade,
                decoration: const InputDecoration(labelText: '等级'),
                items: const [
                  DropdownMenuItem(value: 'A', child: Text('A')),
                  DropdownMenuItem(value: 'B', child: Text('B')),
                  DropdownMenuItem(value: 'C', child: Text('C')),
                ],
                onChanged: (v) => setLocal(() => grade = v ?? 'A'),
              ),
            ],
            const SizedBox(height: 8),
            TextField(controller: remarkCtrl, decoration: const InputDecoration(labelText: '备注')),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(
            onPressed: result == null ? null : () => Navigator.pop(ctx, true),
            child: const Text('下一步'),
          ),
        ],
      ),
    ),
  );
  if (ok != true || result == null || !context.mounted) return;

  Map<String, dynamic> body = {
    'qc_result': result,
    'remark': remarkCtrl.text.trim(),
  };
  if (result == 'pass') {
    body['grade'] = grade;
    final next = await pickNextHandler(context, ticketId);
    if (next == null || !context.mounted) return;
    body.addAll(next);
  }

  final r = await context.read<AuthState>().api.post('/purchase/weigh-tickets/$bizId/qc', body);
  if (!context.mounted) return;
  ScaffoldMessenger.of(context).showSnackBar(
    SnackBar(content: Text(r.ok ? '质检已提交' : r.msg)),
  );
  if (r.ok) {
    onActed?.call();
    try {
      context.read<TicketRefreshBus>().bump();
    } catch (_) {}
  }
}

Future<void> openTicketDetail(
  BuildContext context,
  Map<String, dynamic> row, {
  bool allowActions = true,
  VoidCallback? onActed,
}) async {
  final id = (row['id'] as num?)?.toInt();
  if (id == null) return;
  final auth = context.read<AuthState>();
  final r = await auth.api.get('/workflow/tickets/$id');
  if (!context.mounted) return;
  if (!r.ok || r.data is! Map) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.msg)));
    return;
  }
  final d = Map<String, dynamic>.from(r.data as Map);
  final schema = (d['form_schema'] as List?) ?? [];
  final payload =
      d['payload'] is Map ? Map<String, dynamic>.from(d['payload'] as Map) : <String, dynamic>{};
  final st = '${d['status'] ?? ''}';
  // 详情以服务端最新 assignee 为准，非当前处理人只读。
  final canAct = allowActions && ticketCanActByMe(auth, d);
  final bizType = '${d['biz_type'] ?? ''}';
  final isWeigh = bizType == 'weigh_ticket';
  final bizId = (d['biz_id'] as num?)?.toInt() ?? 0;

  // 兜底：工单 payload 无图时，从过磅单补拉现场照片
  if (isWeigh && bizId > 0 && _weighVerifyImages(payload).isEmpty) {
    final wr = await auth.api.get('/purchase/weigh-tickets/$bizId');
    if (context.mounted && wr.ok && wr.data is Map) {
      final w = Map<String, dynamic>.from(wr.data as Map);
      final imgs = <String>[];
      void add(dynamic v) {
        final s = v?.toString().trim() ?? '';
        if (s.isNotEmpty && !imgs.contains(s)) imgs.add(s);
      }
      add(w['image_url']);
      final urls = w['image_urls'] ?? w['verify_images'];
      if (urls is List) {
        for (final e in urls) {
          add(e);
        }
      }
      final evidences = w['evidences'];
      if (evidences is List) {
        for (final e in evidences) {
          if (e is Map) add(e['file_url'] ?? e['url']);
        }
      }
      if (imgs.isNotEmpty) {
        payload['image_url'] = imgs.first;
        payload['image_urls'] = imgs;
        payload['verify_images'] = imgs;
      }
      if (w['photos'] is Map) {
        payload['photos'] = w['photos'];
      }
      payload['status'] = w['status'] ?? payload['status'];
      payload['qc_result'] = w['qc_result'] ?? payload['qc_result'];
      payload['grade'] = w['grade'] ?? payload['grade'];
      payload['receive_kind'] = w['receive_kind'] ?? payload['receive_kind'];
    }
  }

  final settle = payload['settle_breakdown'] is Map
      ? Map<String, dynamic>.from(payload['settle_breakdown'] as Map)
      : <String, dynamic>{};
  final hasSettle = settle.isNotEmpty || payload['settlement_id'] != null;
  final verifyImgs = _weighVerifyImages(payload, resolve: auth.api.resolveMediaUrl);

  await showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    builder: (ctx) => Padding(
      padding: EdgeInsets.only(
        left: 16,
        right: 16,
        top: 16,
        bottom: MediaQuery.of(ctx).viewInsets.bottom + 16,
      ),
      child: ListView(
        shrinkWrap: true,
        children: [
          Text('${d['title']}', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
          Text(
            '${d['doc_no']} · ${d['category_name']} · ${ticketStatusLabel[st] ?? st}',
          ),
          const Divider(),
          if (isWeigh) ...[
            if (payload['batch_no'] != null) ListTile(dense: true, title: const Text('批号'), trailing: Text('${payload['batch_no']}')),
            if (payload['trace_code'] != null) ListTile(dense: true, title: const Text('溯源码'), trailing: Text('${payload['trace_code']}')),
            if (payload['variety'] != null || payload['product_name'] != null)
              ListTile(
                dense: true,
                title: const Text('品种'),
                trailing: Text('${payload['variety'] ?? payload['product_name'] ?? '-'}'),
              ),
            if (payload['net_weight'] != null)
              ListTile(dense: true, title: const Text('净重(kg)'), trailing: Text('${payload['net_weight']}')),
            if (verifyImgs.isNotEmpty) ...[
              const SizedBox(height: 8),
              const Text('现场照片（核对用）', style: TextStyle(fontWeight: FontWeight.w600)),
              const SizedBox(height: 8),
              SizedBox(
                height: 110,
                child: ListView.separated(
                  scrollDirection: Axis.horizontal,
                  itemCount: verifyImgs.length,
                  separatorBuilder: (_, __) => const SizedBox(width: 8),
                  itemBuilder: (_, i) {
                    final url = verifyImgs[i];
                    return GestureDetector(
                      onTap: () => _previewImage(context, url),
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(8),
                        child: Image.network(
                          url,
                          width: 110,
                          height: 110,
                          fit: BoxFit.cover,
                          errorBuilder: (_, __, ___) => Container(
                            width: 110,
                            height: 110,
                            color: Colors.black12,
                            alignment: Alignment.center,
                            child: const Icon(Icons.broken_image),
                          ),
                        ),
                      ),
                    );
                  },
                ),
              ),
            ],
            if (_hasRole(auth, 'finance') && settle.isNotEmpty) ...[
              const Divider(),
              const Text('结算明细', style: TextStyle(fontWeight: FontWeight.w600)),
              ListTile(dense: true, title: const Text('净重×单价'), trailing: Text('${settle['net_weight']} × ${settle['unit_price']}')),
              ListTile(dense: true, title: const Text('货款'), trailing: Text('${settle['goods_amount']}')),
              ListTile(dense: true, title: const Text('运费/装卸/过磅'), trailing: Text('${settle['freight_fee']}/${settle['loading_fee']}/${settle['weigh_fee']}')),
              ListTile(dense: true, title: const Text('应付合计'), trailing: Text('${settle['amount']}')),
              if (settle['farmer_name'] != null)
                ListTile(dense: true, title: const Text('农户'), trailing: Text('${settle['farmer_name']}')),
            ],
          ] else
            ...schema.map((raw) {
              final f = Map<String, dynamic>.from(raw as Map);
              final key = '${f['key']}';
              return ListTile(
                dense: true,
                title: Text('${f['label']}'),
                trailing: Text('${payload[key] ?? '-'}'),
              );
            }),
          if (canAct) ...[
            const SizedBox(height: 8),
            if (isWeigh && (_hasRole(auth, 'qc') || _hasRole(auth, 'purchase')) &&
                !_hasRole(auth, 'warehouse') &&
                (payload['qc_result'] == null || '${payload['qc_result']}'.isEmpty ||
                    '${payload['status']}' == 'qc_pending' ||
                    '${payload['status']}' == 'draft' ||
                    '${payload['status']}' == 'pending_confirm'))
              FilledButton(
                onPressed: () async {
                  Navigator.pop(ctx);
                  await _qcJudgmentFlow(context, d, onActed: onActed);
                },
                child: const Text('质检判定'),
              )
            else if (isWeigh && _hasRole(auth, 'warehouse') && !hasSettle && !_isQcOnly(auth))
              FilledButton(
                onPressed: () async {
                  Navigator.pop(ctx);
                  final needHandoff = '${payload['receive_kind'] ?? 'gate'}'.toLowerCase() == 'stockin';
                  await ticketAct(
                    context,
                    d,
                    'warehouse_confirm',
                    requireHandoff: needHandoff,
                    onDone: onActed,
                  );
                },
                child: const Text('核对并确认入库'),
              )
            else if (isWeigh && _hasRole(auth, 'finance') && hasSettle)
              FilledButton(
                onPressed: () async {
                  Navigator.pop(ctx);
                  await _settlePayDialog(context, d, onActed: onActed);
                },
                child: const Text('确认付款'),
              )
            else if (!isWeigh || (!_hasRole(auth, 'warehouse') && !_hasRole(auth, 'finance') && !_hasRole(auth, 'qc'))) ...[
              FilledButton(
                onPressed: () async {
                  Navigator.pop(ctx);
                  await ticketAct(context, d, 'approve', requireHandoff: true, onDone: onActed);
                },
                child: const Text('通过并交办'),
              ),
              if (!isWeigh) ...[
                const SizedBox(height: 8),
                OutlinedButton(
                  onPressed: () async {
                    Navigator.pop(ctx);
                    await ticketAct(context, d, 'return_confirm', onDone: onActed);
                  },
                  child: const Text('确认归还'),
                ),
              ],
            ] else if (isWeigh && _hasRole(auth, 'finance') && !hasSettle)
              const Text('待仓管入库生成结算后再付款', style: TextStyle(color: Colors.black54))
            else if (isWeigh && _hasRole(auth, 'warehouse') && hasSettle)
              const Text('已入库，等待财务付款', style: TextStyle(color: Colors.black54)),
            const SizedBox(height: 8),
            TextButton(
              onPressed: () async {
                Navigator.pop(ctx);
                await ticketAct(context, d, 'reject', onDone: onActed);
              },
              child: const Text('驳回'),
            ),
          ],
        ],
      ),
    ),
  );
}

Future<void> _settlePayDialog(
  BuildContext context,
  Map<String, dynamic> d, {
  VoidCallback? onActed,
}) async {
  final transfer = TextEditingController();
  String evidenceUrl = '';
  String err = '';
  bool uploading = false;
  int fundAccountId = 0;
  List<Map<String, dynamic>> funds = [];

  try {
    final fr = await context.read<AuthState>().api.get('/finance/fund-accounts');
    if (fr.ok) {
      funds = ApiClient.listOf(fr.data)
          .whereType<Map>()
          .map((e) => Map<String, dynamic>.from(e))
          .toList();
      if (funds.isNotEmpty) {
        fundAccountId = (funds.first['id'] as num?)?.toInt() ?? 0;
      }
    }
  } catch (_) {}

  final ok = await showDialog<bool>(
    context: context,
    builder: (ctx) {
      return StatefulBuilder(
        builder: (ctx, setLocal) => AlertDialog(
          title: const Text('确认付款'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                if (funds.isNotEmpty)
                  DropdownButtonFormField<int>(
                    // ignore: deprecated_member_use
                    value: fundAccountId > 0 ? fundAccountId : null,
                    decoration: const InputDecoration(labelText: '资金账户 *'),
                    items: [
                      for (final f in funds)
                        DropdownMenuItem(
                          value: (f['id'] as num?)?.toInt(),
                          child: Text('${f['name']}（余额 ${f['balance'] ?? 0}）'),
                        ),
                    ],
                    onChanged: (v) => setLocal(() => fundAccountId = v ?? 0),
                  )
                else
                  const Text('未拉到资金账户，请确认财务资金接口已开放', style: TextStyle(color: Colors.orange, fontSize: 13)),
                const SizedBox(height: 8),
                TextField(
                  controller: transfer,
                  decoration: const InputDecoration(labelText: '转账号 *'),
                  onTap: () {
                    final t = transfer.text;
                    transfer.selection = TextSelection.collapsed(offset: t.length);
                  },
                ),
                const SizedBox(height: 12),
                const Text('发票 / 转账截图 *', style: TextStyle(fontSize: 13, color: Colors.black54)),
                const SizedBox(height: 6),
                Row(
                  children: [
                    FilledButton.tonalIcon(
                      onPressed: uploading
                          ? null
                          : () async {
                              try {
                                final picker = ImagePicker();
                                final file = await picker.pickImage(
                                  source: ImageSource.camera,
                                  imageQuality: 85,
                                );
                                if (file == null) return;
                                setLocal(() {
                                  uploading = true;
                                  err = '';
                                });
                                final bytes = await file.readAsBytes();
                                if (!ctx.mounted) return;
                                final r = await ctx.read<AuthState>().api.postMultipart(
                                      '/biz/uploads',
                                      bytes,
                                      filename: file.name.isEmpty ? 'pay_receipt.jpg' : file.name,
                                    );
                                if (!ctx.mounted) return;
                                setLocal(() => uploading = false);
                                if (!r.ok || r.data is! Map) {
                                  setLocal(() => err = '上传失败：${r.msg}');
                                  return;
                                }
                                final url = (r.data as Map)['url']?.toString() ??
                                    (r.data as Map)['file_url']?.toString() ??
                                    '';
                                if (url.isEmpty) {
                                  setLocal(() => err = '上传无返回地址');
                                  return;
                                }
                                setLocal(() {
                                  evidenceUrl = url;
                                  err = '';
                                });
                              } catch (e) {
                                setLocal(() {
                                  uploading = false;
                                  err = '拍照失败：$e';
                                });
                              }
                            },
                      icon: const Icon(Icons.photo_camera),
                      label: Text(uploading ? '上传中…' : '拍照上传'),
                    ),
                    const SizedBox(width: 8),
                    TextButton(
                      onPressed: uploading
                          ? null
                          : () async {
                              try {
                                final picker = ImagePicker();
                                final file = await picker.pickImage(
                                  source: ImageSource.gallery,
                                  imageQuality: 85,
                                );
                                if (file == null) return;
                                setLocal(() {
                                  uploading = true;
                                  err = '';
                                });
                                final bytes = await file.readAsBytes();
                                if (!ctx.mounted) return;
                                final r = await ctx.read<AuthState>().api.postMultipart(
                                      '/biz/uploads',
                                      bytes,
                                      filename: file.name.isEmpty ? 'pay_receipt.jpg' : file.name,
                                    );
                                if (!ctx.mounted) return;
                                setLocal(() => uploading = false);
                                if (!r.ok || r.data is! Map) {
                                  setLocal(() => err = '上传失败：${r.msg}');
                                  return;
                                }
                                final url = (r.data as Map)['url']?.toString() ??
                                    (r.data as Map)['file_url']?.toString() ??
                                    '';
                                if (url.isEmpty) {
                                  setLocal(() => err = '上传无返回地址');
                                  return;
                                }
                                setLocal(() {
                                  evidenceUrl = url;
                                  err = '';
                                });
                              } catch (e) {
                                setLocal(() {
                                  uploading = false;
                                  err = '选择失败：$e';
                                });
                              }
                            },
                      child: const Text('相册'),
                    ),
                  ],
                ),
                if (evidenceUrl.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  ClipRRect(
                    borderRadius: BorderRadius.circular(8),
                    child: AspectRatio(
                      aspectRatio: 16 / 9,
                      child: Image.network(evidenceUrl, fit: BoxFit.cover, errorBuilder: (_, __, ___) => const Center(child: Text('预览失败'))),
                    ),
                  ),
                  TextButton(
                    onPressed: () => setLocal(() => evidenceUrl = ''),
                    child: const Text('清除图片'),
                  ),
                ],
                if (err.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: Text(err, style: const TextStyle(color: Colors.red, fontSize: 13)),
                  ),
              ],
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
            FilledButton(
              onPressed: uploading
                  ? null
                  : () {
                      if (fundAccountId <= 0) {
                        setLocal(() => err = '请选择资金账户');
                        return;
                      }
                      if (transfer.text.trim().isEmpty) {
                        setLocal(() => err = '请填写转账号');
                        return;
                      }
                      if (evidenceUrl.isEmpty) {
                        setLocal(() => err = '请拍照或选择发票/转账截图');
                        return;
                      }
                      Navigator.pop(ctx, true);
                    },
              child: const Text('提交'),
            ),
          ],
        ),
      );
    },
  );
  final transferNo = transfer.text.trim();
  transfer.dispose();
  if (ok != true || !context.mounted) return;
  await ticketAct(
    context,
    d,
    'settle_pay',
    extra: {
      'fund_account_id': fundAccountId,
      'transfer_no': transferNo,
      'pay_evidence_url': evidenceUrl,
    },
    onDone: onActed,
  );
}

class TicketListCard extends StatelessWidget {
  const TicketListCard({
    super.key,
    required this.row,
    this.showActions = false,
    this.emphasizeAssignee = false,
    this.onTap,
    this.onAction,
  });

  final Map<String, dynamic> row;
  final bool showActions;
  /// 「处理中」跟踪视角：突出当前处理人
  final bool emphasizeAssignee;
  final VoidCallback? onTap;
  final void Function(String action)? onAction;

  @override
  Widget build(BuildContext context) {
    final st = '${row['status'] ?? ''}';
    final isWeigh = '${row['biz_type'] ?? ''}' == 'weigh_ticket';
    final auth = context.watch<AuthState>();
    final canMine = showActions && ticketCanActByMe(auth, row);
    final assignee = '${row['assignee_name'] ?? '-'}';
    final statusLine = emphasizeAssignee
        ? '${ticketStatusLabel[st] ?? st} · 当前在 $assignee 手上'
        : '${ticketStatusLabel[st] ?? st} · 处理人 $assignee';
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      child: ListTile(
        title: Text('${row['title'] ?? ''}'),
        subtitle: Text(
          '${row['doc_no']} · ${row['category_name']}\n$statusLine',
        ),
        isThreeLine: true,
        onTap: onTap,
        trailing: canMine
            ? PopupMenuButton<String>(
                onSelected: onAction,
                itemBuilder: (_) => [
                  if (isWeigh && _hasRole(auth, 'warehouse'))
                    const PopupMenuItem(value: 'warehouse_confirm', child: Text('确认入库')),
                  if (!isWeigh) ...[
                    const PopupMenuItem(value: 'approve', child: Text('通过并交办')),
                    const PopupMenuItem(value: 'return_confirm', child: Text('确认归还')),
                  ],
                  const PopupMenuItem(value: 'reject', child: Text('驳回')),
                ],
              )
            : null,
      ),
    );
  }
}

/// AppBar 消息入口（与现网位置一致）
List<Widget> ticketShellMessageActions(BuildContext context, int unread) {
  return [
    IconButton(
      onPressed: () => Navigator.of(context).pushNamed('/inbox'),
      icon: Badge(
        isLabelVisible: unread > 0,
        label: Text('${unread > 99 ? '99+' : unread}'),
        child: const Icon(Icons.notifications_outlined),
      ),
    ),
  ];
}
