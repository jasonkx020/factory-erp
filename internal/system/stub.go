package system

import (
	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// RegisterSkeleton mounts routes that return NOT_IMPLEMENTED until domain packages are filled.
func RegisterSkeleton(r *gin.RouterGroup) {
	paths := []string{
		"/payroll/wage-rates",
		"/hr/employees",
		"/hr/attendance/records",
		"/hr/leave-requests",
		"/approval/tasks",
		"/system/settings",
		"/system/operation-logs",
		"/crm/customers",
		"/sales/orders",
		"/sales/inquiries",
		"/purchase/suppliers",
		"/purchase/inbounds",
		"/report/dashboards/boss",
		"/finance/vouchers",
		"/asset/fixed-assets",
	}
	for _, p := range paths {
		path := p
		r.GET(path, api.NotImplemented)
		r.POST(path, api.NotImplemented)
	}
	r.POST("/payroll/sheets", api.NotImplemented)
	r.GET("/payroll/sheets", api.NotImplemented)
	r.POST("/approval/tasks/:id/approve", api.NotImplemented)
	r.POST("/approval/tasks/:id/reject", api.NotImplemented)
	r.PUT("/system/settings", api.NotImplemented)
	r.POST("/finance/receipt-writeoffs", api.NotImplemented)
}
