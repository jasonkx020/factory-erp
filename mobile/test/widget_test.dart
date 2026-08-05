import 'package:flutter_test/flutter_test.dart';

import 'package:erp_employee/core/employee_modules.dart';

void main() {
  test('admin sees all employee modules', () {
    final mods = visibleEmployeeModules(['*:*:*'], ['sys_admin']);
    expect(mods.length, employeeModules.length);
    expect(mods.map((m) => m.key), containsAll([
      EmployeeModule.workshop,
      EmployeeModule.worker,
      EmployeeModule.receiving,
      EmployeeModule.warehouse,
      EmployeeModule.sales,
      EmployeeModule.assets,
      EmployeeModule.collab,
      EmployeeModule.knowledge,
      EmployeeModule.mine,
    ]));
  });

  test('knowledge and mine always visible', () {
    final mods = visibleEmployeeModules([], []);
    expect(mods.map((m) => m.key), containsAll([EmployeeModule.knowledge, EmployeeModule.mine]));
  });
}
