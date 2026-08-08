package biz

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"erp/internal/middleware"
	"erp/internal/security"
)

func TestCanCreateReportWorkBackfill(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Services{}

	mk := func(roles, perms []string, body map[string]interface{}) *gin.Context {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(middleware.CtxClaims, &security.Claims{Roles: roles, Permissions: perms})
		if body != nil {
			c.Set("body", body)
		}
		return c
	}

	if s.canCreateReportWorkBackfill(mk([]string{"foreman"}, nil, nil), map[string]interface{}{}) {
		t.Fatal("foreman should not backfill")
	}
	if s.canCreateReportWorkBackfill(mk([]string{"sys_admin"}, nil, nil), map[string]interface{}{}) {
		t.Fatal("admin without reason should not backfill")
	}
	if !s.canCreateReportWorkBackfill(mk([]string{"sys_admin"}, nil, nil), map[string]interface{}{"backfill_reason": "现场补录"}) {
		t.Fatal("admin with reason should backfill")
	}
}
