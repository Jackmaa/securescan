package scanner

import (
	"bufio"         // TruffleHog emits NDJSON; scanner reads line-by-line.
	"bytes"         // Wrap raw output bytes in an io.Reader for scanning.
	"context"       // Enables cancellation/timeouts for the trufflehog process.
	"encoding/json" // Each NDJSON line is a JSON object.
	"os/exec"       // Run the trufflehog CLI as an external process.

	"securescan/models" // Normalized Finding model used throughout the system.

	"github.com/google/uuid" // Scan IDs are assigned to each produced Finding.
)

// TruffleHogAdapter integrates the TruffleHog CLI for secret detection.
//
// Secrets scanning is intentionally language-agnostic: credentials and tokens can
// appear anywhere (configs, docs, source). TruffleHog provides strong detectors for
// common secret formats and can flag verified secrets.
type TruffleHogAdapter struct{}

// Name identifies this tool in the database/UI.
func (t *TruffleHogAdapter) Name() string { return "trufflehog" }

// Secrets can exist in any language, so always run trufflehog
func (t *TruffleHogAdapter) IsApplicable(_ []string) bool { return true }

// Run executes trufflehog against the repository path and returns raw NDJSON output.
//
// Flags:
// - `filesystem`: scans local files rather than git history.
// - `--json`: line-delimited JSON suitable for streaming parsing.
// - `--no-update`: avoids network/tool self-update during scans for determinism.
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
	DetectorName   string `json:"DetectorName"`
	DecoderName    string `json:"DecoderName"`
	Verified       bool   `json:"Verified"`
	Raw            string `json:"Raw"`
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

// Parse converts TruffleHog NDJSON output into normalized findings.
//
// Parsing is intentionally tolerant: a single malformed line is skipped rather than
// failing the entire scan, because tool output can include noise (partial writes,
// unexpected detectors, etc.) and we prefer partial results over none.
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

		// Verified secrets indicate the detector was able to validate the credential
		// (e.g., token format + online validation). We treat those as higher severity.
		severity := "high"
		if result.Verified {
			severity = "critical"
		}

		rawJSON := json.RawMessage(line)
		filePath := result.SourceMetadata.Data.Filesystem.File
		lineNum := result.SourceMetadata.Data.Filesystem.Line

		f := models.Finding{
			ID:        uuid.New(),
			ScanID:    scanID,
			ToolName:  "trufflehog",
			RuleID:    strPtr(result.DetectorName),
			FilePath:  strPtr(filePath),
			LineStart: intPtr(lineNum),
			Message:   "Exposed secret: " + result.DetectorName,
			Severity:  severity,
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
