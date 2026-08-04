package persistence

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"

	"erp/internal/config"
)

type DB struct {
	SQL    *sql.DB
	Driver string
}

func Open(cfg *config.Config) (*DB, error) {
	switch strings.ToLower(cfg.Database.Driver) {
	case "sqlite", "sqlite3":
		return openSQLite(cfg.Database.SQLitePath)
	case "mysql":
		if cfg.Database.MySQLDSN == "" {
			return nil, fmt.Errorf("mysql_dsn required")
		}
		db, err := sql.Open("mysql", cfg.Database.MySQLDSN)
		if err != nil {
			return nil, err
		}
		if err := db.Ping(); err != nil {
			_ = db.Close()
			return nil, err
		}
		return &DB{SQL: db, Driver: "mysql"}, nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}
}

func openSQLite(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	needInit := false
	if _, err := os.Stat(path); os.IsNotExist(err) {
		needInit = true
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, err
	}
	out := &DB{SQL: db, Driver: "sqlite"}
	if !needInit {
		var n int
		_ = db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='schema_meta'`).Scan(&n)
		if n == 0 {
			needInit = true
		}
	}
	if needInit {
		if err := out.MigrateSQLite(); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return out, nil
}

func (d *DB) MigrateSQLite() error {
	schemaPath := filepath.Join("db", "sqlite", "schema.sql")
	seedPath := filepath.Join("db", "sqlite", "seed.sql")
	if err := execSQLFile(d.SQL, schemaPath); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	if err := execSQLFile(d.SQL, seedPath); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	return nil
}

func execSQLFile(db *sql.DB, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	stmts := splitSQL(string(b))
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("%s: %w\n--- sql ---\n%s", path, err, truncate(s, 200))
		}
	}
	return nil
}

func splitSQL(script string) []string {
	var out []string
	var b strings.Builder
	lines := strings.Split(script, "\n")
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
		if strings.HasSuffix(strings.TrimSpace(b.String()), ";") {
			stmt := strings.TrimSpace(b.String())
			stmt = strings.TrimSuffix(stmt, ";")
			stmt = strings.TrimSpace(stmt)
			if stmt != "" {
				out = append(out, stmt)
			}
			b.Reset()
		}
	}
	if rest := strings.TrimSpace(b.String()); rest != "" {
		out = append(out, rest)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (d *DB) Close() error {
	if d == nil || d.SQL == nil {
		return nil
	}
	return d.SQL.Close()
}
