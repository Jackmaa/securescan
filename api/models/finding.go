package models

import (
	"encoding/json" // Raw tool output is stored as JSON for transparency and debugging.
	"time"          // CreatedAt is stored and returned for ordering/auditing.

	"github.com/google/uuid" // Finding IDs and scan linkage use UUIDs.
)

// Finding is the normalized vulnerability/issue record produced by scanners.
//
// Different tools output very different schemas. We normalize into this shape so:
// - the database schema is stable across tools,
// - the UI can render a consistent list/detail view, and
// - reporting/scoring can work across heterogeneous sources.
//
// Many fields are optional pointers:
// - Some tools don’t provide rule IDs, precise locations, or snippets.
// - Using pointers keeps API responses clean with `omitempty` and preserves “unknown”.
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
