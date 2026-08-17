package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"erp/internal/dbmigrate"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	switch cmd {
	case "baseline":
		os.Exit(runWithRunner(args, func(r *dbmigrate.Runner) error {
			return r.Baseline(context.Background())
		}))
	case "upgrade":
		os.Exit(runUpgrade(args))
	case "status":
		os.Exit(runStatus(args))
	case "seed-dev":
		os.Exit(runSeedDev(args))
	case "validate":
		os.Exit(runValidate(args))
	case "create":
		os.Exit(runCreate(args))
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `erp-db — factory-erp PostgreSQL migration tool

Usage:
  erp-db baseline   [--dsn URL] [--migrations-dir PATH] [--dry-run]
  erp-db upgrade    [--all|--to VERSION|--file PATH] [--dry-run]
  erp-db status
  erp-db seed-dev [--reset]
  erp-db validate   [--migrations-dir PATH]
  erp-db create     VERSION "description"

Environment:
  ERP_DATABASE_DSN, ERP_MIGRATIONS_DIR, DATABASE_URL
  PGHOST PGPORT PGUSER PGPASSWORD PGDATABASE

Notes:
  seed-dev --reset  先清理工价等易重复种子表，再重新写入 data-dev.sql（开发环境用）
`)
}

type commonFlags struct {
	dsn            string
	migrationsRoot string
	dryRun         bool
}

func parseCommonFlags(args []string) (commonFlags, []string) {
	fs := flag.NewFlagSet("erp-db", flag.ExitOnError)
	var cf commonFlags
	fs.StringVar(&cf.dsn, "dsn", "", "PostgreSQL DSN")
	fs.StringVar(&cf.migrationsRoot, "migrations-dir", "", "migrations root directory")
	fs.BoolVar(&cf.dryRun, "dry-run", false, "print actions without applying")
	_ = fs.Parse(args)
	return cf, fs.Args()
}

func newRunnerFromFlags(cf commonFlags) (*dbmigrate.Runner, error) {
	dsn, err := dbmigrate.ResolveDSN(cf.dsn)
	if err != nil {
		return nil, err
	}
	return dbmigrate.NewRunner(dbmigrate.Options{
		Role:           dbmigrate.RoleERP,
		DSN:            dsn,
		MigrationsRoot: cf.migrationsRoot,
		DryRun:         cf.dryRun,
	})
}

func runWithRunner(args []string, fn func(*dbmigrate.Runner) error) int {
	cf, _ := parseCommonFlags(args)
	runner, err := newRunnerFromFlags(cf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer runner.Close()
	if err := fn(runner); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runSeedDev(args []string) int {
	fs := flag.NewFlagSet("seed-dev", flag.ExitOnError)
	var cf commonFlags
	var reset bool
	fs.StringVar(&cf.dsn, "dsn", "", "PostgreSQL DSN")
	fs.StringVar(&cf.migrationsRoot, "migrations-dir", "", "migrations root directory")
	fs.BoolVar(&cf.dryRun, "dry-run", false, "print actions without applying")
	fs.BoolVar(&reset, "reset", false, "clear wage-rate and related seed tables before seeding")
	_ = fs.Parse(args)
	runner, err := newRunnerFromFlags(cf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer runner.Close()
	var runErr error
	if reset {
		runErr = runner.SeedDevReset(context.Background())
	} else {
		runErr = runner.SeedDev(context.Background())
	}
	if runErr != nil {
		fmt.Fprintln(os.Stderr, runErr)
		return 1
	}
	return 0
}

func runUpgrade(args []string) int {
	fs := flag.NewFlagSet("upgrade", flag.ExitOnError)
	var cf commonFlags
	var upgradeAll bool
	var upgradeTo string
	var upgradeFile string
	fs.StringVar(&cf.dsn, "dsn", "", "PostgreSQL DSN")
	fs.StringVar(&cf.migrationsRoot, "migrations-dir", "", "migrations root directory")
	fs.BoolVar(&cf.dryRun, "dry-run", false, "print actions without applying")
	fs.BoolVar(&upgradeAll, "all", false, "apply all pending upgrades")
	fs.StringVar(&upgradeTo, "to", "", "apply upgrades up to version")
	fs.StringVar(&upgradeFile, "file", "", "apply a single upgrade file")
	_ = fs.Parse(args)
	if !upgradeAll && upgradeTo == "" && upgradeFile == "" {
		if len(fs.Args()) > 0 && (fs.Args()[0] == "--all" || fs.Args()[0] == "all") {
			upgradeAll = true
		} else if len(fs.Args()) == 1 {
			if v, err := dbmigrate.ParseVersion(fs.Args()[0]); err == nil {
				upgradeTo = v.Raw
			} else if strings.HasSuffix(fs.Args()[0], ".sql") {
				upgradeFile = fs.Args()[0]
			} else {
				upgradeAll = true
			}
		} else {
			upgradeAll = true
		}
	}
	runner, err := newRunnerFromFlags(cf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer runner.Close()
	ctx := context.Background()
	var runErr error
	switch {
	case upgradeFile != "":
		runErr = runner.UpgradeFile(ctx, upgradeFile)
	case upgradeTo != "":
		runErr = runner.UpgradeTo(ctx, upgradeTo)
	default:
		runErr = runner.UpgradeAll(ctx)
	}
	if runErr != nil {
		fmt.Fprintln(os.Stderr, runErr)
		return 1
	}
	return 0
}

func runStatus(args []string) int {
	cf, _ := parseCommonFlags(args)
	runner, err := newRunnerFromFlags(cf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer runner.Close()
	status, err := runner.Status(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("role: %s\n", status.Role)
	fmt.Printf("table_exists: %t\n", status.TableExists)
	fmt.Printf("latest: %s\n", status.Latest)
	fmt.Println("applied:")
	for _, r := range status.Applied {
		fmt.Printf("  %s  %s  %s\n", r.Version, r.Description, r.AppliedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println("pending:")
	for _, p := range status.Pending {
		fmt.Printf("  %s  %s\n", p.Version.Raw, p.Path)
	}
	return 0
}

func runValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	migrationsRoot := fs.String("migrations-dir", "", "migrations root directory")
	_ = fs.Parse(args)
	root := *migrationsRoot
	if root == "" {
		var err error
		root, err = dbmigrate.DefaultMigrationsRoot()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	results, err := dbmigrate.ValidateUpgrades(root, dbmigrate.RoleERP)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	failed := 0
	for _, res := range results {
		if res.OK {
			fmt.Printf("OK  %s (%s)\n", res.File, res.Version)
			continue
		}
		failed++
		fmt.Printf("FAIL %s: %s\n", res.File, res.Error)
	}
	if failed > 0 {
		return 1
	}
	fmt.Println("validate: ok")
	return 0
}

func runCreate(args []string) int {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	migrationsRoot := fs.String("migrations-dir", "", "migrations root directory")
	_ = fs.Parse(args)
	if len(fs.Args()) < 2 {
		fmt.Fprintln(os.Stderr, `usage: erp-db create v1.0.1 "description"`)
		return 1
	}
	path, err := dbmigrate.CreateUpgradeTemplate(dbmigrate.RoleERP, *migrationsRoot, fs.Args()[0], strings.Join(fs.Args()[1:], " "))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(path)
	return 0
}
