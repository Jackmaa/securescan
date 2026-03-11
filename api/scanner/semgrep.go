package scanner

import (
	"context"
	"encoding/json"
	"os/exec"

	"securescan/models"

	"github.com/google/uuid"
)

type SemgrepAdapter struct{}

func (s *SemgrepAdapter) Name() string { return "semgrep" }

// Semgrep supports most languages out of the box, so it's always applicable
func (s *SemgrepAdapter) IsApplicable(_ []string) bool { return true }

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

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
