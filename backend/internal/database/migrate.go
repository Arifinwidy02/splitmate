package database

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationLockKey int64 = 736281

type Migration struct {
	Version int64
	Name    string
	Up      string
	Down    string
}

func ParseMigrations(fsys fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	byVersion := map[int64]*Migration{}
	var versions []int64

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		base := strings.TrimSuffix(entry.Name(), ".sql")
		dot := strings.LastIndex(base, ".")
		if dot < 0 {
			return nil, fmt.Errorf("invalid migration filename %q", entry.Name())
		}

		prefix, kind := base[:dot], base[dot+1:]
		if kind != "up" && kind != "down" {
			return nil, fmt.Errorf("invalid migration kind in %q", entry.Name())
		}

		sep := strings.Index(prefix, "_")
		if sep < 0 {
			return nil, fmt.Errorf("invalid migration filename %q", entry.Name())
		}

		version, err := strconv.ParseInt(prefix[:sep], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid migration version in %q: %w", entry.Name(), err)
		}

		content, err := fs.ReadFile(fsys, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}

		if _, ok := byVersion[version]; !ok {
			byVersion[version] = &Migration{Version: version, Name: prefix[sep+1:]}
			versions = append(versions, version)
		}

		switch kind {
		case "up":
			byVersion[version].Up = string(content)
		case "down":
			byVersion[version].Down = string(content)
		}
	}

	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })

	migrations := make([]Migration, 0, len(versions))
	for _, version := range versions {
		m := byVersion[version]
		if m.Up == "" {
			return nil, fmt.Errorf("migration %d has no up file", version)
		}
		migrations = append(migrations, *m)
	}

	return migrations, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, migrations []Migration, logger *slog.Logger) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockKey); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", migrationLockKey)

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    BIGINT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	applied := map[int64]bool{}
	rows, err := conn.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("query applied migrations: %w", err)
	}
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			rows.Close()
			return fmt.Errorf("scan applied migration: %w", err)
		}
		applied[version] = true
	}
	rows.Close()
	if rows.Err() != nil {
		return fmt.Errorf("iterate applied migrations: %w", rows.Err())
	}

	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", m.Version, err)
		}

		if _, err := tx.Exec(ctx, m.Up); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("apply migration %d (%s): %w", m.Version, m.Name, err)
		}

		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", m.Version); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("record migration %d: %w", m.Version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.Version, err)
		}

		logger.Info("migration applied", "version", m.Version, "name", m.Name)
	}

	return nil
}
