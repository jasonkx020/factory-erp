import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/auth_state.dart';
import '../../core/notify_service.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _login = TextEditingController(text: 'admin');
  final _password = TextEditingController(text: 'admin123');

  @override
  void dispose() {
    _login.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final auth = context.read<AuthState>();
    final ok = await auth.login(_login.text.trim(), _password.text);
    if (!ok && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(auth.error)));
      return;
    }
    if (ok && mounted) {
      await context.read<NotifyService>().start();
    }
  }

  @override
  Widget build(BuildContext context) {
    final loading = context.watch<AuthState>().loading;
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 360),
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text('员工端', style: Theme.of(context).textTheme.headlineMedium),
                  const SizedBox(height: 8),
                  Text(
                    'Android / iOS · client_type=mobile · 按权限开放模块',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  const SizedBox(height: 24),
                  TextField(controller: _login, decoration: const InputDecoration(labelText: '用户名')),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _password,
                    obscureText: true,
                    decoration: const InputDecoration(labelText: '密码'),
                  ),
                  const SizedBox(height: 20),
                  FilledButton(
                    onPressed: loading ? null : _submit,
                    child: loading ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2)) : const Text('登录'),
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
