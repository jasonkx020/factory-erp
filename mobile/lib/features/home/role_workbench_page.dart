import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/auth_state.dart';
import '../../core/role_codes.dart';
import '../../core/role_workbench.dart';

/// 按角色展示的直线步骤工作台（仅导航，不强制工作流）。
class RoleWorkbenchBody extends StatelessWidget {
  const RoleWorkbenchBody({super.key});

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthState>();
    final role = auth.primaryRole;
    final steps = stepsForWorkbenchRole(role);
    final switchable = auth.switchableRoles;

    if (role == WorkbenchRole.none) {
      return const Center(child: Text('当前账号无可访问作业角色，请联系管理员授权。'));
    }

    if (role == WorkbenchRole.admin || steps.isEmpty) {
      return const SizedBox.shrink(); // home 用模块网格兜底
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (switchable.length > 1)
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
            child: Row(
              children: [
                for (final r in switchable)
                  Padding(
                    padding: const EdgeInsets.only(right: 8),
                    child: ChoiceChip(
                      label: Text(workbenchRoleLabel(r)),
                      selected: auth.primaryRole == r,
                      onSelected: (_) => auth.setPrimaryRole(r),
                    ),
                  ),
              ],
            ),
          ),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
          child: Text(
            '${workbenchRoleLabel(role)} · 按步骤推进',
            style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600),
          ),
        ),
        Expanded(
          child: ListView.separated(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
            itemCount: steps.length,
            separatorBuilder: (_, _) => const SizedBox(height: 10),
            itemBuilder: (context, i) {
              final s = steps[i];
              return Card(
                child: ListTile(
                  leading: CircleAvatar(
                    backgroundColor: Theme.of(context).colorScheme.primaryContainer,
                    child: Text('${i + 1}', style: const TextStyle(fontWeight: FontWeight.bold)),
                  ),
                  title: Text(s.title, style: const TextStyle(fontWeight: FontWeight.w600)),
                  subtitle: Text(s.subtitle),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => Navigator.of(context).pushNamed(s.route),
                ),
              );
            },
          ),
        ),
      ],
    );
  }
}
