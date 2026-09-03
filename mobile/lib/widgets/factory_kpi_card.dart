import 'package:flutter/material.dart';

import '../theme/plant_colors.dart';

enum FactoryKpiTone { normal, ok, warn, danger }

/// 工业 KPI 卡：左侧色条 + 等宽数字，对齐 Web factory-kpi。
class FactoryKpiCard extends StatelessWidget {
  const FactoryKpiCard({
    super.key,
    required this.label,
    required this.value,
    this.tone = FactoryKpiTone.normal,
    this.subtitle,
  });

  final String label;
  final String value;
  final FactoryKpiTone tone;
  final String? subtitle;

  Color get _bar {
    switch (tone) {
      case FactoryKpiTone.ok:
        return PlantColors.leaf;
      case FactoryKpiTone.warn:
        return PlantColors.warn;
      case FactoryKpiTone.danger:
        return PlantColors.danger;
      case FactoryKpiTone.normal:
        return PlantColors.leafLight;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: PlantColors.border),
      ),
      clipBehavior: Clip.antiAlias,
      child: IntrinsicHeight(
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Container(width: 3, color: _bar),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(12, 10, 12, 10),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      label.toUpperCase(),
                      style: const TextStyle(
                        fontSize: 11,
                        letterSpacing: 0.5,
                        color: PlantColors.muted,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      value,
                      style: const TextStyle(
                        fontSize: 20,
                        fontWeight: FontWeight.w700,
                        fontFeatures: [FontFeature.tabularFigures()],
                        color: PlantColors.text,
                      ),
                    ),
                    if (subtitle != null && subtitle!.isNotEmpty) ...[
                      const SizedBox(height: 2),
                      Text(subtitle!, style: const TextStyle(fontSize: 12, color: PlantColors.muted)),
                    ],
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class StatusLedChip extends StatelessWidget {
  const StatusLedChip({
    super.key,
    required this.label,
    this.run = false,
    this.warn = false,
  });

  final String label;
  final bool run;
  final bool warn;

  @override
  Widget build(BuildContext context) {
    final color = warn ? PlantColors.warn : (run ? PlantColors.leaf : PlantColors.idle);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: PlantColors.panel,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: PlantColors.border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 7,
            height: 7,
            decoration: BoxDecoration(
              color: color,
              shape: BoxShape.circle,
              boxShadow: [
                BoxShadow(color: color.withValues(alpha: 0.45), blurRadius: 4, spreadRadius: 0.5),
              ],
            ),
          ),
          const SizedBox(width: 6),
          Text(label, style: const TextStyle(fontSize: 12, color: PlantColors.text)),
        ],
      ),
    );
  }
}
