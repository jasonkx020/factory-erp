# 加工厂 ERP · 员工 App（Flutter）

Android / iOS **唯一**员工现场端。与管理端共用 `/api/v1` 与 IAM；登录 `client_type=mobile`，按权限显隐车间 / 工人 / 仓管 / 销售模块。

原 Web 统一员工端（`web/apps/employee`）已下线。

## 开发

```bash
# API 默认 http://10.0.2.2:18080/api/v1 （Android 模拟器访问本机）
cd mobile
flutter pub get
flutter run
```

自定义 API：

```bash
flutter run --dart-define=API_BASE=http://192.168.1.10:18080/api/v1
```

演示账号：`admin` / `admin123`

## 模块能力摘要

| 模块 | 能力 |
|------|------|
| 车间 | 双扫报工、流转/任务/派工、工序、库存；确认过账含次品类型与袋数 |
| 工人 | 双扫报工、今日计件核对、领料/工具只读、提醒 |
| 仓管 | 过磅待办认领、溯源核对入库 |
| 销售 | 订单/复购、询价、出厂结算补录、客户跟进、提醒 |

## 不包含

- Flutter Web 目标平台
- 客户自助下单前端（相关接口仍在 OpenAPI / 后端）
