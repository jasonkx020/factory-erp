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
        RoleStep(title: '采购任务', subtitle: '认领或完成采购任务', route: '/receiving'),
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
        RoleStep(title: '接收派工', subtitle: '查看并接收派工单', route: '/workshop'),
        RoleStep(title: '报工与质检', subtitle: '扫码报工、质检', route: '/workshop'),
        RoleStep(title: '废料返修', subtitle: '废料登记与返修单', route: '/workshop'),
        RoleStep(title: '图纸分发', subtitle: '查看工艺图纸', route: '/workshop'),
      ];
    case WorkbenchRole.worker:
      return const [
        RoleStep(title: '扫码报工', subtitle: '工牌+箱码双扫报工', route: '/worker'),
        RoleStep(title: '今日核对', subtitle: '核对当日报工与计件', route: '/worker'),
        RoleStep(title: '联动领料', subtitle: '报工后联动领料过账', route: '/worker'),
      ];
    case WorkbenchRole.sales:
      return const [
        RoleStep(title: '下单与询价', subtitle: '新建订单或询价', route: '/sales'),
        RoleStep(title: '发货进度', subtitle: '跟踪发货与出厂', route: '/sales'),
        RoleStep(title: '收款协同', subtitle: '预警处理与认款', route: '/collab'),
      ];
    case WorkbenchRole.collab:
      return const [
        RoleStep(title: '收款预警', subtitle: '处理逾期收款预警', route: '/collab'),
        RoleStep(title: '销售认款', subtitle: '新建并确认认款', route: '/collab'),
      ];
    case WorkbenchRole.admin:
    case WorkbenchRole.none:
      return const [];
  }
}
