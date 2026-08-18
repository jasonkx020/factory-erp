package biz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"erp/internal/middleware"
	"erp/internal/security"
)

func TestCheckSystemAPIPerm(t *testing.T) {
	gin.SetMode(gin.TestMode)

	withClaims := func(roles, perms []string) *gin.Context {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set(middleware.CtxClaims, &security.Claims{Roles: roles, Permissions: perms})
		return c
	}

	// 非保护资源放行
	c := withClaims(nil, nil)
	if !CheckSystemAPIPerm(c, "sales/orders", "list", "GET") {
		t.Fatal("sales should pass")
	}

	// 无权限拒绝
	c = withClaims([]string{"piece"}, nil)
	if CheckSystemAPIPerm(c, "system/print-templates", "list", "GET") {
		t.Fatal("expected deny")
	}

	// 查看权限放行 GET
	c = withClaims(nil, []string{"系统管理:自定义打印:查看"})
	if !CheckSystemAPIPerm(c, "system/print-templates", "list", "GET") {
		t.Fatal("view should pass list")
	}

	// 仅查看拒绝 PUT
	c = withClaims(nil, []string{"系统管理:自定义打印:查看"})
	if CheckSystemAPIPerm(c, "system/print-templates", "replace", "PUT") {
		t.Fatal("view should deny put")
	}

	// 编辑放行 PUT
	c = withClaims(nil, []string{"系统管理:自定义打印:编辑"})
	if !CheckSystemAPIPerm(c, "system/print-templates", "replace", "PUT") {
		t.Fatal("edit should pass put")
	}

	// sys_admin 放行
	c = withClaims([]string{"sys_admin"}, nil)
	if !CheckSystemAPIPerm(c, "system/settings", "replace", "PUT") {
		t.Fatal("sys_admin should pass")
	}

	c = withClaims(nil, []string{"*:*:*"})
	if !CheckSystemAPIPerm(c, "system/settings", "replace", "PUT") {
		t.Fatal("wildcard perm should pass")
	}

	c = withClaims(nil, nil)
	c.Set(middleware.CtxClaims, &security.Claims{UserID: 1, Roles: nil, Permissions: nil})
	if !CheckSystemAPIPerm(c, "system/settings", "replace", "PUT") {
		t.Fatal("user id 1 should pass")
	}
}
