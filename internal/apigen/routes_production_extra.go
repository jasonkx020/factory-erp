package apigen

import "github.com/gin-gonic/gin"

// RegisterProductionExtra mounts production delivery actions beyond OpenAPI gen.
func RegisterProductionExtra(r *gin.RouterGroup, h Handler) {
	r.POST("/production/consignments/:id/progress", h.Dispatch("POST", "/api/v1/production/consignments/{id}/progress", "/production/consignments/:id/progress", "production/consignments", "action:progress"))
	r.POST("/production/cost-hide-policies", h.Dispatch("POST", "/api/v1/production/cost-hide-policies", "/production/cost-hide-policies", "production/cost-hide-policies", "create"))
	r.GET("/production/drawing-links/:id", h.Dispatch("GET", "/api/v1/production/drawing-links/{id}", "/production/drawing-links/:id", "production/drawing-links", "get"))
	r.GET("/production/shifts", h.Dispatch("GET", "/api/v1/production/shifts", "/production/shifts", "production/shifts", "list"))
	r.POST("/production/shifts", h.Dispatch("POST", "/api/v1/production/shifts", "/production/shifts", "production/shifts", "create"))
	r.GET("/production/shifts/:id", h.Dispatch("GET", "/api/v1/production/shifts/{id}", "/production/shifts/:id", "production/shifts", "get"))
	r.PUT("/production/shifts/:id", h.Dispatch("PUT", "/api/v1/production/shifts/{id}", "/production/shifts/:id", "production/shifts", "update"))
	r.POST("/production/shifts/:id/close", h.Dispatch("POST", "/api/v1/production/shifts/{id}/close", "/production/shifts/:id/close", "production/shifts", "action:close"))
	r.POST("/production/shifts/:id/members", h.Dispatch("POST", "/api/v1/production/shifts/{id}/members", "/production/shifts/:id/members", "production/shifts", "action:add-member"))
	r.DELETE("/production/shifts/:id/members/:memberId", h.Dispatch("DELETE", "/api/v1/production/shifts/{id}/members/{memberId}", "/production/shifts/:id/members/:memberId", "production/shifts", "action:remove-member"))
	r.PUT("/production/flow-rules", h.Dispatch("PUT", "/api/v1/production/flow-rules", "/production/flow-rules", "production/flow-rules", "update"))
}
