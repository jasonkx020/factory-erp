import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

/// Default: Android emulator → host machine. Override with --dart-define=API_BASE=...
const String kDefaultApiBase = String.fromEnvironment(
  'API_BASE',
  defaultValue: 'http://10.0.2.2:18080/api/v1',
);

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

class ApiClient {
  ApiClient({this.baseUrl = kDefaultApiBase});

  final String baseUrl;
  String? accessToken;
  String? refreshToken;

  Future<void> loadToken() async {
    final p = await SharedPreferences.getInstance();
    accessToken = p.getString('access_token');
    refreshToken = p.getString('refresh_token');
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
