import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

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

  /// 金额/数量类数字键盘（不含电话，避免规范化去掉手机号前导 0）。
  static bool isNumberKeyboard(TextInputType? type) {
    if (type == null) return false;
    if (type == TextInputType.phone) return false;
    if (type == TextInputType.number) return true;
    // TextInputType.numberWithOptions(...)
    return type.decimal != null || type.signed != null;
  }

  static bool looksLikeDefaultZero(String text) {
    final s = text.trim();
    if (s.isEmpty) return false;
    return (double.tryParse(s) ?? double.nan) == 0.0;
  }

  /// 规范数字：0100 → 100；保留正在输入的小数点（如 `0.`）。
  static String normalizeNumericInput(String raw, {bool allowDecimal = true}) {
    var s = raw.trim();
    if (s.isEmpty) return s;
    final neg = s.startsWith('-');
    if (neg) s = s.substring(1);
    final keepTrailingDot = allowDecimal && s.endsWith('.') && '.'.allMatches(s).length == 1;
    if (allowDecimal && s.contains('.')) {
      final i = s.indexOf('.');
      var intPart = s.substring(0, i);
      final frac = s.substring(i + 1).replaceAll('.', '');
      intPart = intPart.replaceFirst(RegExp(r'^0+(?=\d)'), '');
      if (intPart.isEmpty) intPart = '0';
      final out = '$intPart.$frac';
      return '${neg ? '-' : ''}${keepTrailingDot && frac.isEmpty ? '$intPart.' : out}';
    }
    s = s.replaceFirst(RegExp(r'^0+(?=\d)'), '');
    if (s.isEmpty) s = '0';
    return '${neg ? '-' : ''}$s';
  }

  static void applyNormalizedNumber(TextEditingController c, {bool allowDecimal = true}) {
    final next = normalizeNumericInput(c.text, allowDecimal: allowDecimal);
    if (next == c.text) return;
    c.value = TextEditingValue(
      text: next,
      selection: TextSelection.collapsed(offset: next.length),
    );
  }

  /// 点击输入框时：默认 0 全选便于覆盖；否则光标到末尾。
  static void onFieldTap(TextEditingController c, {required bool numeric}) {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final text = c.text;
      if (numeric && looksLikeDefaultZero(text)) {
        c.selection = TextSelection(baseOffset: 0, extentOffset: text.length);
      } else {
        c.selection = TextSelection.collapsed(offset: text.length);
      }
    });
  }

  static void moveCursorToEnd(TextEditingController c) => onFieldTap(c, numeric: false);

  static void dismissKeyboard() {
    FocusManager.instance.primaryFocus?.unfocus();
  }

  /// 对齐过磅入厂的右对齐文本行。
  static Widget text({
    required String label,
    required TextEditingController controller,
    String? hint,
    TextInputType? keyboardType,
    bool requiredMark = false,
    int maxLines = 1,
    bool readOnly = false,
    ValueChanged<String>? onChanged,
    TextInputAction? textInputAction,
  }) {
    final numeric = isNumberKeyboard(keyboardType);
    final allowDecimal = keyboardType?.decimal == true;
    final action = textInputAction ?? (maxLines > 1 ? TextInputAction.newline : TextInputAction.done);

    void finishEdit() {
      if (numeric) applyNormalizedNumber(controller, allowDecimal: allowDecimal);
      dismissKeyboard();
    }

    return FormRow(
      label: label,
      requiredMark: requiredMark,
      child: TextField(
        controller: controller,
        textAlign: maxLines > 1 ? TextAlign.left : TextAlign.right,
        keyboardType: keyboardType,
        textInputAction: action,
        maxLines: maxLines,
        readOnly: readOnly,
        style: const TextStyle(fontSize: 15),
        decoration: fieldDecoration(hint: hint),
        inputFormatters: numeric
            ? [
                FilteringTextInputFormatter.allow(
                  allowDecimal ? RegExp(r'[0-9.-]') : RegExp(r'[0-9-]'),
                ),
              ]
            : null,
        onTap: readOnly ? null : () => onFieldTap(controller, numeric: numeric),
        onChanged: (v) {
          if (numeric) {
            final next = normalizeNumericInput(v, allowDecimal: allowDecimal);
            if (next != v) {
              final sel = next.length;
              controller.value = TextEditingValue(
                text: next,
                selection: TextSelection.collapsed(offset: sel),
              );
            }
          }
          onChanged?.call(controller.text);
        },
        onEditingComplete: finishEdit,
        onSubmitted: (_) => finishEdit(),
      ),
    );
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
