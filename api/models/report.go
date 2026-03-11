package models

import (
	"time"

	"github.com/google/uuid"
)

type Report struct {
	ID        uuid.UUID `json:"id"`
	ScanID    uuid.UUID `json:"scan_id"`
	Format    string    `json:"format"`
	FilePath  string    `json:"file_path"`
	CreatedAt time.Time `json:"created_at"`
}
