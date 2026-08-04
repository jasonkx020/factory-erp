package product

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

func Register(r *gin.RouterGroup, db *sql.DB) {
	g := r.Group("/product")
	g.GET("/products", func(c *gin.Context) {
		rows, err := db.Query(`SELECT id, code, name, product_type, cost_price, sale_price, status FROM prd_product WHERE is_deleted=0`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var code, name, ptype, status string
			var cost, sale sql.NullFloat64
			_ = rows.Scan(&id, &code, &name, &ptype, &cost, &sale, &status)
			list = append(list, gin.H{
				"id": id, "code": code, "name": name, "product_type": ptype, "status": status,
				"cost_price": cost.Float64, "sale_price": sale.Float64,
			})
		}
		api.OK(c, gin.H{"list": list, "total": len(list)})
	})
	g.POST("/products", api.NotImplemented)
	g.GET("/products/:id", api.NotImplemented)
	g.PUT("/products/:id", api.NotImplemented)
	g.GET("/products/:id/units", api.NotImplemented)
	g.PUT("/products/:id/units", api.NotImplemented)
}
