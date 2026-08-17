import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

/// 仓管分板未提交草稿（按溯源码/过磅单缓存）。
class BoxStockinDraftStore {
  BoxStockinDraftStore._();

  static String _key(String id) => 'erp.warehouse.box_draft.$id';

  static Future<Map<String, dynamic>?> load(String id) async {
    if (id.trim().isEmpty) return null;
    final p = await SharedPreferences.getInstance();
    final raw = p.getString(_key(id.trim()));
    if (raw == null || raw.isEmpty) return null;
    try {
      final m = jsonDecode(raw);
      if (m is Map<String, dynamic>) return m;
      if (m is Map) return Map<String, dynamic>.from(m);
    } catch (_) {}
    return null;
  }

  static Future<void> save(String id, Map<String, dynamic> data) async {
    if (id.trim().isEmpty) return;
    final p = await SharedPreferences.getInstance();
    await p.setString(_key(id.trim()), jsonEncode(data));
  }

  static Future<void> clear(String id) async {
    if (id.trim().isEmpty) return;
    final p = await SharedPreferences.getInstance();
    await p.remove(_key(id.trim()));
  }
}
