package production

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

func Register(r *gin.RouterGroup, db *sql.DB) {
	g := r.Group("/production")
	g.GET("/processes", func(c *gin.Context) {
		rows, err := db.Query(`SELECT id, code, name, process_type, is_piecework, is_handover_point, status FROM pd_process WHERE is_deleted=0`)
		if err != nil {
			api.FailJSON(c, "DB_ERROR")
			return
		}
		defer rows.Close()
		list := []gin.H{}
		for rows.Next() {
			var id int64
			var code, name, ptype, status string
			var piece, hand int
			_ = rows.Scan(&id, &code, &name, &ptype, &piece, &hand, &status)
			list = append(list, gin.H{
				"id": id, "code": code, "name": name, "process_type": ptype, "status": status,
				"is_piecework": piece == 1, "is_handover_point": hand == 1,
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
