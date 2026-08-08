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
    this.scannerTitle = '扫描溯源号',
    /// true：左标签右输入（单行）
    this.compact = false,
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
  final String scannerTitle;
  final bool compact;

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
        requiredMark: true,
        child: TextField(
          controller: controller,
          textCapitalization: textCapitalization,
          textAlign: TextAlign.right,
          style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
          decoration: FormRow.fieldDecoration(hint: hint, suffixIcon: scanIcon),
          onTap: () => FormRow.moveCursorToEnd(controller),
          onChanged: onChanged,
          onEditingComplete: onEditingComplete,
        ),
      );
    }
    return TextField(
      controller: controller,
      textCapitalization: textCapitalization,
      style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
      decoration: InputDecoration(
        labelText: label,
        hintText: hint,
        filled: true,
        fillColor: Colors.teal.withValues(alpha: 0.06),
        border: OutlineInputBorder(borderRadius: BorderRadius.circular(8)),
        suffixIcon: scanIcon,
      ),
      onChanged: onChanged,
      onEditingComplete: onEditingComplete,
    );
  }
}
