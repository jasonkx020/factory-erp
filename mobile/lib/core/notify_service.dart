import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:mqtt_client/mqtt_client.dart';
import 'package:mqtt_client/mqtt_server_client.dart';

import 'api_client.dart';

/// Inbox + MQTT (TCP) + local notifications. Falls back to HTTP poll.
class NotifyService extends ChangeNotifier {
  NotifyService(this.api);

  final ApiClient api;
  Timer? _timer;
  MqttServerClient? _client;
  final FlutterLocalNotificationsPlugin _local = FlutterLocalNotificationsPlugin();

  int unread = 0;
  List<dynamic> inbox = [];
  Map<String, dynamic>? mqttInfo;
  String mqttStatus = 'idle';
  int tick = 0;
  bool _started = false;
  bool _notifReady = false;

  Future<void> start() async {
    if (_started) {
      await refresh();
      return;
    }
    _started = true;
    await _initLocalNotifications();
    await refresh();
    await _connectMqtt();
    _timer?.cancel();
    _timer = Timer.periodic(const Duration(seconds: 15), (_) => refresh());
  }

  Future<void> stop() async {
    _started = false;
    _timer?.cancel();
    _timer = null;
    try {
      _client?.disconnect();
    } catch (_) {}
    _client = null;
    mqttStatus = 'idle';
    inbox = [];
    unread = 0;
    notifyListeners();
  }

  Future<void> _initLocalNotifications() async {
    if (_notifReady) return;
    const android = AndroidInitializationSettings('@mipmap/ic_launcher');
    const ios = DarwinInitializationSettings();
    await _local.initialize(
      settings: const InitializationSettings(android: android, iOS: ios),
    );
    if (Platform.isAndroid) {
      await _local
          .resolvePlatformSpecificImplementation<AndroidFlutterLocalNotificationsPlugin>()
          ?.requestNotificationsPermission();
    }
    _notifReady = true;
  }

  Future<void> _showLocal(String title, String body) async {
    if (!_notifReady) return;
    const android = AndroidNotificationDetails(
      'erp_workflow',
      '工作流通知',
      channelDescription: '下一岗待办与知会',
      importance: Importance.high,
      priority: Priority.high,
    );
    const details = NotificationDetails(android: android, iOS: DarwinNotificationDetails());
    await _local.show(
      id: DateTime.now().millisecondsSinceEpoch ~/ 1000,
      title: title,
      body: body,
      notificationDetails: details,
    );
  }

  Future<void> _connectMqtt() async {
    final res = await api.get('/notify/mqtt-connect');
    if (!res.ok || res.data is! Map) {
      mqttStatus = 'error';
      notifyListeners();
      return;
    }
    final mqtt = (res.data as Map)['mqtt'];
    if (mqtt is! Map) {
      mqttStatus = 'closed';
      notifyListeners();
      return;
    }
    mqttInfo = Map<String, dynamic>.from(mqtt);
    if (mqttInfo!['enabled'] != true) {
      mqttStatus = 'closed';
      notifyListeners();
      return;
    }
    final brokerUrl = mqttInfo!['broker_url']?.toString() ?? '';
    final hostPort = _parseBroker(brokerUrl);
    if (hostPort == null) {
      mqttStatus = 'error';
      notifyListeners();
      return;
    }
    mqttStatus = 'connecting';
    notifyListeners();
    final clientId = mqttInfo!['client_id']?.toString() ?? 'erp-mobile-${DateTime.now().millisecondsSinceEpoch}';
    final client = MqttServerClient.withPort(hostPort.$1, clientId, hostPort.$2);
    client.logging(on: false);
    client.keepAlivePeriod = (mqttInfo!['keep_alive_seconds'] as num?)?.toInt() ?? 60;
    client.onDisconnected = () {
      mqttStatus = 'closed';
      notifyListeners();
    };
    client.onConnected = () {
      mqttStatus = 'connected';
      notifyListeners();
    };
    final connMess = MqttConnectMessage()
        .withClientIdentifier(clientId)
        .authenticateAs(
          mqttInfo!['username']?.toString() ?? '',
          mqttInfo!['password']?.toString() ?? '',
        )
        .startClean()
        .withWillQos(MqttQos.atLeastOnce);
    client.connectionMessage = connMess;
    try {
      await client.connect();
    } catch (e) {
      if (kDebugMode) debugPrint('mqtt connect failed: $e');
      mqttStatus = 'error';
      try {
        client.disconnect();
      } catch (_) {}
      notifyListeners();
      return;
    }
    if (client.connectionStatus?.state != MqttConnectionState.connected) {
      mqttStatus = 'error';
      notifyListeners();
      return;
    }
    _client = client;
    final topics = (mqttInfo!['subscribe_topics'] as List?) ?? [];
    for (final t in topics) {
      final topic = t.toString();
      if (topic.isEmpty) continue;
      client.subscribe(topic, MqttQos.atLeastOnce);
    }
    client.updates?.listen((events) async {
      for (final ev in events) {
        final rec = ev.payload as MqttPublishMessage;
        final raw = MqttPublishPayload.bytesToStringAsString(rec.payload.message);
        String title = '工作流通知';
        String body = raw;
        try {
          final j = jsonDecode(raw);
          if (j is Map) {
            title = j['title']?.toString() ?? j['event_key']?.toString() ?? title;
            body = j['body']?.toString() ?? body;
          }
        } catch (_) {}
        await _showLocal(title, body);
      }
      await refresh();
    });
  }

  /// tcp://host:1883 or host:1883
  (String, int)? _parseBroker(String url) {
    var s = url.trim();
    if (s.isEmpty) return null;
    s = s.replaceFirst(RegExp(r'^tcp://'), '');
    s = s.replaceFirst(RegExp(r'^mqtt://'), '');
    final parts = s.split(':');
    if (parts.length >= 2) {
      return (parts[0], int.tryParse(parts[1]) ?? 1883);
    }
    return (s, 1883);
  }

  Future<void> refresh() async {
    final res = await api.get('/notify/inbox?page_num=1&page_size=30');
    if (!res.ok || res.data is! Map) return;
    final m = res.data as Map;
    inbox = (m['list'] as List?) ?? [];
    unread = (m['unread'] as num?)?.toInt() ?? 0;
    tick++;
    notifyListeners();
  }

  Future<void> markRead(int id) async {
    await api.post('/notify/inbox/$id/read', {});
    await refresh();
  }

  static String? routeForEvent(String? eventKey) {
    switch (eventKey) {
      case 'purchase.weigh_confirmed':
        return '/warehouse';
      case 'production.report_confirmed':
        return '/workshop';
      case 'payroll.labor_paid':
        return '/worker';
      default:
        return null;
    }
  }

  static Map<String, dynamic> parsePayload(dynamic raw) {
    if (raw is Map) return Map<String, dynamic>.from(raw);
    if (raw is String && raw.trim().isNotEmpty) {
      try {
        final v = jsonDecode(raw);
        if (v is Map) return Map<String, dynamic>.from(v);
      } catch (_) {}
    }
    return {};
  }
}
