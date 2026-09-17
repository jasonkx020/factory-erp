package production

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

func Register(r *gin.RouterGroup, db *sql.DB) {
	g := r.Group("/production")
	g.GET("/processes", func(c *gin.Context) {
		rows, err := db.Query(`SELECT p.id, p.code, p.name, p.is_handover_point,
			COALESCE(NULLIF(r.pay_mode,''),'none'), r.id
			FROM pd_process p
			LEFT JOIN pay_process_wage_rate r ON r.process_id=p.id AND r.status='active'
			WHERE p.is_deleted=0`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var code, name, payMode string
			var hand int
			var rateID sql.NullInt64
			_ = rows.Scan(&id, &code, &name, &hand, &payMode, &rateID)
			hasWage := rateID.Valid && rateID.Int64 > 0
			billable := payMode == "weight" || payMode == "piece"
			status := "inactive"
			if hasWage {
				status = "active"
			}
			list = append(list, gin.H{
				"id": id, "code": code, "name": name, "status": status,
				"pay_mode": payMode, "is_piecework": billable, "has_wage": hasWage, "is_handover_point": hand == 1,
			})
		}
		api.OK(c, gin.H{"list": list})
	})
	g.POST("/processes", api.NotImplemented)
	g.GET("/routings", api.NotImplemented)
	g.POST("/routings", api.NotImplemented)
	g.GET("/tasks", api.NotImplemented)
	g.POST("/tasks", api.NotImplemented)
	g.GET("/dispatches", api.NotImplemented)
	g.POST("/dispatches", api.NotImplemented)
	g.POST("/requisitions", api.NotImplemented)
	g.GET("/report-works", api.NotImplemented)
	g.POST("/report-works", api.NotImplemented)
	g.GET("/qc-orders", api.NotImplemented)
	g.POST("/qc-orders", api.NotImplemented)
	g.GET("/progress", api.NotImplemented)
}
