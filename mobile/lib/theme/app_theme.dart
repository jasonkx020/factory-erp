import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'plant_colors.dart';

abstract final class AppTheme {
  static ThemeData get light {
    final scheme = ColorScheme.fromSeed(
      seedColor: PlantColors.leaf,
      primary: PlantColors.leaf,
      secondary: PlantColors.soil,
      surface: Colors.white,
      error: PlantColors.danger,
      brightness: Brightness.light,
    ).copyWith(
      primaryContainer: PlantColors.soft,
      onPrimaryContainer: PlantColors.forest,
      secondaryContainer: const Color(0xFFF3E8D8),
      onSecondaryContainer: PlantColors.soil,
      tertiary: PlantColors.leafLight,
      outline: PlantColors.border,
      surfaceContainerHighest: PlantColors.panel,
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: PlantColors.bg,
      appBarTheme: const AppBarTheme(
        backgroundColor: PlantColors.forest,
        foregroundColor: PlantColors.onForest,
        elevation: 0,
        centerTitle: false,
        systemOverlayStyle: SystemUiOverlayStyle.light,
        titleTextStyle: TextStyle(
          color: PlantColors.onForest,
          fontSize: 18,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.2,
        ),
        iconTheme: IconThemeData(color: PlantColors.onForest),
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: PlantColors.forest.withValues(alpha: 0.82),
        elevation: 0,
        shadowColor: Colors.transparent,
        surfaceTintColor: Colors.transparent,
        indicatorColor: PlantColors.leaf.withValues(alpha: 0.35),
        labelTextStyle: WidgetStateProperty.resolveWith((states) {
          final selected = states.contains(WidgetState.selected);
          return TextStyle(
            fontSize: 12,
            fontWeight: selected ? FontWeight.w600 : FontWeight.w500,
            color: selected ? Colors.white : PlantColors.onForest.withValues(alpha: 0.72),
          );
        }),
        iconTheme: WidgetStateProperty.resolveWith((states) {
          final selected = states.contains(WidgetState.selected);
          return IconThemeData(
            color: selected ? Colors.white : PlantColors.onForest.withValues(alpha: 0.72),
          );
        }),
      ),
      bottomAppBarTheme: BottomAppBarThemeData(
        color: PlantColors.forest.withValues(alpha: 0.82),
        elevation: 0,
        shadowColor: Colors.transparent,
        surfaceTintColor: Colors.transparent,
      ),
      floatingActionButtonTheme: const FloatingActionButtonThemeData(
        backgroundColor: PlantColors.leaf,
        foregroundColor: Colors.white,
      ),
      cardTheme: CardThemeData(
        color: Colors.white,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
          side: const BorderSide(color: PlantColors.border),
        ),
      ),
      chipTheme: ChipThemeData(
        backgroundColor: PlantColors.panel,
        selectedColor: PlantColors.soft,
        side: const BorderSide(color: PlantColors.border),
        labelStyle: const TextStyle(color: PlantColors.text, fontSize: 13),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: PlantColors.leaf,
          foregroundColor: Colors.white,
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: Colors.white,
        border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: const BorderSide(color: PlantColors.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: const BorderSide(color: PlantColors.leaf, width: 1.5),
        ),
      ),
      dividerTheme: const DividerThemeData(color: PlantColors.border),
      snackBarTheme: SnackBarThemeData(
        backgroundColor: PlantColors.forest,
        contentTextStyle: const TextStyle(color: PlantColors.onForest),
        behavior: SnackBarBehavior.floating,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
      ),
    );
  }
}
