package handlers

import (
	"securescan/models"
	"securescan/services"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type FixHandler struct {
	Service *services.FixService
}

func NewFixHandler(s *services.FixService) *FixHandler {
	return &FixHandler{Service: s}
}

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
