//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"erp/internal/persistence"
)

func main() {
	dsn := os.Getenv("ERP_DATABASE_DSN")
	if len(os.Args) > 1 {
		dsn = os.Args[1]
	}
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "usage: go run scripts/repair_flow_encoding.go <postgres-dsn>")
		os.Exit(2)
	}
	persistence.RegisterRebindDriverForTest()
	db, err := sql.Open("pgx_rebind", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	var name, step string
	_ = db.QueryRow(`SELECT name FROM pd_routing WHERE code='RT-CASSAVA'`).Scan(&name)
	_ = db.QueryRow(`SELECT step_name FROM pd_routing_step WHERE routing_id=1 AND step_code='S3'`).Scan(&step)
	fmt.Printf("routing=%q step_S3=%q\n", name, step)
}
