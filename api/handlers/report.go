package handlers

import (
	"securescan/services"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type ReportHandler struct {
	ReportSvc  *services.ReportService
	ScanSvc    *services.ScanService
	ProjectSvc *services.ProjectService
}

func NewReportHandler(reportSvc *services.ReportService, scanSvc *services.ScanService, projectSvc *services.ProjectService) *ReportHandler {
	return &ReportHandler{ReportSvc: reportSvc, ScanSvc: scanSvc, ProjectSvc: projectSvc}
}

// Generate creates an HTML report for a scan.
func (h *ReportHandler) Generate(c fiber.Ctx) error {
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

	report, err := h.ReportSvc.GenerateHTML(c.Context(), scanID, project.Name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(report)
}

// Download serves the latest report file for a scan.
func (h *ReportHandler) Download(c fiber.Ctx) error {
	scanID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scan ID"})
	}

	report, err := h.ReportSvc.GetLatestReport(c.Context(), scanID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "no report found"})
	}

	return c.Download(report.FilePath)
}
