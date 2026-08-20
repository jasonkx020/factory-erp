package apigen

import "github.com/gin-gonic/gin"

// RegisterPurchaseExtra mounts purchase delivery actions beyond OpenAPI gen.
func RegisterPurchaseExtra(r *gin.RouterGroup, h Handler) {
	r.POST("/purchase/requests/:id/approve", h.Dispatch("POST", "/api/v1/purchase/requests/{id}/approve", "/purchase/requests/:id/approve", "purchase/requests", "action:approve"))
	r.POST("/purchase/requests/:id/reject", h.Dispatch("POST", "/api/v1/purchase/requests/{id}/reject", "/purchase/requests/:id/reject", "purchase/requests", "action:reject"))
	r.POST("/purchase/requests/:id/to-plan", h.Dispatch("POST", "/api/v1/purchase/requests/{id}/to-plan", "/purchase/requests/:id/to-plan", "purchase/requests", "action:to-plan"))
	r.POST("/purchase/plans/:id/approve", h.Dispatch("POST", "/api/v1/purchase/plans/{id}/approve", "/purchase/plans/:id/approve", "purchase/plans", "action:approve"))
	r.POST("/purchase/plans/:id/to-inbound", h.Dispatch("POST", "/api/v1/purchase/plans/{id}/to-inbound", "/purchase/plans/:id/to-inbound", "purchase/plans", "action:to-inbound"))

	r.GET("/purchase/trace-batch-codes", h.Dispatch("GET", "/api/v1/purchase/trace-batch-codes", "/purchase/trace-batch-codes", "purchase/trace-batch-codes", "list"))
	r.POST("/purchase/trace-batch-codes/generate", h.Dispatch("POST", "/api/v1/purchase/trace-batch-codes/generate", "/purchase/trace-batch-codes/generate", "purchase/trace-batch-codes", "action:generate"))
	r.POST("/purchase/trace-batch-codes/validate", h.Dispatch("POST", "/api/v1/purchase/trace-batch-codes/validate", "/purchase/trace-batch-codes/validate", "purchase/trace-batch-codes", "action:validate"))
	r.POST("/purchase/trace-batch-codes/void", h.Dispatch("POST", "/api/v1/purchase/trace-batch-codes/void", "/purchase/trace-batch-codes/void", "purchase/trace-batch-codes", "action:void"))
	r.POST("/purchase/trace-batch-codes/end", h.Dispatch("POST", "/api/v1/purchase/trace-batch-codes/end", "/purchase/trace-batch-codes/end", "purchase/trace-batch-codes", "action:end"))

	r.GET("/purchase/weigh-flow/next-options", h.Dispatch("GET", "/api/v1/purchase/weigh-flow/next-options", "/purchase/weigh-flow/next-options", "purchase/weigh-tickets", "list"))

	// 完成本批分板（仓管）；须单独注册，Gin 不会把 /box-stock-in 匹配到 /box-stock-in/complete
	r.POST("/purchase/weigh-tickets/:id/box-stock-in/complete", h.Dispatch(
		"POST",
		"/api/v1/purchase/weigh-tickets/{id}/box-stock-in/complete",
		"/purchase/weigh-tickets/:id/box-stock-in/complete",
		"purchase/weigh-tickets",
		"action:box-stock-in-complete",
	))

	r.POST("/biz/uploads", h.Dispatch("POST", "/api/v1/biz/uploads", "/biz/uploads", "biz/uploads", "create"))
}
