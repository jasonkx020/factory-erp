# -*- coding: utf-8 -*-
"""Inject field-ledger OpenAPI paths and regenerate routes."""
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OPENAPI = ROOT / "docs" / "openapi3.0-加工厂ERP.yaml"

SNIPPET = """
  /api/v1/sales/outbound-settles:
    get:
      operationId: get_api_v1_sales_outbound_settles
      summary: 销售出厂结算列表
      responses: { '200': { description: ok } }
    post:
      operationId: post_api_v1_sales_outbound_settles
      summary: 新建销售出厂结算
      responses: { '200': { description: ok } }
  /api/v1/sales/outbound-settles/{id}:
    get:
      operationId: get_api_v1_sales_outbound_settles_id
      summary: 出厂结算详情
      parameters: [{ name: id, in: path, required: true, schema: { type: integer } }]
      responses: { '200': { description: ok } }
    put:
      operationId: put_api_v1_sales_outbound_settles_id
      summary: 更新出厂结算
      parameters: [{ name: id, in: path, required: true, schema: { type: integer } }]
      responses: { '200': { description: ok } }
  /api/v1/sales/outbound-settles/{id}/close:
    post:
      operationId: post_api_v1_sales_outbound_settles_id_close
      summary: 关单出厂结算
      parameters: [{ name: id, in: path, required: true, schema: { type: integer } }]
      responses: { '200': { description: ok } }
  /api/v1/production/piece-issue-sheets:
    get:
      operationId: get_api_v1_production_piece_issue_sheets
      summary: 计件领料表列表
      responses: { '200': { description: ok } }
    post:
      operationId: post_api_v1_production_piece_issue_sheets
      summary: 新建计件领料表
      responses: { '200': { description: ok } }
  /api/v1/production/piece-issue-sheets/generate:
    post:
      operationId: post_api_v1_production_piece_issue_sheets_generate
      summary: 从报工生成领料表草稿
      responses: { '200': { description: ok } }
  /api/v1/production/piece-issue-sheets/{id}:
    get:
      operationId: get_api_v1_production_piece_issue_sheets_id
      summary: 计件领料表详情
      parameters: [{ name: id, in: path, required: true, schema: { type: integer } }]
      responses: { '200': { description: ok } }
  /api/v1/production/process-reports:
    get:
      operationId: get_api_v1_production_process_reports
      summary: 加工记录列表
      responses: { '200': { description: ok } }
  /api/v1/hr/tool-items:
    get:
      operationId: get_api_v1_hr_tool_items
      summary: 工具品类
      responses: { '200': { description: ok } }
  /api/v1/hr/tool-issues:
    get:
      operationId: get_api_v1_hr_tool_issues
      summary: 工具领还列表
      responses: { '200': { description: ok } }
    post:
      operationId: post_api_v1_hr_tool_issues
      summary: 工具领取
      responses: { '200': { description: ok } }
  /api/v1/hr/tool-issues/{id}/return:
    post:
      operationId: post_api_v1_hr_tool_issues_id_return
      summary: 工具交还
      parameters: [{ name: id, in: path, required: true, schema: { type: integer } }]
      responses: { '200': { description: ok } }
  /api/v1/inventory/weighbridges:
    get:
      operationId: get_api_v1_inventory_weighbridges
      summary: 地磅列表
      responses: { '200': { description: ok } }
    post:
      operationId: post_api_v1_inventory_weighbridges
      summary: 新建地磅
      responses: { '200': { description: ok } }
  /api/v1/inventory/weighbridges/{id}:
    put:
      operationId: put_api_v1_inventory_weighbridges_id
      summary: 更新地磅
      parameters: [{ name: id, in: path, required: true, schema: { type: integer } }]
      responses: { '200': { description: ok } }
"""

def main():
    text = OPENAPI.read_text(encoding="utf-8")
    if "/api/v1/sales/outbound-settles:" in text:
        print("already injected")
    else:
        marker = "\ncomponents:"
        if marker not in text:
            raise SystemExit("components: not found")
        text = text.replace(marker, SNIPPET + marker, 1)
        OPENAPI.write_text(text, encoding="utf-8")
        print("injected")
    import subprocess
    subprocess.check_call(["python", str(ROOT / "scripts" / "gen_routes.py")])

if __name__ == "__main__":
    main()
