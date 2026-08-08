import 'package:flutter/material.dart';

/// 单行表单项：左侧名称，右侧数值/控件。
class FormRow extends StatelessWidget {
  const FormRow({
    super.key,
    required this.label,
    required this.child,
    this.labelWidth = 108,
    this.requiredMark = false,
  });

  final String label;
  final Widget child;
  final double labelWidth;
  final bool requiredMark;

  static InputDecoration fieldDecoration({String? hint, Widget? suffixIcon, Widget? prefixIcon}) {
    return InputDecoration(
      isDense: true,
      hintText: hint,
      border: InputBorder.none,
      contentPadding: const EdgeInsets.symmetric(horizontal: 4, vertical: 10),
      suffixIcon: suffixIcon,
      prefixIcon: prefixIcon,
      suffixIconConstraints: const BoxConstraints(minWidth: 36, minHeight: 36),
      prefixIconConstraints: const BoxConstraints(minWidth: 36, minHeight: 36),
    );
  }

  /// 点击输入框时把光标移到文本末尾，方便改已有数值。
  /// 延后一帧，避免被 TextField 默认点选位置覆盖。
  static void moveCursorToEnd(TextEditingController c) {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final text = c.text;
      c.selection = TextSelection.collapsed(offset: text.length);
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      margin: const EdgeInsets.only(bottom: 1),
      padding: const EdgeInsets.symmetric(horizontal: 4),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: Colors.black.withValues(alpha: 0.08))),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          SizedBox(
            width: labelWidth,
            child: Text.rich(
              TextSpan(
                children: [
                  TextSpan(
                    text: label,
                    style: TextStyle(fontSize: 14, color: theme.colorScheme.onSurface.withValues(alpha: 0.75)),
                  ),
                  if (requiredMark)
                    const TextSpan(text: ' *', style: TextStyle(color: Colors.redAccent, fontSize: 14)),
                ],
              ),
            ),
          ),
          Expanded(child: child),
        ],
      ),
    );
  }
}
