import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../theme/plant_colors.dart';
import '../../widgets/factory_kpi_card.dart';

/// 产线结算财务：农户应付 / 资金 / 流水 / 成本期间预览（对齐 Web FinanceHub）。
class FinanceHubPage extends StatefulWidget {
  const FinanceHubPage({super.key, this.asTab = false});

  final bool asTab;

  @override
  State<FinanceHubPage> createState() => _FinanceHubPageState();
}

class _FinanceHubPageState extends State<FinanceHubPage> with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  bool _loading = false;
  String _msg = '';

  List<Map<String, dynamic>> _payables = [];
  List<Map<String, dynamic>> _funds = [];
  List<Map<String, dynamic>> _ledger = [];
  Map<String, dynamic>? _costPreview;
  final _period = TextEditingController(
    text: '${DateTime.now().year}-${DateTime.now().month.toString().padLeft(2, '0')}',
  );

  double get _pendingAmt =>
      _payables.fold<double>(0, (s, r) => s + ((r['amount'] as num?)?.toDouble() ?? 0));
  double get _fundBal =>
      _funds.fold<double>(0, (s, r) => s + ((r['balance'] as num?)?.toDouble() ?? 0));

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 4, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) => refresh());
  }

  @override
  void dispose() {
    _tabs.dispose();
    _period.dispose();
    super.dispose();
  }

  bool _isPending(Map r) {
    final st = '${r['status'] ?? ''}';
    return st != 'settle_paid' && st != 'paid' && st != 'void';
  }

  Future<void> refresh() async {
    setState(() {
      _loading = true;
      _msg = '';
    });
    final api = context.read<AuthState>().api;
    try {
      final results = await Future.wait([
        api.get('/purchase/farmer-settlements'),
        api.get('/finance/fund-accounts'),
        api.get('/finance/ledger-entries'),
        api.get('/finance/cost-period-preview?period=${Uri.encodeComponent(_period.text.trim())}'),
      ]);
      if (!mounted) return;
      final settles = ApiClient.listOf(results[0].data).whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
      setState(() {
        _payables = settles.where(_isPending).toList();
        _funds = ApiClient.listOf(results[1].data).whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
        _ledger = ApiClient.listOf(results[2].data).whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
        _costPreview = results[3].ok && results[3].data is Map
            ? Map<String, dynamic>.from(results[3].data as Map)
            : null;
        if (!results[1].ok) _msg = results[1].msg;
        if (!results[0].ok) _msg = results[0].msg;
      });
    } catch (e) {
      if (mounted) setState(() => _msg = '$e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  String _money(dynamic v) {
    final n = (v as num?)?.toDouble();
    if (n == null) return '-';
    return n.toStringAsFixed(2);
  }

  Future<void> _paySettlement(Map<String, dynamic> row) async {
    if (_funds.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请先在管理端维护资金账户')));
      return;
    }
    int fundId = (_funds.first['id'] as num?)?.toInt() ?? 0;
    final transfer = TextEditingController();
    final evidence = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setLocal) => AlertDialog(
          title: Text('支付关单 ${row['doc_no'] ?? ''}'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('应付 ¥${_money(row['amount'])}', style: const TextStyle(fontWeight: FontWeight.w600)),
                const SizedBox(height: 12),
                DropdownButtonFormField<int>(
                  // ignore: deprecated_member_use
                  value: fundId > 0 ? fundId : null,
                  decoration: const InputDecoration(labelText: '资金账户 *'),
                  items: [
                    for (final f in _funds)
                      DropdownMenuItem(
                        value: (f['id'] as num?)?.toInt(),
                        child: Text('${f['name']}（${_money(f['balance'])}）'),
                      ),
                  ],
                  onChanged: (v) => setLocal(() => fundId = v ?? 0),
                ),
                TextField(controller: transfer, decoration: const InputDecoration(labelText: '转账单号 *')),
                TextField(controller: evidence, decoration: const InputDecoration(labelText: '回单/截图 URL *')),
              ],
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
            FilledButton(
              onPressed: () {
                if (fundId <= 0 || transfer.text.trim().isEmpty || evidence.text.trim().isEmpty) {
                  ScaffoldMessenger.of(ctx).showSnackBar(const SnackBar(content: Text('账户、转账单号、回单必填')));
                  return;
                }
                Navigator.pop(ctx, true);
              },
              child: const Text('确认支付'),
            ),
          ],
        ),
      ),
    );
    final transferNo = transfer.text.trim();
    final evidenceUrl = evidence.text.trim();
    transfer.dispose();
    evidence.dispose();
    if (ok != true || !mounted) return;
    final id = (row['id'] as num?)?.toInt();
    if (id == null) return;
    final r = await context.read<AuthState>().api.post('/purchase/farmer-settlements/$id/pay', {
      'fund_account_id': fundId,
      'transfer_no': transferNo,
      'pay_evidence_url': evidenceUrl,
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.ok ? '已支付关单' : r.msg)));
    if (r.ok) await refresh();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: PlantColors.bg,
      appBar: AppBar(
        title: const Text('结算财务'),
        automaticallyImplyLeading: !widget.asTab,
        actions: [
          IconButton(onPressed: _loading ? null : refresh, icon: const Icon(Icons.refresh)),
        ],
        bottom: TabBar(
          controller: _tabs,
          isScrollable: true,
          indicatorColor: PlantColors.soil,
          labelColor: PlantColors.onForest,
          unselectedLabelColor: PlantColors.onForest.withValues(alpha: 0.65),
          tabs: const [
            Tab(text: '农户应付'),
            Tab(text: '资金账户'),
            Tab(text: '资金流水'),
            Tab(text: '成本预览'),
          ],
        ),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 12, 12, 0),
            child: Row(
              children: [
                Expanded(
                  child: FactoryKpiCard(
                    label: '待付笔数',
                    value: '${_payables.length}',
                    tone: _payables.isEmpty ? FactoryKpiTone.ok : FactoryKpiTone.warn,
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: FactoryKpiCard(
                    label: '待付金额',
                    value: _money(_pendingAmt),
                    tone: _payables.isEmpty ? FactoryKpiTone.ok : FactoryKpiTone.warn,
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: FactoryKpiCard(
                    label: '资金余额',
                    value: _money(_fundBal),
                    tone: FactoryKpiTone.ok,
                  ),
                ),
              ],
            ),
          ),
          if (_msg.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
              child: Text(_msg, style: const TextStyle(color: PlantColors.warn, fontSize: 12)),
            ),
          const SizedBox(height: 8),
          Expanded(
            child: _loading
                ? const Center(child: CircularProgressIndicator())
                : TabBarView(
                    controller: _tabs,
                    children: [
                      _payablesTab(),
                      _fundsTab(),
                      _ledgerTab(),
                      _costTab(),
                    ],
                  ),
          ),
        ],
      ),
    );
  }

  Widget _payablesTab() {
    if (_payables.isEmpty) {
      return const Center(child: Text('暂无待付农户结算', style: TextStyle(color: PlantColors.muted)));
    }
    return ListView.separated(
      padding: const EdgeInsets.all(12),
      itemCount: _payables.length,
      separatorBuilder: (_, __) => const SizedBox(height: 8),
      itemBuilder: (_, i) {
        final r = _payables[i];
        return Card(
          child: ListTile(
            title: Text('${r['doc_no'] ?? '-'} · ${r['farmer_name'] ?? '农户'}'),
            subtitle: Text('${r['biz_date'] ?? ''} · ${_money(r['amount'])} 元'),
            trailing: FilledButton(
              onPressed: () => _paySettlement(r),
              child: const Text('支付'),
            ),
          ),
        );
      },
    );
  }

  Widget _fundsTab() {
    if (_funds.isEmpty) {
      return const Center(child: Text('暂无资金账户', style: TextStyle(color: PlantColors.muted)));
    }
    return ListView.separated(
      padding: const EdgeInsets.all(12),
      itemCount: _funds.length,
      separatorBuilder: (_, __) => const SizedBox(height: 8),
      itemBuilder: (_, i) {
        final r = _funds[i];
        return Card(
          child: ListTile(
            leading: const Icon(Icons.account_balance_wallet_outlined, color: PlantColors.leaf),
            title: Text('${r['name'] ?? r['code'] ?? '-'}'),
            subtitle: Text('${r['code'] ?? ''} · ${r['currency'] ?? 'CNY'}'),
            trailing: Text(
              _money(r['balance']),
              style: const TextStyle(
                fontWeight: FontWeight.w700,
                fontFeatures: [FontFeature.tabularFigures()],
                color: PlantColors.leaf,
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _ledgerTab() {
    if (_ledger.isEmpty) {
      return const Center(child: Text('暂无资金流水', style: TextStyle(color: PlantColors.muted)));
    }
    return ListView.separated(
      padding: const EdgeInsets.all(12),
      itemCount: _ledger.length.clamp(0, 80),
      separatorBuilder: (_, __) => const SizedBox(height: 8),
      itemBuilder: (_, i) {
        final r = _ledger[i];
        final out = '${r['direction']}' == 'out';
        return Card(
          child: ListTile(
            title: Text('${r['doc_no'] ?? '-'} · ${r['account_name'] ?? ''}'),
            subtitle: Text('${r['biz_date'] ?? ''} · ${r['counterparty'] ?? r['remark'] ?? ''}'),
            trailing: Text(
              '${out ? '-' : '+'}${_money(r['amount'])}',
              style: TextStyle(
                fontWeight: FontWeight.w700,
                color: out ? PlantColors.warn : PlantColors.leaf,
                fontFeatures: const [FontFeature.tabularFigures()],
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _costTab() {
    final p = _costPreview;
    return ListView(
      padding: const EdgeInsets.all(12),
      children: [
        Row(
          children: [
            Expanded(
              child: TextField(
                controller: _period,
                decoration: const InputDecoration(labelText: '期间 YYYY-MM'),
              ),
            ),
            const SizedBox(width: 8),
            FilledButton(onPressed: refresh, child: const Text('汇入')),
          ],
        ),
        const SizedBox(height: 12),
        if (p == null)
          const Text('暂无期间数据', style: TextStyle(color: PlantColors.muted))
        else ...[
          FactoryKpiCard(label: '建议物料', value: _money(p['material_cost']), tone: FactoryKpiTone.ok),
          const SizedBox(height: 8),
          FactoryKpiCard(label: '建议人工', value: _money(p['labor_cost'])),
          const SizedBox(height: 8),
          FactoryKpiCard(label: '合计', value: _money(p['total_cost']), tone: FactoryKpiTone.ok),
          const SizedBox(height: 12),
          Text(
            '农户已付 ${_money(p['farmer_paid'])}（${p['farmer_paid_count'] ?? 0}）'
            ' · 待付 ${_money(p['farmer_pending'])}'
            ' · 计件 ${_money(p['piecework_amount'])}',
            style: const TextStyle(fontSize: 12, color: PlantColors.muted),
          ),
          const SizedBox(height: 8),
          const Text('完整建单与核算请在管理端「财务管理」操作。', style: TextStyle(fontSize: 12, color: PlantColors.soil)),
        ],
      ],
    );
  }
}
