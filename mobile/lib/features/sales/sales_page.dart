import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../widgets/form_row.dart';
import '../../widgets/form_section_header.dart';
import '../../widgets/form_sticky_actions.dart';
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
  final _qty = TextEditingController(text: '1');
  final _remark = TextEditingController();
  final _productName = TextEditingController();
  final _settleProduct = TextEditingController();
  final _plate = TextEditingController();
  final _weight = TextEditingController(text: '0');
  final _unitPrice = TextEditingController(text: '0');
  final _freight = TextEditingController(text: '0');
  final _loadingFee = TextEditingController(text: '0');
  final _weighFee = TextEditingController(text: '0');
  final _followContent = TextEditingController();
  List<dynamic> _orders = [];
  List<dynamic> _inquiries = [];
  List<dynamic> _customers = [];
  List<dynamic> _products = [];
  List<dynamic> _settles = [];
  List<dynamic> _follows = [];
  List<dynamic> _deliveries = [];
  List<dynamic> _preShips = [];
  int? _customerId;
  int? _productId;
  String _msg = '';
  Map<String, dynamic>? _quote;
  final _margin = TextEditingController(text: '0.2');
  final _baseCost = TextEditingController();

  static const _titles = ['我的订单', '询价', '出厂结算', '发货进度', '报价试算', '客户跟进'];

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _boot());
  }

  @override
  void dispose() {
    _qty.dispose();
    _remark.dispose();
    _productName.dispose();
    _settleProduct.dispose();
    _plate.dispose();
    _weight.dispose();
    _unitPrice.dispose();
    _freight.dispose();
    _loadingFee.dispose();
    _weighFee.dispose();
    _followContent.dispose();
    _margin.dispose();
    _baseCost.dispose();
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
      api.get('/sales/orders?page_size=50'),
      api.get('/sales/inquiries?page_size=50'),
      api.get('/crm/customers?page_size=100'),
      api.get('/sales/outbound-settles?page_size=50'),
      api.get('/product/products?page_size=100'),
      api.get('/crm/follow-ups?page_size=30'),
      api.get('/sales/deliveries?page_size=50'),
      api.get('/sales/pre-shipments?page_size=50'),
    ]);
    if (!mounted) return;
    setState(() {
      _orders = ApiClient.listOf(results[0].data);
      _inquiries = ApiClient.listOf(results[1].data);
      _customers = ApiClient.listOf(results[2].data);
      _settles = ApiClient.listOf(results[3].data);
      _products = ApiClient.listOf(results[4].data);
      _follows = ApiClient.listOf(results[5].data);
      _deliveries = ApiClient.listOf(results[6].data);
      _preShips = ApiClient.listOf(results[7].data);
      if (_customerId == null && _customers.isNotEmpty) {
        _customerId = (_customers.first as Map)['id'] is num ? ((_customers.first as Map)['id'] as num).toInt() : null;
      }
      if (_productId == null && _products.isNotEmpty) {
        final p = Map<String, dynamic>.from(_products.first as Map);
        _productId = (p['id'] as num?)?.toInt();
        _settleProduct.text = p['name']?.toString() ?? '';
        if (_baseCost.text.isEmpty) {
          _baseCost.text = '${p['cost_price'] ?? p['sale_price'] ?? 0}';
        }
      }
    });
  }

  Future<void> _createOrder() async {
    if (_customerId == null || _productId == null) {
      setState(() => _msg = '请选择客户与产品');
      return;
    }
    final api = context.read<AuthState>().api;
    final r = await api.post('/sales/orders', {
      'customer_id': _customerId,
      'lines': [
        {'product_id': _productId, 'qty': double.tryParse(_qty.text) ?? 1},
      ],
      'remark': _remark.text.trim(),
    });
    setState(() => _msg = r.ok ? '订单已创建' : r.msg);
    if (r.ok) await _refresh();
  }

  Future<void> _rebuy(Map<String, dynamic> row) async {
    final api = context.read<AuthState>().api;
    final cid = (row['customer_id'] as num?)?.toInt() ?? _customerId;
    if (cid == null) {
      setState(() => _msg = '缺少客户');
      return;
    }
    // 拉详情取行
    final det = await api.get('/sales/orders/${row['id']}');
    List lines = [];
    if (det.ok && det.data is Map && (det.data as Map)['lines'] is List) {
      lines = (det.data as Map)['lines'] as List;
    }
    if (lines.isEmpty) {
      lines = [
        {'product_id': _productId ?? 1, 'qty': 1},
      ];
    }
    final r = await api.post('/sales/orders', {
      'customer_id': cid,
      'rebuy_from': row['id'],
      'lines': lines
          .map((e) {
            final m = Map<String, dynamic>.from(e as Map);
            return {'product_id': m['product_id'], 'qty': m['qty'] ?? 1, 'price': m['price']};
          })
          .toList(),
    });
    setState(() => _msg = r.ok ? '复购成功' : r.msg);
    if (r.ok) await _refresh();
  }

  Future<void> _createInquiry() async {
    if (_customerId == null) {
      setState(() => _msg = '请选择客户');
      return;
    }
    final api = context.read<AuthState>().api;
    final r = await api.post('/sales/inquiries', {
      'customer_id': _customerId,
      'lines': [
        {
          'product_id': _productId ?? 1,
          'qty': double.tryParse(_qty.text) ?? 1,
          'name': _productName.text.trim(),
        },
      ],
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
      'product_id': _productId,
      'product_name': _settleProduct.text.trim(),
      'plate_no': _plate.text.trim(),
      'weight': w,
      'qty': w,
      'unit': 'kg',
      'unit_price': double.tryParse(_unitPrice.text) ?? 0,
      'freight_fee': double.tryParse(_freight.text) ?? 0,
      'loading_fee': double.tryParse(_loadingFee.text) ?? 0,
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

  Future<void> _createFollow() async {
    if (_customerId == null) {
      setState(() => _msg = '请选择客户');
      return;
    }
    final r = await context.read<AuthState>().api.post('/crm/follow-ups', {
      'customer_id': _customerId,
      'follow_type': 'visit',
      'content': _followContent.text.trim().isEmpty ? '外勤跟进' : _followContent.text.trim(),
    });
    setState(() => _msg = r.ok ? '跟进已登记' : r.msg);
    if (r.ok) {
      _followContent.clear();
      await _refresh();
    }
  }

  Future<void> _calcQuote() async {
    final r = await context.read<AuthState>().api.post('/sales/quote-calculator/calc', {
      'product_id': _productId ?? 3,
      'qty': double.tryParse(_qty.text) ?? 1,
      'base_cost': double.tryParse(_baseCost.text) ?? 0,
      'margin_rate': double.tryParse(_margin.text) ?? 0.2,
      'customer_id': _customerId,
    });
    setState(() {
      if (r.ok && r.data is Map) {
        _quote = Map<String, dynamic>.from(r.data as Map);
        _msg = '试算完成';
      } else {
        _msg = r.msg;
      }
    });
  }

  Future<void> _applyQuote() async {
    if (_quote == null) {
      setState(() => _msg = '请先试算');
      return;
    }
    final r = await context.read<AuthState>().api.post('/sales/quote-calculator/apply', {
      ..._quote!,
      'customer_id': _customerId,
    });
    setState(() => _msg = r.ok ? '报价已保存' : r.msg);
  }

  Widget _customerPicker() {
    return FormRow(
      label: '客户',
      requiredMark: true,
      child: DropdownButtonHideUnderline(
        child: DropdownButton<int>(
          isExpanded: true,
          value: _customerId,
          alignment: Alignment.centerRight,
          hint: const Text('请选择', textAlign: TextAlign.right),
          items: _customers.map((e) {
            final m = Map<String, dynamic>.from(e as Map);
            return DropdownMenuItem(
              value: (m['id'] as num?)?.toInt(),
              child: Text('${m['name'] ?? m['code'] ?? m['id']}', textAlign: TextAlign.right),
            );
          }).toList(),
          onChanged: (v) => setState(() => _customerId = v),
        ),
      ),
    );
  }

  Widget _productPicker() {
    return FormRow(
      label: '产品',
      requiredMark: true,
      child: DropdownButtonHideUnderline(
        child: DropdownButton<int>(
          isExpanded: true,
          value: _productId,
          alignment: Alignment.centerRight,
          hint: const Text('请选择', textAlign: TextAlign.right),
          items: _products.map((e) {
            final m = Map<String, dynamic>.from(e as Map);
            return DropdownMenuItem(
              value: (m['id'] as num?)?.toInt(),
              child: Text('${m['name'] ?? m['code'] ?? m['id']}', textAlign: TextAlign.right),
            );
          }).toList(),
          onChanged: (v) {
            setState(() => _productId = v);
            final hit = _products.cast<dynamic>().map((e) => Map<String, dynamic>.from(e as Map)).where((m) => (m['id'] as num?)?.toInt() == v);
            if (hit.isNotEmpty) {
              _settleProduct.text = hit.first['name']?.toString() ?? '';
              _productName.text = hit.first['name']?.toString() ?? '';
            }
          },
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(_titles[_tab])),
      body: IndexedStack(
        index: _tab,
        children: [
          ListView(
            keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
            padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
            children: [
              _customerPicker(),
              _productPicker(),
              FormRow.text(label: '数量', controller: _qty, keyboardType: TextInputType.number, requiredMark: true),
              FormRow.text(label: '备注', controller: _remark),
              FormStickyActions(primaryLabel: '新建订单', onPrimary: _createOrder),
              if (_msg.isNotEmpty) Text(_msg),
              const Divider(),
              ..._orders.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['doc_no'] ?? '#'}${m['id']}'),
                  subtitle: Text('${m['customer_name'] ?? ''} · ${m['status'] ?? ''} · ¥${m['total_amount'] ?? 0}'),
                  trailing: TextButton(onPressed: () => _rebuy(m), child: const Text('复购')),
                );
              }),
            ],
          ),
          ListView(
            keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
            padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
            children: [
              _customerPicker(),
              _productPicker(),
              FormRow.text(label: '数量', controller: _qty, keyboardType: TextInputType.number, requiredMark: true),
              FormRow.text(label: '备注', controller: _remark),
              FormStickyActions(primaryLabel: '提交询价', onPrimary: _createInquiry),
              if (_msg.isNotEmpty) Text(_msg),
              const Divider(),
              ..._inquiries.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['doc_no'] ?? m['id']}'),
                  subtitle: Text('${m['customer_name'] ?? ''} · ${m['status'] ?? ''}'),
                );
              }),
            ],
          ),
          ListView(
            keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
            padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
            children: [
              const FormSectionHeader('出厂结算补录'),
              Text('金额=重量×单价+运/装/磅费', style: TextStyle(fontSize: 12, color: Colors.black54)),
              _productPicker(),
              FormRow.text(label: '产品名称', controller: _settleProduct),
              FormRow.text(label: '车牌', controller: _plate),
              FormRow.text(label: '重量', controller: _weight, keyboardType: TextInputType.number, requiredMark: true),
              FormRow.text(label: '单价', controller: _unitPrice, keyboardType: TextInputType.number, requiredMark: true),
              FormRow.text(label: '运费', controller: _freight, keyboardType: TextInputType.number),
              FormRow.text(label: '装卸', controller: _loadingFee, keyboardType: TextInputType.number),
              FormRow.text(label: '过磅费', controller: _weighFee, keyboardType: TextInputType.number),
              FormStickyActions(primaryLabel: '保存补录', onPrimary: _createSettle),
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
            padding: const EdgeInsets.all(12),
            children: [
              const Text('发货审批进度', style: TextStyle(fontWeight: FontWeight.bold)),
              if (_deliveries.isEmpty) const Text('暂无发货单'),
              ..._deliveries.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['doc_no'] ?? m['id']}'),
                  subtitle: Text('${m['status'] ?? ''} · 物流 ${m['logistics_no'] ?? m['tracking_no'] ?? '-'}'),
                  trailing: Text('¥${m['amount'] ?? ''}'),
                );
              }),
              const Divider(),
              const Text('预发货', style: TextStyle(fontWeight: FontWeight.bold)),
              if (_preShips.isEmpty) const Text('暂无预发货'),
              ..._preShips.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['doc_no'] ?? m['id']}'),
                  subtitle: Text('${m['status'] ?? ''} · 仓 ${m['warehouse_id'] ?? ''}'),
                );
              }),
            ],
          ),
          ListView(
            keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
            padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
            children: [
              const FormSectionHeader('报价试算'),
              Text('成本×(1+毛利率)', style: TextStyle(fontSize: 12, color: Colors.black54)),
              _customerPicker(),
              _productPicker(),
              FormRow.text(label: '数量', controller: _qty, keyboardType: TextInputType.number, requiredMark: true),
              FormRow.text(label: '成本单价', controller: _baseCost, keyboardType: TextInputType.number, requiredMark: true),
              FormRow.text(label: '毛利率', controller: _margin, keyboardType: TextInputType.number, hint: '如 0.2'),
              FormStickyButtonBar(
                children: [
                  FilledButton(onPressed: _calcQuote, child: const Text('试算')),
                  OutlinedButton(onPressed: _applyQuote, child: const Text('保存报价')),
                ],
              ),
              if (_quote != null)
                Card(
                  margin: const EdgeInsets.only(top: 12),
                  child: Padding(
                    padding: const EdgeInsets.all(12),
                    child: Text(
                      '报价单价 ¥${_quote!['quote_price']}\n'
                      '合计 ¥${_quote!['amount']}\n'
                      '成本 ${_quote!['base_cost']} · 毛利 ${_quote!['margin_rate']}',
                      style: const TextStyle(fontSize: 16),
                    ),
                  ),
                ),
              if (_msg.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_msg)),
            ],
          ),
          ListView(
            keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
            padding: EdgeInsets.fromLTRB(12, 8, 12, 16 + MediaQuery.viewInsetsOf(context).bottom),
            children: [
              _customerPicker(),
              FormRow.text(label: '跟进内容', controller: _followContent, maxLines: 3, requiredMark: true),
              FormStickyActions(primaryLabel: '登记跟进', onPrimary: _createFollow),
              if (_msg.isNotEmpty) Text(_msg),
              const Divider(),
              ..._follows.map((e) {
                final m = Map<String, dynamic>.from(e as Map);
                return ListTile(
                  title: Text('${m['customer_name'] ?? m['customer_id']}'),
                  subtitle: Text('${m['content'] ?? ''}\n${m['follow_at'] ?? ''}'),
                  isThreeLine: true,
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
          if (i != 4) _refresh();
        },
        destinations: const [
          NavigationDestination(icon: Icon(Icons.receipt_long), label: '订单'),
          NavigationDestination(icon: Icon(Icons.request_quote), label: '询价'),
          NavigationDestination(icon: Icon(Icons.local_shipping), label: '出厂'),
          NavigationDestination(icon: Icon(Icons.local_mall), label: '发货'),
          NavigationDestination(icon: Icon(Icons.calculate), label: '报价'),
          NavigationDestination(icon: Icon(Icons.people), label: '跟进'),
        ],
      ),
    );
  }
}
