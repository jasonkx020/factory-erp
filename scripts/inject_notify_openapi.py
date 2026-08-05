# -*- coding: utf-8 -*-
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
p = ROOT / "docs" / "openapi3.0-加工厂ERP.yaml"
text = p.read_text(encoding="utf-8")
marker = "  /api/v1/biz/evidences:"
if marker not in text:
    marker = "  /api/v1/report/accounts:"
block = r'''
  /api/v1/mqtt/auth:
    post:
      tags: [系统管理]
      summary: NanoMQ-HTTP认证
      operationId: post_api_v1_mqtt_auth
      description: |
        功能模块：工作流推送；分期：2
      x-erp-phase: 2
      x-erp-module: 工作流推送
      security: []
      responses:
        "200":
          description: OK
        "403":
          description: Forbidden

  /api/v1/mqtt/superuser:
    post:
      tags: [系统管理]
      summary: NanoMQ-超管校验
      operationId: post_api_v1_mqtt_superuser
      description: |
        功能模块：工作流推送；分期：2
      x-erp-phase: 2
      x-erp-module: 工作流推送
      security: []
      responses:
        "200":
          description: OK
        "403":
          description: Forbidden

  /api/v1/mqtt/acl:
    post:
      tags: [系统管理]
      summary: NanoMQ-ACL
      operationId: post_api_v1_mqtt_acl
      description: |
        功能模块：工作流推送；分期：2
      x-erp-phase: 2
      x-erp-module: 工作流推送
      security: []
      responses:
        "200":
          description: OK
        "403":
          description: Forbidden

  /api/v1/notify/mqtt-connect:
    get:
      tags: [系统管理]
      summary: MQTT连接凭证下发
      operationId: get_api_v1_notify_mqtt_connect
      description: |
        功能模块：工作流推送；分期：2
      x-erp-phase: 2
      x-erp-module: 工作流推送
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OkAnyResponse"

  /api/v1/notify/inbox:
    get:
      tags: [系统管理]
      summary: 通知收件箱
      operationId: get_api_v1_notify_inbox
      description: |
        功能模块：工作流推送；分期：2
      x-erp-phase: 2
      x-erp-module: 工作流推送
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

  /api/v1/notify/inbox/{id}/read:
    post:
      tags: [系统管理]
      summary: 通知已读
      operationId: post_api_v1_notify_inbox_id_read
      description: |
        功能模块：工作流推送；分期：2
      x-erp-phase: 2
      x-erp-module: 工作流推送
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

  /api/v1/workflow/tasks:
    get:
      tags: [系统管理]
      summary: 工作流待办
      operationId: get_api_v1_workflow_tasks
      description: |
        功能模块：工作流推送；分期：2
      x-erp-phase: 2
      x-erp-module: 工作流推送
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

  /api/v1/workflow/tasks/{id}/claim:
    post:
      tags: [系统管理]
      summary: 认领待办
      operationId: post_api_v1_workflow_tasks_id_claim
      description: |
        功能模块：工作流推送；分期：2
      x-erp-phase: 2
      x-erp-module: 工作流推送
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

'''
if "/api/v1/notify/mqtt-connect:" in text:
    print("already inserted")
else:
    if marker not in text:
        raise SystemExit("marker missing")
    p.write_text(text.replace(marker, block + marker, 1), encoding="utf-8")
    print("inserted ok")
