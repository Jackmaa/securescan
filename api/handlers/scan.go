package handlers

import (
	"securescan/services" // Scan orchestration + query operations for scans/projects.

	"github.com/gofiber/fiber/v3" // HTTP framework used for request context + JSON responses.
	"github.com/google/uuid"      // Scan/project IDs are UUIDs passed via URL params.
)

// ScanHandler exposes scan lifecycle endpoints: trigger, fetch, and stats.
//
// Scans are long-running. Triggering a scan returns quickly with a scan ID; progress
// is delivered out-of-band via SSE (see `SSEHandler`).
type ScanHandler struct {
	ScanService    *services.ScanService
	ProjectService *services.ProjectService
}

// NewScanHandler wires scan/project services into an HTTP handler.
//
// We pass both services because triggering a scan needs the project record, while
// scan reads/stats only need the scan service.
func NewScanHandler(ss *services.ScanService, ps *services.ProjectService) *ScanHandler {
	return &ScanHandler{ScanService: ss, ProjectService: ps}
}

// TriggerScan creates a scan record and starts the scan asynchronously.
//
// Why async:
// - Tooling can take seconds/minutes; keeping HTTP requests open is brittle.
// - Clients can immediately transition UI state to “running” and subscribe to SSE.
func (h *ScanHandler) TriggerScan(c fiber.Ctx) error {
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid project ID"})
	}

	project, err := h.ProjectService.GetByID(c.Context(), projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "project not found"})
	}

	scan, err := h.ScanService.CreateAndRun(c.Context(), project)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(scan)
}

// GetScan returns the persisted scan record (status + timestamps + score/grade).
//
// The UI uses this for refresh/polling and for pages that don’t keep an SSE
// connection open.
func (h *ScanHandler) GetScan(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scan ID"})
	}

	scan, err := h.ScanService.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "scan not found"})
	}

	return c.JSON(scan)
}

// GetStats returns aggregated dashboard metrics derived from the scan’s findings.
//
// Aggregation happens server-side to avoid downloading all findings just to compute
// counts and groupings on the client.
func (h *ScanHandler) GetStats(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scan ID"})
	}

	stats, err := h.ScanService.GetStats(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(stats)
}
