package database

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// RunMigrations applies embedded SQL migrations in lexical order exactly once.
//
// Each migration is executed in its own transaction and tracked in
// schema_migrations to guarantee idempotent startup runs.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return err
	}

	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return err
	}

	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		versions = append(versions, entry.Name())
	}

	sort.Strings(versions)

	for _, version := range versions {
		applied, err := isMigrationApplied(ctx, pool, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := migrationFiles.ReadFile(filepath.Join("migrations", version))
		if err != nil {
			return err
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %s failed: %w", version, err)
		}

		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			tx.Rollback(ctx)
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
}

func isMigrationApplied(ctx context.Context, pool *pgxpool.Pool, version string) (bool, error) {
	row := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
