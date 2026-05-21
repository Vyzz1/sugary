package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"

	"sugary/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <up|down|version|create> [args]", filepath.Base(os.Args[0]))
	}

	cfg := config.Load()
	cmd := os.Args[1]

	switch cmd {
	case "create":
		if err := createMigration(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		return
	case "version", "up", "down":
		if err := runMigration(cfg, cmd, os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		return
	default:
		log.Fatalf("unknown command %q", cmd)
	}
}

func runMigration(cfg config.Config, cmd string, args []string) error {
	db, err := sql.Open("pgx", cfg.Postgres.DSN())
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://db/migrations", "postgres", driver)
	if err != nil {
		return err
	}

	switch cmd {
	case "up":
		if len(args) == 0 {
			return m.Up()
		}
		steps, err := parseSteps(args[0])
		if err != nil {
			return err
		}
		return m.Steps(steps)
	case "down":
		if len(args) == 0 {
			return m.Down()
		}
		steps, err := parseSteps(args[0])
		if err != nil {
			return err
		}
		return m.Steps(-steps)
	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				fmt.Println("version: none")
				return nil
			}
			return err
		}
		fmt.Printf("version: %d dirty: %t\n", version, dirty)
		return nil
	default:
		return nil
	}
}

func parseSteps(value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid steps %q: %w", value, err)
	}
	if n < 1 {
		return 0, fmt.Errorf("steps must be positive, got %d", n)
	}
	return n, nil
}

func createMigration(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: migrate create <name>")
	}

	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		return errors.New("migration name is required")
	}

	version, err := nextMigrationVersion("db/migrations")
	if err != nil {
		return err
	}

	slug := migrationSlug(name)
	upPath := filepath.Join("db/migrations", fmt.Sprintf("%06d_%s.up.sql", version, slug))
	downPath := filepath.Join("db/migrations", fmt.Sprintf("%06d_%s.down.sql", version, slug))

	if err := os.WriteFile(upPath, []byte(fmt.Sprintf("-- %s\n", name)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(downPath, []byte(fmt.Sprintf("-- rollback %s\n", name)), 0o644); err != nil {
		return err
	}

	fmt.Println(upPath)
	fmt.Println(downPath)
	return nil
}

func nextMigrationVersion(dir string) (int64, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return 0, err
	}
	var maxVersion int64
	for _, entry := range entries {
		base := filepath.Base(entry)
		parts := strings.SplitN(base, "_", 2)
		if len(parts) != 2 {
			continue
		}
		version, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}
		if version > maxVersion {
			maxVersion = version
		}
	}
	return maxVersion + 1, nil
}

func migrationSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "-", "_")
	return strings.Trim(value, "_")
}
