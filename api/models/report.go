package models

import (
	"time" // Report generation time is stored for auditing and retention workflows.

	"github.com/google/uuid" // Report IDs and scan linkage use UUIDs.
)

// Report represents an exported artifact (PDF/HTML/JSON/etc.) derived from a scan.
//
// This model is a placeholder for a future reporting feature:
// - generate a report after a scan completes
// - store it on disk/object storage
// - reference it via a DB record for download and audit trails
type Report struct {
	ID        uuid.UUID `json:"id"`
	ScanID    uuid.UUID `json:"scan_id"`
	Format    string    `json:"format"`
	FilePath  string    `json:"file_path"`
	CreatedAt time.Time `json:"created_at"`
}
