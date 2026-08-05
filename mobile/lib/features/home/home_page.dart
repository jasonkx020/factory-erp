import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';

class HomePage extends StatefulWidget {
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      final auth = context.read<AuthState>();
      final notify = context.read<NotifyService>();
      await auth.fetchMe();
      await notify.start();
      // 仅单一作业角色时自动进入；资料/我的始终留在首页
      final mods = visibleEmployeeModules(auth.permissions, auth.roles)
          .where((m) => m.key != EmployeeModule.mine && m.key != EmployeeModule.knowledge)
          .toList();
      if (mods.length == 1 && mounted) {
        Navigator.of(context).pushNamed(mods.first.route);
      }
    });
  }

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
    final notify = context.watch<NotifyService>();
    final mods = visibleEmployeeModules(auth.permissions, auth.roles);
    return Scaffold(
      appBar: AppBar(
        title: const Text('员工工作台'),
        actions: [
          IconButton(
            onPressed: () => Navigator.of(context).pushNamed('/inbox'),
            icon: Badge(
              isLabelVisible: notify.unread > 0,
              label: Text('${notify.unread > 99 ? '99+' : notify.unread}'),
              child: const Icon(Icons.notifications_outlined),
            ),
          ),
          TextButton(
            onPressed: () async {
              await context.read<NotifyService>().stop();
              await auth.logout();
            },
            child: Text(auth.name?.isNotEmpty == true ? auth.name! : (auth.loginName ?? '退出')),
          ),
        ],
      ),
      body: mods.isEmpty
          ? const Center(child: Text('当前账号无可访问员工模块，请联系管理员授权。'))
          : ListView.separated(
              padding: const EdgeInsets.all(16),
              itemCount: mods.length,
              separatorBuilder: (_, _) => const SizedBox(height: 12),
              itemBuilder: (context, i) {
                final m = mods[i];
                return Card(
                  child: ListTile(
                    leading: CircleAvatar(
                      backgroundColor: Theme.of(context).colorScheme.primaryContainer,
                      child: Icon(_icon(m.key)),
                    ),
                    title: Text(m.title, style: const TextStyle(fontWeight: FontWeight.w600)),
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
