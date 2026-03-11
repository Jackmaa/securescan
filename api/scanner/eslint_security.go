package scanner

import (
	"context"
	"encoding/json"
	"os/exec"
	"slices"
	"strings"

	"securescan/models"

	"github.com/google/uuid"
)

type ESLintSecurityAdapter struct{}

func (e *ESLintSecurityAdapter) Name() string { return "eslint_security" }

func (e *ESLintSecurityAdapter) IsApplicable(languages []string) bool {
	return slices.Contains(languages, "JavaScript") || slices.Contains(languages, "TypeScript")
}

func (e *ESLintSecurityAdapter) Run(ctx context.Context, repoPath string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "eslint",
		"--format", "json",
		"--no-eslintrc",
		"--plugin", "security",
		"--rule", `{"security/detect-eval-with-expression":"warn","security/detect-non-literal-fs-filename":"warn","security/detect-non-literal-regexp":"warn","security/detect-possible-timing-attacks":"warn","security/detect-unsafe-regex":"warn","security/detect-buffer-noassert":"warn","security/detect-child-process":"warn","security/detect-no-csrf-before-method-override":"warn","security/detect-object-injection":"warn","security/detect-pseudoRandomBytes":"warn"}`,
		"--ext", ".js,.ts,.jsx,.tsx",
		repoPath,
	)
	// ESLint returns exit code 1 when there are warnings/errors — expected
	output, _ := cmd.Output()
	return output, nil
}

// eslintFile maps one file entry in ESLint's JSON output array.
type eslintFile struct {
	FilePath string          `json:"filePath"`
	Messages []eslintMessage `json:"messages"`
}

type eslintMessage struct {
	RuleID   string `json:"ruleId"`
	Severity int    `json:"severity"` // 1=warn, 2=error
	Message  string `json:"message"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	EndLine  int    `json:"endLine"`
	EndColumn int   `json:"endColumn"`
	Source   string `json:"source"`
}

// eslintOwaspMap connects ESLint security rule names to OWASP categories.
// This is the heuristic fallback layer for tools without CWE metadata.
var eslintOwaspMap = map[string]string{
	"security/detect-eval-with-expression":              "A03",
	"security/detect-non-literal-fs-filename":           "A01",
	"security/detect-non-literal-regexp":                "A05",
	"security/detect-possible-timing-attacks":           "A02",
	"security/detect-unsafe-regex":                      "A05",
	"security/detect-buffer-noassert":                   "A05",
	"security/detect-child-process":                     "A03",
	"security/detect-no-csrf-before-method-override":    "A01",
	"security/detect-object-injection":                  "A03",
	"security/detect-pseudoRandomBytes":                 "A02",
}

func (e *ESLintSecurityAdapter) Parse(scanID uuid.UUID, raw []byte) ([]models.Finding, error) {
	var files []eslintFile
	if err := json.Unmarshal(raw, &files); err != nil {
		return nil, err
	}

	var findings []models.Finding
	for _, file := range files {
		for _, msg := range file.Messages {
			if msg.RuleID == "" {
				continue
			}

			severity := "medium"
			if msg.Severity == 2 {
				severity = "high"
			}

			rawJSON := json.RawMessage(mustMarshal(msg))
			f := models.Finding{
				ID:          uuid.New(),
				ScanID:      scanID,
				ToolName:    "eslint_security",
				RuleID:      strPtr(msg.RuleID),
				FilePath:    strPtr(file.FilePath),
				LineStart:   intPtr(msg.Line),
				LineEnd:     intPtr(msg.EndLine),
				ColStart:    intPtr(msg.Column),
				ColEnd:      intPtr(msg.EndColumn),
				Message:     msg.Message,
				Severity:    severity,
				RawOutput:   &rawJSON,
				CodeSnippet: strPtr(msg.Source),
			}

			if owasp, ok := eslintOwaspMap[msg.RuleID]; ok {
				f.OwaspCategory = strPtr(owasp)
			}

			// Derive CWE from rule name as a best-effort heuristic
			if cwe := eslintRuleToCWE(msg.RuleID); cwe != "" {
				f.CweID = strPtr(cwe)
			}

			findings = append(findings, f)
		}
	}

	return findings, nil
}

func eslintRuleToCWE(ruleID string) string {
	cweMap := map[string]string{
		"detect-eval-with-expression": "CWE-95",
		"detect-unsafe-regex":         "CWE-1333",
		"detect-child-process":        "CWE-78",
		"detect-object-injection":     "CWE-94",
		"detect-pseudoRandomBytes":    "CWE-338",
	}
	for pattern, cwe := range cweMap {
		if strings.Contains(ruleID, pattern) {
			return cwe
		}
	}
	return ""
}
