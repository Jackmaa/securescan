package services

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"securescan/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportData aggregates everything needed to render the HTML report template.
type ReportData struct {
	ProjectName string
	ScanDate    string
	Score       int
	Grade       string
	TotalFindings int
	BySeverity  map[string]int
	ByOwasp     map[string]int
	ByTool      map[string]int
	Findings    []models.Finding
	Fixes       []models.Fix
}

type ReportService struct {
	DB         *pgxpool.Pool
	ScanSvc    *ScanService
	FindingSvc *FindingService
	FixSvc     *FixService
	OutputDir  string
}

func NewReportService(db *pgxpool.Pool, scanSvc *ScanService, findingSvc *FindingService, fixSvc *FixService, outputDir string) *ReportService {
	return &ReportService{
		DB:         db,
		ScanSvc:    scanSvc,
		FindingSvc: findingSvc,
		FixSvc:     fixSvc,
		OutputDir:  outputDir,
	}
}

// GenerateHTML produces an HTML report and persists a report record.
func (s *ReportService) GenerateHTML(ctx context.Context, scanID uuid.UUID, projectName string) (*models.Report, error) {
	data, err := s.gatherData(ctx, scanID, projectName)
	if err != nil {
		return nil, err
	}

	funcMap := template.FuncMap{
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"deref_int": func(i *int) int {
			if i == nil {
				return 0
			}
			return *i
		},
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(reportTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	// Write to file
	os.MkdirAll(s.OutputDir, 0755)
	filename := fmt.Sprintf("report-%s.html", scanID)
	outPath := filepath.Join(s.OutputDir, filename)
	if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("write report: %w", err)
	}

	// Persist report record
	report := &models.Report{
		ID:       uuid.New(),
		ScanID:   scanID,
		Format:   "html",
		FilePath: outPath,
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO reports (id, scan_id, format, file_path)
		VALUES ($1, $2, $3, $4)
	`, report.ID, report.ScanID, report.Format, report.FilePath)
	if err != nil {
		return nil, fmt.Errorf("persist report record: %w", err)
	}

	return report, nil
}

func (s *ReportService) gatherData(ctx context.Context, scanID uuid.UUID, projectName string) (*ReportData, error) {
	stats, err := s.ScanSvc.GetStats(ctx, scanID)
	if err != nil {
		return nil, err
	}

	result, err := s.FindingSvc.List(ctx, FindingFilter{
		ScanID: scanID,
		Sort:   "severity",
		Page:   1,
		Limit:  500,
	})
	if err != nil {
		return nil, err
	}

	fixes, err := s.FixSvc.ListByScan(ctx, scanID)
	if err != nil {
		return nil, err
	}

	score := 0
	grade := "?"
	if stats.Score != nil {
		score = *stats.Score
	}
	if stats.Grade != nil {
		grade = *stats.Grade
	}

	return &ReportData{
		ProjectName:   projectName,
		ScanDate:      time.Now().Format("January 2, 2006"),
		Score:         score,
		Grade:         grade,
		TotalFindings: stats.TotalFindings,
		BySeverity:    stats.BySeverity,
		ByOwasp:       stats.ByOwasp,
		ByTool:        stats.ByTool,
		Findings:      result.Findings,
		Fixes:         fixes,
	}, nil
}

// GetLatestReport returns the most recent report for a scan.
func (s *ReportService) GetLatestReport(ctx context.Context, scanID uuid.UUID) (*models.Report, error) {
	report := &models.Report{}
	err := s.DB.QueryRow(ctx, `
		SELECT id, scan_id, format, file_path, created_at
		FROM reports WHERE scan_id = $1
		ORDER BY created_at DESC LIMIT 1
	`, scanID).Scan(&report.ID, &report.ScanID, &report.Format, &report.FilePath, &report.CreatedAt)
	if err != nil {
		return nil, err
	}
	return report, nil
}

// Inline HTML template with all CSS inlined for PDF rendering compatibility.
var reportTemplate = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>SecureScan Report - {{.ProjectName}}</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; color: #1a1a2e; padding: 40px; max-width: 900px; margin: 0 auto; }
  h1 { font-size: 28px; margin-bottom: 4px; }
  h2 { font-size: 20px; margin: 32px 0 16px; padding-bottom: 8px; border-bottom: 2px solid #e0e0e0; }
  h3 { font-size: 16px; margin: 16px 0 8px; }
  .header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 32px; padding-bottom: 20px; border-bottom: 3px solid #6366f1; }
  .score-box { text-align: center; padding: 16px 24px; border: 2px solid #e0e0e0; border-radius: 12px; }
  .grade { font-size: 48px; font-weight: bold; }
  .grade-A { color: #22c55e; } .grade-B { color: #3b82f6; } .grade-C { color: #eab308; }
  .grade-D { color: #f97316; } .grade-F { color: #ef4444; }
  .score-num { font-size: 24px; color: #666; }
  .stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin-bottom: 24px; }
  .stat-card { background: #f8f9fa; border-radius: 8px; padding: 16px; text-align: center; }
  .stat-value { font-size: 24px; font-weight: bold; color: #6366f1; }
  .stat-label { font-size: 12px; color: #666; margin-top: 4px; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 24px; font-size: 13px; }
  th { background: #f1f5f9; text-align: left; padding: 10px 12px; font-weight: 600; }
  td { padding: 8px 12px; border-bottom: 1px solid #e2e8f0; vertical-align: top; }
  .sev-critical { color: #ef4444; font-weight: bold; }
  .sev-high { color: #f97316; font-weight: bold; }
  .sev-medium { color: #eab308; }
  .sev-low { color: #3b82f6; }
  .sev-info { color: #6b7280; }
  .footer { margin-top: 40px; padding-top: 16px; border-top: 1px solid #e0e0e0; font-size: 12px; color: #999; text-align: center; }
  .owasp-tag { background: #eef2ff; color: #6366f1; padding: 2px 6px; border-radius: 4px; font-size: 11px; font-family: monospace; }
</style>
</head>
<body>

<div class="header">
  <div>
    <h1>SecureScan Report</h1>
    <p style="color:#666">{{.ProjectName}} &mdash; {{.ScanDate}}</p>
  </div>
  <div class="score-box">
    <div class="grade grade-{{.Grade}}">{{.Grade}}</div>
    <div class="score-num">{{.Score}}/100</div>
  </div>
</div>

<div class="stats">
  <div class="stat-card">
    <div class="stat-value">{{.TotalFindings}}</div>
    <div class="stat-label">Total Findings</div>
  </div>
  <div class="stat-card">
    <div class="stat-value">{{len .Fixes}}</div>
    <div class="stat-label">Fixes Generated</div>
  </div>
  <div class="stat-card">
    <div class="stat-value">{{len .ByOwasp}}</div>
    <div class="stat-label">OWASP Categories</div>
  </div>
</div>

<h2>Severity Distribution</h2>
<table>
  <tr><th>Severity</th><th>Count</th></tr>
  {{range $sev, $count := .BySeverity}}
  <tr><td class="sev-{{$sev}}">{{$sev}}</td><td>{{$count}}</td></tr>
  {{end}}
</table>

<h2>OWASP Top 10 Coverage</h2>
<table>
  <tr><th>Category</th><th>Findings</th></tr>
  {{range $cat, $count := .ByOwasp}}
  <tr><td><span class="owasp-tag">{{$cat}}</span></td><td>{{$count}}</td></tr>
  {{end}}
</table>

<h2>Findings</h2>
<table>
  <tr><th>Severity</th><th>Tool</th><th>Message</th><th>File</th><th>OWASP</th></tr>
  {{range .Findings}}
  <tr>
    <td class="sev-{{.Severity}}">{{.Severity}}</td>
    <td>{{.ToolName}}</td>
    <td>{{.Message}}</td>
    <td style="font-family:monospace;font-size:11px">{{if .FilePath}}{{deref .FilePath}}{{if .LineStart}}:{{deref_int .LineStart}}{{end}}{{end}}</td>
    <td>{{if .OwaspCategory}}<span class="owasp-tag">{{deref .OwaspCategory}}</span>{{end}}</td>
  </tr>
  {{end}}
</table>

{{if .Fixes}}
<h2>Fix Suggestions</h2>
<table>
  <tr><th>Type</th><th>Description</th><th>File</th><th>Status</th></tr>
  {{range .Fixes}}
  <tr>
    <td>{{.FixType}}</td>
    <td>{{.Description}}</td>
    <td style="font-family:monospace;font-size:11px">{{.FilePath}}</td>
    <td>{{.Status}}</td>
  </tr>
  {{end}}
</table>
{{end}}

<div class="footer">
  Generated by SecureScan &mdash; {{.ScanDate}}
</div>

</body>
</html>`

