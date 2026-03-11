package handlers

import (
	"securescan/models"   // Request/response shapes shared across handlers and services.
	"securescan/services" // Business logic and database interaction for projects.

	"github.com/gofiber/fiber/v3" // HTTP framework used for binding and responses.
	"github.com/google/uuid"      // Project IDs are UUIDs; validate/parse from URL parameters.
)

// ProjectHandler exposes HTTP endpoints for creating and fetching projects.
//
// The handler layer is responsible for:
// - Validating user input (shape + basic constraints)
// - Translating HTTP/JSON into service calls
// - Mapping domain/service errors into appropriate HTTP statuses
//
// It deliberately does not do filesystem/database work itself; that stays in `services`.
type ProjectHandler struct {
	Service *services.ProjectService
}

// NewProjectHandler constructs a ProjectHandler with its dependencies injected.
//
// Keeping construction separate makes it easier to test handlers by swapping in a
// real or fake ProjectService.
func NewProjectHandler(s *services.ProjectService) *ProjectHandler {
	return &ProjectHandler{Service: s}
}

// Create registers a new project and prepares a local workspace for scanning.
//
// Why we validate here (instead of deeper in the service):
// - It provides fast feedback to API clients with clean 400 responses.
// - It keeps service logic focused on business rules and side effects.
//
// Supported source types:
// - `git`: clone a repository into the scan workspace
// - `zip`: reserved for later (upload/import pipeline not implemented yet)
func (h *ProjectHandler) Create(c fiber.Ctx) error {
	var req models.CreateProjectRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name is required"})
	}
	if req.SourceType != "git" && req.SourceType != "zip" {
		return c.Status(400).JSON(fiber.Map{"error": "source_type must be 'git' or 'zip'"})
	}
	if req.SourceType == "git" && req.SourceURL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "source_url is required for git projects"})
	}

	project, err := h.Service.Create(c.Context(), req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(project)
}

// GetByID returns a previously created project record.
//
// We return 404 if lookup fails so clients can treat missing projects as a normal
// condition (e.g., stale URLs) rather than a server error.
func (h *ProjectHandler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid project ID"})
	}

	project, err := h.Service.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "project not found"})
	}

	return c.JSON(project)
}
