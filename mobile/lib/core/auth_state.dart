import 'package:flutter/foundation.dart';

import 'api_client.dart';

class AuthState extends ChangeNotifier {
  AuthState(this.api);

  final ApiClient api;

  String? name;
  String? loginName;
  List<String> roles = [];
  List<String> permissions = [];
  String error = '';
  bool loading = false;

  bool get isLoggedIn => api.accessToken != null && api.accessToken!.isNotEmpty;

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
      final user = data['user'] as Map<String, dynamic>?;
      name = user?['name']?.toString();
      loginName = user?['login_name']?.toString() ?? login;
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
    final user = data['user'] as Map<String, dynamic>?;
    loginName = user?['login_name']?.toString() ?? loginName;
    notifyListeners();
    return true;
  }

  Future<void> logout() async {
    await api.clearTokens();
    name = null;
    loginName = null;
    roles = [];
    permissions = [];
    notifyListeners();
  }
}
