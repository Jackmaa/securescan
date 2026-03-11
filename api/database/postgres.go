package database

import (
	"context"       // Used for connection setup, pings, and migration execution.
	"fmt"           // Wrap errors with context so callers get actionable messages.
	"log"           // Migration progress is logged for operational visibility.
	"os"            // Read migration files from disk at startup.
	"path/filepath" // Build OS-correct paths to migration directories/files.
	"sort"          // Ensure deterministic migration ordering.
	"strings"       // Filter filenames by extension.

	"github.com/jackc/pgx/v5/pgxpool" // PostgreSQL driver + connection pooling.
)

// Connect creates a connection pool to PostgreSQL.
//
// Why a pool:
// - The API serves concurrent requests; a single connection would serialize queries.
// - The driver manages connection reuse and health checks efficiently.
func Connect(dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

// Migrate runs all SQL files in the migrations directory in lexicographic order.
// Each migration is idempotent (uses IF NOT EXISTS, etc.) so running them
// multiple times is safe. No migration tracking table — keep it simple.
//
// This is intentionally lightweight:
//   - For early-stage projects, a minimal “run every .sql file” approach reduces
//     operational overhead.
//   - Idempotent migrations avoid the complexity of version tables while remaining safe.
//
// If/when migrations become non-idempotent, this should evolve into a tracked system
// (e.g., goose, golang-migrate, or a custom schema_migrations table).
func Migrate(pool *pgxpool.Pool, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, f := range files {
		path := filepath.Join(migrationsDir, f)
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}

		if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
			return fmt.Errorf("execute %s: %w", f, err)
		}

		log.Printf("migrated: %s", f)
	}

	return nil
}
