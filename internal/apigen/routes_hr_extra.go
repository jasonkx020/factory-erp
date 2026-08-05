package apigen

import "github.com/gin-gonic/gin"

// RegisterHRExtra mounts HR delivery routes not yet in OpenAPI gen.
func RegisterHRExtra(r *gin.RouterGroup, h Handler) {
	r.POST("/hr/leave-requests/:id/approve", h.Dispatch("POST", "/api/v1/hr/leave-requests/{id}/approve", "/hr/leave-requests/:id/approve", "hr/leave-requests", "action:approve"))
	r.POST("/hr/leave-requests/:id/reject", h.Dispatch("POST", "/api/v1/hr/leave-requests/{id}/reject", "/hr/leave-requests/:id/reject", "hr/leave-requests", "action:reject"))
	r.POST("/hr/overtime-patches/:id/approve", h.Dispatch("POST", "/api/v1/hr/overtime-patches/{id}/approve", "/hr/overtime-patches/:id/approve", "hr/overtime-patches", "action:approve"))
	r.POST("/hr/overtime-patches/:id/reject", h.Dispatch("POST", "/api/v1/hr/overtime-patches/{id}/reject", "/hr/overtime-patches/:id/reject", "hr/overtime-patches", "action:reject"))
	r.POST("/hr/overtime-patches/:id/cancel", h.Dispatch("POST", "/api/v1/hr/overtime-patches/{id}/cancel", "/hr/overtime-patches/:id/cancel", "hr/overtime-patches", "action:cancel"))
	r.POST("/hr/attendance-perf-summaries/recalc", h.Dispatch("POST", "/api/v1/hr/attendance-perf-summaries/recalc", "/hr/attendance-perf-summaries/recalc", "hr/attendance-perf-summaries", "action:recalc"))
}
