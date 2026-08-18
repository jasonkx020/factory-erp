import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/auth_state.dart';
import '../receiving/gate_inbound_prefs.dart';
import '../receiving/weigh_ticket_local_store.dart';
import '../warehouse/box_stockin_draft.dart';

/// 个人账户：改密 + 第三方绑定占位
class AccountCenterPage extends StatefulWidget {
  const AccountCenterPage({super.key});

  @override
  State<AccountCenterPage> createState() => _AccountCenterPageState();
}

class _AccountCenterPageState extends State<AccountCenterPage> {
  final _oldPwd = TextEditingController();
  final _newPwd = TextEditingController();
  final _confirmPwd = TextEditingController();
  String _msg = '';
  bool _busy = false;
  List<dynamic> _providers = [];

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadBindings());
  }

  @override
  void dispose() {
    _oldPwd.dispose();
    _newPwd.dispose();
    _confirmPwd.dispose();
    super.dispose();
  }

  Future<void> _loadBindings() async {
    final r = await context.read<AuthState>().api.get('/auth/oauth/bindings');
    if (!mounted) return;
    setState(() {
      if (r.ok && r.data is Map) {
        final m = Map<String, dynamic>.from(r.data as Map);
        _providers = (m['providers'] as List?) ?? [];
      }
    });
  }

  Future<void> _changePassword() async {
    final oldP = _oldPwd.text;
    final newP = _newPwd.text;
    final conf = _confirmPwd.text;
    if (oldP.isEmpty || newP.isEmpty) {
      setState(() => _msg = '请填写旧密码与新密码');
      return;
    }
    if (newP != conf) {
      setState(() => _msg = '两次新密码不一致');
      return;
    }
    setState(() {
      _busy = true;
      _msg = '';
    });
    final r = await context.read<AuthState>().api.post('/auth/password/change', {
      'old_password': oldP,
      'new_password': newP,
    });
    if (!mounted) return;
    setState(() {
      _busy = false;
      _msg = r.ok ? '密码已修改，请使用新密码登录' : r.msg;
      if (r.ok) {
        _oldPwd.clear();
        _newPwd.clear();
        _confirmPwd.clear();
      }
    });
  }

  Future<void> _bind(String provider) async {
    final r = await context.read<AuthState>().api.post('/auth/oauth/bind', {
      'provider': provider,
      'code': 'stub',
    });
    if (!mounted) return;
    final tip = r.msg.contains('OAUTH_NOT_CONFIGURED') || r.msg.contains('NOT_IMPLEMENTED')
        ? '第三方绑定暂未开通，请联系管理员配置'
        : r.msg;
    setState(() => _msg = r.ok ? '绑定成功' : tip);
    await _loadBindings();
  }

  Future<void> _unbind(String provider) async {
    final r = await context.read<AuthState>().api.delete('/auth/oauth/bind/$provider');
    if (!mounted) return;
    setState(() => _msg = r.ok ? '已解绑' : r.msg);
    await _loadBindings();
  }

  Future<bool> _confirm(String title, String body) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: Text(body),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('确定')),
        ],
      ),
    );
    return ok == true;
  }

  Future<void> _clearCache() async {
    final ok = await _confirm(
      '清理缓存',
      '将清除入厂向导记住的单价/扣损/运费等，以及仓管分板未提交草稿。过磅单本机备份不受影响。',
    );
    if (!ok || !mounted) return;
    await GateInboundPrefs.clear();
    await BoxStockinDraftStore.clearAll();
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('缓存已清理')));
  }

  Future<void> _clearLocalBackup() async {
    final ok = await _confirm(
      '清除备份数据',
      '将删除本机近 3 个月过磅备份，不可恢复；服务端单据不受影响。',
    );
    if (!ok || !mounted) return;
    final auth = context.read<AuthState>();
    if (auth.userId <= 0) auth.syncUserIdFromToken();
    await WeighTicketLocalStore.clear(auth.userId);
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('本机过磅备份已清除')));
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthState>();
    return Scaffold(
      appBar: AppBar(title: const Text('账户')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: ListTile(
              title: Text(auth.name?.isNotEmpty == true ? auth.name! : (auth.loginName ?? '-')),
              subtitle: Text(
                '登录名 ${auth.loginName ?? '-'}\n角色 ${auth.roles.isEmpty ? '-' : auth.roles.join(', ')}',
              ),
              isThreeLine: true,
            ),
          ),
          const SizedBox(height: 16),
          const Text('修改密码', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
          const SizedBox(height: 8),
          TextField(
            controller: _oldPwd,
            obscureText: true,
            decoration: const InputDecoration(labelText: '旧密码', border: OutlineInputBorder()),
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _newPwd,
            obscureText: true,
            decoration: const InputDecoration(labelText: '新密码', border: OutlineInputBorder()),
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _confirmPwd,
            obscureText: true,
            decoration: const InputDecoration(labelText: '确认新密码', border: OutlineInputBorder()),
          ),
          const SizedBox(height: 12),
          FilledButton(
            onPressed: _busy ? null : _changePassword,
            child: Text(_busy ? '提交中…' : '保存新密码'),
          ),
          const Divider(height: 32),
          const Text('缓存与备份', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
          const SizedBox(height: 4),
          const Text('仅清理本机数据，不影响服务端单据', style: TextStyle(color: Colors.black54, fontSize: 12)),
          const SizedBox(height: 8),
          ListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('清理缓存'),
            subtitle: const Text('入厂常用单价/扣损/运费，以及仓管分板草稿'),
            trailing: TextButton(onPressed: _clearCache, child: const Text('清理')),
          ),
          ListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('清除备份数据'),
            subtitle: const Text('删除本机近 3 个月过磅备份，不可恢复'),
            trailing: TextButton(onPressed: _clearLocalBackup, child: const Text('清除')),
          ),
          const Divider(height: 32),
          const Text('第三方账号', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
          const SizedBox(height: 4),
          const Text('未开通时仅提示，不影响正常使用', style: TextStyle(color: Colors.black54, fontSize: 12)),
          const SizedBox(height: 8),
          if (_providers.isEmpty)
            ListTile(
              title: const Text('微信'),
              subtitle: const Text('未绑定'),
              trailing: TextButton(onPressed: () => _bind('wechat'), child: const Text('绑定')),
            )
          else
            ..._providers.map((e) {
              final m = Map<String, dynamic>.from(e as Map);
              final provider = '${m['provider'] ?? 'wechat'}';
              final bound = m['bound'] == true;
              final label = '${m['label'] ?? provider}';
              return ListTile(
                title: Text(label),
                subtitle: Text(bound ? '已绑定' : '未绑定'),
                trailing: bound
                    ? TextButton(onPressed: () => _unbind(provider), child: const Text('解绑'))
                    : TextButton(onPressed: () => _bind(provider), child: const Text('绑定')),
              );
            }),
          if (_msg.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: Text(_msg, style: TextStyle(color: _msg.contains('已') || _msg.contains('成功') ? Colors.teal : Colors.orange.shade800)),
            ),
        ],
      ),
    );
  }
}
