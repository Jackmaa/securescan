package scanner

import (
	"context"       // Allows cancellation/timeouts for npm processes.
	"encoding/json" // npm audit emits JSON; we parse only the fields we need.
	"os"            // Used to detect/generate lockfiles.
	"os/exec"       // Run npm as an external process.
	"path/filepath" // Build OS-correct path to package-lock.json.
	"slices"        // Convenience helpers for language applicability checks.

	"securescan/models" // Normalized Finding model used downstream.

	"github.com/google/uuid" // Scan IDs are assigned to each produced Finding.
)

// NpmAuditAdapter integrates `npm audit` to detect vulnerable dependencies.
//
// We only run this adapter for JS/TS projects because it requires a Node dependency
// graph. Findings map naturally to OWASP A06 (Vulnerable and Outdated Components).
type NpmAuditAdapter struct{}

// Name identifies this tool in the database/UI.
func (n *NpmAuditAdapter) Name() string { return "npm_audit" }

// IsApplicable returns true when the project uses JavaScript or TypeScript.
func (n *NpmAuditAdapter) IsApplicable(languages []string) bool {
	return slices.Contains(languages, "JavaScript") || slices.Contains(languages, "TypeScript")
}

// Run executes `npm audit --json` in the repo directory and returns its output.
//
// Important behavior:
// - `npm audit` returns exit code 1 when vulnerabilities are found; that is *expected*.
// - We capture output regardless of exit code to parse and persist findings.
//
// Lockfile handling:
//   - `npm audit` is most reliable with a `package-lock.json`.
//   - If missing, we generate a lockfile without installing packages (`--package-lock-only`)
//     and without running lifecycle scripts (`--ignore-scripts`) to reduce risk.
func (n *NpmAuditAdapter) Run(ctx context.Context, repoPath string) ([]byte, error) {
	// npm audit requires a package-lock.json; generate one if missing
	lockPath := filepath.Join(repoPath, "package-lock.json")
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		install := exec.CommandContext(ctx, "npm", "install", "--package-lock-only", "--ignore-scripts")
		install.Dir = repoPath
		if err := install.Run(); err != nil {
			return nil, err
		}
	}

	cmd := exec.CommandContext(ctx, "npm", "audit", "--json")
	cmd.Dir = repoPath
	// npm audit returns exit code 1 when vulnerabilities are found, which is expected.
	// We capture the output regardless of exit code.
	output, _ := cmd.Output()
	return output, nil
}

// npmAuditOutput maps the top-level npm audit JSON response.
type npmAuditOutput struct {
	Vulnerabilities map[string]npmVulnerability `json:"vulnerabilities"`
}

type npmVulnerability struct {
	Name     string            `json:"name"`
	Severity string            `json:"severity"`
	Via      []json.RawMessage `json:"via"`
	// FixAvailable is polymorphic in npm’s schema (bool/object/etc.), so we keep it untyped.
	FixAvailable interface{} `json:"fixAvailable"`
	Range        string      `json:"range"`
}

// npmAdvisory is one entry in the "via" array when it's an object (not a string).
type npmAdvisory struct {
	Source   int      `json:"source"`
	Name     string   `json:"name"`
	Title    string   `json:"title"`
	URL      string   `json:"url"`
	Severity string   `json:"severity"`
	CWE      []string `json:"cwe"`
	Range    string   `json:"range"`
}

// Parse converts the npm audit JSON output into normalized findings.
//
// npm’s "via" array includes both:
// - strings (transitive references / shorthand), and
// - advisory objects (rich data we can present to users).
// We only create findings from advisory objects since they include title/URL/CWE hints.
func (n *NpmAuditAdapter) Parse(scanID uuid.UUID, raw []byte) ([]models.Finding, error) {
	var output npmAuditOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, err
	}

	var findings []models.Finding
	for pkgName, vuln := range output.Vulnerabilities {
		severity := mapNpmSeverity(vuln.Severity)

		// Each "via" entry can be either a string (transitive) or an advisory object
		for _, viaRaw := range vuln.Via {
			var advisory npmAdvisory
			if err := json.Unmarshal(viaRaw, &advisory); err != nil {
				continue // skip string entries (transitive refs)
			}
			if advisory.Title == "" {
				continue
			}

			rawJSON := json.RawMessage(mustMarshal(map[string]any{
				"package":       pkgName,
				"advisory":      advisory,
				"fix_available": vuln.FixAvailable,
			}))

			f := models.Finding{
				ID:       uuid.New(),
				ScanID:   scanID,
				ToolName: "npm_audit",
				RuleID:   strPtr(advisory.URL),
				FilePath: strPtr("package.json"),
				Message:  pkgName + ": " + advisory.Title,
				Severity: severity,
				// Vulnerable dependencies → A06 (Vulnerable and Outdated Components)
				OwaspCategory: strPtr("A06"),
				RawOutput:     &rawJSON,
			}

			if len(advisory.CWE) > 0 {
				f.CweID = strPtr(advisory.CWE[0])
			}

			findings = append(findings, f)
		}
	}

	return findings, nil
}

// mapNpmSeverity normalizes npm’s labels into our severity scale.
func mapNpmSeverity(s string) string {
	switch s {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "moderate":
		return "medium"
	case "low":
		return "low"
	default:
		return "info"
	}
}
