package services

import (
	"context"
	"fmt"
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
//
// Deduplication: fixes that produce identical remediation (same file_path + fixed_code)
// are merged into a single record. This avoids e.g. 30 identical "npm audit fix"
// suggestions when npm audit reports 30 vulnerable packages. The merged fix keeps
// the first finding_id as representative and aggregates descriptions.
func GenerateFixes(ctx context.Context, fixSvc *FixService, findings []models.Finding, scanID uuid.UUID, repoPath string) {
	generators := []FixGenerator{
		&SQLInjectionFixer{},
		&XSSFixer{},
		&VulnerableDepFixer{},
		&ExposedSecretFixer{},
		&PlaintextPasswordFixer{},
	}

	// Generate all candidate fixes first, then deduplicate before inserting.
	var candidates []*models.Fix
	for _, finding := range findings {
		for _, gen := range generators {
			if !gen.CanFix(finding) {
				continue
			}

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
			candidates = append(candidates, fix)
			break // one template fix per finding
		}
	}

	// Deduplicate: group by (file_path, fixed_code). Fixes with unique code
	// (e.g. SQL injection at a specific line) naturally stay 1:1.
	deduplicated := deduplicateFixes(candidates)

	for _, fix := range deduplicated {
		if err := fixSvc.Insert(ctx, fix); err != nil {
			log.Printf("fix insert failed: %v", err)
		}
	}
}

// deduplicateFixes merges fixes that target the same file with identical remediation code.
//
// Why this matters: a single scan can produce dozens of findings that all resolve to the
// same action (e.g. "npm audit fix --force" on package.json). Storing and displaying them
// individually clutters the review UI and makes the git-apply step attempt the same
// replacement repeatedly.
func deduplicateFixes(fixes []*models.Fix) []*models.Fix {
	type groupKey struct {
		filePath  string
		fixedCode string
	}

	seen := make(map[groupKey]*models.Fix)
	counts := make(map[groupKey]int)
	var order []groupKey // preserve insertion order

	for _, fix := range fixes {
		code := ""
		if fix.FixedCode != nil {
			code = *fix.FixedCode
		}
		key := groupKey{filePath: fix.FilePath, fixedCode: code}

		if existing, ok := seen[key]; ok {
			// Merge: bump count, keep the first fix's ID and finding_id
			counts[key]++
			// Accumulate descriptions for the merged explanation
			if existing.Explanation != nil && fix.Explanation != nil {
				// Keep the first explanation — they're identical for same-template fixes
			}
		} else {
			seen[key] = fix
			counts[key] = 1
			order = append(order, key)
		}
	}

	var result []*models.Fix
	for _, key := range order {
		fix := seen[key]
		count := counts[key]
		if count > 1 {
			fix.Description = fmt.Sprintf("%s (%d findings)", fix.Description, count)
		}
		result = append(result, fix)
	}
	return result
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
