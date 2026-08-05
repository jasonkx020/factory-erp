import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';
import '../../core/notify_service.dart';

class SalesPage extends StatefulWidget {
  const SalesPage({super.key});

  @override
  State<SalesPage> createState() => _SalesPageState();
}

class _SalesPageState extends State<SalesPage> {
  int _tab = 0;
  final _customer = TextEditingController();
  final _product = TextEditingController();
  final _qty = TextEditingController(text: '1');
  final _remark = TextEditingController();
  List<dynamic> _orders = [];
  List<dynamic> _inquiries = [];
  List<dynamic> _customers = [];
  String _msg = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _boot());
  }

  @override
  void dispose() {
    _customer.dispose();
    _product.dispose();
    _qty.dispose();
    _remark.dispose();
    super.dispose();
  }

  Future<void> _boot() async {
    final auth = context.read<AuthState>();
    if (!canAccessEmployeeModule(EmployeeModule.sales, auth.permissions, auth.roles)) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('无销售模块权限')));
        Navigator.of(context).pop();
      }
      return;
    }
    await context.read<NotifyService>().start();
    await _refresh();
  }

  Future<void> _refresh() async {
    final api = context.read<AuthState>().api;
    final results = await Future.wait([
      api.get('/sales/orders'),
      api.get('/sales/inquiries'),
      api.get('/crm/customers'),
    ]);
    setState(() {
      _orders = (results[0].data is Map ? (results[0].data as Map)['list'] as List? : null) ?? [];
      _inquiries = (results[1].data is Map ? (results[1].data as Map)['list'] as List? : null) ?? [];
      _customers = (results[2].data is Map ? (results[2].data as Map)['list'] as List? : null) ?? [];
    });
  }

  Future<void> _createOrder() async {
    final api = context.read<AuthState>().api;
    final r = await api.post('/sales/orders', {
      'customer': _customer.text.trim(),
      'lines': [
        {'product_id': 1, 'qty': int.tryParse(_qty.text) ?? 1},
      ],
    });
    setState(() => _msg = r.ok ? '订单已创建' : r.msg);
    if (r.ok) await _refresh();
  }

  Future<void> _createInquiry() async {
    final api = context.read<AuthState>().api;
    final r = await api.post('/sales/inquiries', {
      'customer': _customer.text.trim(),
      'product': _product.text.trim(),
      'qty': int.tryParse(_qty.text) ?? 1,
      'remark': _remark.text.trim(),
    });
    setState(() => _msg = r.ok ? '询价已提交' : r.msg);
    if (r.ok) await _refresh();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(['我的订单', '询价', '客户跟进', '任务提醒'][_tab])),
      body: IndexedStack(
        index: _tab,
        children: [
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              TextField(controller: _customer, decoration: const InputDecoration(labelText: '客户')),
              TextField(controller: _qty, decoration: const InputDecoration(labelText: '数量'), keyboardType: TextInputType.number),
              FilledButton(onPressed: _createOrder, child: const Text('新建订单')),
              if (_msg.isNotEmpty) Text(_msg),
              const Divider(),
              ..._orders.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('#${m['id']} ${m['doc_no'] ?? ''}'),
                  trailing: Text('${m['status'] ?? ''}'),
                );
              }),
            ],
          ),
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              TextField(controller: _customer, decoration: const InputDecoration(labelText: '客户')),
              TextField(controller: _product, decoration: const InputDecoration(labelText: '产品')),
              TextField(controller: _remark, decoration: const InputDecoration(labelText: '备注')),
              FilledButton(onPressed: _createInquiry, child: const Text('提交询价')),
              const Divider(),
              ..._inquiries.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(title: Text('#${m['id']}'), subtitle: Text('${m['product'] ?? m['customer'] ?? ''}'));
              }),
            ],
          ),
          ListView(
            children: _customers
                .map((e) {
                  final m = Map<String, dynamic>.from(e as Map);
                  return ListTile(
                    title: Text('${m['name'] ?? m['code'] ?? m['id']}'),
                    subtitle: Text('${m['status'] ?? ''}'),
                  );
                })
                .toList(),
          ),
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Text('MQTT ${context.watch<NotifyService>().mqttStatus} · 未读 ${context.watch<NotifyService>().unread}',
                  style: const TextStyle(color: Colors.black54, fontSize: 12)),
              const SizedBox(height: 8),
              if (context.watch<NotifyService>().inbox.isEmpty) const Text('暂无提醒'),
              ...context.watch<NotifyService>().inbox.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text(m['title']?.toString() ?? m['event_key']?.toString() ?? ''),
                  subtitle: Text(m['body']?.toString() ?? ''),
                  trailing: Text((m['read_at']?.toString() ?? '').isEmpty ? '未读' : '已读'),
                  onTap: () {
                    final id = (m['id'] as num?)?.toInt();
                    if (id != null) context.read<NotifyService>().markRead(id);
                  },
                );
              }),
            ],
          ),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _tab,
        onDestinationSelected: (i) {
          setState(() => _tab = i);
          if (i == 3) context.read<NotifyService>().refresh();
        },
        destinations: const [
          NavigationDestination(icon: Icon(Icons.receipt_long), label: '订单'),
          NavigationDestination(icon: Icon(Icons.request_quote), label: '询价'),
          NavigationDestination(icon: Icon(Icons.people), label: '跟进'),
          NavigationDestination(icon: Icon(Icons.alarm), label: '提醒'),
        ],
      ),
    );
  }
}
