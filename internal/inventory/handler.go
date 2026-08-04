package inventory

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

func Register(r *gin.RouterGroup, db *sql.DB) {
	g := r.Group("/inventory")
	g.GET("/balances", func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT b.warehouse_id, b.product_id, b.batch_no, b.qty, p.code, p.name
			FROM inv_balance b JOIN prd_product p ON p.id = b.product_id`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var wh, pid int64
			var batch, code, name string
			var qty float64
			_ = rows.Scan(&wh, &pid, &batch, &qty, &code, &name)
			list = append(list, gin.H{
				"warehouse_id": wh, "product_id": pid, "batch_no": batch, "qty": qty,
				"product_code": code, "product_name": name,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
	})
	g.GET("/availability", func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT b.product_id, b.warehouse_id, b.qty,
			  COALESCE((SELECT SUM(r.qty) FROM inv_reservation r WHERE r.product_id=b.product_id AND r.warehouse_id=b.warehouse_id AND r.status='active'),0) AS reserved
			FROM inv_balance b`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var pid, wh int64
			var onHand, reserved float64
			_ = rows.Scan(&pid, &wh, &onHand, &reserved)
			list = append(list, gin.H{
				"product_id": pid, "warehouse_id": wh, "on_hand": onHand,
				"reserved": reserved, "in_transit": 0, "available": onHand - reserved,
			})
		}
		api.OK(c, gin.H{"list": list})
	})
	g.GET("/stock-txns", api.NotImplemented)
	g.POST("/stock-txns", api.NotImplemented)
	g.POST("/stock-txns/:id/post", api.NotImplemented)
	g.GET("/transfers", api.NotImplemented)
	g.POST("/transfers", api.NotImplemented)
	g.GET("/stocktakes", api.NotImplemented)
	g.POST("/stocktakes", api.NotImplemented)
	g.GET("/box-codes", api.NotImplemented)
}
