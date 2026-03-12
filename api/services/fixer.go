package services

import (
	"context"
	"log"
	"os"
	"strings"

	"securescan/models"

	"github.com/google/uuid"
)

// FixGenerator is the interface shared by template-based and AI-based fix generators.
//
// Both strategies implement the same contract so the fix engine can try template first
// (instant, no API cost) and offer AI as a fallback per finding.
type FixGenerator interface {
	CanFix(finding models.Finding) bool
	Generate(ctx context.Context, finding models.Finding, sourceCode string) (*models.Fix, error)
}

// GenerateFixes runs all registered fix generators against the scan's findings.
//
// Strategy: try template generators first (fast, deterministic). Each finding gets
// at most one template fix. AI fixes are generated on-demand via a separate endpoint.
func GenerateFixes(ctx context.Context, fixSvc *FixService, findings []models.Finding, scanID uuid.UUID, repoPath string) {
	generators := []FixGenerator{
		&SQLInjectionFixer{},
		&XSSFixer{},
		&VulnerableDepFixer{},
		&ExposedSecretFixer{},
		&PlaintextPasswordFixer{},
	}

	for _, finding := range findings {
		for _, gen := range generators {
			if !gen.CanFix(finding) {
				continue
			}

			// Read source code around the finding for context
			sourceCode := readSourceContext(finding, repoPath)

			fix, err := gen.Generate(ctx, finding, sourceCode)
			if err != nil {
				log.Printf("fix generation failed for %s: %v", finding.ID, err)
				continue
			}
			if fix == nil {
				continue
			}

			fix.ScanID = scanID
			if err := fixSvc.Insert(ctx, fix); err != nil {
				log.Printf("fix insert failed: %v", err)
			}
			break // one template fix per finding
		}
	}
}

// readSourceContext reads source lines around a finding's location.
// Returns empty string if the file can't be read.
func readSourceContext(f models.Finding, repoPath string) string {
	if f.FilePath == nil {
		return ""
	}

	filePath := *f.FilePath
	// Handle both absolute and relative paths
	if !strings.HasPrefix(filePath, "/") {
		filePath = repoPath + "/" + filePath
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")

	// Extract a window around the finding
	start := 0
	end := len(lines)
	if f.LineStart != nil {
		start = *f.LineStart - 3
		if start < 0 {
			start = 0
		}
	}
	if f.LineEnd != nil {
		end = *f.LineEnd + 3
		if end > len(lines) {
			end = len(lines)
		}
	} else if f.LineStart != nil {
		end = *f.LineStart + 3
		if end > len(lines) {
			end = len(lines)
		}
	}

	return strings.Join(lines[start:end], "\n")
}
