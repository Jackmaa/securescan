package handlers

import (
	"strconv" // Query params arrive as strings; we parse pagination inputs.

	"securescan/services" // Business-layer API for listing findings with filters/pagination.

	"github.com/gofiber/fiber/v3" // HTTP framework used for routing, binding, and responses.
	"github.com/google/uuid"      // Scan IDs are UUIDs in URLs; we validate/parse them here.
)

// FindingHandler exposes HTTP endpoints that read findings from storage.
//
// Handlers intentionally stay thin: they validate/normalize request inputs and delegate
// to services for query logic so that the same behavior is reusable outside HTTP (tests,
// background jobs, future gRPC, etc.).
type FindingHandler struct {
	Service *services.FindingService
}

// NewFindingHandler wires a FindingService into an HTTP-facing handler.
//
// We accept the dependency as an argument (instead of constructing it inside) to keep
// the handler easy to test and to make composition in `main` explicit.
func NewFindingHandler(s *services.FindingService) *FindingHandler {
	return &FindingHandler{Service: s}
}

// GetFindings returns a filtered, paginated list of findings for a scan.
//
// Why this endpoint exists:
//   - Findings are often numerous; pagination is required for UI performance.
//   - Filters (severity/OWASP/tool) let the UI and users slice results without downloading
//     the entire dataset.
//
// Query params:
// - `severity`, `owasp`, `tool`: optional filters
// - `sort`: controls server-side ordering (defaults to `severity`)
// - `page`, `limit`: pagination (defaults 1 / 50; `limit` clamped to 1..100)
func (h *FindingHandler) GetFindings(c fiber.Ctx) error {
	scanID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scan ID"})
	}

	filter := services.FindingFilter{
		ScanID:   scanID,
		Severity: c.Query("severity"),
		Owasp:    c.Query("owasp"),
		Tool:     c.Query("tool"),
		Sort:     c.Query("sort", "severity"),
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	filter.Page = page
	filter.Limit = limit

	result, err := h.Service.List(c.Context(), filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}
