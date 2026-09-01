package apigen

import "github.com/gin-gonic/gin"

// RegisterHRExtra mounts HR delivery routes not yet in OpenAPI gen.
func RegisterHRExtra(r *gin.RouterGroup, h Handler) {
	// 岗位管理：实现已在 biz；挂在 extra，避免 gen-routes 覆盖 routes_gen 时丢失
	r.GET("/hr/job-titles", h.Dispatch("GET", "/api/v1/hr/job-titles", "/hr/job-titles", "hr/job-titles", "list"))
	r.POST("/hr/job-titles", h.Dispatch("POST", "/api/v1/hr/job-titles", "/hr/job-titles", "hr/job-titles", "create"))
	r.POST("/hr/job-titles/ensure", h.Dispatch("POST", "/api/v1/hr/job-titles/ensure", "/hr/job-titles/ensure", "hr/job-titles", "ensure"))
	r.GET("/hr/job-titles/:id", h.Dispatch("GET", "/api/v1/hr/job-titles/{id}", "/hr/job-titles/:id", "hr/job-titles", "get"))
	r.PUT("/hr/job-titles/:id", h.Dispatch("PUT", "/api/v1/hr/job-titles/{id}", "/hr/job-titles/:id", "hr/job-titles", "update"))
	r.DELETE("/hr/job-titles/:id", h.Dispatch("DELETE", "/api/v1/hr/job-titles/{id}", "/hr/job-titles/:id", "hr/job-titles", "remove"))

	r.POST("/hr/leave-requests/:id/approve", h.Dispatch("POST", "/api/v1/hr/leave-requests/{id}/approve", "/hr/leave-requests/:id/approve", "hr/leave-requests", "action:approve"))
	r.POST("/hr/leave-requests/:id/reject", h.Dispatch("POST", "/api/v1/hr/leave-requests/{id}/reject", "/hr/leave-requests/:id/reject", "hr/leave-requests", "action:reject"))
	r.POST("/hr/overtime-patches/:id/approve", h.Dispatch("POST", "/api/v1/hr/overtime-patches/{id}/approve", "/hr/overtime-patches/:id/approve", "hr/overtime-patches", "action:approve"))
	r.POST("/hr/overtime-patches/:id/reject", h.Dispatch("POST", "/api/v1/hr/overtime-patches/{id}/reject", "/hr/overtime-patches/:id/reject", "hr/overtime-patches", "action:reject"))
	r.POST("/hr/overtime-patches/:id/cancel", h.Dispatch("POST", "/api/v1/hr/overtime-patches/{id}/cancel", "/hr/overtime-patches/:id/cancel", "hr/overtime-patches", "action:cancel"))
	r.POST("/hr/attendance-perf-summaries/recalc", h.Dispatch("POST", "/api/v1/hr/attendance-perf-summaries/recalc", "/hr/attendance-perf-summaries/recalc", "hr/attendance-perf-summaries", "action:recalc"))
	r.POST("/hr/id-card/ocr", h.Dispatch("POST", "/api/v1/hr/id-card/ocr", "/hr/id-card/ocr", "hr/employees", "create"))
	r.POST("/hr/tool-issues/:id/approve", h.Dispatch("POST", "/api/v1/hr/tool-issues/{id}/approve", "/hr/tool-issues/:id/approve", "hr/tool-issues", "action:approve"))
	r.POST("/hr/tool-issues/:id/reject", h.Dispatch("POST", "/api/v1/hr/tool-issues/{id}/reject", "/hr/tool-issues/:id/reject", "hr/tool-issues", "action:reject"))
	r.POST("/hr/tool-issues/:id/return-request", h.Dispatch("POST", "/api/v1/hr/tool-issues/{id}/return-request", "/hr/tool-issues/:id/return-request", "hr/tool-issues", "action:return-request"))
	r.POST("/hr/tool-issues/:id/return-confirm", h.Dispatch("POST", "/api/v1/hr/tool-issues/{id}/return-confirm", "/hr/tool-issues/:id/return-confirm", "hr/tool-issues", "action:return-confirm"))
}
