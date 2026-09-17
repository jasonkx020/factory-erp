package apigen

import "github.com/gin-gonic/gin"

// RegisterSystemFactoryExtra mounts factory-core system endpoints (setup readiness, plants, product profile).
func RegisterSystemFactoryExtra(r *gin.RouterGroup, e *Engine) {
	r.GET("/system/setup-readiness", func(c *gin.Context) {
		e.Biz.GetSetupReadiness(c)
	})
	r.GET("/system/product-profile", func(c *gin.Context) {
		e.Biz.GetProductProfile(c)
	})
	r.GET("/hr/plants", e.Dispatch("GET", "/api/v1/hr/plants", "/hr/plants", "hr/plants", "list"))
	r.POST("/hr/plants", e.Dispatch("POST", "/api/v1/hr/plants", "/hr/plants", "hr/plants", "create"))
	r.GET("/hr/plants/:id", e.Dispatch("GET", "/api/v1/hr/plants/{id}", "/hr/plants/:id", "hr/plants", "get"))
	r.PUT("/hr/plants/:id", e.Dispatch("PUT", "/api/v1/hr/plants/{id}", "/hr/plants/:id", "hr/plants", "update"))
	r.DELETE("/hr/plants/:id", e.Dispatch("DELETE", "/api/v1/hr/plants/{id}", "/hr/plants/:id", "hr/plants", "delete"))
}
