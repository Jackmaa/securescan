package main

import (
	"flag"          // Small CLI surface for operational tasks (e.g., migrations).
	"log"           // Startup logs are written to stdout/stderr for container friendliness.
	"os"            // Used for filesystem setup and discovering the executable location.
	"path/filepath" // OS-correct path assembly for migrations directory.
	"strings"       // Simple path heuristics for locating migrations in dev vs compiled runs.

	"securescan/config"     // Central configuration loading (env + optional .env).
	"securescan/database"   // PostgreSQL connection + lightweight migration runner.
	"securescan/handlers"   // HTTP handlers (thin HTTP→service adapters).
	"securescan/middleware" // Global middleware (recover/logger/CORS).
	"securescan/routes"     // Route registration and API surface definition.
	"securescan/services"   // Business logic + persistence boundary.

	"github.com/gofiber/fiber/v3" // HTTP server framework.
)

// main is the API server entrypoint.
//
// Why this file is mostly "wiring":
// - The server is composed from small packages (config/database/services/handlers/routes).
// - Keeping construction here makes dependencies explicit and keeps other packages testable.
func main() {
	migrateOnly := flag.Bool("migrate", false, "run migrations and exit")
	flag.Parse()

	cfg := config.Load()

	pool, err := database.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("connected to PostgreSQL")

	// Migrations are stored in `api/database/migrations`.
	//
	// When running via `go run`, the compiled binary lives under a Go build cache
	// directory, so "next to the binary" is *not* the repo checkout. We therefore
	// resolve migrations from a few known locations (cwd, repo layout) before
	// falling back to execDir for real deployments.
	migrationsDir := resolveMigrationsDir()
	if err := database.Migrate(pool, migrationsDir); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	if *migrateOnly {
		log.Println("migrations complete")
		return
	}

	// Ensure scan workspace exists
	os.MkdirAll(cfg.ScanWorkspace, 0755)

	// Services
	projectSvc := services.NewProjectService(pool, cfg.ScanWorkspace)
	findingSvc := services.NewFindingService(pool)
	scanSvc := services.NewScanService(pool, findingSvc)
	fixSvc := services.NewFixService(pool)

	// Handlers
	projectH := handlers.NewProjectHandler(projectSvc)
	scanH := handlers.NewScanHandler(scanSvc, projectSvc)
	findingH := handlers.NewFindingHandler(findingSvc)
	fixH := handlers.NewFixHandler(fixSvc)
	sseH := handlers.NewSSEHandler(scanSvc)

	// Fiber app
	app := fiber.New(fiber.Config{
		AppName: "SecureScan API",
	})

	middleware.Setup(app, cfg.FrontendURL)
	routes.Setup(app, projectH, scanH, findingH, fixH, sseH)

	log.Printf("SecureScan API listening on :%s", cfg.APIPort)
	log.Fatal(app.Listen(":" + cfg.APIPort))
}

// execDir returns the directory containing the running binary,
// falling back to the current working directory. Used to locate
// migration files relative to the binary.
func execDir() string {
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	return filepath.Dir(exe)
}

// resolveMigrationsDir returns the on-disk directory containing SQL migrations.
//
// Resolution strategy (first match wins):
// - `MIGRATIONS_DIR` environment variable (explicit override)
// - `<cwd>/database/migrations` when starting from `api/`
// - `<cwd>/api/database/migrations` when starting from repo root
// - `<execDir>/database/migrations` for compiled deployments where migrations are shipped
func resolveMigrationsDir() string {
	if explicit := strings.TrimSpace(os.Getenv("MIGRATIONS_DIR")); explicit != "" {
		return explicit
	}

	wd, err := os.Getwd()
	if err == nil {
		candidates := []string{
			filepath.Join(wd, "database", "migrations"),
			filepath.Join(wd, "api", "database", "migrations"),
		}
		for _, dir := range candidates {
			if _, statErr := os.Stat(dir); statErr == nil {
				return dir
			}
		}
	}

	return filepath.Join(execDir(), "database", "migrations")
}
