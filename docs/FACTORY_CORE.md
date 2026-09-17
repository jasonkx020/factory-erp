# 工厂加工核心（FACTORY_CORE）

本仓库产品形态：**通用工厂加工核心 + 可选行业种子包**。木薯为行业包之一，不进入过站/库存引擎硬编码。

## 部署配置

```yaml
product:
  profile: core        # core | extended
  industry_pack: none  # none | cassava（现网未写时默认 cassava）
```

环境变量：`ERP_PRODUCT_PROFILE`、`ERP_INDUSTRY_PACK`。

| profile | 菜单范围 |
|---------|----------|
| `core` | 采购过磅闭环、生产过站、库存、产品、人事工资、基础财务 |
| `extended` | 另含销售出库/客户、完整总账（凭证/科目/月结/三表等） |

| industry_pack | 行为 |
|---------------|------|
| `none` | 不自动种木薯工艺/品种；厂名默认「加工厂 ERP」；走开厂配置 |
| `cassava` | 幂等补齐鲜木薯工艺、过磅品种、相关报表别名 |

**不做**：多租户 SaaS（库级隔离）。多厂区 = 同一组织下 `sys_plant` + 仓/工序/用户 scope。

## 后台配置后方可使用

现场事务（建过磅单、溯源开工、过站、销售出库）在 **开厂就绪检查** 通过前拒绝，返回 `SETUP_INCOMPLETE:<item>`。

检查入口：`GET /api/v1/system/setup-readiness`；Admin「系统管理 → 开厂配置」。

核心检查项：

1. 组织 / 至少一个厂区  
2. 仓库角色：至少原料仓（`warehouse_type/role=raw`）、半成品仓（`semi`）、成品仓（`fg`）——或工艺步骤已引用具体仓  
3. ≥1 产品与单位  
4. ≥1 已发布生产工艺（步骤含合法 `output_product_id`）  
5. 启用过磅时：≥1 过磅品种（建议绑定仓库）+ 已发布 `purchase_gate` 流程  
6. 班次、现场用户、工序/仓 scope、计件工序工价  

Admin 主数据配置始终可写；仅挡事务写入。

## 配置清单（开厂向导）

1. 公司架构 / 厂区  
2. 仓库（角色 raw/semi/fg）与库位  
3. 产品、单位、规格↔工艺  
4. 工序定义 → 工艺流程画布发布  
5. 过磅品种、过磅/入库流程编排  
6. 产线班次、工序工价、角色权限与 scope  
7. （extended）客户、销售出库、会计科目与期间  

## 行业包约定

- 引擎与表结构不出现行业字面依赖（如默认「鲜木薯」、固定仓 id=1/2/3）。  
- 行业差异仅：`seed` 主数据、品牌文案（`sys_org_setting`）、报表显示别名。  
- 木薯包入口：`biz.EnsureCassavaIndustryPack`（原 `EnsureFreshCassavaRouting` 等）。

## 相关文档

- [ADMIN_DELIVERY.md](./ADMIN_DELIVERY.md) — 交付边界与运维  
- [ERP-持续开发约束.md](./ERP-持续开发约束.md) — 不做多租户等约束  
