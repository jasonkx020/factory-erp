package main

import (
	"flag"
	"log"
	"os"

	"erp/internal/app"
)

func main() {
	cfgPath := flag.String("config", "configs/erp.dev.yaml", "config file path")
	flag.Parse()
	if v := os.Getenv("ERP_CONFIG"); v != "" {
		*cfgPath = v
	}
	application, err := app.New(*cfgPath)
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}
	defer application.Close()
	if err := application.Run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
