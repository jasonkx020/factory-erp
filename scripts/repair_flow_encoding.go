//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"

	"erp/internal/biz"
)

func main() {
	path := "data/erp_dev.db"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	biz.EnsureAutomationSchema(db)

	var name, step string
	_ = db.QueryRow(`SELECT name FROM pd_routing WHERE code='RT-CASSAVA'`).Scan(&name)
	_ = db.QueryRow(`SELECT step_name FROM pd_routing_step WHERE routing_id=1 AND step_code='S3'`).Scan(&step)
	fmt.Printf("routing=%q step_S3=%q\n", name, step)
}
