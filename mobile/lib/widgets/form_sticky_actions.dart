import 'package:flutter/material.dart';

/// 底部固定操作栏（对齐过磅入厂：次要 Outlined + 主 Filled）。
class FormStickyActions extends StatelessWidget {
  const FormStickyActions({
    super.key,
    this.secondaryLabel,
    this.onSecondary,
    required this.primaryLabel,
    required this.onPrimary,
    this.primaryBusy = false,
    this.busyLabel,
  });

  final String? secondaryLabel;
  final VoidCallback? onSecondary;
  final String primaryLabel;
  final VoidCallback? onPrimary;
  final bool primaryBusy;
  final String? busyLabel;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
        child: Row(
          children: [
            if (secondaryLabel != null) ...[
              OutlinedButton(
                onPressed: primaryBusy ? null : onSecondary,
                child: Text(secondaryLabel!),
              ),
              const SizedBox(width: 12),
            ],
            Expanded(
              child: FilledButton(
                onPressed: primaryBusy ? null : onPrimary,
                child: Text(primaryBusy ? (busyLabel ?? '处理中…') : primaryLabel),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// 底栏多个主/次按钮（如过站：预览 + 提交）。
class FormStickyButtonBar extends StatelessWidget {
  const FormStickyButtonBar({super.key, required this.children});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
        child: Row(
          children: [
            for (var i = 0; i < children.length; i++) ...[
              if (i > 0) const SizedBox(width: 8),
              Expanded(child: children[i]),
            ],
          ],
        ),
      ),
    );
  }
}
