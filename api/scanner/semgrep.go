package scanner

import (
	"context"       // Enables cancellation/timeouts for the semgrep process.
	"encoding/json" // Semgrep emits JSON; we unmarshal only the fields we use.
	"os/exec"       // Run the semgrep CLI as an external process.

	"securescan/models" // Normalized Finding model for downstream storage/UI.

	"github.com/google/uuid" // Scan IDs are attached to every produced Finding.
)

// SemgrepAdapter integrates the Semgrep CLI.
//
// Semgrep is used as a broad “SAST baseline” because it supports many languages and
// has high-quality community rulepacks. We intentionally run a known ruleset rather
// than relying on project-local configs so results are consistent across repos.
type SemgrepAdapter struct{}

// Name identifies this tool in the database and UI.
func (s *SemgrepAdapter) Name() string { return "semgrep" }

// Semgrep supports most languages out of the box, so it's always applicable
func (s *SemgrepAdapter) IsApplicable(_ []string) bool { return true }

// Run executes Semgrep against the repository directory and returns raw JSON output.
//
// Notes on flags:
// - `--config p/owasp-top-ten`: uses a curated, security-focused ruleset.
// - `--json`: machine-readable output for parsing into normalized findings.
// - `--quiet`: reduces noise so stderr is mostly reserved for real failures.
func (s *SemgrepAdapter) Run(ctx context.Context, repoPath string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "semgrep",
		"--config", "p/owasp-top-ten",
		"--json",
		"--quiet",
		repoPath,
	)
	return cmd.Output()
}

// semgrepOutput and semgrepResult map the semgrep JSON schema.
// Only the fields we need are included — the raw JSON is preserved in RawOutput.
type semgrepOutput struct {
	Results []semgrepResult `json:"results"`
}

type semgrepResult struct {
	CheckID string `json:"check_id"`
	Path    string `json:"path"`
	Start   struct {
		Line int `json:"line"`
		Col  int `json:"col"`
	} `json:"start"`
	End struct {
		Line int `json:"line"`
		Col  int `json:"col"`
	} `json:"end"`
	Extra struct {
		Message  string `json:"message"`
		Severity string `json:"severity"`
		Metadata struct {
			Owasp    []string `json:"owasp"`
			CWE      []string `json:"cwe"`
			Category string   `json:"category"`
		} `json:"metadata"`
		Lines string `json:"lines"`
	} `json:"extra"`
}

func (s *SemgrepAdapter) Parse(scanID uuid.UUID, raw []byte) ([]models.Finding, error) {
	var output semgrepOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, err
	}

	// We preallocate based on semgrep’s result count to reduce allocations for large scans.
	findings := make([]models.Finding, 0, len(output.Results))
	for _, r := range output.Results {
		severity := normalizeSeverity(r.Extra.Severity)
		rawJSON := json.RawMessage(mustMarshal(r))

		f := models.Finding{
			ID:          uuid.New(),
			ScanID:      scanID,
			ToolName:    "semgrep",
			RuleID:      strPtr(r.CheckID),
			FilePath:    strPtr(r.Path),
			LineStart:   intPtr(r.Start.Line),
			LineEnd:     intPtr(r.End.Line),
			ColStart:    intPtr(r.Start.Col),
			ColEnd:      intPtr(r.End.Col),
			Message:     r.Extra.Message,
			Severity:    severity,
			RawOutput:   &rawJSON,
			CodeSnippet: strPtr(r.Extra.Lines),
		}

		// Semgrep often provides OWASP directly — this is our highest-confidence mapping
		if len(r.Extra.Metadata.Owasp) > 0 {
			f.OwaspCategory = strPtr(extractOwaspCategory(r.Extra.Metadata.Owasp[0]))
		}
		if len(r.Extra.Metadata.CWE) > 0 {
			f.CweID = strPtr(r.Extra.Metadata.CWE[0])
		}

		findings = append(findings, f)
	}

	return findings, nil
}

// normalizeSeverity converts Semgrep’s severity labels into our normalized scale.
//
// We normalize early so sorting, scoring, and UI treatment are consistent across tools.
func normalizeSeverity(s string) string {
	switch s {
	case "ERROR":
		return "high"
	case "WARNING":
		return "medium"
	case "INFO":
		return "info"
	default:
		return "medium"
	}
}

// extractOwaspCategory extracts "A01" from strings like "A01:2021 - Broken Access Control"
func extractOwaspCategory(s string) string {
	if len(s) >= 3 && s[0] == 'A' {
		return s[:3]
	}
	return s
}

// strPtr stores empty strings as NULLs in the database by returning nil when empty.
//
// This helps keep the schema semantically clean (unknown vs empty) and avoids noise
// in API responses that use `omitempty`.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// intPtr stores 0 as NULL by returning nil when i==0, matching “unknown” semantics.
func intPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// mustMarshal is a convenience helper to keep raw tool output embedded in findings.
//
// We intentionally ignore marshal errors here because v is derived from already
// unmarshaled JSON structs; if marshal fails, we prefer returning a best-effort
// empty/partial raw payload rather than failing the entire scan.
func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
