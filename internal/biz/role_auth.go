package biz

import (
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

func (s *Services) requireMobileClient(c *gin.Context) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(cl.ClientType), "mobile") {
		api.FailJSON(c, "APP_ONLY")
		return false
	}
	return true
}

func (s *Services) requireAnyRole(c *gin.Context, roles ...string) bool {
	cl := middleware.Claims(c)
	if cl == nil {
		api.FailJSON(c, "UNAUTHORIZED")
		return false
	}
	aliases := map[string][]string{
		"purchase":  {"purchase", "采购员", "采购"},
		"warehouse": {"warehouse", "仓管员", "仓管"},
		"finance":   {"finance", "财务", "财务员"},
		"qc":        {"qc", "质检", "质检员"},
		"foreman":   {"foreman", "车间主任", "主任", "生管"},
		"piece":     {"piece", "计件工"},
		"fixed":     {"fixed", "固定工"},
		"sales":     {"sales", "销售员", "销售"},
	}
	for _, r := range cl.Roles {
		if r == "sys_admin" || r == "admin" || r == "系统管理员" {
			return true
		}
		for _, need := range roles {
			if strings.EqualFold(r, need) {
				return true
			}
			for _, a := range aliases[strings.ToLower(need)] {
				if strings.EqualFold(r, a) {
					return true
				}
			}
		}
	}
	api.FailJSON(c, "ROLE_FORBIDDEN")
	return false
}
