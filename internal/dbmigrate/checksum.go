package dbmigrate

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var migrationFooterPattern = regexp.MustCompile(`(?is)\n\s*INSERT\s+INTO\s+erp_schema_migration\b`)

// BodyChecksum returns SHA256 hex of SQL content excluding the migration ledger footer.
func BodyChecksum(content string) string {
	body := StripMigrationFooter(content)
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

// StripMigrationFooter removes trailing INSERT INTO erp_schema_migration statement.
func StripMigrationFooter(content string) string {
	loc := migrationFooterPattern.FindStringIndex(content)
	if loc == nil {
		return strings.TrimSpace(content)
	}
	return strings.TrimSpace(content[:loc[0]])
}

// ParseMigrationFooter extracts version, description, checksum from file content.
func ParseMigrationFooter(content string) (version, description, checksum string, ok bool) {
	loc := migrationFooterPattern.FindStringIndex(content)
	if loc == nil {
		return "", "", "", false
	}
	footer := content[loc[0]:]
	valuesPattern := regexp.MustCompile(`VALUES\s*\(\s*'([^']*)'\s*,\s*'([^']*)'\s*,\s*'([^']*)'\s*\)`)
	m := valuesPattern.FindStringSubmatch(footer)
	if len(m) != 4 {
		return "", "", "", false
	}
	return m[1], m[2], m[3], true
}
