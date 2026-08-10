# Web 单端口与内嵌/外置

## 行为

`erp-api` 在同一端口（默认 `:18080`）提供：

- API：`/api/v1/*`
- 上传：`/files/*`
- 门户：`/`
- 管理端：`/admin/`
- 老板舱：`/front/boss/`

前端源码仍为 `web/apps/{portal,admin,boss}`，不合并成一个 SPA。

## 内嵌

发布脚本先 `npm run build:dist`，再同步到 `internal/webui/dist`，最后 `go build`：

- Windows：`powershell -File scripts/build_release.ps1`
- Unix：`./scripts/build_release.sh`

生产配置 `server.web_root` **留空**，使用二进制内资源。

## 外置

```yaml
server:
  web_root: web/dist   # 或绝对路径
```

或环境变量 `ERP_WEB_ROOT`。目录需含 `index.html`；否则回退内嵌。

优先级：**有效外置目录 > 内嵌**。

## 开发

- Vite 多端口：`npm run dev:portal|admin|boss`
- 或 `configs/erp.dev.yaml` 默认 `web_root: web/dist`，先 `cd web && npm run build:dist`，再只开 `go run ./cmd/erp-api`
