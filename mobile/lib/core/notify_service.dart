import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:mqtt_client/mqtt_client.dart';
import 'package:mqtt_client/mqtt_server_client.dart';

import 'api_client.dart';

/// Global navigator for local-notification deep links (cold / background tap).
final GlobalKey<NavigatorState> appNavigatorKey = GlobalKey<NavigatorState>();

/// Resolved deep-link target from inbox / push payload.
class NotifyTarget {
  const NotifyTarget({required this.route, this.arguments = const {}});

  final String route;
  final Map<String, dynamic> arguments;
}

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
  bool _launchHandled = false;
  Map<String, dynamic>? _pendingLaunchArgs;

  Future<void> start() async {
    if (_started) {
      await refresh();
      await consumePendingLaunch();
      return;
    }
    _started = true;
    await _initLocalNotifications();
    await refresh();
    await _connectMqtt();
    _timer?.cancel();
    _timer = Timer.periodic(const Duration(seconds: 15), (_) => refresh());
    await consumePendingLaunch();
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
      onDidReceiveNotificationResponse: _onLocalNotificationResponse,
    );
    if (Platform.isAndroid) {
      await _local
          .resolvePlatformSpecificImplementation<AndroidFlutterLocalNotificationsPlugin>()
          ?.requestNotificationsPermission();
    }
    final launch = await _local.getNotificationAppLaunchDetails();
    if (launch?.didNotificationLaunchApp == true) {
      final payload = launch!.notificationResponse?.payload;
      if (payload != null && payload.isNotEmpty) {
        _pendingLaunchArgs = _decodeNotifyPayload(payload);
      }
    }
    _notifReady = true;
  }

  void _onLocalNotificationResponse(NotificationResponse response) {
    final raw = response.payload;
    if (raw == null || raw.isEmpty) return;
    final data = _decodeNotifyPayload(raw);
    if (data == null) return;
    final ctx = appNavigatorKey.currentContext;
    if (ctx == null) {
      _pendingLaunchArgs = data;
      return;
    }
    navigateFromPayload(ctx, data);
  }

  /// After login / start, apply cold-start notification deep link once.
  Future<void> consumePendingLaunch() async {
    if (_launchHandled) return;
    final data = _pendingLaunchArgs;
    if (data == null) return;
    _pendingLaunchArgs = null;
    // Wait a frame so MaterialApp / home are mounted.
    await Future<void>.delayed(const Duration(milliseconds: 300));
    final nav = appNavigatorKey.currentState;
    if (nav == null || !nav.mounted) {
      _pendingLaunchArgs = data;
      return;
    }
    _launchHandled = true;
    final target = resolveNotifyTarget(data['event_key']?.toString(), Map<String, dynamic>.from(data));
    if (target == null) return;
    nav.pushNamed(target.route, arguments: target.arguments);
  }

  Map<String, dynamic>? _decodeNotifyPayload(String raw) {
    try {
      final v = jsonDecode(raw);
      if (v is Map) return Map<String, dynamic>.from(v);
    } catch (_) {}
    return null;
  }

  Future<void> _showLocal(String title, String body, {Map<String, dynamic>? payload}) async {
    if (!_notifReady) return;
    const android = AndroidNotificationDetails(
      'erp_workflow',
      '工作流通知',
      channelDescription: '下一岗待办与知会',
      importance: Importance.high,
      priority: Priority.high,
    );
    const details = NotificationDetails(android: android, iOS: DarwinNotificationDetails());
    final payloadStr = payload == null || payload.isEmpty ? null : jsonEncode(payload);
    await _local.show(
      id: DateTime.now().millisecondsSinceEpoch ~/ 1000,
      title: title,
      body: body,
      notificationDetails: details,
      payload: payloadStr,
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
        Map<String, dynamic> pushPayload = {};
        try {
          final j = jsonDecode(raw);
          if (j is Map) {
            final m = Map<String, dynamic>.from(j);
            title = m['title']?.toString() ?? m['event_key']?.toString() ?? title;
            body = m['body']?.toString() ?? body;
            final inner = parsePayload(m['payload'] ?? m['payload_json']);
            pushPayload = {
              'event_key': m['event_key']?.toString() ?? '',
              ...inner,
            };
            if (m['ticket_id'] != null) pushPayload['ticket_id'] = m['ticket_id'];
            if (m['task_id'] != null) pushPayload['task_id'] = m['task_id'];
            if (inner['ticket_id'] != null) pushPayload['ticket_id'] = inner['ticket_id'];
            if (inner['employee_route'] != null) pushPayload['employee_route'] = inner['employee_route'];
          }
        } catch (_) {}
        await _showLocal(title, body, payload: pushPayload.isEmpty ? null : pushPayload);
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

  /// Mark read (if any) then navigate to the pending page for this inbox row.
  Future<void> openInboxItem(BuildContext context, Map<String, dynamic> row) async {
    final id = (row['id'] as num?)?.toInt();
    if (id != null) await markRead(id);
    if (!context.mounted) return;
    final eventKey = row['event_key']?.toString();
    final payload = parsePayload(row['payload_json'] ?? row['payload']);
    final target = resolveNotifyTarget(eventKey, payload);
    if (target == null) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('暂无可跳转页面')),
        );
      }
      return;
    }
    navigateToTarget(context, target);
  }

  void navigateFromPayload(BuildContext context, Map<String, dynamic> data) {
    final eventKey = data['event_key']?.toString();
    final payload = Map<String, dynamic>.from(data);
    final target = resolveNotifyTarget(eventKey, payload);
    if (target == null) return;
    navigateToTarget(context, target);
  }

  void navigateToTarget(BuildContext context, NotifyTarget target) {
    Navigator.of(context).pushNamed(target.route, arguments: target.arguments);
  }

  static NotifyTarget? resolveNotifyTarget(String? eventKey, Map<String, dynamic> payload) {
    final ek = (eventKey ?? payload['event_key']?.toString() ?? '').trim();
    var route = payload['employee_route']?.toString().trim();
    if (route != null && route.isEmpty) route = null;
    // Normalize Web /m/* paths if present in payload.
    if (route != null && route.startsWith('/m/')) {
      route = route.substring(2);
    }
    route ??= routeForEvent(ek);
    if (route == null || route.isEmpty) return null;

    final args = <String, dynamic>{
      'event_key': ek,
      'payload': payload,
    };
    final ticketId = payload['ticket_id'];
    if (ticketId is num) {
      args['ticket_id'] = ticketId.toInt();
    } else if (ticketId != null) {
      final n = int.tryParse(ticketId.toString());
      if (n != null) args['ticket_id'] = n;
    }
    return NotifyTarget(route: route, arguments: args);
  }

  static String? routeForEvent(String? eventKey) {
    final k = (eventKey ?? '').trim();
    switch (k) {
      case 'purchase.weigh_confirmed':
        return '/warehouse';
      case 'production.report_confirmed':
        return '/workshop';
      case 'payroll.labor_paid':
        return '/worker';
      case 'purchase.stocked':
      case 'purchase.settle_paid':
        return '/receiving';
      default:
        if (k.startsWith('workflow.ticket')) return '/tickets';
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
