package handlers

import (
	"securescan/services"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type GitHandler struct {
	GitSvc     *services.GitIntegrationService
	ScanSvc    *services.ScanService
	ProjectSvc *services.ProjectService
}

func NewGitHandler(gitSvc *services.GitIntegrationService, scanSvc *services.ScanService, projectSvc *services.ProjectService) *GitHandler {
	return &GitHandler{GitSvc: gitSvc, ScanSvc: scanSvc, ProjectSvc: projectSvc}
}

// ApplyFixes creates a branch with accepted fixes, commits, and pushes.
func (h *GitHandler) ApplyFixes(c fiber.Ctx) error {
	scanID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scan ID"})
	}

	scan, err := h.ScanSvc.GetByID(c.Context(), scanID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "scan not found"})
	}

	project, err := h.ProjectSvc.GetByID(c.Context(), scan.ProjectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "project not found"})
	}

	branch, err := h.GitSvc.ApplyFixes(c.Context(), scanID, project.LocalPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"branch":  branch,
		"message": "Fixes applied and pushed to branch " + branch,
	})
}
