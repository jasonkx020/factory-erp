package apigen

import "github.com/gin-gonic/gin"

func RegisterFinanceExtra(r *gin.RouterGroup, h Handler) {
	r.POST("/finance/vouchers/:id/post", h.Dispatch("POST", "/api/v1/finance/vouchers/{id}/post", "/finance/vouchers/:id/post", "finance/vouchers", "action:post"))
}
