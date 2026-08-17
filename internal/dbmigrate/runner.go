package dbmigrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Options configures migration execution.
type Options struct {
	Role           Role
	DSN            string
	MigrationsRoot string
	DryRun         bool
}

// Runner executes database migrations.
type Runner struct {
	opts  Options
	paths Paths
	db    *sqlx.DB
}

func NewRunner(opts Options) (*Runner, error) {
	if opts.Role == "" {
		opts.Role = RoleERP
	}
	if strings.TrimSpace(opts.DSN) == "" {
		return nil, fmt.Errorf("database DSN is required")
	}
	paths, err := ResolvePaths(opts.Role, opts.MigrationsRoot)
	if err != nil {
		return nil, err
	}
	db, err := sqlx.Open("pgx", opts.DSN)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database ping: %w", err)
	}
	return &Runner{opts: opts, paths: paths, db: db}, nil
}

func (r *Runner) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *Runner) DB() *sqlx.DB { return r.db }

func (r *Runner) Paths() Paths { return r.paths }

// Baseline applies schema.sql.
func (r *Runner) Baseline(ctx context.Context) error {
	raw, err := os.ReadFile(r.paths.SchemaFile)
	if err != nil {
		return err
	}
	if r.opts.DryRun {
		fmt.Printf("dry-run: would apply baseline %s\n", r.paths.SchemaFile)
		return nil
	}
	return r.withLock(ctx, func(tx *sqlx.Tx) error {
		return execStatements(tx, string(raw), true)
	})
}

// SeedDev applies data-dev.sql when present.
func (r *Runner) SeedDev(ctx context.Context) error {
	return r.seedDev(ctx, false)
}

// SeedDevReset clears noisy seed tables then applies data-dev.sql (dev re-init).
func (r *Runner) SeedDevReset(ctx context.Context) error {
	return r.seedDev(ctx, true)
}

func (r *Runner) seedDev(ctx context.Context, reset bool) error {
	if _, err := os.Stat(r.paths.SeedDevFile); err != nil {
		return fmt.Errorf("seed file not found: %s", r.paths.SeedDevFile)
	}
	raw, err := os.ReadFile(r.paths.SeedDevFile)
	if err != nil {
		return err
	}
	if r.opts.DryRun {
		if reset {
			fmt.Printf("dry-run: would RESET seed tables then apply %s\n", r.paths.SeedDevFile)
		} else {
			fmt.Printf("dry-run: would apply seed %s\n", r.paths.SeedDevFile)
		}
		return nil
	}
	return r.withLock(ctx, func(tx *sqlx.Tx) error {
		if reset {
			if err := resetDevSeedTables(tx); err != nil {
				return fmt.Errorf("reset seed tables: %w", err)
			}
			fmt.Println("seed-dev: cleared wage rates / demo noise tables")
		}
		return execStatements(tx, string(raw), true)
	})
}

// resetDevSeedTables removes data that accumulates duplicates on repeated seed-dev.
func resetDevSeedTables(tx *sqlx.Tx) error {
	stmts := []string{
		`DELETE FROM pay_process_wage_rate`,
		`DELETE FROM pd_routing_step WHERE routing_id IN (1,2,3)`,
		`DELETE FROM pd_routing WHERE id IN (1,2,3)`,
	}
	for _, q := range stmts {
		if _, err := tx.Exec(q); err != nil {
			// tolerate missing tables during partial installs
			if isBenignExistsError(err) {
				continue
			}
			// also tolerate undefined_table
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "does not exist") || strings.Contains(msg, "undefined_table") {
				continue
			}
			return fmt.Errorf("%s: %w", truncateStmt(q, 80), err)
		}
	}
	return nil
}

// UpgradeAll applies pending upgrade scripts in version order.
func (r *Runner) UpgradeAll(ctx context.Context) error {
	status, err := r.Status(ctx)
	if err != nil {
		return err
	}
	if !status.TableExists {
		return fmt.Errorf("erp_schema_migration not found; run baseline first")
	}
	for _, up := range status.Pending {
		if err := r.applyUpgrade(ctx, up.Path); err != nil {
			return err
		}
	}
	return nil
}

// UpgradeTo applies upgrades up to and including target version.
func (r *Runner) UpgradeTo(ctx context.Context, target string) error {
	targetV, err := ParseVersion(target)
	if err != nil {
		return err
	}
	status, err := r.Status(ctx)
	if err != nil {
		return err
	}
	if !status.TableExists {
		return fmt.Errorf("erp_schema_migration not found; run baseline first")
	}
	for _, up := range status.Pending {
		if CompareVersions(up.Version, targetV) > 0 {
			break
		}
		if err := r.applyUpgrade(ctx, up.Path); err != nil {
			return err
		}
	}
	return nil
}

// UpgradeFile applies a single upgrade SQL file.
func (r *Runner) UpgradeFile(ctx context.Context, filePath string) error {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(r.paths.UpgradesDir, filePath)
	}
	return r.applyUpgrade(ctx, filePath)
}

func (r *Runner) applyUpgrade(ctx context.Context, filePath string) error {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(raw)
	ver, desc, footerChecksum, ok := ParseMigrationFooter(content)
	if !ok {
		return fmt.Errorf("upgrade file missing erp_schema_migration footer: %s", filePath)
	}
	bodyChecksum := BodyChecksum(content)
	if footerChecksum != "" && footerChecksum != bodyChecksum {
		return fmt.Errorf("checksum mismatch in %s: footer=%s body=%s", filePath, footerChecksum, bodyChecksum)
	}
	if r.opts.DryRun {
		fmt.Printf("dry-run: would apply upgrade %s (%s)\n", filePath, ver)
		return nil
	}
	return r.withLock(ctx, func(tx *sqlx.Tx) error {
		if err := execStatements(tx, content, false); err != nil {
			return err
		}
		_ = desc
		return verifyChecksum(tx, ver, bodyChecksum)
	})
}

// Status returns applied and pending migrations.
func (r *Runner) Status(ctx context.Context) (Status, error) {
	_ = ctx
	tableExists, err := migrationTableExists(r.db)
	if err != nil {
		return Status{}, err
	}
	applied, err := listApplied(r.db)
	if err != nil {
		return Status{}, err
	}
	pending, err := r.listPendingUpgrades(appliedVersionSet(applied))
	if err != nil {
		return Status{}, err
	}
	return Status{
		Role:        r.opts.Role,
		Applied:     applied,
		Pending:     pending,
		Latest:      latestAppliedVersion(applied),
		TableExists: tableExists,
	}, nil
}

func (r *Runner) listPendingUpgrades(applied map[string]struct{}) ([]UpgradeFile, error) {
	entries, err := os.ReadDir(r.paths.UpgradesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []UpgradeFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		ver, err := ParseUpgradeFilename(e.Name())
		if err != nil {
			continue
		}
		if _, ok := applied[ver.Raw]; ok {
			continue
		}
		files = append(files, UpgradeFile{
			Path:    filepath.Join(r.paths.UpgradesDir, e.Name()),
			Version: ver,
		})
	}
	versions := make([]Version, len(files))
	for i, f := range files {
		versions[i] = f.Version
	}
	SortVersions(versions)
	versionOrder := make(map[string]int, len(versions))
	for i, v := range versions {
		versionOrder[v.Raw] = i
	}
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			if versionOrder[files[j].Version.Raw] < versionOrder[files[i].Version.Raw] {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
	return files, nil
}

func (r *Runner) withLock(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := acquireAdvisoryLock(tx, r.opts.Role); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func execStatements(tx *sqlx.Tx, sqlText string, tolerateExists bool) error {
	const sp = "sp_migrate_stmt"
	for _, stmt := range SplitStatements(sqlText) {
		if stmt == "" {
			continue
		}
		if tolerateExists {
			if _, err := tx.Exec("SAVEPOINT " + sp); err != nil {
				return fmt.Errorf("savepoint: %w", err)
			}
		}
		if _, err := tx.Exec(stmt); err != nil {
			if tolerateExists && isBenignExistsError(err) {
				if _, rbErr := tx.Exec("ROLLBACK TO SAVEPOINT " + sp); rbErr != nil {
					return fmt.Errorf("%s: %w (rollback savepoint: %v)", truncateStmt(stmt, 120), err, rbErr)
				}
				continue
			}
			return fmt.Errorf("%s: %w", truncateStmt(stmt, 120), err)
		}
		if tolerateExists {
			if _, err := tx.Exec("RELEASE SAVEPOINT " + sp); err != nil {
				return fmt.Errorf("release savepoint: %w", err)
			}
		}
	}
	return nil
}

// InitDevDatabase runs baseline, all upgrades, and optional seed for local development.
func InitDevDatabase(ctx context.Context, dsn, migrationsRoot, seedPath string) error {
	opts := Options{Role: RoleERP, DSN: dsn, MigrationsRoot: migrationsRoot}
	runner, err := NewRunner(opts)
	if err != nil {
		return err
	}
	defer runner.Close()

	if err := runner.Baseline(ctx); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	if err := runner.UpgradeAll(ctx); err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}
	if seedPath == "" {
		seedPath = runner.paths.SeedDevFile
	}
	if seedPath != "" {
		if _, err := os.Stat(seedPath); err == nil {
			raw, err := os.ReadFile(seedPath)
			if err != nil {
				return err
			}
			if err := runner.withLock(ctx, func(tx *sqlx.Tx) error {
				return execStatements(tx, string(raw), true)
			}); err != nil {
				return fmt.Errorf("seed: %w", err)
			}
		}
	}
	return nil
}

// ResolveDSN builds a PostgreSQL DSN from flag or environment variables.
func ResolveDSN(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}
	if s := os.Getenv("ERP_DATABASE_DSN"); s != "" {
		return s, nil
	}
	if s := os.Getenv("DATABASE_URL"); s != "" {
		return s, nil
	}
	host := envOr("PGHOST", "127.0.0.1")
	port := envOr("PGPORT", "5432")
	user := envOr("PGUSER", "erp")
	pass := envOr("PGPASSWORD", "erp")
	db := envOr("PGDATABASE", "erp_factory")
	sslmode := envOr("PGSSLMODE", "disable")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, db, sslmode), nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
