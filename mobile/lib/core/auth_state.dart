import 'dart:convert';

import 'package:flutter/foundation.dart';

import 'api_client.dart';
import 'role_codes.dart';

class AuthState extends ChangeNotifier {
  AuthState(this.api);

  final ApiClient api;

  String? name;
  String? loginName;
  String? empNo;
  String? badgeCode;
  int userId = 0;
  int employeeId = 0;
  List<String> roles = [];
  List<String> permissions = [];
  WorkbenchRole primaryRole = WorkbenchRole.none;
  String error = '';
  bool loading = false;
  /// 本地有 token 时，等 /auth/me 校验完成再进主壳，避免过期会话拉通知 401
  bool sessionReady = true;

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
    final displayName = user['name']?.toString().isNotEmpty == true
        ? user['name']?.toString()
        : user['employee_name']?.toString();
    if (displayName != null && displayName.isNotEmpty) name = displayName;
    loginName = user['login_name']?.toString() ?? loginName;
    final parsedId = _asInt(user['id']) ?? _asInt(user['user_id']);
    if (parsedId != null && parsedId > 0) userId = parsedId;
    employeeId = _asInt(user['employee_id']) ?? employeeId;
    if (user.containsKey('emp_no')) {
      empNo = user['emp_no']?.toString().trim() ?? '';
    }
    if (user.containsKey('badge_code')) {
      badgeCode = user['badge_code']?.toString().trim() ?? '';
    }
  }

  static int? _asInt(dynamic v) {
    if (v == null) return null;
    if (v is num) return v.toInt();
    return int.tryParse(v.toString().trim());
  }

  /// 从 access token 解析 user_id（防止 /auth/me 未带回 id 时本地为 0）
  void syncUserIdFromToken() {
    final tid = _userIdFromJwt(api.accessToken);
    if (tid != null && tid > 0) userId = tid;
  }

  static int? _userIdFromJwt(String? token) {
    if (token == null || token.isEmpty) return null;
    final parts = token.split('.');
    if (parts.length < 2) return null;
    try {
      var payload = parts[1].replaceAll('-', '+').replaceAll('_', '/');
      switch (payload.length % 4) {
        case 2:
          payload += '==';
          break;
        case 3:
          payload += '=';
          break;
      }
      final json = utf8.decode(base64.decode(payload));
      final map = jsonDecode(json);
      if (map is! Map) return null;
      return _asInt(map['user_id']) ?? _asInt(map['uid']) ?? _asInt(map['sub']);
    } catch (_) {
      return null;
    }
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
    syncUserIdFromToken();
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
      sessionReady = true;
      notifyListeners();
      return false;
    }
    final data = r.data as Map<String, dynamic>;
    roles = (data['roles'] as List?)?.map((e) => e.toString()).toList() ?? roles;
    permissions = (data['permissions'] as List?)?.map((e) => e.toString()).toList() ?? permissions;
    _applyUser(data['user'] as Map<String, dynamic>?);
    syncUserIdFromToken();
    _refreshPrimaryRole(keepSelection: true);
    sessionReady = true;
    notifyListeners();
    return true;
  }

  Future<void> logout() async {
    await api.clearTokens();
    name = null;
    loginName = null;
    empNo = null;
    badgeCode = null;
    userId = 0;
    employeeId = 0;
    roles = [];
    permissions = [];
    primaryRole = WorkbenchRole.none;
    error = '';
    sessionReady = true;
    notifyListeners();
  }
}
