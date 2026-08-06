import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../../core/auth_state.dart';
import '../../core/id_card_ocr.dart';

/// 人事在 App 开户：建档 + 可选 OCR + 确认开户（初始密码 ChangeMe123）
class HrOnboardPage extends StatefulWidget {
  const HrOnboardPage({super.key});

  @override
  State<HrOnboardPage> createState() => _HrOnboardPageState();
}

class _HrOnboardPageState extends State<HrOnboardPage> {
  final _empNo = TextEditingController();
  final _name = TextEditingController();
  final _idCard = TextEditingController();
  final _mobile = TextEditingController();
  final _login = TextEditingController();
  final _job = TextEditingController();
  final _workshop = TextEditingController(text: '1');
  final _dept = TextEditingController(text: '1');
  String _empType = 'piece';
  bool _needAccount = true;
  String _msg = '';
  bool _busy = false;

  static const _types = [
    ('piece', '计件工'),
    ('temp', '临时工'),
    ('fixed', '固定工'),
    ('office', '职能/内勤'),
  ];

  @override
  void initState() {
    super.initState();
    final suffix = DateTime.now().millisecondsSinceEpoch.toString().substring(5);
    _empNo.text = 'E$suffix';
    _login.text = _empNo.text;
  }

  @override
  void dispose() {
    _empNo.dispose();
    _name.dispose();
    _idCard.dispose();
    _mobile.dispose();
    _login.dispose();
    _job.dispose();
    _workshop.dispose();
    _dept.dispose();
    super.dispose();
  }

  IdCardOcr _ocr() {
    final api = context.read<AuthState>().api;
    return HttpIdCardOcr((bytes, filename) async {
      final r = await api.postMultipart('/hr/id-card/ocr', bytes, filename: filename, fieldName: 'file');
      Map<String, dynamic>? data;
      if (r.data is Map) data = Map<String, dynamic>.from(r.data as Map);
      return (ok: r.ok, msg: r.msg, data: data);
    });
  }

  Future<void> _pickAndOcr(ImageSource source) async {
    try {
      final picker = ImagePicker();
      final file = await picker.pickImage(source: source, imageQuality: 85);
      if (file == null) return;
      final bytes = await file.readAsBytes();
      setState(() {
        _busy = true;
        _msg = '识别中…';
      });
      final result = await _ocr().recognizeBytes(Uint8List.fromList(bytes), filename: file.name);
      if (!mounted) return;
      setState(() {
        _busy = false;
        if (result.name.isNotEmpty) _name.text = result.name;
        if (result.idCardNo.isNotEmpty) _idCard.text = result.idCardNo;
        _msg = result.message.isNotEmpty
            ? result.message
            : (result.name.isEmpty && result.idCardNo.isEmpty ? '未识别到信息，请手填' : '已填入识别结果，可手改');
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _busy = false;
        _msg = '选图失败，请手填身份证信息';
      });
    }
  }

  Future<void> _submit() async {
    final empNo = _empNo.text.trim();
    final name = _name.text.trim();
    if (empNo.isEmpty || name.isEmpty) {
      setState(() => _msg = '请填写工号与姓名');
      return;
    }
    setState(() {
      _busy = true;
      _msg = '';
    });
    final api = context.read<AuthState>().api;
    final body = <String, dynamic>{
      'emp_no': empNo,
      'name': name,
      'emp_type': _empType,
      'org_id': 1,
      'dept_id': int.tryParse(_dept.text.trim()) ?? 1,
      'workshop_id': int.tryParse(_workshop.text.trim()) ?? 1,
      'team_id': 0,
      'job_title': _job.text.trim(),
      'mobile': _mobile.text.trim(),
      'id_card_no': _idCard.text.trim(),
      'onboard_date': DateTime.now().toIso8601String().substring(0, 10),
      'need_account': _needAccount,
      'login_name': _login.text.trim().isEmpty ? empNo : _login.text.trim(),
      'remark': 'app_hr_onboard',
    };
    final create = await api.post('/hr/onboards', body);
    if (!mounted) return;
    if (!create.ok || create.data is! Map) {
      setState(() {
        _busy = false;
        _msg = create.msg.isEmpty ? '创建失败' : create.msg;
      });
      return;
    }
    final id = (create.data as Map)['id'];
    if (id == null) {
      setState(() {
        _busy = false;
        _msg = '未返回入职单号';
      });
      return;
    }
    final confirm = await api.post('/hr/onboards/$id/confirm', {
      'need_account': _needAccount,
      'login_name': body['login_name'],
    });
    if (!mounted) return;
    setState(() {
      _busy = false;
      if (confirm.ok) {
        _msg = _needAccount
            ? '开户成功。初始密码 ChangeMe123，请员工在「我的 → 账户」修改密码。'
            : '入职成功（未开登录账号）。';
      } else {
        _msg = '草稿已创建，确认失败：${confirm.msg}';
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('人事开户')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const Text('新建员工并开户', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
          const SizedBox(height: 4),
          const Text('账号仅由人事创建；无 App 自助注册。OCR 未开通时可手填。', style: TextStyle(color: Colors.black54, fontSize: 12)),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: _busy ? null : () => _pickAndOcr(ImageSource.camera),
                  icon: const Icon(Icons.photo_camera),
                  label: const Text('拍照识别'),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: _busy ? null : () => _pickAndOcr(ImageSource.gallery),
                  icon: const Icon(Icons.photo_library),
                  label: const Text('相册识别'),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          TextField(controller: _empNo, decoration: const InputDecoration(labelText: '工号', border: OutlineInputBorder())),
          const SizedBox(height: 8),
          TextField(controller: _name, decoration: const InputDecoration(labelText: '姓名', border: OutlineInputBorder())),
          const SizedBox(height: 8),
          TextField(controller: _idCard, decoration: const InputDecoration(labelText: '身份证号', border: OutlineInputBorder())),
          const SizedBox(height: 8),
          TextField(controller: _mobile, decoration: const InputDecoration(labelText: '手机', border: OutlineInputBorder())),
          const SizedBox(height: 8),
          TextField(controller: _job, decoration: const InputDecoration(labelText: '岗位', border: OutlineInputBorder())),
          const SizedBox(height: 8),
          DropdownButtonFormField<String>(
            initialValue: _empType,
            decoration: const InputDecoration(labelText: '员工类型', border: OutlineInputBorder()),
            items: [for (final t in _types) DropdownMenuItem(value: t.$1, child: Text(t.$2))],
            onChanged: (v) => setState(() => _empType = v ?? 'piece'),
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _workshop,
            decoration: const InputDecoration(labelText: '车间 ID', border: OutlineInputBorder()),
            keyboardType: TextInputType.number,
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _dept,
            decoration: const InputDecoration(labelText: '部门 ID', border: OutlineInputBorder()),
            keyboardType: TextInputType.number,
          ),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('需要登录账号'),
            subtitle: const Text('确认后开户，初始密码 ChangeMe123'),
            value: _needAccount,
            onChanged: (v) => setState(() => _needAccount = v),
          ),
          if (_needAccount)
            TextField(
              controller: _login,
              decoration: const InputDecoration(labelText: '登录名（建议用工号）', border: OutlineInputBorder()),
            ),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: _busy ? null : _submit,
            child: Text(_busy ? '提交中…' : '提交并确认入职'),
          ),
          if (_msg.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: Text(_msg, style: const TextStyle(color: Colors.teal)),
            ),
        ],
      ),
    );
  }
}
