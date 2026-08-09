import 'package:flutter/material.dart';

/// 分区小标题（对齐过磅入厂「常用项…」样式）。
class FormSectionHeader extends StatelessWidget {
  const FormSectionHeader(this.text, {super.key, this.padding = const EdgeInsets.fromLTRB(4, 12, 4, 6)});

  final String text;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: padding,
      child: Text(text, style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
    );
  }
}
