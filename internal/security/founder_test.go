package security

import "testing"

func TestIsFounderEmpNo(t *testing.T) {
	yes := []string{"001", "0001", "E0001", "E001", "EMP-001", "emp001", "1"}
	for _, s := range yes {
		if !IsFounderEmpNo(s) {
			t.Fatalf("want founder emp_no %q", s)
		}
	}
	no := []string{"", "E0002", "002", "E0301", "cust01", "10", "E0010"}
	for _, s := range no {
		if IsFounderEmpNo(s) {
			t.Fatalf("not founder emp_no %q", s)
		}
	}
}

func TestElevateFounder(t *testing.T) {
	roles, perms := elevateFounder(nil, nil)
	if len(roles) != 1 || roles[0] != "sys_admin" {
		t.Fatalf("roles=%v", roles)
	}
	if len(perms) != 1 || perms[0] != WildcardPerm {
		t.Fatalf("perms=%v", perms)
	}
	roles, perms = elevateFounder([]string{"sys_admin", "boss"}, []string{WildcardPerm, "x"})
	if len(roles) != 2 {
		t.Fatalf("should not duplicate role: %v", roles)
	}
	if len(perms) != 2 {
		t.Fatalf("should not duplicate star: %v", perms)
	}
}
