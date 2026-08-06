import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/notify_service.dart';

class InboxPage extends StatelessWidget {
  const InboxPage({super.key});

  @override
  Widget build(BuildContext context) {
    final notify = context.watch<NotifyService>();
    return Scaffold(
      appBar: AppBar(
        title: const Text('收件箱'),
        actions: [
          IconButton(onPressed: notify.refresh, icon: const Icon(Icons.refresh)),
        ],
      ),
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
            child: Text('MQTT: ${notify.mqttStatus} · 未读 ${notify.unread}', style: const TextStyle(color: Colors.black54, fontSize: 12)),
          ),
          Expanded(
            child: notify.inbox.isEmpty
                ? const Center(child: Text('暂无通知'))
                : ListView.separated(
                    itemCount: notify.inbox.length,
                    separatorBuilder: (_, _) => const Divider(height: 1),
                    itemBuilder: (context, i) {
                      final row = Map<String, dynamic>.from(notify.inbox[i] as Map);
                      final unread = (row['read_at']?.toString() ?? '').isEmpty;
                      return ListTile(
                        title: Text(
                          row['title']?.toString() ?? row['event_key']?.toString() ?? '',
                          style: TextStyle(fontWeight: unread ? FontWeight.w600 : FontWeight.normal),
                        ),
                        subtitle: Text('${row['body'] ?? ''}\n${row['created_at'] ?? ''} · ${row['event_key'] ?? ''}'),
                        isThreeLine: true,
                        trailing: const Icon(Icons.chevron_right),
                        onTap: () => notify.openInboxItem(context, row),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}
