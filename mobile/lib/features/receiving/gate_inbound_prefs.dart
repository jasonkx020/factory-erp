import 'package:shared_preferences/shared_preferences.dart';

/// Local sticky defaults for 过磅入厂 step-1 fee/price fields.
class GateInboundPrefs {
  GateInboundPrefs._();

  static const _prefix = 'erp.receiving.gate.';
  static const keyUnitPrice = '${_prefix}unit_price';
  static const keyDeductRate = '${_prefix}deduct_rate';
  static const keyFreight = '${_prefix}freight_fee';
  static const keyLoading = '${_prefix}loading_fee';
  static const keyWeighFee = '${_prefix}weigh_fee';
  static const keyNextRole = '${_prefix}next_role';

  static Future<Map<String, String>> load() async {
    final p = await SharedPreferences.getInstance();
    return {
      'unit_price': p.getString(keyUnitPrice) ?? '',
      'deduct_rate': p.getString(keyDeductRate) ?? '',
      'freight_fee': p.getString(keyFreight) ?? '',
      'loading_fee': p.getString(keyLoading) ?? '',
      'weigh_fee': p.getString(keyWeighFee) ?? '',
      'next_role': p.getString(keyNextRole) ?? '',
    };
  }

  static Future<void> save({
    required String unitPrice,
    required String deductRate,
    required String freight,
    required String loadingFee,
    required String weighFee,
    String? nextRole,
  }) async {
    final p = await SharedPreferences.getInstance();
    await p.setString(keyUnitPrice, unitPrice.trim());
    await p.setString(keyDeductRate, deductRate.trim());
    await p.setString(keyFreight, freight.trim());
    await p.setString(keyLoading, loadingFee.trim());
    await p.setString(keyWeighFee, weighFee.trim());
    if (nextRole != null && nextRole.isNotEmpty) {
      await p.setString(keyNextRole, nextRole);
    }
  }
}
