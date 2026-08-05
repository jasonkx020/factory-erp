package biz

import (
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

func (s *Services) requireAnyRole(c *gin.Context, roles ...string) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return false
	}
	for _, r := range cl.Roles {
		if r == "sys_admin" || r == "admin" || r == "系统管理员" {
			return true
		}
		for _, need := range roles {
			if strings.EqualFold(r, need) {
				return true
			}
		}
	}
	api.FailJSON(c, "ROLE_FORBIDDEN")
	return false
}
