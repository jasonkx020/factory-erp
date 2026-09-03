import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/notify_service.dart';
import '../../theme/plant_colors.dart';
import '../finance/finance_hub_page.dart';
import '../mine/mine_page.dart';
import '../ticket/ticket_todo_page.dart';
import '../ticket/ticket_widgets.dart';

/// 财务角色壳：结算中心 / 付款待办 / 我的
class FinanceShell extends StatefulWidget {
  const FinanceShell({super.key});

  @override
  State<FinanceShell> createState() => _FinanceShellState();
}

class _FinanceShellState extends State<FinanceShell> {
  int _index = 0;
  final _todoKey = GlobalKey<TicketTodoPageState>();
  TicketRefreshBus? _bus;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<NotifyService>().consumePendingLaunch();
      _bus = context.read<TicketRefreshBus>();
      _bus?.addListener(_onRefresh);
    });
  }

  @override
  void dispose() {
    _bus?.removeListener(_onRefresh);
    super.dispose();
  }

  void _onRefresh() => _todoKey.currentState?.reload();

  @override
  Widget build(BuildContext context) {
    final notify = context.watch<NotifyService>();
    return Scaffold(
      extendBody: true,
      body: Stack(
        children: [
          IndexedStack(
            index: _index,
            children: [
              const FinanceHubPage(asTab: true),
              TicketTodoPage(key: _todoKey),
              const MinePage(asTab: true),
            ],
          ),
          SafeArea(
            child: Align(
              alignment: Alignment.topRight,
              child: IconButton(
                tooltip: '消息',
                onPressed: () => Navigator.of(context).pushNamed('/inbox'),
                icon: Badge(
                  isLabelVisible: notify.unread > 0,
                  label: Text('${notify.unread > 99 ? '99+' : notify.unread}'),
                  child: const Icon(Icons.notifications_outlined, color: PlantColors.forest),
                ),
              ),
            ),
          ),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _index,
        onDestinationSelected: (i) {
          setState(() => _index = i);
          if (i == 1) _todoKey.currentState?.reload();
        },
        destinations: const [
          NavigationDestination(icon: Icon(Icons.account_balance_outlined), label: '结算'),
          NavigationDestination(icon: Icon(Icons.checklist_outlined), label: '待办'),
          NavigationDestination(icon: Icon(Icons.person_outline), label: '我的'),
        ],
      ),
    );
  }
}
