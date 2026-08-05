package main

import (
	"fmt"
	"os"

	"erp/internal/devtools/openapi"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	root, err := openapi.FindRepoRoot()
	if err != nil {
		fatal(err)
	}
	switch os.Args[1] {
	case "gen-routes":
		ops, emitted, err := openapi.GenRoutes(root)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("ops=%d routes_emitted=%d -> internal/apigen/routes_gen.go\n", ops, emitted)
	case "openapi-coverage":
		if err := openapi.CheckCoverage(root); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "gen-web-meta":
		n, err := openapi.GenWebMeta(root)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("modules=%d -> web/packages/shared/src/generated/modules.ts\n", n)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `erp-tools — OpenAPI 开发期代码生成与门禁（替代原 Python 脚本）

Usage:
  go run ./cmd/erp-tools gen-routes          # OpenAPI → routes_gen.go + openapi_ops.json
  go run ./cmd/erp-tools openapi-coverage    # 契约路径须 100%% 注册
  go run ./cmd/erp-tools gen-web-meta        # 路径全表 + menus → modules.ts

`)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
