import 'package:flutter/material.dart';

/// `extendBody` + 半透明底栏时，列表需额外垫高，避免末项被 NavigationBar 挡住。
double tabShellBottomPadding(
  BuildContext context, {
  bool asTab = true,
  double extra = 16,
}) {
  if (!asTab) return extra;
  final safe = MediaQuery.viewPaddingOf(context).bottom;
  // Material 3 NavigationBar 默认高度约 80
  const navH = 80.0;
  return extra + navH + safe;
}
