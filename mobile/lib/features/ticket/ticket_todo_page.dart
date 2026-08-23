import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/notify_service.dart';
import 'ticket_widgets.dart';

/// 待办：指派给我且未结案
class TicketTodoPage extends StatefulWidget {
  const TicketTodoPage({super.key});

  @override
  State<TicketTodoPage> createState() => TicketTodoPageState();
}

class TicketTodoPageState extends State<TicketTodoPage> {
  List<Map<String, dynamic>> _list = [];
  String _msg = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => reload());
  }

  Future<void> reload() async {
    final r = await context.read<AuthState>().api.get('/workflow/tickets?scope=mine_assignee');
    if (!mounted) return;
    setState(() {
      _list = ApiClient.listOf(r.data).map((e) => Map<String, dynamic>.from(e as Map)).toList();
      _msg = r.ok ? '' : r.msg;
    });
  }

  @override
  Widget build(BuildContext context) {
    final notify = context.watch<NotifyService>();
    return Scaffold(
      appBar: AppBar(
        title: Text(context.watch<AuthState>().preferQcShell ? '质检待办' : '待办'),
        actions: ticketShellMessageActions(context, notify.unread),
      ),
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Padding(
            padding: EdgeInsets.fromLTRB(16, 8, 16, 0),
            child: Text('指派给我的待处理工单', style: TextStyle(color: Colors.black54, fontSize: 13)),
          ),
          if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.all(12), child: Text(_msg)),
          Expanded(
            child: RefreshIndicator(
              onRefresh: reload,
              child: _list.isEmpty
                  ? ListView(
                      physics: const AlwaysScrollableScrollPhysics(),
                      children: const [
                        SizedBox(height: 120),
                        Center(child: Text('暂无待办')),
                      ],
                    )
                  : ListView.builder(
                      itemCount: _list.length,
                      itemBuilder: (_, i) {
                        final m = _list[i];
                        return TicketListCard(
                          row: m,
                          showActions: true,
                          onTap: () => openTicketDetail(context, m, onActed: reload),
                          onAction: (a) => ticketAct(context, m, a, onDone: reload),
                        );
                      },
                    ),
            ),
          ),
        ],
      ),
    );
  }
}
