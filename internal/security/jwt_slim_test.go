package security

import (
	"strings"
	"testing"
)

func TestSlimTokenOmitsPermissions(t *testing.T) {
	secret := "test-secret-for-slim-jwt"
	roles := []string{"sys_admin"}
	fatPerms := make([]string, 0, 400)
	for i := 0; i < 400; i++ {
		fatPerms = append(fatPerms, "域模块示例:子模块名称:查看")
	}

	fat, err := IssueToken(secret, 60, Claims{
		UserID: 1, LoginName: "admin", UserType: "internal", ClientType: "admin",
		Roles: roles, Permissions: fatPerms,
	})
	if err != nil {
		t.Fatal(err)
	}
	slim, err := IssueToken(secret, 60, SlimForToken(Claims{
		UserID: 1, LoginName: "admin", UserType: "internal", ClientType: "admin",
		Roles: roles, Permissions: fatPerms,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(slim) >= len(fat)/2 {
		t.Fatalf("slim token not much smaller: slim=%d fat=%d", len(slim), len(fat))
	}
	if len(slim) > 800 {
		t.Fatalf("slim token still too large: %d bytes", len(slim))
	}
	parsed, err := ParseToken(secret, slim)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Permissions) != 0 {
		t.Fatalf("expected no permissions in slim token, got %d", len(parsed.Permissions))
	}
	if strings.Join(parsed.Roles, ",") != "sys_admin" {
		t.Fatalf("roles=%v", parsed.Roles)
	}
}
