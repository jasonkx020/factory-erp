import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';
import '../hr/hr_onboard_page.dart';
import 'account_center_page.dart';

/// 个人中心：列表入口（无页内底栏）
class MinePage extends StatefulWidget {
  const MinePage({super.key, this.asTab = false});

  /// 作为主壳 Tab 时隐藏返回、弱化消息入口
  final bool asTab;

  @override
  State<MinePage> createState() => _MinePageState();
}

class _MinePageState extends State<MinePage> {
  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthState>();
    return Scaffold(
      appBar: AppBar(
        title: Text('我的 · ${auth.name ?? auth.loginName ?? ''}'),
        automaticallyImplyLeading: !widget.asTab,
        actions: [
          if (!widget.asTab)
            IconButton(
              tooltip: '账户',
              onPressed: () => Navigator.of(context).push(
                MaterialPageRoute(builder: (_) => const AccountCenterPage()),
              ),
              icon: const Icon(Icons.manage_accounts_outlined),
            ),
          TextButton(
            onPressed: () async {
              await context.read<NotifyService>().stop();
              await auth.logout();
            },
            child: const Text('退出'),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: ListTile(
              leading: const CircleAvatar(child: Icon(Icons.person)),
              title: Text(auth.name?.isNotEmpty == true ? auth.name! : (auth.loginName ?? '-')),
              subtitle: Text('员工ID ${auth.employeeId} · 角色 ${auth.roles.join(', ')}'),
              trailing: const Icon(Icons.chevron_right),
              onTap: () => Navigator.of(context).push(
                MaterialPageRoute(builder: (_) => const AccountCenterPage()),
              ),
            ),
          ),
          const SizedBox(height: 12),
          const Text('常用', style: TextStyle(fontWeight: FontWeight.bold)),
          const SizedBox(height: 8),
          _entry(Icons.fingerprint, '打卡', '上下班打卡 / 今日计件核对', () {
            Navigator.of(context).push(MaterialPageRoute(builder: (_) => const _PunchPage()));
          }),
          _entry(Icons.event_busy, '假勤', '请假 / 加班补卡', () {
            Navigator.of(context).push(MaterialPageRoute(builder: (_) => const _LeaveOtPage()));
          }),
          _entry(Icons.fact_check, '审批', '审批待办', () {
            Navigator.of(context).push(MaterialPageRoute(builder: (_) => const _ApprovalPage()));
          }),
          _entry(Icons.account_balance_wallet, '工资', '计件 / 工资单 / 提成（只读）', () {
            Navigator.of(context).push(MaterialPageRoute(builder: (_) => const _PayPage()));
          }),
          _entry(Icons.edit_note, '日志备忘', '员工日志与个人备忘录', () {
            Navigator.of(context).push(MaterialPageRoute(builder: (_) => const _NotesPage()));
          }),
          _entry(Icons.handyman_outlined, '物料工具', '申请领取 / 归还', () {
            Navigator.of(context).pushNamed('/tools');
          }),
          if (auth.canHrOnboard)
            _entry(Icons.person_add_alt_1, '人事开户', '新建员工档案并开登录账号', () {
              Navigator.of(context).push(MaterialPageRoute(builder: (_) => const HrOnboardPage()));
            }),
          const SizedBox(height: 16),
          const Text('更多业务', style: TextStyle(fontWeight: FontWeight.bold)),
          const SizedBox(height: 8),
          _entry(Icons.apps_outlined, '业务模块', '车间 / 仓管 / 收货等（按权限）', () {
            Navigator.of(context).push(MaterialPageRoute(builder: (_) => const _ModulesPage()));
          }),
          _entry(Icons.menu_book_outlined, '资料中心', '知识库 / 图纸 / 公告', () {
            Navigator.of(context).pushNamed('/knowledge');
          }),
          _entry(Icons.manage_accounts_outlined, '账户中心', '改密 / 第三方绑定', () {
            Navigator.of(context).push(MaterialPageRoute(builder: (_) => const AccountCenterPage()));
          }),
        ],
      ),
    );
  }

  Widget _entry(IconData icon, String title, String subtitle, VoidCallback onTap) {
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: Icon(icon),
        title: Text(title),
        subtitle: Text(subtitle),
        trailing: const Icon(Icons.chevron_right),
        onTap: onTap,
      ),
    );
  }
}

class _PunchPage extends StatefulWidget {
  const _PunchPage();
  @override
  State<_PunchPage> createState() => _PunchPageState();
}

class _PunchPageState extends State<_PunchPage> {
  String _msg = '';
  Map<String, dynamic>? _lastPunch;
  Map<String, dynamic>? _wage;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadWage());
  }

  Future<void> _loadWage() async {
    final r = await context.read<AuthState>().api.get('/production/piecework-summaries/mine');
    if (!mounted) return;
    setState(() => _wage = r.data is Map ? Map<String, dynamic>.from(r.data as Map) : null);
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

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('打卡')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
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
    );
  }
}

class _LeaveOtPage extends StatefulWidget {
  const _LeaveOtPage();
  @override
  State<_LeaveOtPage> createState() => _LeaveOtPageState();
}

class _LeaveOtPageState extends State<_LeaveOtPage> {
  String _msg = '';
  List<dynamic> _leaves = [];
  List<dynamic> _ots = [];
  final _leaveRemark = TextEditingController();
  final _otMinutes = TextEditingController(text: '60');
  final _otRemark = TextEditingController();
  String _leaveType = 'annual';
  String _otType = 'overtime';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      await _loadLeaves();
      await _loadOts();
    });
  }

  @override
  void dispose() {
    _leaveRemark.dispose();
    _otMinutes.dispose();
    _otRemark.dispose();
    super.dispose();
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

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('假勤')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const Text('请假', style: TextStyle(fontWeight: FontWeight.bold)),
          DropdownButtonFormField<String>(
            initialValue: _leaveType,
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
            return ListTile(title: Text('假 ${m['doc_no']} · ${m['leave_type']}'), trailing: Text('${m['status']}'));
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
    );
  }
}

class _ApprovalPage extends StatefulWidget {
  const _ApprovalPage();
  @override
  State<_ApprovalPage> createState() => _ApprovalPageState();
}

class _ApprovalPageState extends State<_ApprovalPage> {
  List<dynamic> _tasks = [];

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
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
    if (r.ok) await _load();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('审批')),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
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
      ),
    );
  }
}

class _PayPage extends StatefulWidget {
  const _PayPage();
  @override
  State<_PayPage> createState() => _PayPageState();
}

class _PayPageState extends State<_PayPage> {
  Map<String, dynamic>? _wage;
  List<dynamic> _sheets = [];
  List<dynamic> _commissions = [];

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
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

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('工资')),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            const Text('本人工资 / 提成（只读）', style: TextStyle(fontWeight: FontWeight.bold)),
            ListTile(title: const Text('今日计件'), trailing: Text('¥${_wage?['total_amount'] ?? 0}')),
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
      ),
    );
  }
}

class _NotesPage extends StatefulWidget {
  const _NotesPage();
  @override
  State<_NotesPage> createState() => _NotesPageState();
}

class _NotesPageState extends State<_NotesPage> {
  String _msg = '';
  List<dynamic> _journals = [];
  List<dynamic> _memos = [];
  final _journalContent = TextEditingController();
  final _memoTitle = TextEditingController();
  final _memoContent = TextEditingController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  @override
  void dispose() {
    _journalContent.dispose();
    _memoTitle.dispose();
    _memoContent.dispose();
    super.dispose();
  }

  Future<void> _load() async {
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
      await _load();
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
      await _load();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('日志备忘')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const Text('员工日志', style: TextStyle(fontWeight: FontWeight.bold)),
          TextField(controller: _journalContent, decoration: const InputDecoration(labelText: '今日工作记录'), maxLines: 3),
          FilledButton(onPressed: _createJournal, child: const Text('提交日志')),
          const SizedBox(height: 8),
          ..._journals.take(10).map((e) {
            final m = Map<String, dynamic>.from(e as Map);
            return ListTile(title: Text('${m['biz_date'] ?? ''}'), subtitle: Text('${m['content'] ?? ''}'));
          }),
          const Divider(height: 32),
          const Text('个人备忘录', style: TextStyle(fontWeight: FontWeight.bold)),
          TextField(controller: _memoTitle, decoration: const InputDecoration(labelText: '标题')),
          TextField(controller: _memoContent, decoration: const InputDecoration(labelText: '内容'), maxLines: 2),
          FilledButton.tonal(onPressed: _createMemo, child: const Text('保存备忘')),
          if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_msg)),
          ..._memos.take(10).map((e) {
            final m = Map<String, dynamic>.from(e as Map);
            return ListTile(title: Text('${m['title'] ?? ''}'), subtitle: Text('${m['content'] ?? ''}'));
          }),
        ],
      ),
    );
  }
}

class _ModulesPage extends StatelessWidget {
  const _ModulesPage();

  IconData _icon(EmployeeModule m) {
    switch (m) {
      case EmployeeModule.workshop:
        return Icons.precision_manufacturing;
      case EmployeeModule.worker:
        return Icons.badge;
      case EmployeeModule.receiving:
        return Icons.scale;
      case EmployeeModule.warehouse:
        return Icons.warehouse;
      case EmployeeModule.sales:
        return Icons.storefront;
      case EmployeeModule.assets:
        return Icons.handyman;
      case EmployeeModule.collab:
        return Icons.payments;
      case EmployeeModule.knowledge:
        return Icons.menu_book;
      case EmployeeModule.mine:
        return Icons.person;
    }
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthState>();
    final mods = visibleEmployeeModules(auth.permissions, auth.roles)
        .where((m) => m.key != EmployeeModule.mine)
        .toList();
    return Scaffold(
      appBar: AppBar(title: const Text('业务模块')),
      body: mods.isEmpty
          ? const Center(child: Text('当前账号无可访问业务模块'))
          : ListView.separated(
              padding: const EdgeInsets.all(16),
              itemCount: mods.length,
              separatorBuilder: (_, _) => const SizedBox(height: 8),
              itemBuilder: (context, i) {
                final m = mods[i];
                return Card(
                  child: ListTile(
                    leading: CircleAvatar(
                      backgroundColor: Theme.of(context).colorScheme.primaryContainer,
                      child: Icon(_icon(m.key)),
                    ),
                    title: Text(m.title),
                    subtitle: Text(m.desc),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: () => Navigator.of(context).pushNamed(m.route),
                  ),
                );
              },
            ),
    );
  }
}
