package handlers

import (
	"securescan/models"   // Request/response shapes (e.g., bulk actions) shared across layers.
	"securescan/services" // Fix persistence + state transitions.

	"github.com/gofiber/fiber/v3" // HTTP framework used for binding and JSON responses.
	"github.com/google/uuid"      // Fix/scan IDs are UUIDs in URL parameters.
)

// FixHandler exposes endpoints for listing fixes and changing their status.
//
// "Fixes" are remediation suggestions or patches generated for findings. The API
// supports per-fix accept/reject decisions as well as a bulk endpoint so the UI can
// apply actions efficiently.
type FixHandler struct {
	Service    *services.FixService
	FindingSvc *services.FindingService
	AIFixer    *services.AIFixGenerator
}

// NewFixHandler constructs a handler with its dependencies injected.
func NewFixHandler(s *services.FixService, findingSvc *services.FindingService, aiFixer *services.AIFixGenerator) *FixHandler {
	return &FixHandler{Service: s, FindingSvc: findingSvc, AIFixer: aiFixer}
}

// GetFixes lists all fixes associated with a scan.
//
// Fixes are returned separately from findings because they can have larger payloads
// (code snippets / explanations) and because not all workflows need them.
func (h *FixHandler) GetFixes(c fiber.Ctx) error {
	scanID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scan ID"})
	}

	fixes, err := h.Service.ListByScan(c.Context(), scanID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fixes)
}

// Accept marks a fix as accepted.
//
// Status changes are modeled as simple strings in the database for now; the service
// performs the update and reports not-found conditions.
func (h *FixHandler) Accept(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid fix ID"})
	}

	if err := h.Service.UpdateStatus(c.Context(), id, "accepted"); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "accepted"})
}

// Reject marks a fix as rejected.
func (h *FixHandler) Reject(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid fix ID"})
	}

	if err := h.Service.UpdateStatus(c.Context(), id, "rejected"); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "rejected"})
}

// Bulk applies accept/reject to a list of fix IDs.
//
// This endpoint exists to reduce round-trips for the UI when applying the same
// decision to many suggested fixes.
func (h *FixHandler) Bulk(c fiber.Ctx) error {
	var req models.BulkFixRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Action != "accept" && req.Action != "reject" {
		return c.Status(400).JSON(fiber.Map{"error": "action must be 'accept' or 'reject'"})
	}

	status := req.Action + "ed"
	for _, id := range req.FixIDs {
		if err := h.Service.UpdateStatus(c.Context(), id, status); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.JSON(fiber.Map{"updated": len(req.FixIDs)})
}

// AIFix generates an AI-powered fix for a specific finding.
// Invoked on-demand via the "Get AI Fix" button to control API costs.
func (h *FixHandler) AIFix(c fiber.Ctx) error {
	scanID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scan ID"})
	}

	findingID, err := uuid.Parse(c.Params("findingId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid finding ID"})
	}

	if h.AIFixer == nil || !h.AIFixer.CanFix(models.Finding{}) {
		return c.Status(501).JSON(fiber.Map{"error": "AI fixes not configured (ANTHROPIC_API_KEY missing)"})
	}

	// Fetch the finding
	result, err := h.FindingSvc.List(c.Context(), services.FindingFilter{
		ScanID: scanID,
		Page:   1,
		Limit:  1,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Find the specific finding
	var finding *models.Finding
	for _, f := range result.Findings {
		if f.ID == findingID {
			finding = &f
			break
		}
	}

	// If not found in first page, query more broadly
	if finding == nil {
		allResult, err := h.FindingSvc.List(c.Context(), services.FindingFilter{
			ScanID: scanID,
			Page:   1,
			Limit:  1000,
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		for _, f := range allResult.Findings {
			if f.ID == findingID {
				finding = &f
				break
			}
		}
	}

	if finding == nil {
		return c.Status(404).JSON(fiber.Map{"error": "finding not found"})
	}

	sourceCode := ""
	if finding.CodeSnippet != nil {
		sourceCode = *finding.CodeSnippet
	}

	fix, err := h.AIFixer.Generate(c.Context(), *finding, sourceCode)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	fix.ScanID = scanID
	if err := h.Service.Insert(c.Context(), fix); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(fix)
}
