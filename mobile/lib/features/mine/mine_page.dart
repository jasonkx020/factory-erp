import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/notify_service.dart';
import 'account_center_page.dart';
import '../hr/hr_onboard_page.dart';

/// 全员：打卡、请假、审批待办、工资只读、消息
class MinePage extends StatefulWidget {
  const MinePage({super.key});

  @override
  State<MinePage> createState() => _MinePageState();
}

class _MinePageState extends State<MinePage> {
  int _tab = 0;
  String _msg = '';
  Map<String, dynamic>? _lastPunch;
  Map<String, dynamic>? _wage;
  List<dynamic> _leaves = [];
  List<dynamic> _tasks = [];
  List<dynamic> _ots = [];
  List<dynamic> _sheets = [];
  List<dynamic> _commissions = [];
  List<dynamic> _journals = [];
  List<dynamic> _memos = [];
  final _leaveRemark = TextEditingController();
  final _otMinutes = TextEditingController(text: '60');
  final _otRemark = TextEditingController();
  final _journalContent = TextEditingController();
  final _memoTitle = TextEditingController();
  final _memoContent = TextEditingController();
  String _leaveType = 'annual';
  String _otType = 'overtime';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      await context.read<NotifyService>().start();
      await _loadWage();
    });
  }

  @override
  void dispose() {
    _leaveRemark.dispose();
    _otMinutes.dispose();
    _otRemark.dispose();
    _journalContent.dispose();
    _memoTitle.dispose();
    _memoContent.dispose();
    super.dispose();
  }

  Future<void> _punch(String type) async {
    final auth = context.read<AuthState>();
    final body = <String, dynamic>{'punch_type': type};
    if (auth.employeeId > 0) body['employee_id'] = auth.employeeId;
    final r = await auth.api.post('/hr/attendance/records/punch', body);
    setState(() {
      if (r.ok) {
        _lastPunch = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : null;
        _msg = type == 'in' ? '上班打卡成功' : '下班打卡成功';
      } else {
        _msg = r.msg;
      }
    });
  }

  Future<void> _loadWage() async {
    final r = await context.read<AuthState>().api.get('/production/piecework-summaries/mine');
    if (!mounted) return;
    setState(() => _wage = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : null);
  }

  Future<void> _loadLeaves() async {
    final r = await context.read<AuthState>().api.get('/hr/leave-requests');
    if (!mounted) return;
    setState(() => _leaves = ApiClient.listOf(r.data));
  }

  Future<void> _loadOts() async {
    final r = await context.read<AuthState>().api.get('/hr/overtime-patches');
    if (!mounted) return;
    setState(() => _ots = ApiClient.listOf(r.data));
  }

  Future<void> _createOt() async {
    final auth = context.read<AuthState>();
    if (auth.employeeId <= 0) {
      setState(() => _msg = '账号未绑定员工');
      return;
    }
    final r = await auth.api.post('/hr/overtime-patches', {
      'employee_id': auth.employeeId,
      'biz_type': _otType,
      'biz_date': DateTime.now().toIso8601String().substring(0, 10),
      'minutes': int.tryParse(_otMinutes.text) ?? 60,
      'remark': _otRemark.text.trim().isEmpty ? '手机端申请' : _otRemark.text.trim(),
    });
    setState(() => _msg = r.ok ? '加班/补卡已提交' : r.msg);
    if (r.ok) await _loadOts();
  }

  Future<void> _loadPay() async {
    final api = context.read<AuthState>().api;
    final results = await Future.wait([
      api.get('/payroll/sheets?page_size=20'),
      api.get('/payroll/commission-calcs?page_size=20'),
      api.get('/production/piecework-summaries/mine'),
    ]);
    if (!mounted) return;
    setState(() {
      _sheets = ApiClient.listOf(results[0].data);
      _commissions = ApiClient.listOf(results[1].data);
      _wage = results[2].data is Map ? Map<String, dynamic>.from(results[2].data as Map) : null;
    });
  }

  Future<void> _createLeave() async {
    final auth = context.read<AuthState>();
    if (auth.employeeId <= 0) {
      setState(() => _msg = '账号未绑定员工，无法请假');
      return;
    }
    final today = DateTime.now();
    final start = '${today.toIso8601String().substring(0, 10)} 09:00:00';
    final end = '${today.toIso8601String().substring(0, 10)} 18:00:00';
    final r = await auth.api.post('/hr/leave-requests', {
      'employee_id': auth.employeeId,
      'leave_type': _leaveType,
      'start_at': start,
      'end_at': end,
      'remark': _leaveRemark.text.trim().isEmpty ? '手机端请假' : _leaveRemark.text.trim(),
    });
    setState(() => _msg = r.ok ? '请假已提交' : r.msg);
    if (r.ok) await _loadLeaves();
  }

  Future<void> _loadTasks() async {
    final r = await context.read<AuthState>().api.get('/approval/tasks');
    if (!mounted) return;
    setState(() => _tasks = ApiClient.listOf(r.data));
  }

  Future<void> _decide(Map row, {required bool approve}) async {
    final id = (row['id'] as num?)?.toInt();
    if (id == null) return;
    final path = approve ? '/approval/tasks/$id/approve' : '/approval/tasks/$id/reject';
    final r = await context.read<AuthState>().api.post(path, {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? (approve ? '已通过' : '已驳回') : r.msg)));
    if (r.ok) await _loadTasks();
  }

  Future<void> _loadNotes() async {
    final auth = context.read<AuthState>();
    final eid = auth.employeeId;
    final results = await Future.wait([
      auth.api.get(eid > 0 ? '/hr/employee-journals?employee_id=$eid' : '/hr/employee-journals'),
      auth.api.get('/hr/memos'),
    ]);
    if (!mounted) return;
    setState(() {
      _journals = ApiClient.listOf(results[0].data);
      _memos = ApiClient.listOf(results[1].data);
    });
  }

  Future<void> _createJournal() async {
    final auth = context.read<AuthState>();
    final content = _journalContent.text.trim();
    if (content.isEmpty) return;
    var eid = auth.employeeId;
    if (eid <= 0) {
      await auth.fetchMe();
      eid = auth.employeeId;
    }
    if (eid <= 0) {
      setState(() => _msg = '无员工档案，无法写日志');
      return;
    }
    final r = await auth.api.post('/hr/employee-journals', {
      'employee_id': eid,
      'biz_date': DateTime.now().toIso8601String().substring(0, 10),
      'content': content,
    });
    if (!mounted) return;
    setState(() => _msg = r.ok ? '日志已保存' : r.msg);
    if (r.ok) {
      _journalContent.clear();
      await _loadNotes();
    }
  }

  Future<void> _createMemo() async {
    final title = _memoTitle.text.trim();
    if (title.isEmpty) return;
    final r = await context.read<AuthState>().api.post('/hr/memos', {
      'title': title,
      'content': _memoContent.text.trim(),
      'biz_date': DateTime.now().toIso8601String().substring(0, 10),
      'scope_type': 'hr',
    });
    if (!mounted) return;
    setState(() => _msg = r.ok ? '备忘录已保存' : r.msg);
    if (r.ok) {
      _memoTitle.clear();
      _memoContent.clear();
      await _loadNotes();
    }
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthState>();
    final notify = context.watch<NotifyService>();
    return Scaffold(
      appBar: AppBar(
        title: Text('我的 · ${auth.name ?? auth.loginName ?? ''}'),
        actions: [
          IconButton(
            tooltip: '账户',
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => const AccountCenterPage()),
            ),
            icon: const Icon(Icons.manage_accounts_outlined),
          ),
        ],
      ),
      body: IndexedStack(
        index: _tab,
        children: [
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Card(
                child: ListTile(
                  title: Text(auth.name?.isNotEmpty == true ? auth.name! : (auth.loginName ?? '-')),
                  subtitle: Text('员工ID ${auth.employeeId} · 角色 ${auth.roles.join(', ')}'),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => Navigator.of(context).push(
                    MaterialPageRoute(builder: (_) => const AccountCenterPage()),
                  ),
                ),
              ),
              if (auth.canHrOnboard) ...[
                const SizedBox(height: 8),
                Card(
                  child: ListTile(
                    leading: const Icon(Icons.person_add_alt_1),
                    title: const Text('人事开户'),
                    subtitle: const Text('新建员工档案并开登录账号'),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: () => Navigator.of(context).push(
                      MaterialPageRoute(builder: (_) => const HrOnboardPage()),
                    ),
                  ),
                ),
              ],
              const SizedBox(height: 8),
              Card(
                child: ListTile(
                  leading: const Icon(Icons.handyman_outlined),
                  title: const Text('物料工具'),
                  subtitle: const Text('申请领取 / 归还'),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => Navigator.of(context).pushNamed('/tools'),
                ),
              ),
              Card(
                child: ListTile(
                  leading: const Icon(Icons.assignment_outlined),
                  title: const Text('工单'),
                  subtitle: const Text('待我处理 / 我发起的'),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => Navigator.of(context).pushNamed('/tickets'),
                ),
              ),
              const SizedBox(height: 12),
              const Text('今日打卡', style: TextStyle(fontWeight: FontWeight.bold)),
              const SizedBox(height: 8),
              Row(
                children: [
                  Expanded(child: FilledButton(onPressed: () => _punch('in'), child: const Text('上班打卡'))),
                  const SizedBox(width: 12),
                  Expanded(child: FilledButton.tonal(onPressed: () => _punch('out'), child: const Text('下班打卡'))),
                ],
              ),
              if (_lastPunch != null)
                Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: Text('上次 ${_lastPunch!['punch_type']} @ ${_lastPunch!['at']}'),
                ),
              const Divider(height: 32),
              const Text('今日计件核对', style: TextStyle(fontWeight: FontWeight.bold)),
              ListTile(
                title: const Text('预计工钱'),
                trailing: Text('¥${_wage?['total_amount'] ?? 0}', style: const TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
              ),
              ListTile(
                title: const Text('完工重量'),
                trailing: Text('${_wage?['total_output_weight'] ?? _wage?['total_qty'] ?? 0} kg'),
              ),
              TextButton(onPressed: _loadWage, child: const Text('刷新核对')),
              if (_msg.isNotEmpty) Text(_msg, style: const TextStyle(color: Colors.teal)),
            ],
          ),
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              const Text('请假', style: TextStyle(fontWeight: FontWeight.bold)),
              DropdownButtonFormField<String>(
                value: _leaveType,
                decoration: const InputDecoration(labelText: '请假类型'),
                items: const [
                  DropdownMenuItem(value: 'annual', child: Text('年假')),
                  DropdownMenuItem(value: 'sick', child: Text('病假')),
                  DropdownMenuItem(value: 'personal', child: Text('事假')),
                ],
                onChanged: (v) => setState(() => _leaveType = v ?? 'annual'),
              ),
              TextField(controller: _leaveRemark, decoration: const InputDecoration(labelText: '备注')),
              const SizedBox(height: 8),
              FilledButton(onPressed: _createLeave, child: const Text('提交今日请假')),
              const Divider(),
              const Text('加班 / 补卡', style: TextStyle(fontWeight: FontWeight.bold)),
              SegmentedButton<String>(
                segments: const [
                  ButtonSegment(value: 'overtime', label: Text('加班')),
                  ButtonSegment(value: 'patch', label: Text('补卡')),
                ],
                selected: {_otType},
                onSelectionChanged: (s) => setState(() => _otType = s.first),
              ),
              TextField(controller: _otMinutes, decoration: const InputDecoration(labelText: '分钟'), keyboardType: TextInputType.number),
              TextField(controller: _otRemark, decoration: const InputDecoration(labelText: '说明')),
              FilledButton.tonal(onPressed: _createOt, child: const Text('提交加班/补卡')),
              if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_msg)),
              const Divider(),
              ..._leaves.take(10).map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('假 ${m['doc_no']} · ${m['leave_type']}'),
                  trailing: Text('${m['status']}'),
                );
              }),
              ..._ots.take(10).map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['biz_type']} ${m['doc_no']}'),
                  subtitle: Text('${m['biz_date']} · ${m['minutes']}分钟'),
                  trailing: Text('${m['status']}'),
                );
              }),
            ],
          ),
          ListView(
            padding: const EdgeInsets.all(12),
            children: [
              if (_tasks.isEmpty) const Padding(padding: EdgeInsets.all(24), child: Center(child: Text('暂无审批待办'))),
              ..._tasks.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                final st = m['status']?.toString() ?? '';
                return Card(
                  child: ListTile(
                    title: Text('${m['title'] ?? m['doc_no'] ?? m['id']}'),
                    subtitle: Text('${m['doc_type'] ?? ''} · ¥${m['amount'] ?? 0}\n$st'),
                    isThreeLine: true,
                    trailing: st == 'pending'
                        ? Wrap(
                            direction: Axis.vertical,
                            children: [
                              TextButton(onPressed: () => _decide(m, approve: true), child: const Text('通过')),
                              TextButton(onPressed: () => _decide(m, approve: false), child: const Text('驳回')),
                            ],
                          )
                        : null,
                  ),
                );
              }),
            ],
          ),
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              const Text('本人工资 / 提成（只读）', style: TextStyle(fontWeight: FontWeight.bold)),
              ListTile(
                title: const Text('今日计件'),
                trailing: Text('¥${_wage?['total_amount'] ?? 0}'),
              ),
              const Divider(),
              const Text('工资单', style: TextStyle(fontWeight: FontWeight.w600)),
              if (_sheets.isEmpty) const Text('暂无工资单'),
              ..._sheets.take(10).map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['doc_no'] ?? m['id']}'),
                  subtitle: Text('${m['period_year']}-${m['period_month']} · ${m['status']}'),
                );
              }),
              const Divider(),
              const Text('销售提成', style: TextStyle(fontWeight: FontWeight.w600)),
              if (_commissions.isEmpty) const Text('暂无提成记录'),
              ..._commissions.take(10).map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('期间 ${m['period'] ?? ''}'),
                  subtitle: Text('基数 ¥${m['base_amount'] ?? 0}'),
                  trailing: Text('¥${m['commission_amount'] ?? 0}'),
                );
              }),
            ],
          ),
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Text('MQTT ${notify.mqttStatus} · 未读 ${notify.unread}', style: const TextStyle(color: Colors.black54)),
              const SizedBox(height: 8),
              FilledButton.tonal(
                onPressed: () => Navigator.of(context).pushNamed('/inbox'),
                child: const Text('打开消息收件箱'),
              ),
              const Divider(),
              if (notify.inbox.isEmpty) const Text('暂无消息'),
              ...notify.inbox.take(15).map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text(m['title']?.toString() ?? m['event_key']?.toString() ?? ''),
                  subtitle: Text(m['body']?.toString() ?? ''),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => notify.openInboxItem(context, m),
                );
              }),
            ],
          ),
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              const Text('员工日志', style: TextStyle(fontWeight: FontWeight.bold)),
              TextField(controller: _journalContent, decoration: const InputDecoration(labelText: '今日工作记录'), maxLines: 3),
              FilledButton(onPressed: _createJournal, child: const Text('提交日志')),
              const SizedBox(height: 8),
              ..._journals.take(10).map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['biz_date'] ?? ''}'),
                  subtitle: Text('${m['content'] ?? ''}'),
                );
              }),
              const Divider(height: 32),
              const Text('个人备忘录', style: TextStyle(fontWeight: FontWeight.bold)),
              TextField(controller: _memoTitle, decoration: const InputDecoration(labelText: '标题')),
              TextField(controller: _memoContent, decoration: const InputDecoration(labelText: '内容'), maxLines: 2),
              FilledButton.tonal(onPressed: _createMemo, child: const Text('保存备忘')),
              if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_msg)),
              ..._memos.take(10).map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['title'] ?? ''}'),
                  subtitle: Text('${m['content'] ?? ''}'),
                );
              }),
            ],
          ),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _tab,
        onDestinationSelected: (i) {
          setState(() => _tab = i);
          if (i == 0) _loadWage();
          if (i == 1) {
            _loadLeaves();
            _loadOts();
          }
          if (i == 2) _loadTasks();
          if (i == 3) _loadPay();
          if (i == 4) notify.refresh();
          if (i == 5) _loadNotes();
        },
        destinations: const [
          NavigationDestination(icon: Icon(Icons.fingerprint), label: '打卡'),
          NavigationDestination(icon: Icon(Icons.event_busy), label: '假勤'),
          NavigationDestination(icon: Icon(Icons.fact_check), label: '审批'),
          NavigationDestination(icon: Icon(Icons.account_balance_wallet), label: '工资'),
          NavigationDestination(icon: Icon(Icons.mail), label: '消息'),
          NavigationDestination(icon: Icon(Icons.edit_note), label: '日志'),
        ],
      ),
    );
  }
}
