# 加工厂 ERP · 员工 App（Flutter）

Android / iOS 统一员工端。与 Web `apps/employee` 共用同一套 `/api/v1` 与 IAM；登录 `client_type=mobile`，按权限显隐车间 / 工人 / 销售模块。

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

## 不包含

- Flutter Web 目标平台
- 客户自助下单前端（相关接口仍在 OpenAPI / 后端）
