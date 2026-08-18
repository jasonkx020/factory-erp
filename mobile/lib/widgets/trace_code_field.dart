import 'package:flutter/material.dart';

import '../features/receiving/batch_code_scanner_page.dart';
import 'form_row.dart';

/// 溯源号/批号/箱码统一录入：点文字区弹键盘，点右侧图标开相机扫码。
class TraceCodeField extends StatelessWidget {
  const TraceCodeField({
    super.key,
    required this.controller,
    this.label = '溯源号',
    this.hint = '点击输入，或点右侧图标扫码',
    this.validated = false,
    this.textCapitalization = TextCapitalization.characters,
    this.onChanged,
    this.onEditingComplete,
    this.onScanned,
    this.onTapManual,
    this.scannerTitle = '扫描溯源号',
    /// 现场默认紧凑（左标签右输入，对齐过磅入厂）；管理向可传 false。
    this.compact = true,
    this.requiredMark = true,
  });

  final TextEditingController controller;
  final String label;
  final String hint;
  final bool validated;
  final TextCapitalization textCapitalization;
  final ValueChanged<String>? onChanged;
  final VoidCallback? onEditingComplete;
  /// Called after a successful camera scan (text already written to [controller]).
  final ValueChanged<String>? onScanned;
  /// 点文字区准备手输（扫码不触发）。
  final VoidCallback? onTapManual;
  final String scannerTitle;
  final bool compact;
  final bool requiredMark;

  Future<void> _openScan(BuildContext context) async {
    final code = await Navigator.of(context).push<String>(
      MaterialPageRoute(builder: (_) => BatchCodeScannerPage(title: scannerTitle)),
    );
    if (!context.mounted || code == null || code.trim().isEmpty) return;
    final normalized = textCapitalization == TextCapitalization.characters
        ? code.trim().toUpperCase()
        : code.trim();
    controller.text = normalized;
    controller.selection = TextSelection.collapsed(offset: normalized.length);
    onChanged?.call(normalized);
    onScanned?.call(normalized);
  }

  @override
  Widget build(BuildContext context) {
    final scanIcon = IconButton(
      tooltip: '扫描',
      onPressed: () => _openScan(context),
      icon: Icon(
        validated ? Icons.check_circle : Icons.qr_code_scanner,
        color: validated ? Colors.teal : Colors.teal.shade700,
      ),
    );
    if (compact) {
      return FormRow(
        label: label,
        requiredMark: requiredMark,
        child: TextField(
          controller: controller,
          textCapitalization: textCapitalization,
          textAlign: TextAlign.right,
          style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
          decoration: FormRow.fieldDecoration(hint: hint, suffixIcon: scanIcon),
          textInputAction: TextInputAction.done,
          onTap: () {
            FormRow.moveCursorToEnd(controller);
            onTapManual?.call();
          },
          onChanged: onChanged,
          onEditingComplete: () {
            onEditingComplete?.call();
            FormRow.dismissKeyboard();
          },
          onSubmitted: (_) {
            onEditingComplete?.call();
            FormRow.dismissKeyboard();
          },
        ),
      );
    }
    return TextField(
      controller: controller,
      textCapitalization: textCapitalization,
      textInputAction: TextInputAction.done,
      style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
      decoration: InputDecoration(
        labelText: label,
        hintText: hint,
        filled: true,
        fillColor: Colors.teal.withValues(alpha: 0.06),
        border: OutlineInputBorder(borderRadius: BorderRadius.circular(8)),
        suffixIcon: scanIcon,
      ),
      onTap: () => onTapManual?.call(),
      onChanged: onChanged,
      onEditingComplete: () {
        onEditingComplete?.call();
        FormRow.dismissKeyboard();
      },
      onSubmitted: (_) {
        onEditingComplete?.call();
        FormRow.dismissKeyboard();
      },
    );
  }
}
