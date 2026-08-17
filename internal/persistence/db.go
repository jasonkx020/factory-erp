package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"erp/internal/config"
	"erp/internal/dbmigrate"
)

type DB struct {
	SQL    *sql.DB
	Driver string
}

func Open(cfg *config.Config) (*DB, error) {
	registerRebindDriver()
	driver := strings.ToLower(strings.TrimSpace(cfg.Database.Driver))
	if driver == "" || driver == "postgres" || driver == "postgresql" || driver == "pgx" {
		driver = "postgres"
	} else {
		return nil, fmt.Errorf("unsupported database driver %q (postgres only)", cfg.Database.Driver)
	}
	dsn := strings.TrimSpace(cfg.Database.DSN)
	if dsn == "" {
		return nil, fmt.Errorf("database.dsn required")
	}
	sqlDB, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database ping: %w", err)
	}
	out := &DB{SQL: sqlDB, Driver: "postgres"}
	if cfg.Database.InitSchema {
		migrationsRoot := strings.TrimSpace(cfg.Database.MigrationsDir)
		if migrationsRoot == "" {
			migrationsRoot = "migrations"
		}
		seedPath := strings.TrimSpace(cfg.Database.DataPath)
		if err := dbmigrate.InitDevDatabase(context.Background(), dsn, migrationsRoot, seedPath); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("init_schema: %w", err)
		}
	}
	return out, nil
}

func (d *DB) Close() error {
	if d == nil || d.SQL == nil {
		return nil
	}
	return d.SQL.Close()
}

// ExecInsertID runs an INSERT and returns the generated id via RETURNING.
// If query has no RETURNING clause, appends " RETURNING id".
func ExecInsertID(db *sql.DB, query string, args ...any) (int64, error) {
	q := strings.TrimSpace(query)
	q = strings.TrimRight(q, ";")
	upper := strings.ToUpper(q)
	if !strings.Contains(upper, "RETURNING") {
		q = q + " RETURNING id"
	}
	var id int64
	if err := db.QueryRow(q, args...).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

// ExecInsertIDTx is ExecInsertID for a transaction.
func ExecInsertIDTx(tx *sql.Tx, query string, args ...any) (int64, error) {
	q := strings.TrimSpace(query)
	q = strings.TrimRight(q, ";")
	upper := strings.ToUpper(q)
	if !strings.Contains(upper, "RETURNING") {
		q = q + " RETURNING id"
	}
	var id int64
	if err := tx.QueryRow(q, args...).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}
