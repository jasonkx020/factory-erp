import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api_client.dart';
import '../../core/auth_state.dart';
import '../../core/employee_modules.dart';

/// 知识库 / 图纸 / 公告 / 学堂（只读）
class KnowledgePage extends StatefulWidget {
  const KnowledgePage({super.key});

  @override
  State<KnowledgePage> createState() => _KnowledgePageState();
}

class _KnowledgePageState extends State<KnowledgePage> {
  int _tab = 0;
  List<dynamic> _knowledge = [];
  List<dynamic> _drawings = [];
  List<dynamic> _announcements = [];
  List<dynamic> _courses = [];
  Map<String, dynamic>? _detail;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load(0));
  }

  Future<void> _load(int tab) async {
    final auth = context.read<AuthState>();
    if (!canAccessEmployeeModule(EmployeeModule.knowledge, auth.permissions, auth.roles)) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('无资料权限')));
        Navigator.of(context).pop();
      }
      return;
    }
    final api = auth.api;
    setState(() => _detail = null);
    switch (tab) {
      case 0:
        final r = await api.get('/system/knowledge?page_size=50');
        if (mounted && r.ok) setState(() => _knowledge = ApiClient.listOf(r.data));
        break;
      case 1:
        final r = await api.get('/system/drawings?page_size=50');
        if (mounted && r.ok) setState(() => _drawings = ApiClient.listOf(r.data));
        break;
      case 2:
        final r = await api.get('/system/announcements?page_size=50');
        if (mounted && r.ok) setState(() => _announcements = ApiClient.listOf(r.data));
        break;
      case 3:
        final r = await api.get('/system/courses?page_size=50');
        if (mounted && r.ok) setState(() => _courses = ApiClient.listOf(r.data));
        break;
    }
  }

  Future<void> _open(String path) async {
    final r = await context.read<AuthState>().api.get(path);
    if (!mounted) return;
    if (r.ok && r.data is Map) {
      setState(() => _detail = Map<String, dynamic>.from(r.data as Map));
    } else {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(r.msg)));
    }
  }

  Widget _list(List<dynamic> rows, {required String titleKey, String? subtitleKey, required String idPath}) {
    if (rows.isEmpty) return const Center(child: Text('暂无内容'));
    return ListView.builder(
      padding: const EdgeInsets.all(12),
      itemCount: rows.length + (_detail == null ? 0 : 1),
      itemBuilder: (context, i) {
        if (_detail != null && i == 0) {
          return Card(
            color: Colors.teal.shade50,
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(child: Text('${_detail!['title'] ?? _detail!['name'] ?? _detail!['code'] ?? ''}', style: const TextStyle(fontWeight: FontWeight.bold))),
                      IconButton(onPressed: () => setState(() => _detail = null), icon: const Icon(Icons.close)),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Text('${_detail!['content'] ?? _detail!['file_url'] ?? _detail!['category'] ?? ''}'),
                ],
              ),
            ),
          );
        }
        final idx = _detail == null ? i : i - 1;
        final m = Map<String, dynamic>.from(rows[idx] as Map);
        final id = (m['id'] as num?)?.toInt();
        return ListTile(
          title: Text('${m[titleKey] ?? m['code'] ?? m['id']}'),
          subtitle: subtitleKey == null ? null : Text('${m[subtitleKey] ?? m['category'] ?? m['status'] ?? ''}'),
          trailing: const Icon(Icons.chevron_right),
          onTap: id == null ? null : () => _open('$idPath/$id'),
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('资料中心')),
      body: IndexedStack(
        index: _tab,
        children: [
          _list(_knowledge, titleKey: 'title', subtitleKey: 'category', idPath: '/system/knowledge'),
          _list(_drawings, titleKey: 'title', subtitleKey: 'version_no', idPath: '/system/drawings'),
          _list(_announcements, titleKey: 'title', subtitleKey: 'status', idPath: '/system/announcements'),
          _list(_courses, titleKey: 'title', subtitleKey: 'category', idPath: '/system/courses'),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _tab,
        onDestinationSelected: (i) {
          setState(() => _tab = i);
          _load(i);
        },
        destinations: const [
          NavigationDestination(icon: Icon(Icons.menu_book), label: '知识库'),
          NavigationDestination(icon: Icon(Icons.architecture), label: '图纸'),
          NavigationDestination(icon: Icon(Icons.campaign), label: '公告'),
          NavigationDestination(icon: Icon(Icons.school), label: '学堂'),
        ],
      ),
    );
  }
}
