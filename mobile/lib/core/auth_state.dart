import 'package:flutter/foundation.dart';

import 'api_client.dart';

class AuthState extends ChangeNotifier {
  AuthState(this.api);

  final ApiClient api;

  String? name;
  String? loginName;
  int userId = 0;
  int employeeId = 0;
  List<String> roles = [];
  List<String> permissions = [];
  String error = '';
  bool loading = false;

  bool get isLoggedIn => api.accessToken != null && api.accessToken!.isNotEmpty;

  void _applyUser(Map<String, dynamic>? user) {
    if (user == null) return;
    name = user['name']?.toString().isNotEmpty == true ? user['name']?.toString() : name;
    loginName = user['login_name']?.toString() ?? loginName;
    userId = (user['id'] as num?)?.toInt() ?? userId;
    employeeId = (user['employee_id'] as num?)?.toInt() ?? employeeId;
  }

  Future<bool> login(String login, String password) async {
    loading = true;
    error = '';
    notifyListeners();
    try {
      final r = await api.post(
        '/auth/login',
        {
          'login_name': login,
          'password': password,
          'client_type': 'mobile',
        },
        auth: false,
      );
      if (!r.ok || r.data is! Map) {
        error = r.msg.isEmpty ? 'LOGIN_FAILED' : r.msg;
        return false;
      }
      final data = r.data as Map<String, dynamic>;
      await api.saveTokens(
        data['access_token']?.toString() ?? '',
        data['refresh_token']?.toString() ?? '',
      );
      roles = (data['roles'] as List?)?.map((e) => e.toString()).toList() ?? [];
      permissions = (data['permissions'] as List?)?.map((e) => e.toString()).toList() ?? [];
      _applyUser(data['user'] as Map<String, dynamic>?);
      await fetchMe();
      return true;
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  Future<bool> fetchMe() async {
    final r = await api.get('/auth/me');
    if (!r.ok || r.data is! Map) {
      if (r.msg == 'UNAUTHORIZED') await logout();
      return false;
    }
    final data = r.data as Map<String, dynamic>;
    roles = (data['roles'] as List?)?.map((e) => e.toString()).toList() ?? roles;
    permissions = (data['permissions'] as List?)?.map((e) => e.toString()).toList() ?? permissions;
    _applyUser(data['user'] as Map<String, dynamic>?);
    notifyListeners();
    return true;
  }

  Future<void> logout() async {
    await api.clearTokens();
    name = null;
    loginName = null;
    userId = 0;
    employeeId = 0;
    roles = [];
    permissions = [];
    notifyListeners();
  }
}
