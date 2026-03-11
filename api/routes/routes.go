package routes

import (
	"securescan/handlers" // HTTP handler structs for each resource group.

	"github.com/gofiber/fiber/v3" // Router + grouping primitives.
)

// Setup registers all HTTP routes for the API.
//
// Why this exists as a separate package:
// - Keeps `main` focused on wiring dependencies and starting the server.
// - Provides a single place to audit public endpoints (useful for security review).
// - Makes it straightforward to version routes or attach middleware at group level.
func Setup(app *fiber.App, ph *handlers.ProjectHandler, sh *handlers.ScanHandler,
	fh *handlers.FindingHandler, fixH *handlers.FixHandler, sseH *handlers.SSEHandler) {

	api := app.Group("/api")

	// Projects
	api.Post("/projects", ph.Create)
	api.Get("/projects/:id", ph.GetByID)

	// Scans
	api.Post("/projects/:id/scan", sh.TriggerScan)
	api.Get("/scans/:id", sh.GetScan)
	api.Get("/scans/:id/stats", sh.GetStats)
	api.Get("/scans/:id/progress", sseH.Progress)

	// Findings
	api.Get("/scans/:id/findings", fh.GetFindings)

	// Fixes
	api.Get("/scans/:id/fixes", fixH.GetFixes)
	api.Post("/fixes/:id/accept", fixH.Accept)
	api.Post("/fixes/:id/reject", fixH.Reject)
	api.Post("/fixes/bulk", fixH.Bulk)
}
