import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/carrier_code_labels.dart';
import '../../core/debug_demo_accounts.dart';
import '../../core/notify_service.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  late final TextEditingController _login;
  late final TextEditingController _password;
  late final TextEditingController _server;
  String _selectedDemoLogin = 'admin';
  String _savedBaseHint = '';

  @override
  void initState() {
    super.initState();
    final initial = showDebugDemoAccounts
        ? kDebugDemoAccounts.firstWhere((a) => a.login == 'admin', orElse: () => kDebugDemoAccounts.first)
        : null;
    _selectedDemoLogin = initial?.login ?? 'admin';
    _login = TextEditingController(text: initial?.login ?? 'admin');
    _password = TextEditingController(text: DebugDemoAccount.password);
    _server = TextEditingController();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final api = context.read<AuthState>().api;
      _server.text = displayApiHost(api.baseUrl);
      setState(() => _savedBaseHint = api.baseUrl);
    });
  }

  @override
  void dispose() {
    _login.dispose();
    _password.dispose();
    _server.dispose();
    super.dispose();
  }

  void _applyDemoAccount(DebugDemoAccount acc) {
    setState(() {
      _selectedDemoLogin = acc.login;
      _login.text = acc.login;
      _password.text = DebugDemoAccount.password;
    });
  }

  Future<void> _applyServer() async {
    final api = context.read<AuthState>().api;
    await api.setBaseUrl(_server.text);
    if (!mounted) return;
    setState(() {
      _server.text = displayApiHost(api.baseUrl);
      _savedBaseHint = api.baseUrl;
    });
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('已保存后台地址：${api.baseUrl}')),
    );
  }

  Future<void> _submit() async {
    final auth = context.read<AuthState>();
    await auth.api.setBaseUrl(_server.text);
    if (!mounted) return;
    setState(() {
      _server.text = displayApiHost(auth.api.baseUrl);
      _savedBaseHint = auth.api.baseUrl;
    });
    final ok = await auth.login(_login.text.trim(), _password.text);
    if (!ok && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(auth.error)));
      return;
    }
    if (ok && mounted) {
      await context.read<CarrierCodeLabels>().load(force: true);
      await context.read<NotifyService>().start();
    }
  }

  Future<void> _loginAsDemo(DebugDemoAccount acc) async {
    _applyDemoAccount(acc);
    await _submit();
  }

  Future<void> _oauthStub() async {
    final auth = context.read<AuthState>();
    await auth.api.setBaseUrl(_server.text);
    if (!mounted) return;
    final ok = await auth.loginWithOAuth(provider: 'wechat', code: 'stub');
    if (!mounted) return;
    if (!ok) {
      final msg = auth.error == 'OAUTH_NOT_CONFIGURED' || auth.error.contains('OAUTH')
          ? '第三方登录暂未开通'
          : auth.error;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
      return;
    }
    await context.read<CarrierCodeLabels>().load(force: true);
    await context.read<NotifyService>().start();
  }

  @override
  Widget build(BuildContext context) {
    final loading = context.watch<AuthState>().loading;
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 400),
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text('员工端', style: Theme.of(context).textTheme.headlineMedium),
                  const SizedBox(height: 8),
                  Text(
                    '登录后按角色展示作业步骤 · client_type=mobile',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: _server,
                    keyboardType: TextInputType.url,
                    decoration: InputDecoration(
                      labelText: '后台地址（IP/主机）',
                      hintText: '192.168.1.100 或 10.0.2.2:18080',
                      helperText: _savedBaseHint.isEmpty ? '默认模拟器网关 10.0.2.2:18080' : _savedBaseHint,
                      helperMaxLines: 2,
                      border: const OutlineInputBorder(),
                      suffixIcon: IconButton(
                        tooltip: '保存地址',
                        onPressed: loading ? null : _applyServer,
                        icon: const Icon(Icons.save_outlined),
                      ),
                    ),
                  ),
                  if (showDebugDemoAccounts) ...[
                    const SizedBox(height: 20),
                    Card(
                      margin: EdgeInsets.zero,
                      child: Padding(
                        padding: const EdgeInsets.fromLTRB(12, 12, 12, 8),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            Text(
                              '调试账号（密码均为 admin123）',
                              style: Theme.of(context).textTheme.titleSmall,
                            ),
                            const SizedBox(height: 10),
                            DropdownButtonFormField<String>(
                              isExpanded: true,
                              initialValue: _selectedDemoLogin,
                              decoration: const InputDecoration(
                                labelText: '选择角色用户',
                                border: OutlineInputBorder(),
                                isDense: true,
                              ),
                              items: [
                                for (final a in kDebugDemoAccounts)
                                  DropdownMenuItem(
                                    value: a.login,
                                    child: Text(a.menuLabel, overflow: TextOverflow.ellipsis),
                                  ),
                              ],
                              onChanged: loading
                                  ? null
                                  : (v) {
                                      if (v == null) return;
                                      final acc = kDebugDemoAccounts.firstWhere((e) => e.login == v);
                                      _applyDemoAccount(acc);
                                    },
                            ),
                            const SizedBox(height: 8),
                            FilledButton.tonal(
                              onPressed: loading
                                  ? null
                                  : () {
                                      final acc = kDebugDemoAccounts.firstWhere(
                                        (e) => e.login == _selectedDemoLogin,
                                        orElse: () => kDebugDemoAccounts.first,
                                      );
                                      _loginAsDemo(acc);
                                    },
                              child: const Text('用所选账号登录'),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ],
                  const SizedBox(height: 20),
                  TextField(
                    controller: _login,
                    decoration: const InputDecoration(labelText: '用户名'),
                    onChanged: (_) {
                      if (showDebugDemoAccounts &&
                          kDebugDemoAccounts.any((a) => a.login == _login.text.trim())) {
                        setState(() => _selectedDemoLogin = _login.text.trim());
                      }
                    },
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _password,
                    obscureText: true,
                    decoration: const InputDecoration(labelText: '密码'),
                  ),
                  const SizedBox(height: 20),
                  FilledButton(
                    onPressed: loading ? null : _submit,
                    child: loading
                        ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                        : const Text('账号登录'),
                  ),
                  const SizedBox(height: 12),
                  OutlinedButton.icon(
                    onPressed: loading ? null : _oauthStub,
                    icon: const Icon(Icons.chat_bubble_outline),
                    label: const Text('第三方登录'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
