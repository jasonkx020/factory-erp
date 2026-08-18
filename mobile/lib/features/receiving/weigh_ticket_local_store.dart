import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

/// 提交成功的过磅入厂单本机快照（按登录用户隔离，保留 90 天）。
class WeighTicketLocalStore {
  WeighTicketLocalStore._();

  static const retention = Duration(days: 90);

  static String _key(int userId) => 'erp.receiving.gate.tickets.$userId';

  static DateTime? _parseTime(dynamic v) {
    final s = (v ?? '').toString().trim();
    if (s.isEmpty) return null;
    return DateTime.tryParse(s);
  }

  static List<Map<String, dynamic>> _prune(List<Map<String, dynamic>> list) {
    final cutoff = DateTime.now().subtract(retention);
    return list.where((m) {
      final t = _parseTime(m['saved_at']) ?? _parseTime(m['created_at']);
      if (t == null) return true;
      return t.isAfter(cutoff);
    }).toList();
  }

  static Future<List<Map<String, dynamic>>> load(int userId) async {
    if (userId <= 0) return [];
    final p = await SharedPreferences.getInstance();
    final raw = p.getString(_key(userId));
    if (raw == null || raw.isEmpty) return [];
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! List) return [];
      final list = <Map<String, dynamic>>[];
      for (final e in decoded) {
        if (e is Map<String, dynamic>) {
          list.add(e);
        } else if (e is Map) {
          list.add(Map<String, dynamic>.from(e));
        }
      }
      final pruned = _prune(list);
      if (pruned.length != list.length) {
        await _write(p, userId, pruned);
      }
      return pruned;
    } catch (_) {
      return [];
    }
  }

  static Future<void> save(int userId, Map<String, dynamic> ticket) async {
    if (userId <= 0) return;
    final list = await load(userId);
    final id = (ticket['id'] as num?)?.toInt() ?? 0;
    final doc = (ticket['doc_no'] ?? '').toString().trim();
    list.removeWhere((m) {
      final mid = (m['id'] as num?)?.toInt() ?? 0;
      final mdoc = (m['doc_no'] ?? '').toString().trim();
      if (id > 0 && mid == id) return true;
      if (doc.isNotEmpty && mdoc == doc) return true;
      return false;
    });
    final row = Map<String, dynamic>.from(ticket);
    row['saved_at'] = DateTime.now().toIso8601String();
    list.insert(0, row);
    final p = await SharedPreferences.getInstance();
    await _write(p, userId, _prune(list));
  }

  static Future<void> clear(int userId) async {
    if (userId <= 0) return;
    final p = await SharedPreferences.getInstance();
    await p.remove(_key(userId));
  }

  static Future<void> _write(SharedPreferences p, int userId, List<Map<String, dynamic>> list) async {
    await p.setString(_key(userId), jsonEncode(list));
  }
}
