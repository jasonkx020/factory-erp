import 'role_codes.dart';

class RoleStep {
  const RoleStep({
    required this.title,
    required this.subtitle,
    required this.route,
  });

  final String title;
  final String subtitle;
  final String route;
}

/// Static step lists per role — navigation only; no workflow enforcement.
/// [codeLabel] defaults to 板码 (sys_setting.carrier_code_unit).
List<RoleStep> stepsForWorkbenchRole(WorkbenchRole role, {String codeLabel = '板码'}) {
  switch (role) {
    case WorkbenchRole.receiving:
      return [
        const RoleStep(title: '建过磅单', subtitle: '农户过磅收货建单', route: '/receiving'),
        const RoleStep(title: '质检确认', subtitle: '过磅单质检与确认', route: '/receiving'),
        RoleStep(title: '出码推仓', subtitle: '生成$codeLabel并推送待入库', route: '/receiving'),
      ];
    case WorkbenchRole.warehouse:
      return [
        const RoleStep(title: '待入库', subtitle: '核对过磅推仓单据', route: '/warehouse'),
        const RoleStep(title: '核对入库', subtitle: '扫码出入库过账', route: '/warehouse'),
        RoleStep(title: '库存与$codeLabel', subtitle: '查库存、$codeLabel管理', route: '/warehouse'),
        const RoleStep(title: '盘点', subtitle: '仓库/车间盘点', route: '/warehouse'),
      ];
    case WorkbenchRole.workshop:
      return [
        RoleStep(title: '工序过站', subtitle: '指定工序后扫工牌+$codeLabel按 kg 领料', route: '/station'),
        const RoleStep(title: '班组异常', subtitle: '返工派岗、废料处理', route: '/workshop'),
      ];
    case WorkbenchRole.worker:
      return [
        RoleStep(title: '工序过站', subtitle: '指定工序后扫工牌+$codeLabel，领取/退库/入库', route: '/station'),
        const RoleStep(title: '今日核对', subtitle: '核对当日报工与计件', route: '/mine'),
      ];
    case WorkbenchRole.sales:
    case WorkbenchRole.collab:
      return const [];
    case WorkbenchRole.admin:
    case WorkbenchRole.none:
      return const [];
  }
}
