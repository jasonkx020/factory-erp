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

	r.GET("/production/process-returns", h.Dispatch("GET", "/api/v1/production/process-returns", "/production/process-returns", "production/process-returns", "list"))
	r.POST("/production/process-returns", h.Dispatch("POST", "/api/v1/production/process-returns", "/production/process-returns", "production/process-returns", "create"))
	r.GET("/production/process-returns/:id", h.Dispatch("GET", "/api/v1/production/process-returns/{id}", "/production/process-returns/:id", "production/process-returns", "get"))
	r.POST("/production/process-returns/:id/submit", h.Dispatch("POST", "/api/v1/production/process-returns/{id}/submit", "/production/process-returns/:id/submit", "production/process-returns", "action:submit"))
	r.POST("/production/process-returns/:id/approve", h.Dispatch("POST", "/api/v1/production/process-returns/{id}/approve", "/production/process-returns/:id/approve", "production/process-returns", "action:approve"))
	r.POST("/production/process-returns/:id/reject", h.Dispatch("POST", "/api/v1/production/process-returns/{id}/reject", "/production/process-returns/:id/reject", "production/process-returns", "action:reject"))
	r.POST("/production/process-returns/:id/transfer", h.Dispatch("POST", "/api/v1/production/process-returns/{id}/transfer", "/production/process-returns/:id/transfer", "production/process-returns", "action:transfer"))
	r.POST("/production/process-returns/:id/warehouse-confirm", h.Dispatch("POST", "/api/v1/production/process-returns/{id}/warehouse-confirm", "/production/process-returns/:id/warehouse-confirm", "production/process-returns", "action:warehouse-confirm"))
	r.POST("/production/report-works/:id/void", h.Dispatch("POST", "/api/v1/production/report-works/{id}/void", "/production/report-works/:id/void", "production/report-works", "action:void"))
	r.POST("/production/board-issues", h.Dispatch("POST", "/api/v1/production/board-issues", "/production/board-issues", "production/board-issues", "create"))
	r.POST("/production/board-issues/return", h.Dispatch("POST", "/api/v1/production/board-issues/return", "/production/board-issues/return", "production/board-issues", "action:return"))
	r.POST("/production/board-moves", h.Dispatch("POST", "/api/v1/production/board-moves", "/production/board-moves", "production/board-moves", "create"))
	r.POST("/production/board-close/preview", h.Dispatch("POST", "/api/v1/production/board-close/preview", "/production/board-close/preview", "production/board-close", "action:preview"))
	r.POST("/production/board-close", h.Dispatch("POST", "/api/v1/production/board-close", "/production/board-close", "production/board-close", "create"))
	r.GET("/production/process-yields", h.Dispatch("GET", "/api/v1/production/process-yields", "/production/process-yields", "production/process-yields", "list"))
	r.GET("/production/process-yields/traces", h.Dispatch("GET", "/api/v1/production/process-yields/traces", "/production/process-yields/traces", "production/process-yields", "list"))
}
