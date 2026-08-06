package apigen

import "github.com/gin-gonic/gin"

// RegisterInventoryExtra mounts inventory helper routes beyond OpenAPI gen.
// warehouses 已由 OpenAPI gen-routes 注册，此处勿重复挂载。
func RegisterInventoryExtra(r *gin.RouterGroup, h Handler) {
	_ = r
	_ = h
}
