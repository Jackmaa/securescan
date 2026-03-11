package models

import (
	"time" // Timestamps are exposed in API responses and stored in DB.

	"github.com/google/uuid" // IDs are UUIDs for uniqueness across distributed systems.
)

// Project represents a scan target along with its staged local workspace.
//
// It acts as the root resource for the API:
// - A project is created from a source (git/zip).
// - Scans are triggered against a project.
// - The LocalPath is where scanning tools run.
type Project struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	SourceType string    `json:"source_type"`
	SourceURL  string    `json:"source_url,omitempty"`
	LocalPath  string    `json:"local_path"`
	Languages  []string  `json:"languages"`
	Frameworks []string  `json:"frameworks"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateProjectRequest is the payload for POST /projects.
//
// It’s intentionally small:
// - Validation and derived fields (LocalPath, Languages, Frameworks) are computed server-side.
type CreateProjectRequest struct {
	Name       string `json:"name"`
	SourceType string `json:"source_type"`
	SourceURL  string `json:"source_url,omitempty"`
}
