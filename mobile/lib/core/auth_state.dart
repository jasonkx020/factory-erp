import 'package:flutter/foundation.dart';

import 'api_client.dart';
import 'role_codes.dart';

class AuthState extends ChangeNotifier {
  AuthState(this.api);

  final ApiClient api;

  String? name;
  String? loginName;
  int userId = 0;
  int employeeId = 0;
  List<String> roles = [];
  List<String> permissions = [];
  WorkbenchRole primaryRole = WorkbenchRole.none;
  String error = '';
  bool loading = false;

  bool get isLoggedIn => api.accessToken != null && api.accessToken!.isNotEmpty;

  /// 人事开户入口：hr / sys_admin / 含「人事管理」权限
  bool get canHrOnboard {
    for (final r in roles) {
      final c = r.toLowerCase();
      if (c == 'hr' || c == 'sys_admin' || r == '人事' || r == '系统管理员') return true;
    }
    for (final p in permissions) {
      if (p.contains('人事管理')) return true;
    }
    return false;
  }

  List<WorkbenchRole> get switchableRoles => availableWorkbenchRoles(roles);

  void _applyUser(Map<String, dynamic>? user) {
    if (user == null) return;
    name = user['name']?.toString().isNotEmpty == true ? user['name']?.toString() : name;
    loginName = user['login_name']?.toString() ?? loginName;
    userId = (user['id'] as num?)?.toInt() ?? userId;
    employeeId = (user['employee_id'] as num?)?.toInt() ?? employeeId;
  }

  void _refreshPrimaryRole({bool keepSelection = false}) {
    final available = availableWorkbenchRoles(roles);
    if (keepSelection && available.contains(primaryRole)) {
      return;
    }
    primaryRole = resolvePrimaryWorkbenchRole(roles);
  }

  void setPrimaryRole(WorkbenchRole role) {
    if (primaryRole == role) return;
    final available = availableWorkbenchRoles(roles);
    if (!available.contains(role) && role != WorkbenchRole.admin) return;
    primaryRole = role;
    notifyListeners();
  }

  Future<bool> _applyLoginData(Map<String, dynamic> data) async {
    await api.saveTokens(
      data['access_token']?.toString() ?? '',
      data['refresh_token']?.toString() ?? '',
    );
    roles = (data['roles'] as List?)?.map((e) => e.toString()).toList() ?? [];
    permissions = (data['permissions'] as List?)?.map((e) => e.toString()).toList() ?? [];
    _applyUser(data['user'] as Map<String, dynamic>?);
    // 登录成功后一律以 /auth/me 为准刷新角色权限
    final ok = await fetchMe();
    _refreshPrimaryRole();
    notifyListeners();
    return ok;
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
      return await _applyLoginData(r.data as Map<String, dynamic>);
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  /// 第三方 OAuth 交换；未开通时后端返回 OAUTH_NOT_CONFIGURED。
  Future<bool> loginWithOAuth({required String provider, required String code}) async {
    loading = true;
    error = '';
    notifyListeners();
    try {
      final r = await api.post(
        '/auth/oauth/token',
        {
          'provider': provider,
          'code': code,
          'client_type': 'mobile',
        },
        auth: false,
      );
      if (!r.ok || r.data is! Map) {
        error = r.msg.isEmpty ? 'OAUTH_FAILED' : r.msg;
        return false;
      }
      return await _applyLoginData(r.data as Map<String, dynamic>);
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
    _refreshPrimaryRole(keepSelection: true);
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
    primaryRole = WorkbenchRole.none;
    notifyListeners();
  }
}
