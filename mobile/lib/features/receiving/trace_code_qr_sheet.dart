import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:qr_flutter/qr_flutter.dart';

/// 现场出示溯源码二维码（供扫码或拍照印签，不接蓝牙打印）。
Future<void> showTraceCodeQrSheet(
  BuildContext context, {
  required String code,
  String farmerName = '',
}) async {
  final text = code.trim().toUpperCase();
  if (text.isEmpty) return;
  await showDialog<void>(
    context: context,
    barrierDismissible: false,
    builder: (ctx) => Dialog(
      insetPadding: const EdgeInsets.all(16),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 24, 20, 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text('现场溯源码', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700)),
            if (farmerName.trim().isNotEmpty) ...[
              const SizedBox(height: 6),
              Text(farmerName.trim(), style: TextStyle(fontSize: 14, color: Colors.grey.shade700)),
            ],
            const SizedBox(height: 16),
            QrImageView(
              data: text,
              version: QrVersions.auto,
              size: 240,
              backgroundColor: Colors.white,
              eyeStyle: const QrEyeStyle(eyeShape: QrEyeShape.square, color: Colors.black),
              dataModuleStyle: const QrDataModuleStyle(
                dataModuleShape: QrDataModuleShape.square,
                color: Colors.black,
              ),
            ),
            const SizedBox(height: 12),
            SelectableText(
              text,
              style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700, letterSpacing: 0.4),
            ),
            const SizedBox(height: 6),
            Text(
              '请对准扫码或拍照印签',
              style: TextStyle(fontSize: 12, color: Colors.grey.shade600),
            ),
            const SizedBox(height: 8),
            TextButton.icon(
              onPressed: () async {
                await Clipboard.setData(ClipboardData(text: text));
                if (!ctx.mounted) return;
                ScaffoldMessenger.of(ctx).showSnackBar(const SnackBar(content: Text('溯源码已复制')));
              },
              icon: const Icon(Icons.copy, size: 18),
              label: const Text('复制码文'),
            ),
            const SizedBox(height: 8),
            SizedBox(
              width: double.infinity,
              child: FilledButton(
                onPressed: () => Navigator.of(ctx).pop(),
                child: const Text('完成'),
              ),
            ),
          ],
        ),
      ),
    ),
  );
}
