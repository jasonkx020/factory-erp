import 'package:flutter/material.dart';

import '../core/recent_code_store.dart';
import '../features/receiving/batch_code_scanner_page.dart';
import 'form_row.dart';

/// 溯源号/批号/板码/箱码统一录入：支持扫码 + 本地最近历史点选。
class TraceCodeField extends StatefulWidget {
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
    /// 现场默认紧凑（左标签右输入）；管理向可传 false。
    this.compact = true,
    this.requiredMark = true,
    /// 非空时启用本地历史，如 [RecentCodeStore.trace] / [RecentCodeStore.board]。
    this.historyKey,
    this.historyLimit = RecentCodeStore.defaultLimit,
  });

  final TextEditingController controller;
  final String label;
  final String hint;
  final bool validated;
  final TextCapitalization textCapitalization;
  final ValueChanged<String>? onChanged;
  final VoidCallback? onEditingComplete;
  final ValueChanged<String>? onScanned;
  final VoidCallback? onTapManual;
  final String scannerTitle;
  final bool compact;
  final bool requiredMark;
  final String? historyKey;
  final int historyLimit;

  @override
  State<TraceCodeField> createState() => _TraceCodeFieldState();
}

class _TraceCodeFieldState extends State<TraceCodeField> {
  List<String> _recent = [];

  bool get _historyEnabled => (widget.historyKey ?? '').trim().isNotEmpty;
  bool get _upper => widget.textCapitalization == TextCapitalization.characters;

  @override
  void initState() {
    super.initState();
    _reloadRecent();
  }

  @override
  void didUpdateWidget(covariant TraceCodeField oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.historyKey != widget.historyKey) {
      _reloadRecent();
    }
  }

  Future<void> _reloadRecent() async {
    if (!_historyEnabled) {
      if (_recent.isNotEmpty && mounted) setState(() => _recent = []);
      return;
    }
    final list = await RecentCodeStore.list(widget.historyKey!, limit: widget.historyLimit);
    if (!mounted) return;
    setState(() => _recent = list);
  }

  Future<void> _rememberCurrent() async {
    if (!_historyEnabled) return;
    final v = widget.controller.text.trim();
    if (v.isEmpty) return;
    final list = await RecentCodeStore.remember(
      widget.historyKey!,
      v,
      limit: widget.historyLimit,
      upper: _upper,
    );
    if (!mounted) return;
    setState(() => _recent = list);
  }

  void _applyValue(String raw, {bool fromScan = false, bool fromHistory = false}) {
    final normalized = _upper ? raw.trim().toUpperCase() : raw.trim();
    if (normalized.isEmpty) return;
    widget.controller.text = normalized;
    widget.controller.selection = TextSelection.collapsed(offset: normalized.length);
    widget.onChanged?.call(normalized);
    if (fromScan) widget.onScanned?.call(normalized);
    // 点选历史后直接走「完成输入」回调（如仓管定位），少点一次按钮。
    if (fromHistory) widget.onEditingComplete?.call();
    _rememberCurrent();
  }

  Future<void> _openScan(BuildContext context) async {
    final code = await Navigator.of(context).push<String>(
      MaterialPageRoute(builder: (_) => BatchCodeScannerPage(title: widget.scannerTitle)),
    );
    if (!context.mounted || code == null || code.trim().isEmpty) return;
    _applyValue(code, fromScan: true);
  }

  Widget _historyChips() {
    if (!_historyEnabled || _recent.isEmpty) return const SizedBox.shrink();
    final current = widget.controller.text.trim().toUpperCase();
    return Padding(
      padding: const EdgeInsets.only(top: 6, bottom: 2),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Text('最近使用', style: TextStyle(fontSize: 12, color: Colors.black.withValues(alpha: 0.55))),
              const Spacer(),
              TextButton(
                style: TextButton.styleFrom(
                  visualDensity: VisualDensity.compact,
                  padding: const EdgeInsets.symmetric(horizontal: 8),
                  minimumSize: Size.zero,
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                ),
                onPressed: () async {
                  await RecentCodeStore.clear(widget.historyKey!);
                  if (mounted) setState(() => _recent = []);
                },
                child: const Text('清空', style: TextStyle(fontSize: 12)),
              ),
            ],
          ),
          const SizedBox(height: 4),
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: [
              for (final code in _recent)
                InputChip(
                  label: Text(code, style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600)),
                  selected: current == code.toUpperCase(),
                  onPressed: () => _applyValue(code, fromHistory: true),
                  onDeleted: () async {
                    await RecentCodeStore.remove(widget.historyKey!, code);
                    await _reloadRecent();
                  },
                  materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  visualDensity: VisualDensity.compact,
                ),
            ],
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final scanIcon = IconButton(
      tooltip: '扫描',
      onPressed: () => _openScan(context),
      icon: Icon(
        widget.validated ? Icons.check_circle : Icons.qr_code_scanner,
        color: widget.validated ? Colors.teal : Colors.teal.shade700,
      ),
    );

    final field = widget.compact
        ? FormRow(
            label: widget.label,
            requiredMark: widget.requiredMark,
            child: TextField(
              controller: widget.controller,
              textCapitalization: widget.textCapitalization,
              textAlign: TextAlign.right,
              style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
              decoration: FormRow.fieldDecoration(hint: widget.hint, suffixIcon: scanIcon),
              textInputAction: TextInputAction.done,
              onTap: () {
                FormRow.moveCursorToEnd(widget.controller);
                widget.onTapManual?.call();
              },
              onChanged: widget.onChanged,
              onEditingComplete: () {
                widget.onEditingComplete?.call();
                _rememberCurrent();
                FormRow.dismissKeyboard();
              },
              onSubmitted: (_) {
                widget.onEditingComplete?.call();
                _rememberCurrent();
                FormRow.dismissKeyboard();
              },
            ),
          )
        : TextField(
            controller: widget.controller,
            textCapitalization: widget.textCapitalization,
            textInputAction: TextInputAction.done,
            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
            decoration: InputDecoration(
              labelText: widget.label,
              hintText: widget.hint,
              filled: true,
              fillColor: Colors.teal.withValues(alpha: 0.06),
              border: OutlineInputBorder(borderRadius: BorderRadius.circular(8)),
              suffixIcon: scanIcon,
            ),
            onTap: () => widget.onTapManual?.call(),
            onChanged: widget.onChanged,
            onEditingComplete: () {
              widget.onEditingComplete?.call();
              _rememberCurrent();
              FormRow.dismissKeyboard();
            },
            onSubmitted: (_) {
              widget.onEditingComplete?.call();
              _rememberCurrent();
              FormRow.dismissKeyboard();
            },
          );

    if (!_historyEnabled) return field;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        field,
        _historyChips(),
      ],
    );
  }
}
