package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Finding struct {
	ID            uuid.UUID        `json:"id"`
	ScanID        uuid.UUID        `json:"scan_id"`
	ToolName      string           `json:"tool_name"`
	RuleID        *string          `json:"rule_id,omitempty"`
	FilePath      *string          `json:"file_path,omitempty"`
	LineStart     *int             `json:"line_start,omitempty"`
	LineEnd       *int             `json:"line_end,omitempty"`
	ColStart      *int             `json:"col_start,omitempty"`
	ColEnd        *int             `json:"col_end,omitempty"`
	Message       string           `json:"message"`
	Severity      string           `json:"severity"`
	OwaspCategory *string          `json:"owasp_category,omitempty"`
	OwaspLabel    *string          `json:"owasp_label,omitempty"`
	CweID         *string          `json:"cwe_id,omitempty"`
	RawOutput     *json.RawMessage `json:"raw_output,omitempty"`
	CodeSnippet   *string          `json:"code_snippet,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
}
