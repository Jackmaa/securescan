package models

import (
	"time"

	"github.com/google/uuid"
)

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

type BulkFixRequest struct {
	FixIDs []uuid.UUID `json:"fix_ids"`
	Action string      `json:"action"` // "accept" or "reject"
}
