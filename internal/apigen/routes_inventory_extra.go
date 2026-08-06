package apigen

import "github.com/gin-gonic/gin"

// RegisterInventoryExtra mounts inventory helper routes beyond OpenAPI gen.
func RegisterInventoryExtra(r *gin.RouterGroup, h Handler) {
	r.GET("/inventory/warehouses", h.Dispatch("GET", "/api/v1/inventory/warehouses", "/inventory/warehouses", "inventory/warehouses", "list"))
}
