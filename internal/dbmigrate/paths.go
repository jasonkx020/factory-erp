package dbmigrate

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths holds resolved migration file locations for a role.
type Paths struct {
	Role        Role
	Root        string
	SchemaFile  string
	SeedDevFile string
	UpgradesDir string
}

func ResolvePaths(role Role, migrationsRoot string) (Paths, error) {
	if role == "" {
		role = RoleERP
	}
	root := filepath.Clean(migrationsRoot)
	if root == "" || root == "." {
		var err error
		root, err = DefaultMigrationsRoot()
		if err != nil {
			return Paths{}, err
		}
	}
	roleDir := filepath.Join(root, string(role))
	schema := filepath.Join(roleDir, "schema.sql")
	if _, err := os.Stat(schema); err != nil {
		return Paths{}, fmt.Errorf("schema not found: %s", schema)
	}
	return Paths{
		Role:        role,
		Root:        root,
		SchemaFile:  schema,
		SeedDevFile: filepath.Join(roleDir, "data-dev.sql"),
		UpgradesDir: filepath.Join(roleDir, "upgrades"),
	}, nil
}

// DefaultMigrationsRoot locates migrations/ from cwd or ERP_MIGRATIONS_DIR.
func DefaultMigrationsRoot() (string, error) {
	if dir := os.Getenv("ERP_MIGRATIONS_DIR"); dir != "" {
		return filepath.Clean(dir), nil
	}
	candidates := []string{"migrations"}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "migrations"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return filepath.Clean(c), nil
		}
	}
	return "", fmt.Errorf("migrations directory not found; set ERP_MIGRATIONS_DIR or run from repo root")
}
