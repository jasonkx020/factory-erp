import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

/// Compiled-in API base from `--dart-define=API_BASE=...` (empty if unset).
const String kApiBaseDefine = String.fromEnvironment('API_BASE');

/// Default: Android emulator → host machine. Override with --dart-define=API_BASE=...
const String kFallbackApiBase = 'http://10.0.2.2:18080/api/v1';

String get kDefaultApiBase =>
    kApiBaseDefine.trim().isEmpty ? kFallbackApiBase : normalizeApiBase(kApiBaseDefine);

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
    // --dart-define=API_BASE 优先于本地缓存，避免旧地址导致启动卡死白屏
    if (kApiBaseDefine.trim().isNotEmpty) {
      baseUrl = normalizeApiBase(kApiBaseDefine);
      return;
    }
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

  Future<ApiEnvelope> post(String path, Map<String, dynamic> body, {bool auth = true}) {
    return _send(
      'POST',
      path,
      () => http.post(
        Uri.parse('$baseUrl$path'),
        headers: _headers(auth: auth),
        body: jsonEncode(body),
      ),
      requestBody: body,
    );
  }

  Future<ApiEnvelope> put(String path, Map<String, dynamic> body, {bool auth = true}) {
    return _send(
      'PUT',
      path,
      () => http.put(
        Uri.parse('$baseUrl$path'),
        headers: _headers(auth: auth),
        body: jsonEncode(body),
      ),
      requestBody: body,
    );
  }

  Future<ApiEnvelope> get(String path, {bool auth = true}) {
    return _send(
      'GET',
      path,
      () => http.get(
        Uri.parse('$baseUrl$path'),
        headers: _headers(auth: auth),
      ),
    );
  }

  Future<ApiEnvelope> delete(String path, {bool auth = true}) {
    return _send(
      'DELETE',
      path,
      () => http.delete(
        Uri.parse('$baseUrl$path'),
        headers: _headers(auth: auth),
      ),
    );
  }

  Future<ApiEnvelope> postMultipart(
    String path,
    List<int> bytes, {
    String filename = 'upload.bin',
    String fieldName = 'file',
    bool auth = true,
  }) {
    return _send(
      'POST',
      path,
      () async {
        final req = http.MultipartRequest('POST', Uri.parse('$baseUrl$path'));
        if (auth && accessToken != null && accessToken!.isNotEmpty) {
          req.headers['Authorization'] = 'Bearer $accessToken';
        }
        req.files.add(http.MultipartFile.fromBytes(fieldName, bytes, filename: filename));
        final streamed = await req.send();
        return http.Response.fromStream(streamed);
      },
      requestBody: {'field': fieldName, 'filename': filename, 'bytes': bytes.length},
    );
  }

  Future<ApiEnvelope> _send(
    String method,
    String path,
    Future<http.Response> Function() send, {
    Object? requestBody,
  }) async {
    final url = '$baseUrl$path';
    try {
      final res = await send().timeout(const Duration(seconds: 12));
      return _parse(method, url, res, requestBody: requestBody);
    } catch (e, st) {
      _logApiError(method, url, error: e, stack: st, requestBody: requestBody);
      return ApiEnvelope(code: 0, msg: e.toString());
    }
  }

  ApiEnvelope _parse(String method, String url, http.Response res, {Object? requestBody}) {
    ApiEnvelope env;
    try {
      final j = jsonDecode(res.body);
      env = j is Map<String, dynamic>
          ? ApiEnvelope.fromJson(j)
          : ApiEnvelope(code: 0, msg: 'HTTP_${res.statusCode}');
    } catch (_) {
      env = ApiEnvelope(code: 0, msg: 'HTTP_${res.statusCode}');
    }
    final httpBad = res.statusCode < 200 || res.statusCode >= 300;
    if (httpBad || !env.ok) {
      _logApiError(
        method,
        url,
        status: res.statusCode,
        code: env.code,
        msg: env.msg,
        responseBody: res.body,
        requestBody: requestBody,
      );
    }
    return env;
  }

  static const int _logBodyLimit = 4000;

  static void _logApiError(
    String method,
    String url, {
    int? status,
    int? code,
    String? msg,
    String? responseBody,
    Object? requestBody,
    Object? error,
    StackTrace? stack,
  }) {
    final line = StringBuffer('[API ERROR] $method $url');
    if (status != null) line.write(' HTTP $status');
    if (code != null) line.write(' code=$code');
    if (msg != null && msg.isNotEmpty) line.write(' msg=$msg');
    if (error != null) line.write(' exception=$error');
    debugPrint(line.toString());
    if (requestBody != null) {
      debugPrint('[API ERROR] request: ${_clipJson(_redact(requestBody))}');
    }
    if (responseBody != null && responseBody.isNotEmpty) {
      debugPrint('[API ERROR] response: ${_clip(responseBody)}');
    }
    if (kDebugMode && stack != null) {
      debugPrint('[API ERROR] $stack');
    }
  }

  static Object? _redact(Object? body) {
    if (body is Map) {
      return body.map((k, v) {
        final key = k.toString().toLowerCase();
        if (key.contains('password') || key.contains('token') || key.contains('secret')) {
          return MapEntry(k, '***');
        }
        return MapEntry(k, v);
      });
    }
    return body;
  }

  static String _clipJson(Object? value) {
    try {
      return _clip(jsonEncode(value));
    } catch (_) {
      return _clip(value.toString());
    }
  }

  static String _clip(String s) {
    if (s.length <= _logBodyLimit) return s;
    return '${s.substring(0, _logBodyLimit)}…(${s.length} chars)';
  }

  static List<dynamic> listOf(dynamic data) {
    if (data is Map) return (data['list'] as List?) ?? [];
    if (data is List) return data;
    return [];
  }
}
