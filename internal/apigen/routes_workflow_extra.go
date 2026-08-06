package apigen

import "github.com/gin-gonic/gin"

// RegisterWorkflowExtra mounts ticket workflow routes.
func RegisterWorkflowExtra(r *gin.RouterGroup, h Handler) {
	r.GET("/workflow/ticket-categories", h.Dispatch("GET", "/api/v1/workflow/ticket-categories", "/workflow/ticket-categories", "workflow/ticket-categories", "list"))
	r.POST("/workflow/ticket-categories", h.Dispatch("POST", "/api/v1/workflow/ticket-categories", "/workflow/ticket-categories", "workflow/ticket-categories", "create"))
	r.GET("/workflow/ticket-categories/:id", h.Dispatch("GET", "/api/v1/workflow/ticket-categories/{id}", "/workflow/ticket-categories/:id", "workflow/ticket-categories", "get"))
	r.PUT("/workflow/ticket-categories/:id", h.Dispatch("PUT", "/api/v1/workflow/ticket-categories/{id}", "/workflow/ticket-categories/:id", "workflow/ticket-categories", "update"))
	r.GET("/workflow/ticket-categories/:id/handlers", h.Dispatch("GET", "/api/v1/workflow/ticket-categories/{id}/handlers", "/workflow/ticket-categories/:id/handlers", "workflow/ticket-categories", "get"))
	r.PUT("/workflow/ticket-categories/:id/handlers", h.Dispatch("PUT", "/api/v1/workflow/ticket-categories/{id}/handlers", "/workflow/ticket-categories/:id/handlers", "workflow/ticket-categories", "update"))
	r.GET("/workflow/ticket-handler-pool", h.Dispatch("GET", "/api/v1/workflow/ticket-handler-pool", "/workflow/ticket-handler-pool", "workflow/tickets", "list"))
	r.GET("/workflow/tickets", h.Dispatch("GET", "/api/v1/workflow/tickets", "/workflow/tickets", "workflow/tickets", "list"))
	r.POST("/workflow/tickets", h.Dispatch("POST", "/api/v1/workflow/tickets", "/workflow/tickets", "workflow/tickets", "create"))
	r.GET("/workflow/tickets/:id", h.Dispatch("GET", "/api/v1/workflow/tickets/{id}", "/workflow/tickets/:id", "workflow/tickets", "get"))
	r.POST("/workflow/tickets/:id/assign", h.Dispatch("POST", "/api/v1/workflow/tickets/{id}/assign", "/workflow/tickets/:id/assign", "workflow/tickets", "action:assign"))
	r.POST("/workflow/tickets/:id/action", h.Dispatch("POST", "/api/v1/workflow/tickets/{id}/action", "/workflow/tickets/:id/action", "workflow/tickets", "action:action"))
	r.GET("/workflow/tickets/:id/handlers-pool", h.Dispatch("GET", "/api/v1/workflow/tickets/{id}/handlers-pool", "/workflow/tickets/:id/handlers-pool", "workflow/tickets", "get"))
}
