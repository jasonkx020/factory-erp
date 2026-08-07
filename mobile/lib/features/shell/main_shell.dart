import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/ticket_create_kinds.dart';
import '../mine/mine_page.dart';
import '../ticket/ticket_done_page.dart';
import '../ticket/ticket_home_page.dart';
import '../ticket/ticket_todo_page.dart';
import '../ticket/ticket_widgets.dart';

/// 经典五 Tab 壳：首页 / 待办 / +发单 / 已办 / 我的
class MainShell extends StatefulWidget {
  const MainShell({super.key});

  @override
  State<MainShell> createState() => _MainShellState();
}

class _MainShellState extends State<MainShell> {
  /// 内容页索引：0 首页 1 待办 2 已办 3 我的（中间加号不占栈）
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

  Future<void> _openCreate() async {
    final ok = await pickAndCreateTicket(context);
    if (ok && mounted) {
      setState(() => _index = 0);
    }
  }

  void _onNavTap(int i) {
    // 底栏 5 项：0 首页 1 待办 2 加号 3 已办 4 我的
    if (i == 2) {
      _openCreate();
      return;
    }
    final mapped = i > 2 ? i - 1 : i;
    setState(() => _index = mapped);
    if (mapped == 0) _homeKey.currentState?.reload();
    if (mapped == 1) _todoKey.currentState?.reload();
    if (mapped == 2) _doneKey.currentState?.reload();
  }

  int get _navSelected {
    // map content index back to nav: 0->0, 1->1, 2->3, 3->4
    if (_index <= 1) return _index;
    return _index + 1;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: IndexedStack(
        index: _index,
        children: [
          TicketHomePage(key: _homeKey),
          TicketTodoPage(key: _todoKey),
          TicketDonePage(key: _doneKey),
          const MinePage(asTab: true),
        ],
      ),
      floatingActionButtonLocation: FloatingActionButtonLocation.centerDocked,
      floatingActionButton: FloatingActionButton(
        onPressed: _openCreate,
        tooltip: '发布工单',
        child: const Icon(Icons.add),
      ),
      bottomNavigationBar: BottomAppBar(
        shape: const CircularNotchedRectangle(),
        notchMargin: 6,
        child: SizedBox(
          height: 56,
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceAround,
            children: [
              _NavItem(
                icon: Icons.home_outlined,
                selectedIcon: Icons.home,
                label: '首页',
                selected: _navSelected == 0,
                onTap: () => _onNavTap(0),
              ),
              _NavItem(
                icon: Icons.checklist_outlined,
                selectedIcon: Icons.checklist,
                label: '待办',
                selected: _navSelected == 1,
                onTap: () => _onNavTap(1),
              ),
              const SizedBox(width: 48),
              _NavItem(
                icon: Icons.history_outlined,
                selectedIcon: Icons.history,
                label: '已办',
                selected: _navSelected == 3,
                onTap: () => _onNavTap(3),
              ),
              _NavItem(
                icon: Icons.person_outline,
                selectedIcon: Icons.person,
                label: '我的',
                selected: _navSelected == 4,
                onTap: () => _onNavTap(4),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _NavItem extends StatelessWidget {
  const _NavItem({
    required this.icon,
    required this.selectedIcon,
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final IconData icon;
  final IconData selectedIcon;
  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final color = selected ? Theme.of(context).colorScheme.primary : Colors.black54;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: SizedBox(
        width: 64,
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(selected ? selectedIcon : icon, color: color, size: 22),
            Text(label, style: TextStyle(fontSize: 11, color: color)),
          ],
        ),
      ),
    );
  }
}
