package apigen

import "github.com/gin-gonic/gin"

// RegisterProductionExtra mounts production delivery actions beyond OpenAPI gen.
func RegisterProductionExtra(r *gin.RouterGroup, h Handler) {
	r.POST("/production/consignments/:id/progress", h.Dispatch("POST", "/api/v1/production/consignments/{id}/progress", "/production/consignments/:id/progress", "production/consignments", "action:progress"))
	r.POST("/production/cost-hide-policies", h.Dispatch("POST", "/api/v1/production/cost-hide-policies", "/production/cost-hide-policies", "production/cost-hide-policies", "create"))
	r.GET("/production/drawing-links/:id", h.Dispatch("GET", "/api/v1/production/drawing-links/{id}", "/production/drawing-links/:id", "production/drawing-links", "get"))
}
