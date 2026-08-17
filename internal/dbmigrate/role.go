package dbmigrate

import "fmt"

// Role identifies the migration set. factory-erp uses a single database role.
type Role string

const (
	RoleERP Role = "erp"
)

func ParseRole(raw string) (Role, error) {
	switch Role(raw) {
	case RoleERP, "":
		return RoleERP, nil
	default:
		return "", fmt.Errorf("invalid role %q: want erp", raw)
	}
}

func (r Role) lockKey() string {
	if r == "" {
		r = RoleERP
	}
	return "erp_migrate_" + string(r)
}
