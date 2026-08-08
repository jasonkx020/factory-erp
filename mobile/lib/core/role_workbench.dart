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
List<RoleStep> stepsForWorkbenchRole(WorkbenchRole role) {
  switch (role) {
    case WorkbenchRole.receiving:
      return const [
        RoleStep(title: '建过磅单', subtitle: '农户过磅收货建单', route: '/receiving'),
        RoleStep(title: '质检确认', subtitle: '过磅单质检与确认', route: '/receiving'),
        RoleStep(title: '出码推仓', subtitle: '生成箱码并推送待入库', route: '/receiving'),
      ];
    case WorkbenchRole.warehouse:
      return const [
        RoleStep(title: '待入库', subtitle: '核对过磅推仓单据', route: '/warehouse'),
        RoleStep(title: '核对入库', subtitle: '扫码出入库过账', route: '/warehouse'),
        RoleStep(title: '库存与箱码', subtitle: '查库存、箱码管理', route: '/warehouse'),
        RoleStep(title: '盘点', subtitle: '仓库/车间盘点', route: '/warehouse'),
      ];
    case WorkbenchRole.workshop:
      return const [
        RoleStep(title: '工序过站', subtitle: '扫工牌+箱码确认过站', route: '/station'),
        RoleStep(title: '班组异常', subtitle: '返工派岗、废料处理', route: '/workshop'),
      ];
    case WorkbenchRole.worker:
      return const [
        RoleStep(title: '工序过站', subtitle: '扫工牌+箱码，确认投料/完工', route: '/station'),
        RoleStep(title: '今日核对', subtitle: '核对当日报工与计件', route: '/mine'),
      ];
    case WorkbenchRole.sales:
    case WorkbenchRole.collab:
      return const [];
    case WorkbenchRole.admin:
    case WorkbenchRole.none:
      return const [];
  }
}
