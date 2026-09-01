package apigen

import "github.com/gin-gonic/gin"

// RegisterPlantingExtra mounts planting management APIs.
func RegisterPlantingExtra(r *gin.RouterGroup, h Handler) {
	r.GET("/planting/dashboard/overview", h.Dispatch("GET", "/api/v1/planting/dashboard/overview", "/planting/dashboard/overview", "planting/dashboard", "list"))

	r.GET("/planting/plots", h.Dispatch("GET", "/api/v1/planting/plots", "/planting/plots", "planting/plots", "list"))
	r.POST("/planting/plots", h.Dispatch("POST", "/api/v1/planting/plots", "/planting/plots", "planting/plots", "create"))
	r.GET("/planting/plots/:id", h.Dispatch("GET", "/api/v1/planting/plots/{id}", "/planting/plots/:id", "planting/plots", "get"))
	r.PUT("/planting/plots/:id", h.Dispatch("PUT", "/api/v1/planting/plots/{id}", "/planting/plots/:id", "planting/plots", "replace"))
	r.DELETE("/planting/plots/:id", h.Dispatch("DELETE", "/api/v1/planting/plots/{id}", "/planting/plots/:id", "planting/plots", "delete"))

	r.GET("/planting/contracts", h.Dispatch("GET", "/api/v1/planting/contracts", "/planting/contracts", "planting/contracts", "list"))
	r.POST("/planting/contracts", h.Dispatch("POST", "/api/v1/planting/contracts", "/planting/contracts", "planting/contracts", "create"))
	r.GET("/planting/contracts/:id", h.Dispatch("GET", "/api/v1/planting/contracts/{id}", "/planting/contracts/:id", "planting/contracts", "get"))
	r.PUT("/planting/contracts/:id", h.Dispatch("PUT", "/api/v1/planting/contracts/{id}", "/planting/contracts/:id", "planting/contracts", "replace"))
	r.DELETE("/planting/contracts/:id", h.Dispatch("DELETE", "/api/v1/planting/contracts/{id}", "/planting/contracts/:id", "planting/contracts", "delete"))

	r.GET("/planting/field-logs", h.Dispatch("GET", "/api/v1/planting/field-logs", "/planting/field-logs", "planting/field-logs", "list"))
	r.POST("/planting/field-logs", h.Dispatch("POST", "/api/v1/planting/field-logs", "/planting/field-logs", "planting/field-logs", "create"))
	r.GET("/planting/field-logs/:id", h.Dispatch("GET", "/api/v1/planting/field-logs/{id}", "/planting/field-logs/:id", "planting/field-logs", "get"))
	r.DELETE("/planting/field-logs/:id", h.Dispatch("DELETE", "/api/v1/planting/field-logs/{id}", "/planting/field-logs/:id", "planting/field-logs", "delete"))

	r.GET("/planting/harvest-plans", h.Dispatch("GET", "/api/v1/planting/harvest-plans", "/planting/harvest-plans", "planting/harvest-plans", "list"))
	r.POST("/planting/harvest-plans", h.Dispatch("POST", "/api/v1/planting/harvest-plans", "/planting/harvest-plans", "planting/harvest-plans", "create"))
	r.GET("/planting/harvest-plans/:id", h.Dispatch("GET", "/api/v1/planting/harvest-plans/{id}", "/planting/harvest-plans/:id", "planting/harvest-plans", "get"))
	r.PUT("/planting/harvest-plans/:id", h.Dispatch("PUT", "/api/v1/planting/harvest-plans/{id}", "/planting/harvest-plans/:id", "planting/harvest-plans", "replace"))
	r.DELETE("/planting/harvest-plans/:id", h.Dispatch("DELETE", "/api/v1/planting/harvest-plans/{id}", "/planting/harvest-plans/:id", "planting/harvest-plans", "delete"))
	r.POST("/planting/harvest-plans/:id/confirm", h.Dispatch("POST", "/api/v1/planting/harvest-plans/{id}/confirm", "/planting/harvest-plans/:id/confirm", "planting/harvest-plans", "action:confirm"))
	r.POST("/planting/harvest-plans/:id/to-arrival", h.Dispatch("POST", "/api/v1/planting/harvest-plans/{id}/to-arrival", "/planting/harvest-plans/:id/to-arrival", "planting/harvest-plans", "action:to-arrival"))
}
