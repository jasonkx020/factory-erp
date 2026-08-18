package apigen

import "github.com/gin-gonic/gin"

// RegisterPayrollExtra mounts payroll confirm/pay/commission-run routes.
func RegisterPayrollExtra(r *gin.RouterGroup, h Handler) {
	r.POST("/payroll/sheets/:id/confirm", h.Dispatch("POST", "/api/v1/payroll/sheets/{id}/confirm", "/payroll/sheets/:id/confirm", "payroll/sheets", "action:confirm"))
	r.POST("/payroll/sheets/:id/pay", h.Dispatch("POST", "/api/v1/payroll/sheets/{id}/pay", "/payroll/sheets/:id/pay", "payroll/sheets", "action:pay"))
	r.POST("/payroll/commission-calcs/run", h.Dispatch("POST", "/api/v1/payroll/commission-calcs/run", "/payroll/commission-calcs/run", "payroll/commission-calcs", "action:run"))
	r.GET("/payroll/work-records", h.Dispatch("GET", "/api/v1/payroll/work-records", "/payroll/work-records", "payroll/work-records", "list"))
}
