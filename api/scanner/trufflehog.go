package scanner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os/exec"

	"securescan/models"

	"github.com/google/uuid"
)

type TruffleHogAdapter struct{}

func (t *TruffleHogAdapter) Name() string { return "trufflehog" }

// Secrets can exist in any language, so always run trufflehog
func (t *TruffleHogAdapter) IsApplicable(_ []string) bool { return true }

func (t *TruffleHogAdapter) Run(ctx context.Context, repoPath string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "trufflehog",
		"filesystem",
		repoPath,
		"--json",
		"--no-update",
	)
	return cmd.Output()
}

// trufflehogResult maps each line-delimited JSON object from trufflehog output.
type trufflehogResult struct {
	DetectorName string `json:"DetectorName"`
	DecoderName  string `json:"DecoderName"`
	Verified     bool   `json:"Verified"`
	Raw          string `json:"Raw"`
	SourceMetadata struct {
		Data struct {
			Filesystem struct {
				File string `json:"file"`
				Line int    `json:"line"`
			} `json:"Filesystem"`
		} `json:"Data"`
	} `json:"SourceMetadata"`
	ExtraData map[string]string `json:"ExtraData"`
}

func (t *TruffleHogAdapter) Parse(scanID uuid.UUID, raw []byte) ([]models.Finding, error) {
	var findings []models.Finding
	scanner := bufio.NewScanner(bytes.NewReader(raw))

	// TruffleHog outputs one JSON object per line (NDJSON)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var result trufflehogResult
		if err := json.Unmarshal(line, &result); err != nil {
			continue
		}

		severity := "high"
		if result.Verified {
			severity = "critical"
		}

		rawJSON := json.RawMessage(line)
		filePath := result.SourceMetadata.Data.Filesystem.File
		lineNum := result.SourceMetadata.Data.Filesystem.Line

		f := models.Finding{
			ID:       uuid.New(),
			ScanID:   scanID,
			ToolName: "trufflehog",
			RuleID:   strPtr(result.DetectorName),
			FilePath: strPtr(filePath),
			LineStart: intPtr(lineNum),
			Message:  "Exposed secret: " + result.DetectorName,
			Severity: severity,
			// Secrets always map to A07 (Identification and Authentication Failures)
			// and A04 (Insecure Design — hardcoded credentials)
			OwaspCategory: strPtr("A07"),
			CweID:         strPtr("CWE-798"),
			RawOutput:     &rawJSON,
		}

		findings = append(findings, f)
	}

	return findings, nil
}
