import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'core/api_client.dart';
import 'core/auth_state.dart';
import 'core/carrier_code_labels.dart';
import 'core/notify_service.dart';
import 'core/role_codes.dart';
import 'features/assets/assets_page.dart';
import 'features/auth/login_page.dart';
import 'features/finance/finance_hub_page.dart';
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
import 'features/shell/finance_shell.dart';
import 'features/shell/qc_shell.dart';
import 'features/station/station_pass_page.dart';
import 'features/station/trace_production_page.dart';
import 'features/ticket/ticket_widgets.dart';
import 'features/warehouse/warehouse_page.dart';
import 'features/workshop/workshop_page.dart';
import 'theme/app_theme.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final api = ApiClient();
  await api.loadToken();
  final auth = AuthState(api);
  final carrier = CarrierCodeLabels(api);
  final notify = NotifyService(api);
  final ticketRefresh = TicketRefreshBus();
  // 本地有旧 token：先挡在校验页，等 /auth/me 成功再进主壳
  if (api.accessToken != null && api.accessToken!.isNotEmpty) {
    auth.sessionReady = false;
  }
  // 先画 UI，再后台拉会话；避免网络超时导致启动白屏
  runApp(ErpEmployeeApp(
    auth: auth,
    carrier: carrier,
    notify: notify,
    ticketRefresh: ticketRefresh,
  ));
  if (api.accessToken != null && api.accessToken!.isNotEmpty) {
    // ignore: unawaited_futures
    _bootstrapSession(auth, carrier, notify);
  }
}

Future<void> _bootstrapSession(
  AuthState auth,
  CarrierCodeLabels carrier,
  NotifyService notify,
) async {
  try {
    final ok = await auth.fetchMe().timeout(const Duration(seconds: 12), onTimeout: () => false);
    if (!ok) {
      // 过期 token：fetchMe 已 logout，勿再拉通知
      await notify.stop();
      return;
    }
    await carrier.load().timeout(const Duration(seconds: 8));
    await notify.start().timeout(const Duration(seconds: 8));
  } catch (_) {
    /* 离线/超时仍可进入登录态壳或登录页 */
    await notify.stop();
  }
}

class ErpEmployeeApp extends StatelessWidget {
  const ErpEmployeeApp({
    super.key,
    required this.auth,
    required this.carrier,
    required this.notify,
    required this.ticketRefresh,
  });

  final AuthState auth;
  final CarrierCodeLabels carrier;
  final NotifyService notify;
  final TicketRefreshBus ticketRefresh;

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: auth),
        ChangeNotifierProvider.value(value: carrier),
        ChangeNotifierProvider.value(value: notify),
        ChangeNotifierProvider.value(value: ticketRefresh),
      ],
      child: MaterialApp(
        title: '加工厂员工端',
        navigatorKey: appNavigatorKey,
        theme: AppTheme.light,
        home: Consumer<AuthState>(
          builder: (context, a, login) {
            if (!a.isLoggedIn) return login!;
            if (!a.sessionReady) {
              return const Scaffold(
                body: Center(child: CircularProgressIndicator()),
              );
            }
            return const _ShellWithLaunchLink();
          },
          child: const LoginPage(),
        ),
        routes: {
          '/workshop': (_) => const WorkshopPage(),
          '/worker': (_) => const StationPassPage(),
          '/station': (_) => const StationPassPage(),
          '/trace-production': (_) => const TraceProductionPage(),
          '/receiving': (_) => const ReceivingPage(),
          '/warehouse': (_) => const WarehousePage(),
          '/sales': (_) => const SalesPage(),
          '/assets': (_) => const AssetsPage(),
          '/collab': (_) => const FinanceHubPage(),
          '/finance': (_) => const FinanceHubPage(),
          '/knowledge': (_) => const KnowledgePage(),
          '/mine': (_) => const MinePage(),
          '/account': (_) => const AccountCenterPage(),
          '/hr-onboard': (_) => const HrOnboardPage(),
          '/tools': (_) => const ToolIssuePage(),
          '/tickets': (_) => const TicketsPage(),
          '/ticket-create': (_) => const TicketCreatePage(),
          '/inbox': (_) => const InboxPage(),
          '/home': (_) => const _ShellWithLaunchLink(),
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
  Widget build(BuildContext context) {
    final auth = context.watch<AuthState>();
    // 纯质检：专用质检壳（待办判定），绝不进采购页
    if (auth.preferQcShell || auth.primaryRole == WorkbenchRole.qc) {
      return const QcShell();
    }
    if (auth.primaryRole == WorkbenchRole.collab) {
      return const FinanceShell();
    }
    return const FactoryShell();
  }
}
