package models

import (
	"time" // Fix creation timestamps are returned for ordering/auditing.

	"github.com/google/uuid" // Fix IDs and relationships use UUIDs.
)

// Fix represents a remediation suggestion for a finding.
//
// A fix can range from “upgrade dependency” to “replace insecure API usage”.
// We store both human-readable explanation and optional code snippets so the UI
// can present a review/approval workflow.
type Fix struct {
	ID           uuid.UUID `json:"id"`
	FindingID    uuid.UUID `json:"finding_id"`
	ScanID       uuid.UUID `json:"scan_id"`
	FixType      string    `json:"fix_type"`
	Description  string    `json:"description"`
	Explanation  *string   `json:"explanation,omitempty"`
	OriginalCode *string   `json:"original_code,omitempty"`
	FixedCode    *string   `json:"fixed_code,omitempty"`
	FilePath     string    `json:"file_path"`
	LineStart    *int      `json:"line_start,omitempty"`
	LineEnd      *int      `json:"line_end,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// BulkFixRequest is the payload for bulk accept/reject operations.
//
// Bulk endpoints exist to reduce API round-trips when the UI applies the same action
// to many fixes at once.
type BulkFixRequest struct {
	FixIDs []uuid.UUID `json:"fix_ids"`
	Action string      `json:"action"` // "accept" or "reject"
}
