# 员工 App 交付清单（现场可交付）

## 登录与账号策略

- **无自助注册**：登录页仅「账号密码 / 第三方登录 stub / Debug 选人」；账号由**人事在 App「人事开户」或管理端入职登记**创建
- 支持**账号密码**登录；**第三方登录**入口已预留（默认未开通，提示「暂未开通」）
- 登录成功后请求 `GET /auth/me` 刷新 **roles / permissions**
- 按主角色展示**直线步骤工作台**（过磅/仓管/工人/车间/销售等），多角色可切换；步骤仅导航，**不强制工作流完成态**（后续优化）
- 管理员账号仍显示模块网格兜底；资料中心 / 我的在顶栏次要入口
- **Debug 快捷账号**（密码均为 `admin123`）：`admin`、`u_purchase`、`u_qc`、`u_warehouse`、`u_foreman`、`u_piece`、`u_fixed`、`u_sales`、`u_finance`、`u_boss`（demo 启动时 `EnsureDemoRoleUsers` 写入）

## 个人中心（账户）

- 入口：我的页顶栏「账户」或资料卡片 → `AccountCenterPage`
- **修改密码**：`POST /auth/password/change`（校验旧密 + 登录策略强度）；成功后可用新密登录
- **第三方绑定**：`GET/POST/DELETE /auth/oauth/bindings|bind`；表 `iam_user_oauth`；未配置 `oauth.enabled` 时返回 `OAUTH_NOT_CONFIGURED`，界面提示不崩溃

## 人事开户（App）

- 可见角色：`hr` / `sys_admin` / 权限含「人事管理」
- 流程：工号姓名类型部门车间 → 身份证 OCR（拍照/相册，可手改）→ 可选开登录账号 → `POST /hr/onboards` + `confirm`
- 初始密码与后台一致：`ChangeMe123`，提示员工在个人中心改密
- OCR：`POST /hr/id-card/ocr`（multipart）；`ocr.enabled=false` 时 `OCR_NOT_CONFIGURED`，允许手填 `id_card_no` 完成开户
- 管理端入职登记已支持身份证号字段

## 工单与物料工具

- 后台 **系统管理 / 工单中心**：分类、处理人池（用户/角色）、创建与跟踪、指定下一手
- App **物料工具** `/tools`：申请领取/归还时从处理人池选择下一手；**工单** `/tickets`：待我处理 / 我发起
- 工具单据字段：日期、序号、姓名、工具（7 类）、领/还/合计数量；状态 pending→open→pending_return→returned
- 通知：指派写 inbox；深链 Admin `/workflow/tickets`、App `/tickets`

## 模块验收（admin 可见全部）

| 模块 | 验收要点 |
|------|----------|
| 过磅收货 | 建单→质检→出码推仓；采购任务认领/完成；**品种下拉（后台「过磅品种」配置）** |
| 仓管作业 | 待入库核对；库存+**亏料/过量预警**；箱码；盘点；**扫码出入库过账** |
| 车间工作台 | 报工、派工接收、灵活派发、质检/废料/返修、**图纸分发** |
| 工人报工 | 双扫、今日核对、联动领料过账、**工具申请入口** |
| 销售外勤 | 下单/询价/出厂；**发货进度**；**报价试算保存**；跟进 |
| **固定资产** | 查询资产；提交转移申请；确认转移 |
| **收款协同** | 收款预警处理；认款新建/确认 |
| **资料中心** | 知识库/图纸/公告/学堂只读详情 |
| 我的 | 打卡、假勤、审批、工资提成、消息、**日志/备忘录**、**账户**、**工具/工单**、人事**开户** |

## 自动化冒烟

```bash
go run ./cmd/erp-api          # :18080
go run ./cmd/mobile_delivery_smoke
# 期望：DELIVERY_SMOKE_OK
```

## 已知边界

- 扫码/拍照可用手输与占位证据 URL；正式环境接相机与上传
- 过磅确认需采购/管理员；仓管入库需仓管/管理员
- 客户自助无前端；老板看板走 boss Web
- 第三方 OAuth（微信等）需配置 `oauth.enabled` 并接入真实 SDK 后方可开通
- 身份证 OCR 需配置 `ocr.enabled` 并接入真实 Provider；本轮仅骨架，手填不阻塞开户
