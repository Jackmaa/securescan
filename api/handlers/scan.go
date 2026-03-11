package handlers

import (
	"securescan/services"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type ScanHandler struct {
	ScanService    *services.ScanService
	ProjectService *services.ProjectService
}

func NewScanHandler(ss *services.ScanService, ps *services.ProjectService) *ScanHandler {
	return &ScanHandler{ScanService: ss, ProjectService: ps}
}

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
