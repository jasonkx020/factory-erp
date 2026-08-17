import 'package:flutter/foundation.dart';

import 'api_client.dart';

/// Display labels for carrier scan codes (板码 / 箱码).
/// Backed by sys_setting.base.carrier_code_unit; API fields stay box_code.
class CarrierCodeLabels extends ChangeNotifier {
  CarrierCodeLabels(this.api);

  final ApiClient api;

  /// board | box
  String unit = 'board';

  String get code => unit == 'box' ? '箱码' : '板码';
  String get short => unit == 'box' ? '箱' : '板';
  String get manageTitle => unit == 'box' ? '箱码管理' : '板码管理';
  String get splitVerb => unit == 'box' ? '分箱' : '分板';

  bool _loaded = false;

  Future<void> load({bool force = false}) async {
    if (_loaded && !force) return;
    try {
      final r = await api.get('/system/settings');
      if (r.ok && r.data is Map) {
        final data = r.data as Map;
        Map<String, dynamic>? row;
        final list = data['list'];
        if (list is List && list.isNotEmpty && list.first is Map) {
          row = Map<String, dynamic>.from(list.first as Map);
        } else {
          row = Map<String, dynamic>.from(data);
        }
        final u = (row['carrier_code_unit'] ?? 'board').toString().trim().toLowerCase();
        unit = u == 'box' ? 'box' : 'board';
        _loaded = true;
        notifyListeners();
      }
    } catch (_) {
      /* keep default board */
    }
  }

  void reset() {
    unit = 'board';
    _loaded = false;
    notifyListeners();
  }
}
