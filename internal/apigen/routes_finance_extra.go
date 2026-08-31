package apigen

import "github.com/gin-gonic/gin"

func RegisterFinanceExtra(r *gin.RouterGroup, h Handler) {
	r.POST("/finance/vouchers/:id/post", h.Dispatch("POST", "/api/v1/finance/vouchers/{id}/post", "/finance/vouchers/:id/post", "finance/vouchers", "action:post"))
	// 产线成本：期间汇入预览（独立路径，避免被 :id 吞掉）
	r.POST("/finance/cost-period-preview", h.Dispatch("POST", "/api/v1/finance/cost-period-preview", "/finance/cost-period-preview", "finance/cost-accountings", "list"))
	r.GET("/finance/cost-period-preview", h.Dispatch("GET", "/api/v1/finance/cost-period-preview", "/finance/cost-period-preview", "finance/cost-accountings", "list"))
}
