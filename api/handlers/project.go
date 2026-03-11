package handlers

import (
	"securescan/models"
	"securescan/services"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type ProjectHandler struct {
	Service *services.ProjectService
}

func NewProjectHandler(s *services.ProjectService) *ProjectHandler {
	return &ProjectHandler{Service: s}
}

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
