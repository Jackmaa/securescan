package handlers

import (
	"strconv"

	"securescan/services"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type FindingHandler struct {
	Service *services.FindingService
}

func NewFindingHandler(s *services.FindingService) *FindingHandler {
	return &FindingHandler{Service: s}
}

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
