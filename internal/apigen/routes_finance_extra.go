package apigen

import "github.com/gin-gonic/gin"

func RegisterFinanceExtra(r *gin.RouterGroup, h Handler) {
	r.POST("/finance/vouchers/:id/post", h.Dispatch("POST", "/api/v1/finance/vouchers/{id}/post", "/finance/vouchers/:id/post", "finance/vouchers", "action:post"))
	// 产线成本：期间汇入预览（独立路径，避免被 :id 吞掉）
	r.POST("/finance/cost-period-preview", h.Dispatch("POST", "/api/v1/finance/cost-period-preview", "/finance/cost-period-preview", "finance/cost-accountings", "list"))
	r.GET("/finance/cost-period-preview", h.Dispatch("GET", "/api/v1/finance/cost-period-preview", "/finance/cost-period-preview", "finance/cost-accountings", "list"))
	// 供应商在线支付单
	r.GET("/finance/payment-orders", h.Dispatch("GET", "/api/v1/finance/payment-orders", "/finance/payment-orders", "finance/payment-orders", "list"))
	r.POST("/finance/payment-orders", h.Dispatch("POST", "/api/v1/finance/payment-orders", "/finance/payment-orders", "finance/payment-orders", "create"))
	r.GET("/finance/payment-orders/:id", h.Dispatch("GET", "/api/v1/finance/payment-orders/{id}", "/finance/payment-orders/:id", "finance/payment-orders", "get"))
	r.POST("/finance/payment-orders/:id/approve-finance", h.Dispatch("POST", "/api/v1/finance/payment-orders/{id}/approve-finance", "/finance/payment-orders/:id/approve-finance", "finance/payment-orders", "action:approve-finance"))
	r.POST("/finance/payment-orders/:id/approve-boss", h.Dispatch("POST", "/api/v1/finance/payment-orders/{id}/approve-boss", "/finance/payment-orders/:id/approve-boss", "finance/payment-orders", "action:approve-boss"))
	r.POST("/finance/payment-orders/:id/reject", h.Dispatch("POST", "/api/v1/finance/payment-orders/{id}/reject", "/finance/payment-orders/:id/reject", "finance/payment-orders", "action:reject"))
	r.POST("/finance/payment-orders/:id/retry", h.Dispatch("POST", "/api/v1/finance/payment-orders/{id}/retry", "/finance/payment-orders/:id/retry", "finance/payment-orders", "action:retry"))
	r.POST("/pay/alipay/notify", h.Dispatch("POST", "/api/v1/pay/alipay/notify", "/pay/alipay/notify", "finance/payment-orders", "action:alipay-notify"))
}

// RegisterPaymentSystemExtra mounts payment settings singleton.
func RegisterPaymentSystemExtra(r *gin.RouterGroup, h Handler) {
	r.GET("/system/payment-settings", h.Dispatch("GET", "/api/v1/system/payment-settings", "/system/payment-settings", "system/payment-settings", "list"))
	r.PUT("/system/payment-settings", h.Dispatch("PUT", "/api/v1/system/payment-settings", "/system/payment-settings", "system/payment-settings", "replace"))
}
