package biz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"erp/internal/middleware"
	"erp/internal/security"
)

func TestCheckAPIPerm(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mk := func(roles, perms []string) *gin.Context {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set(middleware.CtxClaims, &security.Claims{Roles: roles, Permissions: perms})
		return c
	}

	if !CheckAPIPerm(mk(nil, nil), "notify/inbox", "list", "GET") {
		t.Fatal("notify should pass")
	}

	c := mk([]string{"sales"}, nil)
	if CheckAPIPerm(c, "sales/orders", "list", "GET") {
		t.Fatal("sales list without perm should deny")
	}

	c = mk([]string{"sales"}, []string{"销售管理:销售订单:查看"})
	if !CheckAPIPerm(c, "sales/orders", "list", "GET") {
		t.Fatal("sales list with view should pass")
	}
	if CheckAPIPerm(c, "sales/orders", "create", "POST") {
		t.Fatal("sales create without edit should deny")
	}

	c = mk([]string{"fin"}, []string{"财务管理:凭证管理:编辑"})
	if !CheckAPIPerm(c, "finance/vouchers", "create", "POST") {
		t.Fatal("finance edit should pass create")
	}

	c = mk([]string{"sys_admin"}, nil)
	if !CheckAPIPerm(c, "purchase/weigh-tickets", "create", "POST") {
		t.Fatal("sys_admin should pass")
	}

	c = mk([]string{"warehouse"}, []string{"库存管理:仓管待入库:查看"})
	if !CheckAPIPerm(c, "purchase/weigh-tickets/by-trace", "list", "GET") {
		t.Fatal("warehouse stockin view should pass by-trace")
	}
	c = mk([]string{"warehouse"}, nil)
	if !CheckAPIPerm(c, "purchase/weigh-tickets/by-trace", "list", "GET") {
		t.Fatal("warehouse role alone should pass by-trace")
	}
	c = mk([]string{"warehouse"}, []string{"库存管理:仓管待入库:编辑"})
	if !CheckAPIPerm(c, "purchase/weigh-tickets", "action:warehouse-confirm", "POST") {
		t.Fatal("warehouse stockin edit should pass warehouse-confirm")
	}
	c = mk([]string{"sales"}, []string{"库存管理:库存查询:查看"})
	if CheckAPIPerm(c, "purchase/weigh-tickets/by-trace", "list", "GET") {
		t.Fatal("sales with inventory query alone should not pass by-trace")
	}
}
