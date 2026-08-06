package health

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/alert"
	"erp/internal/api"
)

// Register mounts /health (含 DB 探活)、/ready、/live。
func Register(r *gin.RouterGroup, db *sql.DB, driver string) {
	pingDB := func() error {
		if db == nil {
			return sql.ErrConnDone
		}
		return db.Ping()
	}

	r.GET("/live", func(c *gin.Context) {
		api.OK(c, gin.H{"status": "alive", "server_time": time.Now().Format(time.RFC3339)})
	})

	r.GET("/ready", func(c *gin.Context) {
		if err := pingDB(); err != nil {
			c.JSON(http.StatusServiceUnavailable, api.Response{Code: 0, Msg: "NOT_READY", Data: gin.H{
				"status": "not_ready", "driver": driver, "db": "down", "error": err.Error(),
			}})
			return
		}
		api.OK(c, gin.H{"status": "ready", "driver": driver, "db": "up", "server_time": time.Now().Format(time.RFC3339)})
	})

	r.GET("/health", func(c *gin.Context) {
		dbStatus := "up"
		status := "up"
		code := http.StatusOK
		if err := pingDB(); err != nil {
			dbStatus = "down"
			status = "degraded"
			code = http.StatusServiceUnavailable
			alert.Default.Warn("db", err.Error())
			c.JSON(code, api.Response{Code: 0, Msg: "UNHEALTHY", Data: gin.H{
				"status": status, "driver": driver, "db": dbStatus, "server_time": time.Now().Format(time.RFC3339),
			}})
			return
		}
		api.OK(c, gin.H{
			"status":      status,
			"driver":      driver,
			"db":          dbStatus,
			"server_time": time.Now().Format(time.RFC3339),
		})
	})
}
