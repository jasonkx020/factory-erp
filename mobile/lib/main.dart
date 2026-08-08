import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'core/api_client.dart';
import 'core/auth_state.dart';
import 'core/notify_service.dart';
import 'features/assets/assets_page.dart';
import 'features/auth/login_page.dart';
import 'features/collab/collab_finance_page.dart';
import 'features/hr/hr_onboard_page.dart';
import 'features/hr/ticket_create_page.dart';
import 'features/hr/tickets_page.dart';
import 'features/hr/tool_issue_page.dart';
import 'features/knowledge/knowledge_page.dart';
import 'features/mine/account_center_page.dart';
import 'features/mine/mine_page.dart';
import 'features/notify/inbox_page.dart';
import 'features/receiving/receiving_page.dart';
import 'features/sales/sales_page.dart';
import 'features/shell/factory_shell.dart';
import 'features/shell/main_shell.dart';
import 'features/station/station_pass_page.dart';
import 'features/ticket/ticket_widgets.dart';
import 'features/warehouse/warehouse_page.dart';
import 'features/workshop/workshop_page.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final api = ApiClient();
  await api.loadToken();
  final auth = AuthState(api);
  final notify = NotifyService(api);
  final ticketRefresh = TicketRefreshBus();
  if (api.accessToken != null) {
    await auth.fetchMe();
    await notify.start();
  }
  runApp(ErpEmployeeApp(auth: auth, notify: notify, ticketRefresh: ticketRefresh));
}

class ErpEmployeeApp extends StatelessWidget {
  const ErpEmployeeApp({
    super.key,
    required this.auth,
    required this.notify,
    required this.ticketRefresh,
  });

  final AuthState auth;
  final NotifyService notify;
  final TicketRefreshBus ticketRefresh;

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: auth),
        ChangeNotifierProvider.value(value: notify),
        ChangeNotifierProvider.value(value: ticketRefresh),
      ],
      child: MaterialApp(
        title: '加工厂员工端',
        navigatorKey: appNavigatorKey,
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF0D7A6F)),
          useMaterial3: true,
        ),
        home: Consumer<AuthState>(
          builder: (context, a, _) {
            if (!a.isLoggedIn) return const LoginPage();
            return const _ShellWithLaunchLink();
          },
        ),
        routes: {
          '/workshop': (_) => const WorkshopPage(),
          '/worker': (_) => const StationPassPage(),
          '/station': (_) => const StationPassPage(),
          '/receiving': (_) => const ReceivingPage(),
          '/warehouse': (_) => const WarehousePage(),
          '/sales': (_) => const SalesPage(),
          '/assets': (_) => const AssetsPage(),
          '/collab': (_) => const CollabFinancePage(),
          '/knowledge': (_) => const KnowledgePage(),
          '/mine': (_) => const MinePage(),
          '/account': (_) => const AccountCenterPage(),
          '/hr-onboard': (_) => const HrOnboardPage(),
          '/tools': (_) => const ToolIssuePage(),
          '/tickets': (_) => const TicketsPage(),
          '/ticket-create': (_) => const TicketCreatePage(),
          '/inbox': (_) => const InboxPage(),
          '/home': (_) => const MainShell(),
        },
      ),
    );
  }
}

/// 登录后主壳；冷启动通知深链消费一次
class _ShellWithLaunchLink extends StatefulWidget {
  const _ShellWithLaunchLink();

  @override
  State<_ShellWithLaunchLink> createState() => _ShellWithLaunchLinkState();
}

class _ShellWithLaunchLinkState extends State<_ShellWithLaunchLink> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<NotifyService>().consumePendingLaunch();
    });
  }

  @override
  Widget build(BuildContext context) => const FactoryShell();
}
