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
  final _settleProduct = TextEditingController();
  final _plate = TextEditingController();
  final _weight = TextEditingController(text: '0');
  final _unitPrice = TextEditingController(text: '0');
  final _freight = TextEditingController(text: '0');
  final _loading = TextEditingController(text: '0');
  final _weighFee = TextEditingController(text: '0');
  List<dynamic> _orders = [];
  List<dynamic> _inquiries = [];
  List<dynamic> _customers = [];
  List<dynamic> _settles = [];
  String _msg = '';

  static const _titles = ['我的订单', '询价', '出厂结算', '客户跟进', '任务提醒'];

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
    _settleProduct.dispose();
    _plate.dispose();
    _weight.dispose();
    _unitPrice.dispose();
    _freight.dispose();
    _loading.dispose();
    _weighFee.dispose();
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
      api.get('/sales/outbound-settles'),
    ]);
    if (!mounted) return;
    setState(() {
      _orders = (results[0].data is Map ? (results[0].data as Map)['list'] as List? : null) ?? [];
      _inquiries = (results[1].data is Map ? (results[1].data as Map)['list'] as List? : null) ?? [];
      _customers = (results[2].data is Map ? (results[2].data as Map)['list'] as List? : null) ?? [];
      _settles = (results[3].data is Map ? (results[3].data as Map)['list'] as List? : null) ?? [];
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

  Future<void> _rebuy(Map<String, dynamic> row) async {
    final api = context.read<AuthState>().api;
    final r = await api.post('/sales/orders', {
      'customer': row['customer'] ?? '复购客户',
      'rebuy_from': row['id'],
      'lines': row['lines'] is List
          ? row['lines']
          : [
              {'product_id': 1, 'qty': 1},
            ],
    });
    setState(() => _msg = r.ok ? '复购成功' : r.msg);
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

  Future<void> _createSettle() async {
    final api = context.read<AuthState>().api;
    final w = double.tryParse(_weight.text) ?? 0;
    final r = await api.post('/sales/outbound-settles', {
      'biz_date': DateTime.now().toIso8601String().substring(0, 10),
      'product_name': _settleProduct.text.trim(),
      'plate_no': _plate.text.trim(),
      'weight': w,
      'qty': w,
      'unit': 'kg',
      'unit_price': double.tryParse(_unitPrice.text) ?? 0,
      'freight_fee': double.tryParse(_freight.text) ?? 0,
      'loading_fee': double.tryParse(_loading.text) ?? 0,
      'weigh_fee': double.tryParse(_weighFee.text) ?? 0,
    });
    setState(() {
      if (r.ok) {
        final amt = r.data is Map ? (r.data as Map)['amount'] : 0;
        _msg = '出厂结算已录 ¥$amt';
      } else {
        _msg = r.msg;
      }
    });
    if (r.ok) await _refresh();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(_titles[_tab])),
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
                  subtitle: Text('${m['status'] ?? ''}'),
                  trailing: TextButton(onPressed: () => _rebuy(m), child: const Text('复购')),
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
            padding: const EdgeInsets.all(16),
            children: [
              const Text('补录出厂结算；金额=重量×单价+运/装/磅费', style: TextStyle(fontSize: 12, color: Colors.black54)),
              TextField(controller: _settleProduct, decoration: const InputDecoration(labelText: '产品')),
              TextField(controller: _plate, decoration: const InputDecoration(labelText: '车牌')),
              TextField(controller: _weight, decoration: const InputDecoration(labelText: '重量'), keyboardType: TextInputType.number),
              TextField(controller: _unitPrice, decoration: const InputDecoration(labelText: '单价'), keyboardType: TextInputType.number),
              TextField(controller: _freight, decoration: const InputDecoration(labelText: '运费'), keyboardType: TextInputType.number),
              TextField(controller: _loading, decoration: const InputDecoration(labelText: '装卸'), keyboardType: TextInputType.number),
              TextField(controller: _weighFee, decoration: const InputDecoration(labelText: '过磅费'), keyboardType: TextInputType.number),
              FilledButton(onPressed: _createSettle, child: const Text('保存补录')),
              if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_msg)),
              const Divider(),
              ..._settles.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['doc_no'] ?? ''}'),
                  subtitle: Text('${m['product_name'] ?? ''} · ${m['plate_no'] ?? ''}'),
                  trailing: Text('¥${m['amount'] ?? 0}'),
                );
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
          if (i == 2 || i == 0) _refresh();
          if (i == 4) context.read<NotifyService>().refresh();
        },
        destinations: const [
          NavigationDestination(icon: Icon(Icons.receipt_long), label: '订单'),
          NavigationDestination(icon: Icon(Icons.request_quote), label: '询价'),
          NavigationDestination(icon: Icon(Icons.local_shipping), label: '出厂'),
          NavigationDestination(icon: Icon(Icons.people), label: '跟进'),
          NavigationDestination(icon: Icon(Icons.alarm), label: '提醒'),
        ],
      ),
    );
  }
}
