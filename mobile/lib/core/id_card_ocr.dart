/// 身份证 OCR 可插拔适配器。
class IdCardOcrResult {
  const IdCardOcrResult({this.name = '', this.idCardNo = '', this.address = '', this.configured = true, this.message = ''});

  final String name;
  final String idCardNo;
  final String address;
  final bool configured;
  final String message;
}

abstract class IdCardOcr {
  Future<IdCardOcrResult> recognizeBytes(List<int> bytes, {String filename = 'idcard.jpg'});
}

/// 未配置时返回空结果，不阻塞手填开户。
class LocalStubIdCardOcr implements IdCardOcr {
  @override
  Future<IdCardOcrResult> recognizeBytes(List<int> bytes, {String filename = 'idcard.jpg'}) async {
    return const IdCardOcrResult(
      configured: false,
      message: 'OCR 未配置，请手填姓名与身份证号',
    );
  }
}

/// 调用后端 `POST /hr/id-card/ocr`；未开通时返回 OCR_NOT_CONFIGURED。
class HttpIdCardOcr implements IdCardOcr {
  HttpIdCardOcr(this._upload);

  final Future<({bool ok, String msg, Map<String, dynamic>? data})> Function(
    List<int> bytes,
    String filename,
  ) _upload;

  @override
  Future<IdCardOcrResult> recognizeBytes(List<int> bytes, {String filename = 'idcard.jpg'}) async {
    final r = await _upload(bytes, filename);
    if (!r.ok) {
      final notCfg = r.msg.contains('OCR_NOT_CONFIGURED') || r.msg.contains('NOT_IMPLEMENTED');
      return IdCardOcrResult(
        configured: !notCfg,
        message: notCfg ? 'OCR 暂未开通，请手填' : r.msg,
      );
    }
    final d = r.data ?? {};
    return IdCardOcrResult(
      name: '${d['name'] ?? ''}',
      idCardNo: '${d['id_card_no'] ?? ''}',
      address: '${d['address'] ?? ''}',
    );
  }
}
