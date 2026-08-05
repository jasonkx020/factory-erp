package apigen

import "github.com/gin-gonic/gin"

// RegisterApprovalExtra mounts create routes for approval queues (factory delivery).
func RegisterApprovalExtra(r *gin.RouterGroup, h Handler) {
	r.POST("/approval/tasks", h.Dispatch("POST", "/api/v1/approval/tasks", "/approval/tasks", "approval/tasks", "create"))
	r.POST("/approval/doc-reviews", h.Dispatch("POST", "/api/v1/approval/doc-reviews", "/approval/doc-reviews", "approval/doc-reviews", "create"))
	r.POST("/approval/expense-finance", h.Dispatch("POST", "/api/v1/approval/expense-finance", "/approval/expense-finance", "approval/expense-finance", "create"))
	r.POST("/approval/inquiry-finance", h.Dispatch("POST", "/api/v1/approval/inquiry-finance", "/approval/inquiry-finance", "approval/inquiry-finance", "create"))
	r.POST("/approval/inquiry-lines", h.Dispatch("POST", "/api/v1/approval/inquiry-lines", "/approval/inquiry-lines", "approval/inquiry-lines", "create"))
	r.POST("/approval/purchases", h.Dispatch("POST", "/api/v1/approval/purchases", "/approval/purchases", "approval/purchases", "create"))
	r.POST("/approval/purchase-plans", h.Dispatch("POST", "/api/v1/approval/purchase-plans", "/approval/purchase-plans", "approval/purchase-plans", "create"))
	r.POST("/approval/affairs", h.Dispatch("POST", "/api/v1/approval/affairs", "/approval/affairs", "approval/affairs", "create"))
	r.POST("/approval/attendance", h.Dispatch("POST", "/api/v1/approval/attendance", "/approval/attendance", "approval/attendance", "create"))
}
