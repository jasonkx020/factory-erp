import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/auth_state.dart';
import '../../core/notify_service.dart';
import '../../core/role_codes.dart';
import '../mine/mine_page.dart';
import '../receiving/receiving_page.dart';
import '../station/station_pass_page.dart';
import '../warehouse/warehouse_page.dart';
import '../workshop/workshop_page.dart';

/// 木薯产线默认壳：按角色裁剪底部 Tab（生产 / 采购 / 仓管 / 班组 / 我的）。
/// 无外层标题栏，小屏把高度留给业务表单。
class FactoryShell extends StatefulWidget {
  const FactoryShell({super.key});

  @override
  State<FactoryShell> createState() => _FactoryShellState();
}

class _TabSpec {
  const _TabSpec({required this.label, required this.icon, required this.builder});
  final String label;
  final IconData icon;
  final Widget Function() builder;
}

class _FactoryShellState extends State<FactoryShell> {
  int _index = 0;
  WorkbenchRole? _builtFor;

  List<_TabSpec> _tabsFor(WorkbenchRole role) {
    final tabs = <_TabSpec>[];
    void addStation() => tabs.add(_TabSpec(
          label: '生产',
          icon: Icons.precision_manufacturing,
          builder: () => const StationPassPage(asTab: true),
        ));
    void addReceiving() => tabs.add(_TabSpec(
          label: '采购',
          icon: Icons.scale,
          builder: () => const ReceivingPage(asTab: true),
        ));
    void addWarehouse() => tabs.add(_TabSpec(
          label: '仓管',
          icon: Icons.warehouse,
          builder: () => const WarehousePage(asTab: true),
        ));
    void addTeam() => tabs.add(_TabSpec(
          label: '班组',
          icon: Icons.groups,
          builder: () => const WorkshopPage(asTab: true),
        ));

    switch (role) {
      case WorkbenchRole.worker:
        addStation();
      case WorkbenchRole.receiving:
        addReceiving();
      case WorkbenchRole.warehouse:
        addWarehouse();
      case WorkbenchRole.workshop:
        addStation();
        addTeam();
      case WorkbenchRole.admin:
        addStation();
        addReceiving();
        addWarehouse();
        addTeam();
      default:
        addStation();
    }
    tabs.add(_TabSpec(label: '我的', icon: Icons.person, builder: () => const MinePage(asTab: true)));
    return tabs;
  }

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<NotifyService>().consumePendingLaunch();
    });
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthState>();
    final notify = context.watch<NotifyService>();
    final role = auth.primaryRole;
    if (_builtFor != role) {
      _builtFor = role;
      _index = 0;
    }
    final tabs = _tabsFor(role);
    if (_index >= tabs.length) _index = 0;

    final switchable = auth.switchableRoles;
    return Scaffold(
      body: Stack(
        children: [
          IndexedStack(
            index: _index,
            children: tabs.map((t) => t.builder()).toList(),
          ),
          SafeArea(
            child: Align(
              alignment: Alignment.topRight,
              child: Material(
                color: Colors.transparent,
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (switchable.length > 1)
                      PopupMenuButton<WorkbenchRole>(
                        tooltip: '切换角色',
                        onSelected: auth.setPrimaryRole,
                        itemBuilder: (ctx) => switchable
                            .map((r) => PopupMenuItem(value: r, child: Text(workbenchRoleLabel(r))))
                            .toList(),
                        icon: const Icon(Icons.swap_horiz, size: 22),
                        padding: EdgeInsets.zero,
                        constraints: const BoxConstraints(minWidth: 36, minHeight: 36),
                      ),
                    IconButton(
                      tooltip: '消息',
                      visualDensity: VisualDensity.compact,
                      onPressed: () => Navigator.of(context).pushNamed('/inbox'),
                      icon: Badge(
                        isLabelVisible: notify.unread > 0,
                        label: Text('${notify.unread > 99 ? '99+' : notify.unread}'),
                        child: const Icon(Icons.notifications_outlined, size: 22),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _index,
        onDestinationSelected: (i) => setState(() => _index = i),
        destinations: tabs.map((t) => NavigationDestination(icon: Icon(t.icon), label: t.label)).toList(),
      ),
    );
  }
}
