package dbmigrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateResult holds validation outcome for one upgrade file.
type ValidateResult struct {
	File    string
	Version string
	OK      bool
	Error   string
}

// ValidateUpgrades checks upgrade filenames and checksum footers under migrations root.
func ValidateUpgrades(migrationsRoot string, roles ...Role) ([]ValidateResult, error) {
	if len(roles) == 0 {
		roles = []Role{RoleERP}
	}
	var results []ValidateResult
	for _, role := range roles {
		paths, err := ResolvePaths(role, migrationsRoot)
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(paths.UpgradesDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
				continue
			}
			filePath := filepath.Join(paths.UpgradesDir, e.Name())
			res := ValidateResult{File: filePath}
			if _, err := ParseUpgradeFilename(e.Name()); err != nil {
				res.Error = err.Error()
				results = append(results, res)
				continue
			}
			raw, err := os.ReadFile(filePath)
			if err != nil {
				res.Error = err.Error()
				results = append(results, res)
				continue
			}
			content := string(raw)
			ver, _, footerChecksum, ok := ParseMigrationFooter(content)
			if !ok {
				res.Error = "missing erp_schema_migration footer"
				results = append(results, res)
				continue
			}
			res.Version = ver
			bodyChecksum := BodyChecksum(content)
			if footerChecksum != bodyChecksum {
				res.Error = fmt.Sprintf("checksum mismatch: footer=%s body=%s", footerChecksum, bodyChecksum)
				results = append(results, res)
				continue
			}
			res.OK = true
			results = append(results, res)
		}
	}
	return results, nil
}

// CreateUpgradeTemplate writes a new upgrade SQL file with computed checksum footer.
func CreateUpgradeTemplate(role Role, migrationsRoot, version, description string) (string, error) {
	if role == "" {
		role = RoleERP
	}
	if _, err := ParseVersion(version); err != nil {
		return "", err
	}
	slug := slugify(description)
	if slug == "" {
		slug = "change"
	}
	paths, err := ResolvePaths(role, migrationsRoot)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(paths.UpgradesDir, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s_%s.sql", version, slug)
	filePath := filepath.Join(paths.UpgradesDir, filename)
	if _, err := os.Stat(filePath); err == nil {
		return "", fmt.Errorf("upgrade file already exists: %s", filePath)
	}
	body := fmt.Sprintf("-- %s: %s\n\n-- TODO: add migration SQL here\n", version, description)
	checksum := BodyChecksum(body)
	footer := fmt.Sprintf(`
INSERT INTO erp_schema_migration (version, description, checksum)
VALUES ('%s', '%s', '%s')
ON CONFLICT (version) DO NOTHING;
`, version, escapeSQLLiteral(description), checksum)
	content := body + footer
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return "", err
	}
	return filePath, nil
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		case r == ' ' || r == '-' || r == '_':
			if !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

func escapeSQLLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
