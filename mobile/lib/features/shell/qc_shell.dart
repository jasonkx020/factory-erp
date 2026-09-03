import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/auth_state.dart';
import '../../core/notify_service.dart';
import '../mine/mine_page.dart';
import '../ticket/ticket_done_page.dart';
import '../ticket/ticket_home_page.dart';
import '../ticket/ticket_todo_page.dart';
import '../ticket/ticket_widgets.dart';

/// 质检专用壳：待办 / 履历 / 我的。不含采购建单、不含发单加号。
class QcShell extends StatefulWidget {
  const QcShell({super.key});

  @override
  State<QcShell> createState() => _QcShellState();
}

class _QcShellState extends State<QcShell> {
  int _index = 0;
  final _homeKey = GlobalKey<TicketHomePageState>();
  final _todoKey = GlobalKey<TicketTodoPageState>();
  final _doneKey = GlobalKey<TicketDonePageState>();
  TicketRefreshBus? _bus;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _bus = context.read<TicketRefreshBus>();
      _bus?.addListener(_onRefresh);
      context.read<NotifyService>().consumePendingLaunch();
    });
  }

  @override
  void dispose() {
    _bus?.removeListener(_onRefresh);
    super.dispose();
  }

  void _onRefresh() {
    _homeKey.currentState?.reload();
    _todoKey.currentState?.reload();
    _doneKey.currentState?.reload();
  }

  @override
  Widget build(BuildContext context) {
    final notify = context.watch<NotifyService>();
    final auth = context.watch<AuthState>();
    final name = (auth.name?.isNotEmpty == true) ? auth.name! : (auth.loginName ?? '质检');

    return Scaffold(
      extendBody: true,
      body: IndexedStack(
        index: _index,
        children: [
          TicketHomePage(key: _homeKey),
          TicketTodoPage(key: _todoKey),
          TicketDonePage(key: _doneKey),
          const MinePage(asTab: true),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _index,
        onDestinationSelected: (i) {
          setState(() => _index = i);
          if (i == 0) _homeKey.currentState?.reload();
          if (i == 1) _todoKey.currentState?.reload();
          if (i == 2) _doneKey.currentState?.reload();
        },
        destinations: [
          NavigationDestination(
            icon: Badge(
              isLabelVisible: notify.unread > 0 && _index != 0,
              label: Text('${notify.unread}'),
              child: const Icon(Icons.fact_check_outlined),
            ),
            selectedIcon: const Icon(Icons.fact_check),
            label: '质检待办',
          ),
          const NavigationDestination(
            icon: Icon(Icons.checklist_outlined),
            selectedIcon: Icon(Icons.checklist),
            label: '指派给我',
          ),
          const NavigationDestination(
            icon: Icon(Icons.history_outlined),
            selectedIcon: Icon(Icons.history),
            label: '我处理过的',
          ),
          NavigationDestination(
            icon: const Icon(Icons.person_outline),
            selectedIcon: const Icon(Icons.person),
            label: name.length > 4 ? '我的' : name,
          ),
        ],
      ),
    );
  }
}
