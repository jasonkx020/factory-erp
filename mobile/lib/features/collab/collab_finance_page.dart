import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';

/// 收款预警处理 + 销售认款协同
class CollabFinancePage extends StatefulWidget {
  const CollabFinancePage({super.key});

  @override
  State<CollabFinancePage> createState() => _CollabFinancePageState();
}

class _CollabFinancePageState extends State<CollabFinancePage> {
  int _tab = 0;
  List<dynamic> _alerts = [];
  List<dynamic> _recognitions = [];
  List<dynamic> _customers = [];
  int? _customerId;
  final _amount = TextEditingController(text: '1000');
  final _remark = TextEditingController();
  String _msg = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _boot());
  }

  @override
  void dispose() {
    _amount.dispose();
    _remark.dispose();
    super.dispose();
  }

  Future<void> _boot() async {
    final auth = context.read<AuthState>();
    if (!canAccessEmployeeModule(EmployeeModule.collab, auth.permissions, auth.roles)) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('无财务协同权限')));
        Navigator.of(context).pop();
      }
      return;
    }
    await _refresh();
  }

  Future<void> _refresh() async {
    final api = context.read<AuthState>().api;
    final results = await Future.wait([
      api.get('/finance/receipt-alerts?page_size=50'),
      api.get('/finance/payment-recognitions?page_size=50'),
      api.get('/crm/customers?page_size=100'),
    ]);
    if (!mounted) return;
    setState(() {
      _alerts = ApiClient.listOf(results[0].data);
      _recognitions = ApiClient.listOf(results[1].data);
      _customers = ApiClient.listOf(results[2].data);
      if (_customerId == null && _customers.isNotEmpty) {
        _customerId = (_customers.first as Map)['id'] is num ? ((_customers.first as Map)['id'] as num).toInt() : null;
      }
    });
  }

  Future<void> _handleAlert(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/finance/receipt-alerts/$id/handle', {
      'remark': '手机端已跟进',
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '预警已处理' : r.msg)));
    if (r.ok) await _refresh();
  }

  Future<void> _createRecognition() async {
    if (_customerId == null) {
      setState(() => _msg = '请选择客户');
      return;
    }
    final amt = double.tryParse(_amount.text) ?? 0;
    if (amt <= 0) {
      setState(() => _msg = '金额须大于0');
      return;
    }
    final r = await context.read<AuthState>().api.post('/finance/payment-recognitions', {
      'customer_id': _customerId,
      'amount': amt,
      'remark': _remark.text.trim().isEmpty ? '外勤认款' : _remark.text.trim(),
    });
    setState(() => _msg = r.ok ? '认款草稿已建' : r.msg);
    if (r.ok) await _refresh();
  }

  Future<void> _confirmRecognition(Map m) async {
    final id = (m['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/finance/payment-recognitions/$id/confirm', {});
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '认款已确认' : r.msg)));
    if (r.ok) await _refresh();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('收款协同'),
        actions: [IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh))],
      ),
      body: IndexedStack(
        index: _tab,
        children: [
          ListView(
            padding: const EdgeInsets.all(12),
            children: [
              const Text('收款预警', style: TextStyle(fontWeight: FontWeight.bold)),
              if (_alerts.isEmpty) const Padding(padding: EdgeInsets.all(24), child: Center(child: Text('暂无预警'))),
              ..._alerts.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                final st = m['status']?.toString() ?? '';
                return Card(
                  child: ListTile(
                    title: Text('客户 ${m['customer_id']} · 逾期 ${m['overdue_days'] ?? 0} 天'),
                    subtitle: Text('金额 ¥${m['amount'] ?? 0} · 到期 ${m['due_date'] ?? ''}\n$st'),
                    isThreeLine: true,
                    trailing: st == 'open'
                        ? FilledButton.tonal(onPressed: () => _handleAlert(m), child: const Text('处理'))
                        : null,
                  ),
                );
              }),
            ],
          ),
          ListView(
            padding: const EdgeInsets.all(16),
            children: [
              const Text('销售认款', style: TextStyle(fontWeight: FontWeight.bold)),
              DropdownButtonFormField<int>(
                value: _customerId,
                decoration: const InputDecoration(labelText: '客户'),
                items: _customers.map((e) {
                  final m = Map<String, dynamic>.from(e as Map);
                  return DropdownMenuItem(value: (m['id'] as num?)?.toInt(), child: Text('${m['name'] ?? m['id']}'));
                }).toList(),
                onChanged: (v) => setState(() => _customerId = v),
              ),
              TextField(controller: _amount, decoration: const InputDecoration(labelText: '金额'), keyboardType: TextInputType.number),
              TextField(controller: _remark, decoration: const InputDecoration(labelText: '备注')),
              FilledButton(onPressed: _createRecognition, child: const Text('新建认款')),
              if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_msg)),
              const Divider(),
              ..._recognitions.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                final st = m['status']?.toString() ?? '';
                return ListTile(
                  title: Text('${m['doc_no'] ?? m['id']}'),
                  subtitle: Text('客户 ${m['customer_id']} · ¥${m['amount']} · $st'),
                  trailing: (st == 'draft')
                      ? TextButton(onPressed: () => _confirmRecognition(m), child: const Text('确认'))
                      : null,
                );
              }),
            ],
          ),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _tab,
        onDestinationSelected: (i) => setState(() => _tab = i),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.warning_amber), label: '预警'),
          NavigationDestination(icon: Icon(Icons.payments), label: '认款'),
        ],
      ),
    );
  }
}
