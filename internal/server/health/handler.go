package health

import (
	"time"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

func Register(r *gin.RouterGroup, driver string) {
	r.GET("/health", func(c *gin.Context) {
		api.OK(c, gin.H{
			"status":      "up",
			"driver":      driver,
			"server_time": time.Now().Format(time.RFC3339),
		})
	})
}
