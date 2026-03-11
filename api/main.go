package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"securescan/config"
	"securescan/database"
	"securescan/handlers"
	"securescan/middleware"
	"securescan/routes"
	"securescan/services"

	"github.com/gofiber/fiber/v3"
)

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

	// Migrations live next to the database package
	migrationsDir := filepath.Join(execDir(), "database", "migrations")
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
	scanSvc := services.NewScanService(pool)
	findingSvc := services.NewFindingService(pool)
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
