package services

import (
	"context"
	"encoding/json"
	"strings"

	"securescan/models"

	"github.com/google/uuid"
)

// VulnerableDepFixer generates fixes for vulnerable dependency findings.
//
// Detection: tool = npm_audit and fixAvailable is truthy.
// Fix strategy: suggest updating the version in package.json.
type VulnerableDepFixer struct{}

func (f *VulnerableDepFixer) CanFix(finding models.Finding) bool {
	if finding.ToolName != "npm_audit" {
		return false
	}
	// Check if fixAvailable is present in raw output
	if finding.RawOutput == nil {
		return false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(*finding.RawOutput, &raw); err != nil {
		return false
	}
	_, hasFixAvailable := raw["fix_available"]
	return hasFixAvailable
}

func (f *VulnerableDepFixer) Generate(_ context.Context, finding models.Finding, _ string) (*models.Fix, error) {
	explanation := "One or more dependencies have known vulnerabilities with fixes available. " +
		"Run `npm audit fix` to attempt automatic updates, or `npm audit fix --force` " +
		"if breaking changes are acceptable."

	// The fixed_code is intentionally identical for ALL npm audit findings so the
	// deduplication in GenerateFixes() collapses them into a single fix record.
	// The per-package details are visible in the findings table.
	fixedCode := "npm audit fix --force"

	// Extract package name for the description (used in the merged "(N findings)" label)
	pkgName := finding.Message
	if idx := strings.Index(pkgName, ":"); idx > 0 {
		pkgName = pkgName[:idx]
	}

	fix := &models.Fix{
		ID:          uuid.New(),
		FindingID:   finding.ID,
		FixType:     "template",
		Description: "Update vulnerable dependencies",
		Explanation: &explanation,
		FixedCode:   &fixedCode,
		FilePath:    "package.json",
		Status:      "pending",
	}
	return fix, nil
}
