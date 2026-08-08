import 'package:flutter_test/flutter_test.dart';

import 'package:erp_employee/core/employee_modules.dart';

void main() {
  test('admin sees all factory pilot modules', () {
    final mods = visibleEmployeeModules(['*:*:*'], ['sys_admin']);
    expect(mods.length, employeeModules.length);
    expect(mods.map((m) => m.key), containsAll([
      EmployeeModule.station,
      EmployeeModule.receiving,
      EmployeeModule.warehouse,
      EmployeeModule.workshop,
      EmployeeModule.mine,
    ]));
  });

  test('mine always visible', () {
    final mods = visibleEmployeeModules([], []);
    expect(mods.map((m) => m.key), contains(EmployeeModule.mine));
  });

  test('piece worker sees station and mine', () {
    final mods = visibleEmployeeModules([], ['piece']);
    expect(mods.map((m) => m.key), containsAll([EmployeeModule.station, EmployeeModule.mine]));
    expect(mods.length, lessThanOrEqualTo(4));
  });
}
