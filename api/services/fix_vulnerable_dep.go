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
	explanation := "This dependency has a known vulnerability with a fix available. " +
		"Update to the patched version to resolve the security issue. " +
		"Run `npm audit fix` to attempt automatic updates, or manually update the version in package.json."

	// Extract package name from message ("packageName: advisory title")
	pkgName := finding.Message
	if idx := strings.Index(pkgName, ":"); idx > 0 {
		pkgName = pkgName[:idx]
	}

	fixedCode := "# Option 1: Automatic fix\n" +
		"npm audit fix\n\n" +
		"# Option 2: Force update (may include breaking changes)\n" +
		"npm audit fix --force\n\n" +
		"# Option 3: Manual update in package.json\n" +
		"# Update \"" + pkgName + "\" to the latest patched version"

	filePath := "package.json"

	fix := &models.Fix{
		ID:          uuid.New(),
		FindingID:   finding.ID,
		FixType:     "template",
		Description: "Update " + pkgName + " to patched version",
		Explanation: &explanation,
		FixedCode:   &fixedCode,
		FilePath:    filePath,
		Status:      "pending",
	}
	return fix, nil
}
