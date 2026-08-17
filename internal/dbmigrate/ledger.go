package dbmigrate

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// MigrationRecord is one row in erp_schema_migration.
type MigrationRecord struct {
	Version     string    `db:"version"`
	Description string    `db:"description"`
	Checksum    string    `db:"checksum"`
	AppliedAt   time.Time `db:"applied_at"`
}

// Status summarizes applied and pending migrations.
type Status struct {
	Role        Role
	Applied     []MigrationRecord
	Pending     []UpgradeFile
	Latest      string
	TableExists bool
}

type UpgradeFile struct {
	Path    string
	Version Version
}

func migrationTableExists(db *sqlx.DB) (bool, error) {
	var exists bool
	err := db.Get(&exists, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = 'erp_schema_migration'
		)`)
	return exists, err
}

func listApplied(db *sqlx.DB) ([]MigrationRecord, error) {
	exists, err := migrationTableExists(db)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	var rows []MigrationRecord
	err = db.Select(&rows, `
		SELECT version, description, checksum, applied_at
		FROM erp_schema_migration
		ORDER BY applied_at ASC, version ASC`)
	return rows, err
}

func appliedVersionSet(records []MigrationRecord) map[string]struct{} {
	out := make(map[string]struct{}, len(records))
	for _, r := range records {
		out[r.Version] = struct{}{}
	}
	return out
}

func latestAppliedVersion(records []MigrationRecord) string {
	if len(records) == 0 {
		return ""
	}
	latest := records[0].Version
	latestV, err := ParseVersion(latest)
	if err != nil {
		return records[len(records)-1].Version
	}
	for _, r := range records[1:] {
		v, err := ParseVersion(r.Version)
		if err != nil {
			continue
		}
		if CompareVersions(v, latestV) > 0 {
			latestV = v
			latest = r.Version
		}
	}
	return latest
}

func HasBaseline(records []MigrationRecord) bool {
	_, ok := appliedVersionSet(records)["v1.0.0"]
	return ok
}

func acquireAdvisoryLock(tx *sqlx.Tx, role Role) error {
	_, err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext($1))`, role.lockKey())
	return err
}

func isBenignExistsError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") || strings.Contains(msg, "duplicate")
}

func truncateStmt(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func verifyChecksum(q interface {
	Get(dest interface{}, query string, args ...interface{}) error
}, version, expected string) error {
	var stored string
	err := q.Get(&stored, `SELECT checksum FROM erp_schema_migration WHERE version = $1`, version)
	if err == sql.ErrNoRows {
		return fmt.Errorf("migration %s not recorded after apply", version)
	}
	if err != nil {
		return err
	}
	if stored != expected && expected != "" {
		return fmt.Errorf("checksum mismatch for %s: db=%s file=%s", version, stored, expected)
	}
	return nil
}
