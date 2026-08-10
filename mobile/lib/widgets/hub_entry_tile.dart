import 'package:flutter/material.dart';

/// Hub 首页入口行：对齐「我的」`Card` + `ListTile` + chevron。
class HubEntryTile extends StatelessWidget {
  const HubEntryTile({
    super.key,
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
    this.enabled = true,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final muted = Colors.black38;
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        enabled: enabled,
        leading: Icon(icon, color: enabled ? null : muted),
        title: Text(title, style: TextStyle(color: enabled ? null : muted)),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(subtitle, style: TextStyle(color: enabled ? null : Colors.black26)),
            if (!enabled)
              const Padding(
                padding: EdgeInsets.only(top: 2),
                child: Text('当前账号无此权限', style: TextStyle(fontSize: 11, color: Colors.orange)),
              ),
          ],
        ),
        isThreeLine: !enabled,
        trailing: Icon(Icons.chevron_right, color: enabled ? Colors.black45 : Colors.black26),
        onTap: enabled ? onTap : null,
      ),
    );
  }
}
