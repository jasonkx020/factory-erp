# -*- coding: utf-8 -*-
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
p = ROOT / "docs" / "openapi3.0-加工厂ERP.yaml"
text = p.read_text(encoding="utf-8")
marker = "  /api/v1/report/accounts:"
block = r'''
  /api/v1/biz/evidences:
    get:
      tags: [系统管理]
      summary: 业务证据-列表
      operationId: get_api_v1_biz_evidences
      description: |
        功能模块：证据倒查；分期：2
      x-erp-phase: 2
      x-erp-module: 证据倒查
      parameters:
        - name: biz_type
          in: query
          schema: { type: string }
        - name: biz_id
          in: query
          schema: { type: integer, format: int64 }
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"
    post:
      tags: [系统管理]
      summary: 业务证据-上传登记
      operationId: post_api_v1_biz_evidences
      description: |
        功能模块：证据倒查；分期：2
      x-erp-phase: 2
      x-erp-module: 证据倒查
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              additionalProperties: true
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/biz/corrections:
    post:
      tags: [系统管理]
      summary: 业务纠错/作废（禁止物理删除）
      operationId: post_api_v1_biz_corrections
      description: |
        功能模块：证据倒查；分期：2
      x-erp-phase: 2
      x-erp-module: 证据倒查
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: true
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/purchase/inbound-arrivals:
    get:
      tags: [采购管理]
      summary: 到货单-列表
      operationId: get_api_v1_purchase_inbound_arrivals
      description: |
        功能模块：过磅收货；分期：2
      x-erp-phase: 2
      x-erp-module: 过磅收货
      parameters:
        - $ref: "#/components/parameters/PageNum"
        - $ref: "#/components/parameters/PageSize"
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"
    post:
      tags: [采购管理]
      summary: 到货单-新建
      operationId: post_api_v1_purchase_inbound_arrivals
      description: |
        功能模块：过磅收货；分期：2
      x-erp-phase: 2
      x-erp-module: 过磅收货
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              additionalProperties: true
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/purchase/inbound-arrivals/{id}:
    get:
      tags: [采购管理]
      summary: 到货单-详情
      operationId: get_api_v1_purchase_inbound_arrivals_id
      description: |
        功能模块：过磅收货；分期：2
      x-erp-phase: 2
      x-erp-module: 过磅收货
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: integer, format: int64 }
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/purchase/inbound-arrivals/{id}/qc:
    post:
      tags: [采购管理]
      summary: 到货单-质检定级确认
      operationId: post_api_v1_purchase_inbound_arrivals_id_qc
      description: |
        功能模块：过磅收货；分期：2
      x-erp-phase: 2
      x-erp-module: 过磅收货
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: integer, format: int64 }
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              additionalProperties: true
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/purchase/weigh-tickets/{id}/confirm:
    post:
      tags: [采购管理]
      summary: 过磅单-确认出码
      operationId: post_api_v1_purchase_weigh_tickets_id_confirm
      description: |
        功能模块：过磅收货；分期：2
      x-erp-phase: 2
      x-erp-module: 过磅收货
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: integer, format: int64 }
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              additionalProperties: true
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/purchase/weigh-tickets/{id}/label:
    get:
      tags: [采购管理]
      summary: 过磅单-标签打印数据
      operationId: get_api_v1_purchase_weigh_tickets_id_label
      description: |
        功能模块：原料溯源；分期：2
      x-erp-phase: 2
      x-erp-module: 原料溯源
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: integer, format: int64 }
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/purchase/weigh-tickets/{id}/warehouse-confirm:
    post:
      tags: [采购管理]
      summary: 过磅单-仓管确认入库
      operationId: post_api_v1_purchase_weigh_tickets_id_warehouse_confirm
      description: |
        功能模块：过磅收货；分期：2
      x-erp-phase: 2
      x-erp-module: 过磅收货
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: integer, format: int64 }
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              additionalProperties: true
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/purchase/farmer-settlements/summary:
    get:
      tags: [采购管理]
      summary: 农户结算-汇总
      operationId: get_api_v1_purchase_farmer_settlements_summary
      description: |
        功能模块：农户结算；分期：2
      x-erp-phase: 2
      x-erp-module: 农户结算
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/purchase/farmer-settlements/{id}/pay:
    post:
      tags: [采购管理]
      summary: 农户结算-支付关单
      operationId: post_api_v1_purchase_farmer_settlements_id_pay
      description: |
        功能模块：农户结算；分期：2
      x-erp-phase: 2
      x-erp-module: 农户结算
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: integer, format: int64 }
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              additionalProperties: true
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/purchase/trace-lots/verify:
    post:
      tags: [采购管理]
      summary: 追溯码-验签
      operationId: post_api_v1_purchase_trace_lots_verify
      description: |
        功能模块：原料溯源；分期：2
      x-erp-phase: 2
      x-erp-module: 原料溯源
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              additionalProperties: true
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/production/report-works/{id}/confirm:
    post:
      tags: [生产管理]
      summary: 报工-确认过账
      operationId: post_api_v1_production_report_works_id_confirm
      description: |
        功能模块：扫码报工；分期：1
      x-erp-phase: 1
      x-erp-module: 扫码报工
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: integer, format: int64 }
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              additionalProperties: true
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/production/piecework-summaries/{id}/pay:
    post:
      tags: [生产管理]
      summary: 计件汇总-劳动支付
      operationId: post_api_v1_production_piecework_summaries_id_pay
      description: |
        功能模块：扫码报工；分期：2
      x-erp-phase: 2
      x-erp-module: 扫码报工
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: integer, format: int64 }
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              additionalProperties: true
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

'''

if "/api/v1/biz/evidences:" in text:
    print("already inserted")
else:
    if marker not in text:
        raise SystemExit("marker missing")
    p.write_text(text.replace(marker, block + marker, 1), encoding="utf-8")
    print("inserted ok")
