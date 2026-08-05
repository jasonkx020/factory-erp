import 'package:flutter_test/flutter_test.dart';

import 'package:erp_employee/core/employee_modules.dart';

void main() {
  test('admin sees all employee modules', () {
    final mods = visibleEmployeeModules(['*:*:*'], ['sys_admin']);
    expect(mods.length, 3);
  });
}
