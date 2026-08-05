# 加工厂 ERP — 常用开发命令
# 需要：GNU Make + Go（见 go.mod）+ Node/npm；可选 Flutter
# Windows 建议在 Git Bash / WSL 下执行，或安装 make（scoop/choco）
#
#   make help

.DEFAULT_GOAL := help
.PHONY: help build build-api build-tools run run-api test test-go tidy \
	gen-routes openapi-coverage gen-web-meta gen-all gate \
	web-install web-dev web-dev-admin web-dev-portal web-dev-boss web-build web-dist \
	e2e e2e-flow smoke mobile-get mobile-run clean reseed-demo

CONFIG  ?= configs/erp.dev.yaml
BIN_DIR ?= bin
API_URL ?= http://127.0.0.1:18080/api/v1
WEB_DIR := web

ifeq ($(OS),Windows_NT)
  EXE := .exe
else
  EXE :=
endif

API_OUT   := $(BIN_DIR)/erp-api$(EXE)
TOOLS_OUT := $(BIN_DIR)/erp-tools$(EXE)

help: ## 显示可用目标
	@printf '%s\n' \
	  '加工厂 ERP Makefile' \
	  '' \
	  '  make build            编译 erp-api → $(API_OUT)' \
	  '  make run              启动 API（CONFIG=$(CONFIG)）' \
	  '  make reseed-demo      清除演示数据版本标记（重启 API 会重灌）' \
	  '  make gate             契约门禁：gen-routes + openapi-coverage' \
	  '  make web-install      安装前端依赖' \
	  '  make web-dev-admin    启动管理后台' \
	  '  make web-dev-portal   启动入口' \
	  '  make web-dev-boss     启动老板驾驶舱' \
	  '  make e2e              Web 冒烟（需 API 已启动）' \
	  '  make smoke            GET /health' \
	  '  make clean            清理构建产物' \
	  '' \
	  '变量示例: make run CONFIG=configs/erp.dev.yaml'

# ---------- Go API ----------

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

build: build-api ## 编译 API 二进制

build-api: | $(BIN_DIR) ## 编译 cmd/erp-api
	go build -o $(API_OUT) ./cmd/erp-api
	@echo built $(API_OUT)

build-tools: | $(BIN_DIR) ## 编译 cmd/erp-tools
	go build -o $(TOOLS_OUT) ./cmd/erp-tools
	@echo built $(TOOLS_OUT)

run: run-api ## 启动开发 API

run-api: ## go run API
	go run ./cmd/erp-api -config $(CONFIG)

test: test-go ## 运行 Go 测试

test-go: ## go test ./...
	go test ./...

tidy: ## go mod tidy
	go mod tidy

# ---------- 契约 / 代码生成 ----------

gen-routes: ## OpenAPI → Gin 路由
	go run ./cmd/erp-tools gen-routes

openapi-coverage: ## OpenAPI 路径覆盖率（须 100%）
	go run ./cmd/erp-tools openapi-coverage

gen-web-meta: ## 生成管理端 modules.ts
	go run ./cmd/erp-tools gen-web-meta

gen-all: gen-routes gen-web-meta ## 生成路由 + Web meta

gate: gen-routes openapi-coverage ## 改 OpenAPI 后必跑

# ---------- Web ----------

web-install: ## npm install
	cd $(WEB_DIR) && npm install

web-dev: web-dev-portal ## 默认 portal

web-dev-portal: ## 入口 portal
	cd $(WEB_DIR) && npm run dev:portal

web-dev-admin: ## 管理后台
	cd $(WEB_DIR) && npm run dev:admin

web-dev-boss: ## 老板驾驶舱
	cd $(WEB_DIR) && npm run dev:boss

web-build: ## 构建三端
	cd $(WEB_DIR) && npm run build

web-dist: ## 打包 dist
	cd $(WEB_DIR) && npm run build:dist

e2e: ## Web e2e（API :18080）
	cd $(WEB_DIR) && npm run e2e

e2e-flow: ## 业务闭环 e2e
	cd $(WEB_DIR) && npm run e2e:flow

smoke: ## 健康检查
	@curl -sf "$(API_URL)/health" >/dev/null && echo "health OK ($(API_URL)/health)" || (echo "health FAILED — 先 make run"; exit 1)

# ---------- Flutter ----------

mobile-get: ## flutter pub get
	cd mobile && flutter pub get

mobile-run: ## flutter run
	cd mobile && flutter run --dart-define=API_BASE=$(API_URL)

# ---------- 清理 ----------

clean: ## 删除 bin 与前端 dist
	rm -rf $(BIN_DIR) \
		$(WEB_DIR)/dist \
		$(WEB_DIR)/apps/admin/dist \
		$(WEB_DIR)/apps/portal/dist \
		$(WEB_DIR)/apps/boss/dist
	@echo cleaned

reseed-demo: ## 清除 demo_showcase_version，下次启动 API 会重灌展示假数据
	@echo "Deleting demo_showcase_version from SQLite (if present)..."
	@sqlite3 data/erp_dev.db "DELETE FROM schema_meta WHERE key='demo_showcase_version';" 2>/dev/null || true
	@echo "Done. Restart API (make run) to re-apply EnsureDemoData."
