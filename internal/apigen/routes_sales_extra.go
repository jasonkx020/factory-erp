package apigen

import "github.com/gin-gonic/gin"

// RegisterSalesExtra mounts sales loop actions beyond OpenAPI gen.
func RegisterSalesExtra(r *gin.RouterGroup, h Handler) {
	r.POST("/sales/inquiries/:id/submit", h.Dispatch("POST", "/api/v1/sales/inquiries/{id}/submit", "/sales/inquiries/:id/submit", "sales/inquiries", "action:submit"))
	r.POST("/sales/inquiries/:id/reject", h.Dispatch("POST", "/api/v1/sales/inquiries/{id}/reject", "/sales/inquiries/:id/reject", "sales/inquiries", "action:reject"))
	r.POST("/sales/inquiries/:id/withdraw", h.Dispatch("POST", "/api/v1/sales/inquiries/{id}/withdraw", "/sales/inquiries/:id/withdraw", "sales/inquiries", "action:withdraw"))

	r.POST("/sales/pre-shipments/:id/cancel", h.Dispatch("POST", "/api/v1/sales/pre-shipments/{id}/cancel", "/sales/pre-shipments/:id/cancel", "sales/pre-shipments", "action:cancel"))

	r.POST("/sales/deliveries/:id/resubmit", h.Dispatch("POST", "/api/v1/sales/deliveries/{id}/resubmit", "/sales/deliveries/:id/resubmit", "sales/deliveries", "action:resubmit"))
	r.POST("/sales/deliveries/:id/receive", h.Dispatch("POST", "/api/v1/sales/deliveries/{id}/receive", "/sales/deliveries/:id/receive", "sales/deliveries", "action:receive"))

	r.POST("/sales/price-locks/:id/activate", h.Dispatch("POST", "/api/v1/sales/price-locks/{id}/activate", "/sales/price-locks/:id/activate", "sales/price-locks", "action:activate"))
	r.POST("/sales/price-locks/:id/deactivate", h.Dispatch("POST", "/api/v1/sales/price-locks/{id}/deactivate", "/sales/price-locks/:id/deactivate", "sales/price-locks", "action:deactivate"))

	r.POST("/sales/contracts/:id/activate", h.Dispatch("POST", "/api/v1/sales/contracts/{id}/activate", "/sales/contracts/:id/activate", "sales/contracts", "action:activate"))

	r.POST("/sales/sales-boms/:id/deactivate", h.Dispatch("POST", "/api/v1/sales/sales-boms/{id}/deactivate", "/sales/sales-boms/:id/deactivate", "sales/sales-boms", "action:deactivate"))

	r.POST("/sales/cost-budgets/:id/recalc", h.Dispatch("POST", "/api/v1/sales/cost-budgets/{id}/recalc", "/sales/cost-budgets/:id/recalc", "sales/cost-budgets", "action:recalc"))

	r.POST("/sales/outbound-settles/:id/reopen", h.Dispatch("POST", "/api/v1/sales/outbound-settles/{id}/reopen", "/sales/outbound-settles/:id/reopen", "sales/outbound-settles", "action:reopen"))

	r.GET("/sales/self-order-rules", h.Dispatch("GET", "/api/v1/sales/self-order-rules", "/sales/self-order-rules", "sales/self-orders", "list"))
	r.POST("/sales/self-order-rules", h.Dispatch("POST", "/api/v1/sales/self-order-rules", "/sales/self-order-rules", "sales/self-orders", "action:save-rule"))
	r.PUT("/sales/self-order-rules/:id", h.Dispatch("PUT", "/api/v1/sales/self-order-rules/{id}", "/sales/self-order-rules/:id", "sales/self-orders", "action:save-rule"))
}
