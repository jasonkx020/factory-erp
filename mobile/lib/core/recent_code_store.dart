import 'package:shared_preferences/shared_preferences.dart';

/// 本地最近码值（溯源码 / 板码 / 工牌等），减少重复扫码。
class RecentCodeStore {
  RecentCodeStore._();

  static const int defaultLimit = 12;

  static String _prefKey(String namespace) => 'erp.recent_codes.$namespace';

  static const String trace = 'trace_code';
  static const String board = 'board_code';
  static const String badge = 'badge_code';

  static Future<List<String>> list(String namespace, {int limit = defaultLimit}) async {
    final p = await SharedPreferences.getInstance();
    final raw = p.getStringList(_prefKey(namespace)) ?? const <String>[];
    if (raw.length <= limit) return List<String>.from(raw);
    return raw.take(limit).toList();
  }

  /// 写入最近使用（去重置顶）。返回最新列表。
  static Future<List<String>> remember(
    String namespace,
    String value, {
    int limit = defaultLimit,
    bool upper = true,
  }) async {
    var v = value.trim();
    if (v.isEmpty) return list(namespace, limit: limit);
    if (upper) v = v.toUpperCase();
    final p = await SharedPreferences.getInstance();
    final key = _prefKey(namespace);
    final cur = p.getStringList(key) ?? <String>[];
    cur.removeWhere((e) => e.toUpperCase() == v);
    cur.insert(0, v);
    while (cur.length > limit) {
      cur.removeLast();
    }
    await p.setStringList(key, cur);
    return List<String>.from(cur);
  }

  static Future<void> remove(String namespace, String value) async {
    final p = await SharedPreferences.getInstance();
    final key = _prefKey(namespace);
    final cur = p.getStringList(key) ?? <String>[];
    cur.removeWhere((e) => e.toUpperCase() == value.trim().toUpperCase());
    await p.setStringList(key, cur);
  }

  static Future<void> clear(String namespace) async {
    final p = await SharedPreferences.getInstance();
    await p.remove(_prefKey(namespace));
  }
}
