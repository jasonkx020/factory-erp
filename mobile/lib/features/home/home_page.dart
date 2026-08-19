import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/auth_state.dart';
import '../../core/carrier_code_labels.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';
import '../../core/role_codes.dart';
import 'role_workbench_page.dart';

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
      final carrier = context.read<CarrierCodeLabels>();
      final ok = await auth.fetchMe();
      if (!ok || !mounted) return;
      await carrier.load();
      if (!mounted) return;
      await notify.start();
    });
  }

  IconData _icon(EmployeeModule m) {
    switch (m) {
      case EmployeeModule.station:
        return Icons.precision_manufacturing;
      case EmployeeModule.workshop:
        return Icons.groups;
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

  bool _useStepWorkbench(AuthState auth) {
    final role = auth.primaryRole;
    return role != WorkbenchRole.none && role != WorkbenchRole.admin;
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthState>();
    final notify = context.watch<NotifyService>();
    final codeLabel = context.watch<CarrierCodeLabels>().code;
    final mods = visibleEmployeeModules(auth.permissions, auth.roles, codeLabel: codeLabel);
    final useSteps = _useStepWorkbench(auth);

    return Scaffold(
      appBar: AppBar(
        title: Text(useSteps ? workbenchRoleLabel(auth.primaryRole) : '员工工作台'),
        actions: [
          IconButton(
            tooltip: '资料中心',
            onPressed: () => Navigator.of(context).pushNamed('/knowledge'),
            icon: const Icon(Icons.menu_book_outlined),
          ),
          IconButton(
            tooltip: '我的',
            onPressed: () => Navigator.of(context).pushNamed('/mine'),
            icon: const Icon(Icons.person_outline),
          ),
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
          : useSteps
              ? const RoleWorkbenchBody()
              : _ModuleGrid(mods: mods, iconFor: _icon),
    );
  }
}

class _ModuleGrid extends StatelessWidget {
  const _ModuleGrid({required this.mods, required this.iconFor});

  final List<EmployeeModuleInfo> mods;
  final IconData Function(EmployeeModule) iconFor;

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: mods.length,
      separatorBuilder: (_, _) => const SizedBox(height: 12),
      itemBuilder: (context, i) {
        final m = mods[i];
        return Card(
          child: ListTile(
            leading: CircleAvatar(
              backgroundColor: Theme.of(context).colorScheme.primaryContainer,
              child: Icon(iconFor(m.key)),
            ),
            title: Text(m.title, style: const TextStyle(fontWeight: FontWeight.w600)),
            subtitle: Text(m.desc),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => Navigator.of(context).pushNamed(m.route),
          ),
        );
      },
    );
  }
}
