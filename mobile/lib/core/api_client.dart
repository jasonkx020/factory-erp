import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

/// Default: Android emulator → host machine. Override with --dart-define=API_BASE=...
const String kDefaultApiBase = String.fromEnvironment(
  'API_BASE',
  defaultValue: 'http://10.0.2.2:18080/api/v1',
);

const String _kApiBasePref = 'api_base_url';

class ApiEnvelope {
  ApiEnvelope({required this.code, required this.msg, this.data});

  final int code;
  final String msg;
  final dynamic data;

  bool get ok => code == 1;

  factory ApiEnvelope.fromJson(Map<String, dynamic> j) => ApiEnvelope(
        code: (j['code'] as num?)?.toInt() ?? 0,
        msg: j['msg']?.toString() ?? '',
        data: j['data'],
      );
}

/// 把用户输入规范成 `http(s)://host:port/api/v1`
String normalizeApiBase(String raw) {
  var s = raw.trim();
  if (s.isEmpty) return kDefaultApiBase;
  if (!s.contains('://')) {
    s = 'http://$s';
  }
  final uri = Uri.tryParse(s);
  if (uri == null || uri.host.isEmpty) return kDefaultApiBase;
  final scheme = uri.scheme.isEmpty ? 'http' : uri.scheme;
  final port = uri.hasPort ? uri.port : 18080;
  var path = uri.path;
  if (path.isEmpty || path == '/') {
    path = '/api/v1';
  } else if (!path.contains('/api/v1')) {
    path = path.endsWith('/') ? '${path}api/v1' : '$path/api/v1';
  }
  path = path.replaceAll(RegExp(r'/+$'), '');
  return Uri(scheme: scheme, host: uri.host, port: port, path: path).toString();
}

/// 登录页展示用：尽量显示 host:port
String displayApiHost(String baseUrl) {
  final u = Uri.tryParse(baseUrl);
  if (u == null || u.host.isEmpty) return baseUrl;
  if (u.hasPort) return '${u.host}:${u.port}';
  return u.host;
}

class ApiClient {
  ApiClient({String? baseUrl}) : baseUrl = baseUrl ?? kDefaultApiBase;

  String baseUrl;
  String? accessToken;
  String? refreshToken;

  /// 相对路径 `/files/...` → 绝对 URL（去掉 `/api/v1` 前缀挂到同源）
  String resolveMediaUrl(String? path) {
    final p = (path ?? '').trim();
    if (p.isEmpty) return '';
    if (p.startsWith('http://') || p.startsWith('https://')) return p;
    final origin = baseUrl.replaceFirst(RegExp(r'/api/v1/?$'), '');
    if (p.startsWith('/')) return '$origin$p';
    return '$origin/$p';
  }

  Future<void> loadToken() async {
    final p = await SharedPreferences.getInstance();
    accessToken = p.getString('access_token');
    refreshToken = p.getString('refresh_token');
    final saved = p.getString(_kApiBasePref);
    if (saved != null && saved.trim().isNotEmpty) {
      baseUrl = normalizeApiBase(saved);
    }
  }

  Future<void> setBaseUrl(String raw) async {
    baseUrl = normalizeApiBase(raw);
    final p = await SharedPreferences.getInstance();
    await p.setString(_kApiBasePref, baseUrl);
  }

  Future<void> saveTokens(String access, String refresh) async {
    accessToken = access;
    refreshToken = refresh;
    final p = await SharedPreferences.getInstance();
    await p.setString('access_token', access);
    await p.setString('refresh_token', refresh);
  }

  Future<void> clearTokens() async {
    accessToken = null;
    refreshToken = null;
    final p = await SharedPreferences.getInstance();
    await p.remove('access_token');
    await p.remove('refresh_token');
  }

  Map<String, String> _headers({bool auth = true}) {
    final h = <String, String>{'Content-Type': 'application/json'};
    if (auth && accessToken != null && accessToken!.isNotEmpty) {
      h['Authorization'] = 'Bearer $accessToken';
    }
    return h;
  }

  Future<ApiEnvelope> post(String path, Map<String, dynamic> body, {bool auth = true}) async {
    final res = await http.post(
      Uri.parse('$baseUrl$path'),
      headers: _headers(auth: auth),
      body: jsonEncode(body),
    );
    return _parse(res);
  }

  Future<ApiEnvelope> put(String path, Map<String, dynamic> body, {bool auth = true}) async {
    final res = await http.put(
      Uri.parse('$baseUrl$path'),
      headers: _headers(auth: auth),
      body: jsonEncode(body),
    );
    return _parse(res);
  }

  Future<ApiEnvelope> get(String path, {bool auth = true}) async {
    final res = await http.get(
      Uri.parse('$baseUrl$path'),
      headers: _headers(auth: auth),
    );
    return _parse(res);
  }

  Future<ApiEnvelope> delete(String path, {bool auth = true}) async {
    final res = await http.delete(
      Uri.parse('$baseUrl$path'),
      headers: _headers(auth: auth),
    );
    return _parse(res);
  }

  Future<ApiEnvelope> postMultipart(
    String path,
    List<int> bytes, {
    String filename = 'upload.bin',
    String fieldName = 'file',
    bool auth = true,
  }) async {
    final req = http.MultipartRequest('POST', Uri.parse('$baseUrl$path'));
    if (auth && accessToken != null && accessToken!.isNotEmpty) {
      req.headers['Authorization'] = 'Bearer $accessToken';
    }
    req.files.add(http.MultipartFile.fromBytes(fieldName, bytes, filename: filename));
    final streamed = await req.send();
    final res = await http.Response.fromStream(streamed);
    return _parse(res);
  }

  ApiEnvelope _parse(http.Response res) {
    try {
      final j = jsonDecode(res.body);
      if (j is Map<String, dynamic>) return ApiEnvelope.fromJson(j);
    } catch (_) {}
    return ApiEnvelope(code: 0, msg: 'HTTP_${res.statusCode}');
  }

  static List<dynamic> listOf(dynamic data) {
    if (data is Map) return (data['list'] as List?) ?? [];
    if (data is List) return data;
    return [];
  }
}
